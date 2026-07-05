package audit_handlers

import (
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuditLogHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAuditLogHandler(db *gorm.DB, logger *zap.Logger) *AuditLogHandler {
	return &AuditLogHandler{db: db, logger: logger}
}

// GetAuditLogs returns the tenant-scoped compliance audit trail, filterable by
// patient_id, entity_type, entity_id, action, and a from/to date range.
func (h *AuditLogHandler) GetAuditLogs(c *gin.Context) envelope.Response {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&audit.AuditLog{})

	if patientID := c.Query("patient_id"); patientID != "" {
		id, err := strconv.ParseUint(patientID, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("patient_id = ?", uint(id))
	}
	if entityType := c.Query("entity_type"); entityType != "" {
		baseQuery = baseQuery.Where("entity_type = ?", entityType)
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		id, err := strconv.ParseUint(entityID, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("entity_id = ?", uint(id))
	}
	if action := c.Query("action"); action != "" {
		baseQuery = baseQuery.Where("action = ?", action)
	}
	if from := c.Query("from"); from != "" {
		fromDate, err := time.Parse("2006-01-02", from)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("created_at >= ?", fromDate)
	}
	if to := c.Query("to"); to != "" {
		toDate, err := time.Parse("2006-01-02", to)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("created_at <= ?", toDate.Add(24*time.Hour))
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count audit logs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "audit.log.error.fetch_failed", core_errors.ErrAuditFetchError)
	}

	var logs []audit.AuditLog
	if err := baseQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		h.logger.Error("Failed to fetch audit logs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "audit.log.error.fetch_failed", core_errors.ErrAuditFetchError)
	}

	return envelope.PagedSuccessResponse(logs, int(total), page, limit, "audit.log.list.success")
}
