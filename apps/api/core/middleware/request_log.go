package core_middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UnmatchedRoute is the route logged (and used as a metrics label) for
// requests no route matched, so scanners don't create one value per path.
const UnmatchedRoute = "unmatched"

// requestLogSkip are probes polled too often to be worth a line each.
var requestLogSkip = map[string]bool{"/health": true, "/metrics": true}

// RequestLogger logs every request with its latency through Zap: method,
// route template (c.FullPath(), never the raw path or query string, which
// carry IDs and tokens), status, duration and the X-Tenant-Slug header when
// present. Requests slower than slow are logged at Warn; slow <= 0 disables
// the flag. Register it before gin.Recovery so a panic is logged as its 500.
func RequestLogger(log *zap.Logger, slow time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if requestLogSkip[c.Request.URL.Path] {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		route := c.FullPath()
		if route == "" {
			route = UnmatchedRoute
		}
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("route", route),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
		}
		if tenant := c.GetHeader("X-Tenant-Slug"); tenant != "" {
			fields = append(fields, zap.String("tenant", tenant))
		}
		if slow > 0 && duration > slow {
			log.Warn("slow request", fields...)
			return
		}
		log.Info("request", fields...)
	}
}
