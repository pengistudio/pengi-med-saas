package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	backoffice_handlers "pengi-med-saas/features/backoffice/handlers"
	"pengi-med-saas/i18n/catalog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers all the routes for /api/**/*
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, messages *catalog.Catalog) {
	RegisterI18nRoutes(router, messages)
	RegisterCompanyRoutes(router, db)
	RegisterUserRoutes(router, db)
	RegisterClinicalRoutes(router, db)
	RegisterBackofficeRoutes(router, db)
	RegisterBillingRoutes(router, db)
	RegisterTenantRoutes(router, db)
	RegisterIntegrationRoutes(router, db)
	RegisterKanbanRoutes(router, db)
	RegisterContactRoutes(router, db)
	RegisterAuditRoutes(router, db)
	RegisterNotificationRoutes(router, db)
	RegisterSignatureRoutes(router, db)
	RegisterDocumentTemplateRoutes(router, db)

	webhookHandler := backoffice_handlers.NewBackofficePaymentHandler(db, logger.Log)
	router.POST("/webhooks/dlocal", envelope.Handle(webhookHandler.HandleDlocalWebhook))

	// WhatsApp: tenant routes + public Meta webhook (/webhooks/whatsapp).
	RegisterWhatsAppRoutes(router, db)
}
