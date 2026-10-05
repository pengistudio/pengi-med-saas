package core_middleware

import (
	"io"

	"github.com/gin-gonic/gin"
)

// AccessLogger is gin's default access log (same format as gin.Default) without
// the query string: credentials travel there, e.g. the waiting-room TV's
// ?token= on /public/appointments/today. out nil means gin.DefaultWriter.
func AccessLogger(out io.Writer) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{Output: out, SkipQueryString: true})
}
