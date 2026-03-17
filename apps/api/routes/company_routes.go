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

	group := router.Group("/companies")
	{
		group.GET("", envelope.Handle(companyHandler.GetCompanies))
	}

	// Team management — requires auth + tenant + active subscription
	teamGroup := router.Group("/companies/team",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	{
		teamGroup.GET("", envelope.Handle(companyHandler.GetTeamMembers))
		teamGroup.GET("/roles", envelope.Handle(companyHandler.GetTeamRoles))
		teamGroup.POST("/invite-link", envelope.Handle(companyHandler.GenerateInviteLink))
	}
}
