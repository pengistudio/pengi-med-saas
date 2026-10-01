package backoffice_middleware

import (
	"net/http"
	"strings"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BackofficeAuthMiddleware lets through only backoffice access tokens whose
// user still exists in backoffice_users. Clinic app tokens, refresh tokens and
// tokens of deleted admins get 401.
func BackofficeAuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	db = tenantdb.System(db)
	return func(c *gin.Context) {
		unauthorized := func() {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidRequest))
		}

		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			unauthorized()
			return
		}
		claims, err := auth.ParseBackofficeAccessToken(token)
		if err != nil {
			unauthorized()
			return
		}
		var count int64
		if err := db.Model(&backoffice_models.BackofficeUser{}).
			Where("id = ? AND user_name = ?", claims.UserID, claims.Username).
			Count(&count).Error; err != nil || count == 0 {
			unauthorized()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("auth_token", token)
		c.Set("authenticated", true)
		c.Next()
	}
}
