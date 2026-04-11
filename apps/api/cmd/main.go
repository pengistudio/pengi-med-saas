package main

import (
	"os"
	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/logger"
	billing_workers "pengi-med-saas/features/billing/workers"
	kanban_workers "pengi-med-saas/features/kanban/workers"
	"pengi-med-saas/features/health"
	i18n_middleware "pengi-med-saas/i18n/middleware"
	"pengi-med-saas/migrations"
	"pengi-med-saas/routes"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

	// Initialize RabbitMQ
	rabbitConn, rabbitChannel, err := rabbitmq.StartRabbitMQWithChannel()
	if err != nil {
		logger.Log.Warn("RabbitMQ failed to start. Queues will be unavailable.", zap.Error(err))
	} else {
		defer rabbitConn.Close()
		defer rabbitChannel.Close()
		billing_workers.InitInvoiceBroker(rabbitChannel, DB_CONNECTION, logger.Log)
	}

	// Initialize archive scheduler
	archiveScheduler := kanban_workers.NewArchiveScheduler(DB_CONNECTION, logger.Log)
	go archiveScheduler.Start()
	logger.Log.Info("archive scheduler started")

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-tenant-Slug"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(i18n_middleware.I18nMiddleware(DB_CONNECTION))

	// Inject RabbitMQ channel into context if available
	r.Use(func(c *gin.Context) {
		if rabbitChannel != nil {
			c.Set("invoice_channel", rabbitChannel)
		}
		c.Next()
	})

	r.GET("/health", health.Health)

	routes.RegisterRoutes(r.Group("/api/v1"), DB_CONNECTION)

	r.Run() // listen and serve on 0.0.0.0:8080
}
