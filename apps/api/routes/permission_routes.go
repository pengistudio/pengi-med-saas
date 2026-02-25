package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	permission_handlers "pengi-med-saas/features/permissions/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterPermissionRoutes(router *gin.RouterGroup, db *gorm.DB) {
	permissionHandler := permission_handlers.NewPermissionHandler(db, logger.Log)

	group := router.Group("/permissions")
	{
		group.GET("", envelope.Handle(permissionHandler.GetAllPermissions))
		group.GET("/categories", envelope.Handle(permissionHandler.GetAllPermissionWithCategory))
	}
}
