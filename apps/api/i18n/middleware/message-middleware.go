package i18n_middleware

import (
	message_cache "pengi-med-saas/i18n/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func I18nMiddleware(db *gorm.DB) gin.HandlerFunc {
	_ = message_cache.Init(db)

	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = c.Query("lang")
		}
		if lang == "" {
			lang = "es"
		}

		c.Set("lang", lang)
		c.Set("translator", func(key string) string {
			return message_cache.Get(lang, key)
		})

		c.Next()
	}
}
