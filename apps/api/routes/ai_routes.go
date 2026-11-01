package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	ai_handlers "pengi-med-saas/features/ai/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterAIRoutes exposes the clinic-side AI functions. Gating: plan + role
// (USE_AI_* in the AI Feature), plus the clinic's opt-in checked by the
// handler. The provider is resolved per call (backoffice settings, else
// AI_* env); without a key the routes answer E-AI-001.
func RegisterAIRoutes(router *gin.RouterGroup, db *gorm.DB) {
	h := ai_handlers.NewAIHandler(db, logger.Log, nil)
	group := router.Group("/ai",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	registerAITenantRoutes(group, db, h)
}

// registerAITenantRoutes mounts the session routes on group, which must
// already carry auth, tenant and subscription middleware.
func registerAITenantRoutes(group *gin.RouterGroup, db *gorm.DB, h *ai_handlers.AIHandler) {
	rp := subscription_middleware.RequirePermission

	group.GET("/status", subscription_middleware.RequireAnyPermission(db, "USE_AI_SUMMARY", "USE_AI_SCRIBE"), envelope.Handle(h.GetStatus))

	// The summary reads the whole history: it needs READ_MEDICAL_RECORD too.
	summary := []gin.HandlerFunc{rp(db, "USE_AI_SUMMARY"), rp(db, "READ_MEDICAL_RECORD")}
	group.GET("/patients/:id/summary", append(summary, envelope.Handle(h.GetPatientSummary))...)
	group.POST("/patients/:id/summary", append(summary, envelope.Handle(h.GenerateSummary))...)
}
