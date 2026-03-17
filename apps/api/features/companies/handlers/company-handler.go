package company_handlers

import (
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"

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

// GetTeamMembers returns all users linked to the current tenant's company.
func (h *CompanyHandler) GetTeamMembers(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	var environments []user_models.Environment
	if err := h.db.
		Where("company_id = ?", company.ID).
		Preload("Role").
		Find(&environments).Error; err != nil {
		h.logger.Error("Failed to fetch team members", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching team", core_errors.ErrInternal)
	}

	userIDs := make([]uint, len(environments))
	for i, env := range environments {
		userIDs[i] = env.UserID
	}

	var users []user_models.User
	if len(userIDs) > 0 {
		h.db.Where("id IN ?", userIDs).Find(&users)
	}

	userMap := make(map[uint]user_models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type TeamMember struct {
		EnvironmentID   uint   `json:"environment_id"`
		UserID          uint   `json:"user_id"`
		UserName        string `json:"user_name"`
		Email           string `json:"email"`
		RoleName        string `json:"role_name"`
		EnvironmentName string `json:"environment_name"`
	}

	result := make([]TeamMember, 0, len(environments))
	for _, env := range environments {
		u := userMap[env.UserID]
		result = append(result, TeamMember{
			EnvironmentID:   env.ID,
			UserID:          env.UserID,
			UserName:        u.UserName,
			Email:           u.Email,
			RoleName:        env.Role.Role,
			EnvironmentName: env.Name,
		})
	}

	return envelope.SuccessResponse(result, "company.team.list.success")
}

// GetTeamRoles returns all available roles so the frontend can populate the role selector.
func (h *CompanyHandler) GetTeamRoles(c *gin.Context) envelope.Response {
	var roles []user_models.Role
	if err := h.db.Find(&roles).Error; err != nil {
		h.logger.Error("Failed to fetch roles", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching roles", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(roles, "company.team.roles.success")
}

type GenerateInviteLinkRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

// GenerateInviteLink generates a company signup token with an embedded role.
// Only admins should call this endpoint (enforced via role check).
func (h *CompanyHandler) GenerateInviteLink(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var req GenerateInviteLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "role_id is required", core_errors.ErrInvalidRequest)
	}

	// Verify the role exists
	var role user_models.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	token, err := auth.GenerateCompanySignupToken(company.ID, req.RoleID)
	if err != nil {
		h.logger.Error("Failed to generate invite token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error generating invite link", core_errors.ErrAuthTokenGenerateError)
	}

	h.logger.Info("Invite link generated", zap.Uint("company_id", company.ID), zap.Uint("role_id", req.RoleID))
	return envelope.SuccessResponse(gin.H{
		"token": token,
	}, "company.invite_link.success")
}
