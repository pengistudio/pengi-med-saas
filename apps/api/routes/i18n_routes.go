package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/i18n/catalog"
	i18n_handlers "pengi-med-saas/i18n/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterI18nRoutes(router *gin.RouterGroup, messages *catalog.Catalog) {
	i18nHandler := i18n_handlers.NewMessageHandler(messages)

	group := router.Group("/i18n")
	{
		group.GET("/messages", envelope.Handle(i18nHandler.GetAllMessages))
	}
}
