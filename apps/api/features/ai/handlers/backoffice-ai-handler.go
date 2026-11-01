package ai_handlers

import (
	"errors"
	"net/http"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	ai_services "pengi-med-saas/features/ai/services"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BackofficeAIHandler serves the platform-side AI configuration (provider,
// model, reasoning effort and prices per function) and the usage report with tokens and cost, which
// clinics never see.
type BackofficeAIHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeAIHandler(db *gorm.DB, logger *zap.Logger) *BackofficeAIHandler {
	return &BackofficeAIHandler{db: tenantdb.System(db), logger: logger}
}

// GetSettings answers GET /backoffice/ai/settings.
func (h *BackofficeAIHandler) GetSettings(c *gin.Context) envelope.Response {
	return envelope.SuccessResponse(ai_services.Settings(h.db), "backoffice.ai.settings.fetch.success")
}

// UpdateSettings answers PUT /backoffice/ai/settings with the list of
// functions to save. Every entry is validated before any is saved. The API
// key is write-only: an empty one keeps the stored key, clear_api_key
// removes it; it is never logged nor returned.
func (h *BackofficeAIHandler) UpdateSettings(c *gin.Context) envelope.Response {
	var req []ai_services.SettingsUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.ai.settings.invalid", core_errors.ErrAIInvalidRequest)
	}
	for i := range req {
		if err := ai_services.ValidateUpdate(&req[i]); err != nil {
			return settingsError(err)
		}
	}
	for _, s := range req {
		if err := ai_services.SaveSettings(h.db, s); err != nil {
			if resp := settingsError(err); resp.Code != http.StatusInternalServerError {
				return resp
			}
			// The error never carries the key (sealing fails before or without it).
			h.logger.Error("failed to save AI settings", zap.String("function", s.Function), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAIInternal)
		}
	}
	return envelope.SuccessResponse(ai_services.Settings(h.db), "backoffice.ai.settings.update.success")
}

func settingsError(err error) envelope.Response {
	switch {
	case errors.Is(err, ai_services.ErrInvalidBaseURL):
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.ai.settings.invalid_base_url", core_errors.ErrAIInvalidRequest)
	case errors.Is(err, ai_services.ErrEncryptionUnavailable):
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.ai.settings.encryption_key_missing", core_errors.ErrAIEncryptionUnavailable)
	case errors.Is(err, ai_services.ErrUnknownFunction), errors.Is(err, ai_services.ErrInvalidSettings):
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.ai.settings.invalid", core_errors.ErrAIInvalidRequest)
	default:
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAIInternal)
	}
}

// UsageRow is one tenant's uses of one function in the month.
type UsageRow struct {
	TenantID         uint    `json:"tenant_id"`
	TenantName       string  `json:"tenant_name"`
	Function         string  `json:"function"`
	Uses             int64   `json:"uses"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CostUSD          float64 `json:"cost_usd"`
}

// UsageReport is the AI usage of every tenant in one month.
type UsageReport struct {
	Month string     `json:"month"` // YYYY-MM
	Rows  []UsageRow `json:"rows"`
}

// GetUsage answers GET /backoffice/ai/usage?month=YYYY-MM (default: this
// month, clinic time zone).
func (h *BackofficeAIHandler) GetUsage(c *gin.Context) envelope.Response {
	loc := whatsapp_services.ClinicLocation()
	ref := time.Now()
	if m := c.Query("month"); m != "" {
		parsed, err := time.ParseInLocation("2006-01", m, loc)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.ai.usage.invalid_month", core_errors.ErrAIInvalidRequest)
		}
		ref = parsed
	}
	start, end := whatsapp_services.Period(ref, loc)
	rows := []UsageRow{}
	err := h.db.Table("ai_usage_logs AS u").
		Select("u.tenant_id, t.name AS tenant_name, u.function, COUNT(*) AS uses, "+
			"COALESCE(SUM(u.prompt_tokens),0) AS prompt_tokens, COALESCE(SUM(u.completion_tokens),0) AS completion_tokens, "+
			"COALESCE(SUM(u.cost_usd),0) AS cost_usd").
		Joins("LEFT JOIN tenants AS t ON t.id = u.tenant_id").
		Where("u.created_at >= ? AND u.created_at < ?", start.UTC(), end.UTC()).
		Group("u.tenant_id, t.name, u.function").
		Order("cost_usd DESC, uses DESC").
		Scan(&rows).Error
	if err != nil {
		h.logger.Error("failed to compute AI usage", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAIInternal)
	}
	return envelope.SuccessResponse(UsageReport{Month: start.Format("2006-01"), Rows: rows}, "backoffice.ai.usage.fetch.success")
}
