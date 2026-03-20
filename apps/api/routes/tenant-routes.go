package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	tenant_handlers "pengi-med-saas/features/tenants/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTenantRoutes(router *gin.RouterGroup, db *gorm.DB) {
	tenantHandler := tenant_handlers.NewTenantHandler(db, logger.Log)

	tenantGroup := router.Group("/tenants", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))

	tenantGroup.PUT("/sri/signature", envelope.Handle(tenantHandler.UploadSignature))
	tenantGroup.GET("/sri/status", envelope.Handle(tenantHandler.GetSriStatus))
	tenantGroup.PUT("/sri/info", envelope.Handle(tenantHandler.UpdateSriInfo))
	tenantGroup.GET("/settings", envelope.Handle(tenantHandler.GetUISettings))
	tenantGroup.PUT("/settings", envelope.Handle(tenantHandler.UpdateUISettings))
	tenantGroup.POST("/display-token", envelope.Handle(tenantHandler.GenerateDisplayToken))

	// Public — no auth required, validated via display token
	publicGroup := router.Group("/public")
	publicGroup.GET("/appointments/today", envelope.Handle(tenantHandler.GetTodayAppointmentsPublic))
}
