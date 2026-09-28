package envelope

import (
	core_errors "pengi-med-saas/core/errors"

	"github.com/gin-gonic/gin"
)

type Action func(r *gin.Context) Response

// Handle runs action and serializes its Response. When the request carries a
// "translator" (set by the i18n middleware, backed by the message catalog) it
// translates Message, and the ErrorMessage of an AppError from its code; an
// untranslated code keeps the English default given to NewAppError.
func Handle(action Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := action(c)

		if val, exists := c.Get("translator"); exists {
			if translate, ok := val.(func(string) string); ok {
				response.Message = translate(response.Message)
				if appErr, ok := response.Data.(core_errors.AppError); ok {
					if msg := translate(appErr.ErrorCode); msg != appErr.ErrorCode {
						appErr.ErrorMessage = msg
					}
					response.Data = appErr
				}
			}
		}

		// gin writes no body for statuses that don't allow one (304, 204).
		c.JSON(response.Code, response)
	}
}
