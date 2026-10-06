package main

import (
	"os"
	"strings"
	"time"

	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	core_middleware "pengi-med-saas/core/middleware"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/whatsapp"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_workers "pengi-med-saas/features/clinical/workers"
	"pengi-med-saas/features/health"
	kanban_workers "pengi-med-saas/features/kanban/workers"
	notifications_workers "pengi-med-saas/features/notifications/workers"
	settings_models "pengi-med-saas/features/settings/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
	whatsapp_workers "pengi-med-saas/features/whatsapp/workers"
	"pengi-med-saas/i18n/catalog"
	i18n_messages "pengi-med-saas/i18n/messages"
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

	// Electronic signatures stay disabled (E-SIGN-006) until the key is set.
	if _, err := secretbox.FromEnv(); err != nil {
		logger.Log.Warn("electronic signatures unavailable", zap.Error(err))
	}

	DB_CONNECTION, err := database.Connect()
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}

	err = migrations.RunAllMigrations(DB_CONNECTION)

	if err != nil {
		panic("Failed to run migrations: " + err.Error())
	}

	// Message catalog: the texts embedded in the binary (docs/adr/0003).
	messages, err := catalog.Load(i18n_messages.FS)
	if err != nil {
		panic("Failed to load the message catalog: " + err.Error())
	}
	messages.WithLogger(logger.Log)
	logger.Log.Info("message catalog loaded", zap.Strings("languages", messages.Languages()))

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
	// WhatsApp appointment reminders: one consumer for whatsapp.send.
	whatsappSender := whatsapp_services.NewSender(DB_CONNECTION, logger.Log, whatsapp.NewFromEnv())
	consumers := append(sriConsumers, whatsappSender.StartConsumer)
	go rabbitmq.Run(consumers...)

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

	// Initialize announcement scheduler (scheduled backoffice announcements)
	announcementScheduler := notifications_workers.NewAnnouncementScheduler(DB_CONNECTION, logger.Log)
	go announcementScheduler.Start()
	logger.Log.Info("announcement scheduler started")

	// Initialize WhatsApp reminder scheduler (queues due appointment reminders)
	reminderScheduler := whatsapp_workers.NewReminderScheduler(DB_CONNECTION, logger.Log, whatsapp_services.RabbitPublisher{})
	go reminderScheduler.Start()
	logger.Log.Info("whatsapp reminder scheduler started")

	// gin.Default() minus the query string in the access log (it carries the
	// waiting-room TV token).
	r := gin.New()
	r.Use(core_middleware.AccessLogger(nil), gin.Recovery())

	// c.ClientIP() (per-IP rate limits) trusts X-Forwarded-For only from these
	// peers: Caddy in production, see core_middleware.DefaultTrustedProxies.
	trustedProxies := core_middleware.TrustedProxiesFromEnv()
	if err := core_middleware.ConfigureClientIP(r, trustedProxies); err != nil {
		panic("Invalid TRUSTED_PROXIES: " + err.Error())
	}
	logger.Log.Info("trusted proxies configured",
		zap.Strings("trusted_proxies", trustedProxies),
		zap.Strings("remote_ip_headers", r.RemoteIPHeaders))

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-tenant-Slug", "If-None-Match"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition", "ETag"},
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

	r.Use(i18n_middleware.I18nMiddleware(messages))

	r.GET("/health", envelope.Handle(health.Health))

	routes.RegisterRoutes(r.Group("/api/v1"), DB_CONNECTION, messages)

	r.Run() // listen and serve on 0.0.0.0:8080
}
