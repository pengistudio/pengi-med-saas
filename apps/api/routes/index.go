package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers all the routes for /api/**/*
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	RegisterI18nRoutes(router, db)
	RegisterCompanyRoutes(router, db)
	RegisterUserRoutes(router, db)
	RegisterClinicalRoutes(router, db)
	RegisterPermissionRoutes(router, db)
}
