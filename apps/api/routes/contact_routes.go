package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	contact_handlers "pengi-med-saas/features/contact/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterContactRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := contact_handlers.NewContactHandler(logger.Log)
	router.POST("/contact", envelope.Handle(handler.SendContact))
}
