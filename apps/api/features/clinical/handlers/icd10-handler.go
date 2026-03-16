package clinical_handlers

import (
	"net/http"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ICD10Handler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewICD10Handler(db *gorm.DB, logger *zap.Logger) *ICD10Handler {
	return &ICD10Handler{db: db, logger: logger}
}

type icd10SearchResult struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

func (h *ICD10Handler) Search(c *gin.Context) envelope.Response {
	q := c.Query("q")
	if q == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "query param 'q' is required", core_errors.ErrAuthInvalidRequest)
	}

	var codes []clinical_models.Cie10Code
	if err := h.db.
		Where("code ILIKE ? OR title ILIKE ?", q+"%", "%"+q+"%").
		Order("code ASC").
		Limit(50).
		Find(&codes).Error; err != nil {
		h.logger.Error("Failed to search ICD-10 codes", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "icd10 search error", core_errors.ErrInternal)
	}

	results := make([]icd10SearchResult, len(codes))
	for i, c := range codes {
		results[i] = icd10SearchResult{Code: c.Code, Title: c.Title}
	}

	return envelope.SuccessResponse(results, "clinical.icd10.search.success")
}
