package routes

import (
	"pengi-med-saas/core/envelope"
	i18n_handlers "pengi-med-saas/i18n/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterI18nRoutes(router *gin.RouterGroup, db *gorm.DB) {
	i18nHandler := i18n_handlers.NewMessageHandler(db)

	group := router.Group("/i18n")
	{
		group.GET("/messages", envelope.Handle(i18nHandler.GetAllMessages))
		group.GET("/version", envelope.Handle(i18nHandler.GetMessageVersion))
		group.POST("/reload", envelope.Handle(i18nHandler.ReloadMessages))
	}
}
