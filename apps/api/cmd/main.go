package main

import (
	"os"
	"strings"
	"time"

	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/logger"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_workers "pengi-med-saas/features/clinical/workers"
	"pengi-med-saas/features/health"
	kanban_workers "pengi-med-saas/features/kanban/workers"
	settings_models "pengi-med-saas/features/settings/models"
	message_cache "pengi-med-saas/i18n/cache"
	i18n_middleware "pengi-med-saas/i18n/middleware"
	"pengi-med-saas/migrations"
	"pengi-med-saas/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var DB_CONNECTION *gorm.DB

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		mode = "production"
	} else {
		mode = "development"
	}

	logger.Init(mode)
	logger.Info("Starting application...", zap.String("env", mode))

	DB_CONNECTION, err := database.Connect()
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}

	err = migrations.RunAllMigrations(DB_CONNECTION)

	if err != nil {
		panic("Failed to run migrations: " + err.Error())
	}

	// Initialize message cache
	if err := message_cache.Init(DB_CONNECTION); err != nil {
		logger.Log.Warn("Failed to initialize message cache", zap.Error(err))
	}
	logger.Log.Info("message cache initialized")

	// Initialize RabbitMQ (reconnects on its own). HTTP handlers publish on a shared
	// channel (rabbitmq.PublishChannel); each background consumer gets its own
	// dedicated channel — amqp.Channel is not safe for concurrent use, and declaring
	// the next queue on the same channel while a previous StartConsumer goroutine is
	// still finishing its Consume() handshake races and closes the channel with a 503
	// "unexpected command received".
	sriDocuments := sri_document.NewDefault(DB_CONNECTION, logger.Log)
	var sriConsumers []func(ch *amqp.Channel) error
	for _, kind := range sri_document.Kinds {
		sriConsumers = append(sriConsumers, func(ch *amqp.Channel) error { return sriDocuments.StartConsumer(ch, kind) })
	}
	go rabbitmq.Run(sriConsumers...)

	go sriDocuments.RunSweeper(5*time.Minute, sri_document.Kinds...)
	logger.Log.Info("SRI document sweeper started")

	// Initialize archive scheduler
	archiveScheduler := kanban_workers.NewArchiveScheduler(DB_CONNECTION, logger.Log)
	go archiveScheduler.Start()
	logger.Log.Info("archive scheduler started")

	// Initialize stale draft scheduler
	staleDraftScheduler := clinical_workers.NewStaleDraftScheduler(DB_CONNECTION, logger.Log)
	go staleDraftScheduler.Start()
	logger.Log.Info("stale draft scheduler started")

	r := gin.Default()

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-tenant-Slug"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if mode == "production" {
		var corsSetting settings_models.SystemSetting
		allowedOrigins := []string{}
		if err := DB_CONNECTION.Where("key = ?", "allowed_origins").First(&corsSetting).Error; err == nil {
			for _, o := range strings.Split(corsSetting.Value, ",") {
				if trimmed := strings.TrimSpace(o); trimmed != "" {
					allowedOrigins = append(allowedOrigins, trimmed)
				}
			}
		}
		if len(allowedOrigins) == 0 {
			logger.Log.Warn("no allowed_origins configured — CORS will deny all cross-origin requests")
		}
		corsConfig.AllowOrigins = allowedOrigins
	} else {
		corsConfig.AllowOriginFunc = func(origin string) bool { return true }
	}

	r.Use(cors.New(corsConfig))

	r.Use(i18n_middleware.I18nMiddleware(DB_CONNECTION))

	r.GET("/health", health.Health)

	routes.RegisterRoutes(r.Group("/api/v1"), DB_CONNECTION)

	r.Run() // listen and serve on 0.0.0.0:8080
}
