package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	company_handlers "pengi-med-saas/features/companies/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterCompanyRoutes(router *gin.RouterGroup, db *gorm.DB) {
	companyHandler := company_handlers.NewCompanyHandler(db, logger.Log)
	companyPaymentHandler := company_handlers.NewCompanyPaymentHandler(db, logger.Log)
	dashboardHandler := company_handlers.NewDashboardHandler(db, logger.Log)

	group := router.Group("/companies")
	{
		group.GET("", envelope.Handle(companyHandler.GetCompanies))
	}

	// Auth + tenant only (no subscription middleware — accessible even with expired subscription)
	authedGroup := router.Group("/companies",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
	)
	{
		authedGroup.GET("/dashboard/stats", envelope.Handle(dashboardHandler.GetDashboardStats))
		authedGroup.GET("/subscriptions/plans", envelope.Handle(companyPaymentHandler.GetAvailablePlans))
		authedGroup.POST("/subscriptions/pay", envelope.Handle(companyPaymentHandler.PaySubscription))
		authedGroup.POST("/subscriptions/confirm-payment", envelope.Handle(companyPaymentHandler.ConfirmPayment))
		authedGroup.GET("/subscriptions/me", envelope.Handle(companyPaymentHandler.GetMySubscription))
		authedGroup.GET("/subscriptions/payments", envelope.Handle(companyPaymentHandler.GetSubscriptionPayments))
		authedGroup.DELETE("/subscriptions/plan-change", envelope.Handle(companyPaymentHandler.CancelPlanChange))
	}

	// Auth only — user creates a brand-new, independent company. Must NOT
	// require X-Tenant-Slug/tenant middleware: the user isn't inside a
	// tenant yet, they're creating one.
	selfServiceGroup := router.Group("/companies",
		auth_middleware.AuthMiddleware(),
	)
	{
		selfServiceGroup.POST("", envelope.Handle(companyHandler.CreateAdditionalCompany))
	}

	// Team management — requires auth + tenant + active subscription
	teamGroup := router.Group("/companies/team",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	{
		// Team management is basic account administration, not a paid feature
		// tier — RequireRolePermission checks the caller's role only, skipping
		// the plan/feature gate that RequirePermission would otherwise apply.
		rp := subscription_middleware.RequireRolePermission
		teamGroup.GET("", rp(db, "READ_TEAM"), envelope.Handle(companyHandler.GetTeamMembers))
		teamGroup.GET("/roles", rp(db, "READ_TEAM"), envelope.Handle(companyHandler.GetTeamRoles))
		teamGroup.POST("/invite-link", rp(db, "MANAGE_TEAM_MEMBERS"), envelope.Handle(companyHandler.GenerateInviteLink))
		teamGroup.PUT("/:environment_id/role", rp(db, "MANAGE_TEAM_MEMBERS"), envelope.Handle(companyHandler.UpdateTeamMemberRole))
	}
}
