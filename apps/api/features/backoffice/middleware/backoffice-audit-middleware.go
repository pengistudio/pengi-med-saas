package backoffice_middleware

import (
	"net/http"

	"pengi-med-saas/core/tenantdb"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const auditActorKey = "backoffice_audit_actor"

type auditActor struct {
	id       uint
	userName string
}

// SetAuditActor names the admin behind a request that carries no access
// token, so BackofficeAuditMiddleware can record it: the sign-in that just
// succeeded, the sign-out identified by its refresh cookie.
func SetAuditActor(c *gin.Context, id uint, userName string) {
	c.Set(auditActorKey, auditActor{id: id, userName: userName})
}

func actorOf(c *gin.Context) (auditActor, bool) {
	if v, ok := c.Get(auditActorKey); ok {
		if actor, ok := v.(auditActor); ok {
			return actor, true
		}
	}
	if !c.GetBool("authenticated") {
		return auditActor{}, false
	}
	return auditActor{id: uint(c.GetInt64("user_id")), userName: c.GetString("username")}, true
}

func cut(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// BackofficeAuditMiddleware writes the backoffice audit trail
// (backoffice_models.BackofficeAuditLog). It goes on the group that holds
// every backoffice route, before BackofficeAuthMiddleware: it runs after the
// handler and records the request when an admin is known (authenticated, or
// named with SetAuditActor) and the method is not a read. Requests rejected
// by the handler are recorded too, with their status. A failed write is
// logged and never changes the response.
func BackofficeAuditMiddleware(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	db = tenantdb.System(db)
	return func(c *gin.Context) {
		c.Next()

		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return
		}
		actor, ok := actorOf(c)
		if !ok {
			return
		}
		entry := backoffice_models.BackofficeAuditLog{
			BackofficeUserID: actor.id,
			UserName:         cut(actor.userName, 255),
			Method:           c.Request.Method,
			Route:            cut(c.FullPath(), 255),
			Path:             cut(c.Request.URL.Path, 255),
			Status:           c.Writer.Status(),
			IP:               cut(c.ClientIP(), 64),
			UserAgent:        cut(c.Request.UserAgent(), 255),
		}
		if err := db.Create(&entry).Error; err != nil {
			logger.Error("Failed to write the backoffice audit log",
				zap.Uint("backoffice_user_id", actor.id), zap.String("method", entry.Method), zap.String("path", entry.Path), zap.Error(err))
		}
	}
}
