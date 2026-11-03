package ai_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/ai"
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
const (
	auditEntitySummary = "ai_summary"
	auditEntityScribe  = "ai_scribe"
)

// AIHandler serves the clinic-side AI functions.
type AIHandler struct {
	db         *gorm.DB
	logger     *zap.Logger
	summarizer *ai_services.Summarizer
	scribe     *ai_services.Scribe
}

// NewAIHandler builds the handler; dial nil means the real provider client,
// built per call from the backoffice settings with env fallback, and
// resolvers nil the real diagnosis catalogs.
func NewAIHandler(db *gorm.DB, logger *zap.Logger, dial ai_services.Dialer, resolvers ...ai_services.ResolverFactory) *AIHandler {
	var r ai_services.ResolverFactory
	if len(resolvers) > 0 {
		r = resolvers[0]
	}
	return &AIHandler{db: db, logger: logger, summarizer: ai_services.NewSummarizer(dial), scribe: ai_services.NewScribe(dial, r)}
}

// Status is what the UI needs to show or hide AI actions.
type Status struct {
	Configured bool                `json:"configured"` // the summary has a provider key (backoffice or env)
	Enabled    bool                `json:"enabled"`    // the clinic opted in
	Quotas     []ai_services.Quota `json:"quotas"`
	// Functions: whether each function has a provider key (backoffice or env).
	Functions map[string]bool `json:"functions"`
}

// GetStatus answers GET /ai/status.
func (h *AIHandler) GetStatus(c *gin.Context) envelope.Response {
	tenantID := tenantdb.TenantID(c)
	enabled, err := ai_services.Enabled(h.db, tenantID)
	if err != nil {
		h.logger.Error("failed to read AI opt-in", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "ai.status.error", core_errors.ErrAIInternal)
	}
	status := Status{Enabled: enabled, Quotas: []ai_services.Quota{}, Functions: map[string]bool{}}
	for _, fn := range []string{ai_models.FunctionSummary, ai_models.FunctionScribe} {
		quota, err := ai_services.MonthlyQuota(h.db, tenantID, fn, time.Now())
		if err != nil {
			h.logger.Error("failed to compute AI quota", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "ai.status.error", core_errors.ErrAIInternal)
		}
		status.Quotas = append(status.Quotas, quota)
		provider, err := ai_services.ResolveProvider(h.db, fn)
		if err != nil {
			// A stored key that can't be opened: report not configured.
			h.logger.Warn("AI provider settings unusable", zap.String("function", fn), zap.Error(err))
		}
		status.Functions[fn] = err == nil && provider.Configured()
	}
	status.Configured = status.Functions[ai_models.FunctionSummary]
	return envelope.SuccessResponse(status, "ai.status.fetch.success")
}

// Scribe answers POST /ai/scribe: the doctor's quick notes ordered into a
// SOAP draft plus diagnosis suggestions resolved against the clinic's
// catalog. One "scribe" use once the provider answers. The notes are never
// stored, logged or audited.
func (h *AIHandler) Scribe(c *gin.Context) envelope.Response {
	var req ai_services.ScribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "ai.scribe.error.invalid_notes", core_errors.ErrAIInvalidNotes)
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(int64)
	draft, err := h.scribe.Organize(c.Request.Context(), h.db, tenantdb.For(c, h.db), tenantdb.TenantID(c), uint(uid), req)
	if err != nil {
		if errors.Is(err, ai_services.ErrInvalidNotes) {
			return envelope.ErrorResponse(http.StatusBadRequest, "ai.scribe.error.invalid_notes", core_errors.ErrAIInvalidNotes)
		}
		return h.errorResponse(err)
	}
	patientID := req.PatientID
	audit.RecordAccess(h.db, c, auditEntityScribe, patientID, &patientID)
	return envelope.SuccessResponse(draft, "ai.scribe.success")
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
		h.logger.Warn("AI provider failed", zap.String("cause", providerCause(err)))
		return envelope.ErrorResponse(http.StatusBadGateway, "ai.error.provider", core_errors.ErrAIProviderFailed)
	default:
		h.logger.Error("AI summary failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "ai.summary.error", core_errors.ErrAIInternal)
	}
}

// providerCause is what is safe to log about a provider failure: the status
// code only, never the upstream body (a provider may echo the prompt back).
func providerCause(err error) string {
	var apiErr *ai.APIError
	if errors.As(err, &apiErr) {
		return "provider answered " + strconv.Itoa(apiErr.Status)
	}
	return "provider call failed"
}

func patientIDParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}
