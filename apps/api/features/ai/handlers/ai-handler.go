package ai_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	ai_models "pengi-med-saas/features/ai/models"
	ai_services "pengi-med-saas/features/ai/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// auditEntitySummary is the audit_logs entity of a summary view or
// generation: the audit trail records that AI was used on the patient, never
// the prompt or the answer.
const auditEntitySummary = "ai_summary"

// AIHandler serves the clinic-side AI functions.
type AIHandler struct {
	db         *gorm.DB
	logger     *zap.Logger
	summarizer *ai_services.Summarizer
}

// NewAIHandler builds the handler; dial nil means the real provider client,
// built per call from the backoffice settings with env fallback.
func NewAIHandler(db *gorm.DB, logger *zap.Logger, dial ai_services.Dialer) *AIHandler {
	return &AIHandler{db: db, logger: logger, summarizer: ai_services.NewSummarizer(dial)}
}

// Status is what the UI needs to show or hide AI actions.
type Status struct {
	Configured bool                `json:"configured"` // the summary has a provider key (backoffice or env)
	Enabled    bool                `json:"enabled"`    // the clinic opted in
	Quotas     []ai_services.Quota `json:"quotas"`
}

// GetStatus answers GET /ai/status.
func (h *AIHandler) GetStatus(c *gin.Context) envelope.Response {
	tenantID := tenantdb.TenantID(c)
	enabled, err := ai_services.Enabled(h.db, tenantID)
	if err != nil {
		h.logger.Error("failed to read AI opt-in", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "ai.status.error", core_errors.ErrAIInternal)
	}
	quota, err := ai_services.MonthlyQuota(h.db, tenantID, ai_models.FunctionSummary, time.Now())
	if err != nil {
		h.logger.Error("failed to compute AI quota", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "ai.status.error", core_errors.ErrAIInternal)
	}
	provider, err := ai_services.ResolveProvider(h.db, ai_models.FunctionSummary)
	if err != nil {
		// A stored key that can't be opened: report not configured.
		h.logger.Warn("AI provider settings unusable", zap.Error(err))
	}
	status := Status{
		Configured: err == nil && provider.Configured(),
		Enabled:    enabled,
		Quotas:     []ai_services.Quota{quota},
	}
	return envelope.SuccessResponse(status, "ai.status.fetch.success")
}

// GetPatientSummary answers GET /ai/patients/:id/summary: the cached summary
// if the patient's data didn't change since it was generated, else null. It
// never calls the model nor counts a use.
func (h *AIHandler) GetPatientSummary(c *gin.Context) envelope.Response {
	patientID, ok := patientIDParam(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	if resp := h.requireEnabled(c); resp != nil {
		return *resp
	}
	db := tenantdb.For(c, h.db)
	summary, err := h.summarizer.Cached(db, patientID)
	if err != nil {
		return h.errorResponse(err)
	}
	if summary != nil {
		audit.RecordAccess(h.db, c, auditEntitySummary, patientID, &patientID)
	}
	return envelope.SuccessResponse(summary, "ai.summary.fetch.success")
}

// GenerateSummary answers POST /ai/patients/:id/summary: the cached summary
// when still valid, else a new one (one use of the monthly quota).
func (h *AIHandler) GenerateSummary(c *gin.Context) envelope.Response {
	patientID, ok := patientIDParam(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(int64)
	db := tenantdb.For(c, h.db)
	summary, err := h.summarizer.Generate(c.Request.Context(), h.db, db, tenantdb.TenantID(c), uint(uid), patientID)
	if err != nil {
		return h.errorResponse(err)
	}
	audit.RecordAccess(h.db, c, auditEntitySummary, patientID, &patientID)
	message := "ai.summary.generate.success"
	if summary.Cached {
		message = "ai.summary.fetch.success"
	}
	return envelope.SuccessResponse(summary, message)
}

func (h *AIHandler) requireEnabled(c *gin.Context) *envelope.Response {
	enabled, err := ai_services.Enabled(h.db, tenantdb.TenantID(c))
	if err != nil {
		h.logger.Error("failed to read AI opt-in", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "ai.summary.error", core_errors.ErrAIInternal)
		return &resp
	}
	if !enabled {
		resp := envelope.ErrorResponse(http.StatusForbidden, "ai.error.not_enabled", core_errors.ErrAINotEnabled)
		return &resp
	}
	return nil
}

func (h *AIHandler) errorResponse(err error) envelope.Response {
	switch {
	case errors.Is(err, ai_services.ErrPatientNotFound):
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	case errors.Is(err, ai_services.ErrNotEnabled):
		return envelope.ErrorResponse(http.StatusForbidden, "ai.error.not_enabled", core_errors.ErrAINotEnabled)
	case errors.Is(err, ai_services.ErrNotConfigured):
		if err != ai_services.ErrNotConfigured { // wrapped: there is a cause worth logging
			h.logger.Warn("AI provider not usable", zap.Error(err)) // e.g. a stored key that can't be opened; never the key
		}
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "ai.error.not_configured", core_errors.ErrAINotConfigured)
	case errors.Is(err, ai_services.ErrQuotaExceeded):
		return envelope.ErrorResponse(http.StatusTooManyRequests, "ai.error.quota_exceeded", core_errors.ErrAIQuotaExceeded)
	case errors.Is(err, ai_services.ErrNoHistory):
		return envelope.ErrorResponse(http.StatusUnprocessableEntity, "ai.error.no_history", core_errors.ErrAINoHistory)
	case errors.Is(err, ai_services.ErrProvider):
		h.logger.Warn("AI provider failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadGateway, "ai.error.provider", core_errors.ErrAIProviderFailed)
	default:
		h.logger.Error("AI summary failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "ai.summary.error", core_errors.ErrAIInternal)
	}
}

func patientIDParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}
