package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/whatsapp"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"
	whatsapp_handlers "pengi-med-saas/features/whatsapp/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterWhatsAppRoutes exposes the tenant's WhatsApp connection
// (MANAGE_WHATSAPP), the conversations inbox (USE_WHATSAPP_INBOX) and Meta's
// public webhook, which authenticates requests by their HMAC signature
// instead of a session.
func RegisterWhatsAppRoutes(router *gin.RouterGroup, db *gorm.DB) {
	client := whatsapp.NewFromEnv()
	h := whatsapp_handlers.NewWhatsAppHandler(db, logger.Log, client)

	group := router.Group("/whatsapp",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	registerWhatsAppTenantRoutes(group, db, h)

	webhook := whatsapp_handlers.NewWebhookHandler(db, logger.Log, clinical_handlers.NewAppointmentHandler(db, logger.Log)).WithClient(client)
	router.GET("/webhooks/whatsapp", webhook.Verify)
	router.POST("/webhooks/whatsapp", envelope.Handle(webhook.Receive))
}

// registerWhatsAppTenantRoutes mounts the session routes on group, which must
// already carry auth, tenant and subscription middleware.
func registerWhatsAppTenantRoutes(group *gin.RouterGroup, db *gorm.DB, h *whatsapp_handlers.WhatsAppHandler) {
	rp := subscription_middleware.RequirePermission

	group.GET("/config", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.GetConfig))
	group.GET("/account", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.GetAccount))
	group.DELETE("/account", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.Disconnect))
	group.POST("/connect/embedded", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.ConnectEmbedded))
	group.POST("/connect/manual", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.ConnectManual))
	group.POST("/template/sync", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.SyncTemplate))
	group.PUT("/settings", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.UpdateSettings))
	group.POST("/test", rp(db, "MANAGE_WHATSAPP"), envelope.Handle(h.SendTest))
	group.GET("/usage", subscription_middleware.RequireAnyPermission(db, "MANAGE_WHATSAPP", "USE_WHATSAPP_INBOX"), envelope.Handle(h.GetUsage))
	// The sent-messages log is the inbox's "Envíos" tab, and still readable
	// from the settings card.
	group.GET("/messages", subscription_middleware.RequireAnyPermission(db, "MANAGE_WHATSAPP", "USE_WHATSAPP_INBOX"), envelope.Handle(h.ListMessages))

	inbox := rp(db, "USE_WHATSAPP_INBOX")
	group.GET("/templates", inbox, envelope.Handle(h.ListTemplates))
	group.GET("/conversations", inbox, envelope.Handle(h.ListConversations))
	group.POST("/conversations/start", inbox, envelope.Handle(h.StartConversation))
	group.GET("/conversations/unread-count", inbox, envelope.Handle(h.UnreadCount))
	group.GET("/conversations/:id", inbox, envelope.Handle(h.GetConversation))
	group.GET("/conversations/:id/messages", inbox, envelope.Handle(h.ListConversationMessages))
	group.POST("/conversations/:id/messages", inbox, envelope.Handle(h.SendReply))
	group.POST("/conversations/:id/template", inbox, envelope.Handle(h.SendTemplate))
	group.POST("/conversations/:id/read", inbox, envelope.Handle(h.MarkConversationRead))
	group.PUT("/conversations/:id/patient", inbox, envelope.Handle(h.LinkPatient))
}
