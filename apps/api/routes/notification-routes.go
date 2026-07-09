package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	notifications_handlers "pengi-med-saas/features/notifications/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterNotificationRoutes registers a user's personal notification inbox.
// Only AuthMiddleware + TenantMiddleware are applied here (no RBAC permission,
// no SubscriptionMiddleware) since this always scopes to the requesting
// user's own notifications, and a notification (e.g. a subscription warning)
// shouldn't be gated behind the very thing it might need to warn about.
func RegisterNotificationRoutes(router *gin.RouterGroup, db *gorm.DB) {
	notificationHandler := notifications_handlers.NewNotificationHandler(db, logger.Log)

	group := router.Group("/notifications", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))
	{
		group.GET("", envelope.Handle(notificationHandler.ListNotifications))
		group.PATCH("/:id/read", envelope.Handle(notificationHandler.MarkAsRead))
		group.POST("/mark-all-read", envelope.Handle(notificationHandler.MarkAllAsRead))
	}
}
