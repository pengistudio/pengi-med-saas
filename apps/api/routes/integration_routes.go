package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	integration_handlers "pengi-med-saas/features/integrations/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterIntegrationRoutes(router *gin.RouterGroup, db *gorm.DB) {
	googleHandler := integration_handlers.NewGoogleIntegrationHandler(db, logger.Log)

	// Public callback — no auth, Google redirects here
	router.GET("/integrations/google/callback", googleHandler.Callback)

	// Authenticated integration routes
	integrationGroup := router.Group("/integrations", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))
	{
		integrationGroup.GET("/google/auth-url", envelope.Handle(googleHandler.GetAuthURL))
		integrationGroup.GET("/google/status", envelope.Handle(googleHandler.GetStatus))
		integrationGroup.DELETE("/google/disconnect", envelope.Handle(googleHandler.Disconnect))
	}
}
