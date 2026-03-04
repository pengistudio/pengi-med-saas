package backoffice_handlers

import (
	"fmt"
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeCompanyHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeCompanyHandler(db *gorm.DB, logger *zap.Logger) *BackofficeCompanyHandler {
	return &BackofficeCompanyHandler{
		db:     db,
		logger: logger,
	}
}

// ── Request / Response DTOs ─────────────────────────────────────────────────

type CreateCompanyRequest struct {
	LegalName string `json:"legal_name" binding:"required"`
	TradeName string `json:"trade_name" binding:"required"`
	PlanCode  string `json:"plan_code"  binding:"required"`
}

type UpdateCompanyRequest struct {
	LegalName string `json:"legal_name"`
	TradeName string `json:"trade_name"`
	PlanCode  string `json:"plan_code"`
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	// remove non-alphanumeric chars except hyphens
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// ── Handlers ────────────────────────────────────────────────────────────────

func (h *BackofficeCompanyHandler) GetCompanies(c *gin.Context) envelope.Response {
	var companies []company_models.Company
	if err := h.db.Preload("Tenant").Find(&companies).Error; err != nil {
		h.logger.Error("Failed to fetch companies", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining companies", core_errors.ErrCompanyNotFound)
	}

	h.logger.Info("Companies fetched successfully", zap.Int("count", len(companies)))
	return envelope.SuccessResponse(companies, "backoffice.company.list.success")
}

func (h *BackofficeCompanyHandler) GetCompanyByID(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var company company_models.Company
	if err := h.db.Preload("Tenant").First(&company, id).Error; err != nil {
		h.logger.Error("Company not found", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	return envelope.SuccessResponse(company, "backoffice.company.found")
}

func (h *BackofficeCompanyHandler) CreateCompany(c *gin.Context) envelope.Response {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	var company company_models.Company

	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create Tenant
		tenant := tenant_models.Tenant{
			Name: req.TradeName,
			Slug: slugify(req.TradeName),
		}
		if err := tx.Create(&tenant).Error; err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
		}

		// 2. Create Company
		company = company_models.Company{
			LegalName: req.LegalName,
			TradeName: req.TradeName,
			PlanCode:  req.PlanCode,
			TenantID:  tenant.ID,
		}
		if err := tx.Create(&company).Error; err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}

		return nil
	})

	if err != nil {
		h.logger.Error("Failed to create company", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating company", core_errors.ErrInternal)
	}

	// Reload with Tenant
	h.db.Preload("Tenant").First(&company, company.ID)

	h.logger.Info("Company created successfully", zap.String("trade_name", req.TradeName))
	return envelope.New(http.StatusCreated, "backoffice.company.create.success", company)
}

func (h *BackofficeCompanyHandler) UpdateCompany(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var company company_models.Company
	if err := h.db.First(&company, id).Error; err != nil {
		h.logger.Error("Company not found for update", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	var req UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.LegalName != "" {
		updates["legal_name"] = req.LegalName
	}
	if req.TradeName != "" {
		updates["trade_name"] = req.TradeName
	}
	if req.PlanCode != "" {
		updates["plan_code"] = req.PlanCode
	}

	if err := h.db.Model(&company).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update company", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating company", core_errors.ErrInternal)
	}

	h.db.Preload("Tenant").First(&company, company.ID)

	h.logger.Info("Company updated successfully", zap.String("id", id))
	return envelope.SuccessResponse(company, "backoffice.company.update.success")
}

func (h *BackofficeCompanyHandler) DeleteCompany(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var company company_models.Company
	if err := h.db.First(&company, id).Error; err != nil {
		h.logger.Error("Company not found for deletion", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	if err := h.db.Delete(&company).Error; err != nil {
		h.logger.Error("Failed to delete company", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting company", core_errors.ErrInternal)
	}

	h.logger.Info("Company deleted successfully", zap.String("id", id))
	return envelope.SuccessResponse(nil, "backoffice.company.delete.success")
}
