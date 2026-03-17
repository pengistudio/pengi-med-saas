package backoffice_handlers

import (
	"fmt"
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
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
	PlanCode  string `json:"plan_code"`
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
	if err := h.db.Preload("Tenant").Preload("Subscriptions", "status = ? AND expires_at > NOW()", "active").Find(&companies).Error; err != nil {
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

func (h *BackofficeCompanyHandler) GenerateCompanySignupToken(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var company company_models.Company
	if err := h.db.First(&company, id).Error; err != nil {
		h.logger.Error("Company not found for signup token", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	token, err := auth.GenerateCompanySignupToken(company.ID, 0)
	if err != nil {
		h.logger.Error("Failed to generate company signup token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error generating signup token", core_errors.ErrAuthTokenGenerateError)
	}

	h.logger.Info("Company signup token generated", zap.String("company_id", id))
	return envelope.SuccessResponse(gin.H{
		"token":      token,
		"company_id": company.ID,
		"trade_name": company.TradeName,
	}, "backoffice.company.signup_token.success")
}

// GetCompanyUsers returns all users linked to a company via Environment records.
func (h *BackofficeCompanyHandler) GetCompanyUsers(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var environments []user_models.Environment
	if err := h.db.
		Where("company_id = ?", id).
		Preload("Role").
		Find(&environments).Error; err != nil {
		h.logger.Error("Failed to fetch company users", zap.String("company_id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching company users", core_errors.ErrInternal)
	}

	// Collect user IDs to fetch user details
	userIDs := make([]uint, len(environments))
	for i, env := range environments {
		userIDs[i] = env.UserID
	}

	var users []user_models.User
	if len(userIDs) > 0 {
		if err := h.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			h.logger.Error("Failed to fetch users for company", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching users", core_errors.ErrInternal)
		}
	}

	// Build a user map for quick lookup
	userMap := make(map[uint]user_models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Build the response combining environment + user data
	type CompanyUserResponse struct {
		EnvironmentID   uint   `json:"environment_id"`
		UserID          uint   `json:"user_id"`
		UserName        string `json:"user_name"`
		Email           string `json:"email"`
		RoleID          uint   `json:"role_id"`
		RoleName        string `json:"role_name"`
		EnvironmentName string `json:"environment_name"`
	}

	result := make([]CompanyUserResponse, 0, len(environments))
	for _, env := range environments {
		u := userMap[env.UserID]
		result = append(result, CompanyUserResponse{
			EnvironmentID:   env.ID,
			UserID:          env.UserID,
			UserName:        u.UserName,
			Email:           u.Email,
			RoleID:          env.RoleID,
			RoleName:        env.Role.Role,
			EnvironmentName: env.Name,
		})
	}

	h.logger.Info("Company users fetched", zap.String("company_id", id), zap.Int("count", len(result)))
	return envelope.SuccessResponse(result, "backoffice.company.users.list.success")
}

// UpdateCompanyUser updates a user's attributes and/or their role within a company.
func (h *BackofficeCompanyHandler) UpdateCompanyUser(c *gin.Context) envelope.Response {
	companyID := c.Param("id")
	userID := c.Param("user_id")

	type UpdateCompanyUserRequest struct {
		UserName string `json:"user_name"`
		Email    string `json:"email"`
		RoleID   *uint  `json:"role_id"`
	}

	var req UpdateCompanyUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	// Find the environment linking this user to this company
	var env user_models.Environment
	if err := h.db.Where("company_id = ? AND user_id = ?", companyID, userID).First(&env).Error; err != nil {
		h.logger.Error("Environment not found", zap.String("company_id", companyID), zap.String("user_id", userID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "User not found in this company", core_errors.ErrUserNotFound)
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		// Update user attributes if provided
		userUpdates := map[string]interface{}{}
		if req.UserName != "" {
			userUpdates["user_name"] = req.UserName
		}
		if req.Email != "" {
			userUpdates["email"] = req.Email
		}
		if len(userUpdates) > 0 {
			if err := tx.Model(&user_models.User{}).Where("id = ?", userID).Updates(userUpdates).Error; err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}
		}

		// Update role if provided
		if req.RoleID != nil {
			if err := tx.Model(&env).Update("role_id", *req.RoleID).Error; err != nil {
				return fmt.Errorf("failed to update role: %w", err)
			}
		}

		return nil
	})

	if txErr != nil {
		h.logger.Error("Failed to update company user", zap.Error(txErr))
		return envelope.ErrorResponse(http.StatusInternalServerError, txErr.Error(), core_errors.ErrInternal)
	}

	h.logger.Info("Company user updated", zap.String("company_id", companyID), zap.String("user_id", userID))
	return envelope.SuccessResponse(nil, "backoffice.company.users.update.success")
}

// GetRoles returns all available roles.
func (h *BackofficeCompanyHandler) GetRoles(c *gin.Context) envelope.Response {
	var roles []user_models.Role
	if err := h.db.Find(&roles).Error; err != nil {
		h.logger.Error("Failed to fetch roles", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching roles", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(roles, "backoffice.roles.list.success")
}
