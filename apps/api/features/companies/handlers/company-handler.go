package company_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	role_data "pengi-med-saas/features/users/data"
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
		RoleID          uint   `json:"role_id"`
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
			RoleID:          env.RoleID,
			RoleName:        env.Role.Role,
			EnvironmentName: env.Name,
		})
	}

	return envelope.SuccessResponse(result, "company.team.list.success")
}

// GetTeamRoles returns the fixed, assignable role catalog so the frontend can
// populate the role selector. Legacy/placeholder roles (moderator, user) are
// excluded — they are not part of role_data.CanonicalRoles.
func (h *CompanyHandler) GetTeamRoles(c *gin.Context) envelope.Response {
	var roles []user_models.Role
	if err := h.db.Where("role IN ?", role_data.CanonicalRoles).Find(&roles).Error; err != nil {
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

	// Verify the role exists and is part of the assignable catalog
	var role user_models.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}
	if !role_data.IsCanonicalRole(role.Role) {
		return envelope.ErrorResponse(http.StatusBadRequest, "company.team.role.invalid", core_errors.ErrTeamInvalidRole)
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

type UpdateTeamMemberRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

// UpdateTeamMemberRole reassigns an existing team member's role. Guarded by
// RequireRolePermission(MANAGE_TEAM_MEMBERS) at the route level.
func (h *CompanyHandler) UpdateTeamMemberRole(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	environmentID, err := strconv.ParseUint(c.Param("environment_id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid environment id", core_errors.ErrInvalidRequest)
	}

	var req UpdateTeamMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "role_id is required", core_errors.ErrInvalidRequest)
	}

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	var env user_models.Environment
	if err := h.db.Preload("Role").First(&env, uint(environmentID)).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.team.role.update.forbidden", core_errors.ErrTeamEnvironmentNotFound)
	}
	if env.CompanyID != company.ID {
		return envelope.ErrorResponse(http.StatusForbidden, "company.team.role.update.forbidden", core_errors.ErrTeamEnvironmentNotFound)
	}

	var newRole user_models.Role
	if err := h.db.First(&newRole, req.RoleID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}
	if !role_data.IsCanonicalRole(newRole.Role) {
		return envelope.ErrorResponse(http.StatusBadRequest, "company.team.role.invalid", core_errors.ErrTeamInvalidRole)
	}

	// Prevent demoting the last admin of a company — would lock the account out.
	if env.Role.Role == role_data.RoleAdmin && newRole.Role != role_data.RoleAdmin {
		var otherAdmins int64
		h.db.Model(&user_models.Environment{}).
			Joins("JOIN roles ON roles.id = environments.role_id").
			Where("environments.company_id = ? AND environments.id != ? AND roles.role = ?", company.ID, env.ID, role_data.RoleAdmin).
			Count(&otherAdmins)
		if otherAdmins == 0 {
			return envelope.ErrorResponse(http.StatusConflict, "company.team.role.last_admin", core_errors.ErrTeamLastAdmin)
		}
	}

	// Note: update via a bare Environment{} + explicit Where, not Model(&env) —
	// env has a preloaded Role association, and GORM's association-save
	// behavior on Update would otherwise re-derive role_id from env.Role.ID
	// and silently overwrite req.RoleID.
	if err := h.db.Model(&user_models.Environment{}).Where("id = ?", env.ID).Update("role_id", req.RoleID).Error; err != nil {
		h.logger.Error("Failed to update team member role", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating role", core_errors.ErrInternal)
	}

	h.logger.Info("Team member role updated",
		zap.Uint("environment_id", env.ID), zap.Uint("role_id", req.RoleID))
	return envelope.SuccessResponse(gin.H{
		"environment_id": env.ID,
		"role_id":        req.RoleID,
	}, "company.team.role.update.success")
}
