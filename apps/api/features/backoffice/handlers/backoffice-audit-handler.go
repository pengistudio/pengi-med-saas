package backoffice_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// auditLogPageSize is how many audit entries one request returns.
const auditLogPageSize = 100

type BackofficeAuditHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeAuditHandler(db *gorm.DB, logger *zap.Logger) *BackofficeAuditHandler {
	return &BackofficeAuditHandler{db: tenantdb.System(db), logger: logger}
}

// GetAuditLogs: GET /backoffice/audit-logs. The backoffice audit trail, newest
// first, auditLogPageSize entries at a time. ?before_id= continues from the
// last entry of the previous page; ?user_id= keeps one admin's entries.
func (h *BackofficeAuditHandler) GetAuditLogs(c *gin.Context) envelope.Response {
	invalid := envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)

	query := h.db.Order("id DESC").Limit(auditLogPageSize + 1)
	if raw := c.Query("before_id"); raw != "" {
		beforeID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return invalid
		}
		query = query.Where("id < ?", beforeID)
	}
	if raw := c.Query("user_id"); raw != "" {
		userID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return invalid
		}
		query = query.Where("backoffice_user_id = ?", userID)
	}

	logs := []backoffice_models.BackofficeAuditLog{}
	if err := query.Find(&logs).Error; err != nil {
		h.logger.Error("Failed to fetch the backoffice audit log", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	hasMore := len(logs) > auditLogPageSize
	if hasMore {
		logs = logs[:auditLogPageSize]
	}
	return envelope.SuccessResponse(gin.H{"items": logs, "has_more": hasMore}, "backoffice.audit.list.success")
}
