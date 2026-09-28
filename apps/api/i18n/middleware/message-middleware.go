package i18n_middleware

import (
	"pengi-med-saas/i18n/catalog"

	"github.com/gin-gonic/gin"
)

// I18nMiddleware resolves the request language (?lang=, then Accept-Language,
// then es) and sets "lang" and the "translator" that envelope.Handle uses.
func I18nMiddleware(messages *catalog.Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := messages.ResolveLanguage(c.Query("lang"), c.GetHeader("Accept-Language"))

		c.Set("lang", lang)
		c.Set("translator", func(key string) string {
			return messages.Translate(lang, key)
		})

		c.Next()
	}
}
