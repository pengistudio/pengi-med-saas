package routes

import (
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	reports_handlers "pengi-med-saas/features/reports/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterReportsRoutes mounts the reports: plan + role, Feature REPORTS. Each
// answers JSON, or a CSV / PDF file with ?format=csv|pdf, so the handlers write
// their own responses (errors still go through the envelope).
func RegisterReportsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	h := reports_handlers.NewReportsHandler(db, logger.Log, documentRenderer())

	group := router.Group("/reports",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	rp := subscription_middleware.RequirePermission

	group.GET("/revenue", rp(db, "READ_REPORTS"), h.Revenue)
	group.GET("/receivables", rp(db, "READ_REPORTS"), h.Receivables)
	group.GET("/appointments", rp(db, "READ_REPORTS"), h.Appointments)
	group.GET("/clinical", rp(db, "READ_REPORTS"), h.Clinical)
}
