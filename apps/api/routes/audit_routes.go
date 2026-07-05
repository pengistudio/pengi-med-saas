package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	audit_handlers "pengi-med-saas/features/audit/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuditRoutes(router *gin.RouterGroup, db *gorm.DB) {
	auditHandler := audit_handlers.NewAuditLogHandler(db, logger.Log)

	auditGroup := router.Group("/audit", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db), subscription_middleware.SubscriptionMiddleware(db))
	{
		rp := subscription_middleware.RequirePermission
		auditGroup.GET("/logs", rp(db, "READ_AUDIT_LOG"), envelope.Handle(auditHandler.GetAuditLogs))
	}
}
