package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/whatsapp"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"
	waitlist_handlers "pengi-med-saas/features/waitlist/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterWaitlistRoutes mounts the appointment waitlist under
// /agenda/waitlist. Gating: plan + role, reusing the appointment permissions
// (no new permission nor plan Feature): READ_APPOINTMENT lists entries and a
// freed slot's candidates; MANAGE_APPOINTMENT adds, edits, removes and
// offers. Offering by WhatsApp also needs the plan's WHATSAPP feature, a
// connected account and the slot offer template approved (checked in the
// handler). Patients join from the portal (/portal/waitlist).
func RegisterWaitlistRoutes(router *gin.RouterGroup, db *gorm.DB) {
	group := router.Group("/agenda/waitlist",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	registerWaitlistTenantRoutes(group, db, waitlist_handlers.NewWaitlistHandler(db, logger.Log, whatsapp.NewFromEnv()))
}

// registerWaitlistTenantRoutes is split out so tests can mount the routes
// behind their own tenant/subscription context.
func registerWaitlistTenantRoutes(group *gin.RouterGroup, db *gorm.DB, h *waitlist_handlers.WaitlistHandler) {
	read := subscription_middleware.RequirePermission(db, "READ_APPOINTMENT")
	manage := subscription_middleware.RequirePermission(db, "MANAGE_APPOINTMENT")

	group.GET("", read, envelope.Handle(h.List))
	group.GET("/candidates", read, envelope.Handle(h.Candidates))
	group.POST("", manage, envelope.Handle(h.Create))
	group.PUT("/:id", manage, envelope.Handle(h.Update))
	group.DELETE("/:id", manage, envelope.Handle(h.Remove))
	group.POST("/:id/offer", manage, envelope.Handle(h.Offer))
}
