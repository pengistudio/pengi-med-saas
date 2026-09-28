package health

import (
	"pengi-med-saas/core/envelope"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) envelope.Response {
	return envelope.SuccessResponse(nil, "health.ok")
}
