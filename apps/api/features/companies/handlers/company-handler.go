package company_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CompanyHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCompanyHandler(db *gorm.DB, logger *zap.Logger) *CompanyHandler {
	return &CompanyHandler{
		db:     db,
		logger: logger,
	}
}

func (h *CompanyHandler) GetCompanies(c *gin.Context) envelope.Response {
	companies := []company_models.Company{}
	if err := h.db.Find(&companies).Error; err != nil {
		h.logger.Error("Failed to fetch companies", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining companies", core_errors.ErrCompanyNotFound)
	}

	h.logger.Info("Companies fetched successfully", zap.Int("count", len(companies)))
	return envelope.SuccessResponse(companies, "company.list.success")
}
