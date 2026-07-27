package company_handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/utils"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_dto "pengi-med-saas/features/users/dto"
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

type CreateAdditionalCompanyRequest struct {
	CompanyName string `json:"company_name" binding:"required,min=2,max=100"`
}

// CreateAdditionalCompany lets an already-authenticated user create a second,
// fully independent Company (own Tenant, own TRIAL Subscription) without
// re-registering. Mirrors UserHandler.Register step-by-step, minus the User
// creation step (the user already exists) and gated by a per-user ownership
// limit (User.MaxOwnedCompanies) that backoffice admins can raise manually.
func (h *CompanyHandler) CreateAdditionalCompany(c *gin.Context) envelope.Response {
	userID := c.GetInt64("user_id")

	var req CreateAdditionalCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	var user user_models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.Error("CreateAdditionalCompany: user not found", zap.Int64("user_id", userID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "User not found", core_errors.ErrUserNotFound)
	}

	var ownedCount int64
	h.db.Model(&company_models.Company{}).Where("owner_user_id = ?", user.ID).Count(&ownedCount)
	if int(ownedCount) >= user.MaxOwnedCompanies {
		return envelope.ErrorResponse(http.StatusForbidden, "company.create_additional.limit_reached", core_errors.ErrCompanyOwnershipLimit)
	}

	// Generate a unique slug (same algorithm + retry loop as Register).
	baseSlug := utils.GenerateSlug(req.CompanyName)
	slug := baseSlug
	for i := 1; ; i++ {
		var existing tenant_models.Tenant
		if err := h.db.Where("slug = ?", slug).First(&existing).Error; err != nil {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	var newEnv user_models.Environment
	var newCompany company_models.Company
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		// 1. Tenant — DisplayToken is required (uniqueIndex), same as Register.
		newTenant := tenant_models.Tenant{
			Name:      req.CompanyName,
			Slug:      slug,
			TradeName: req.CompanyName,
			DisplayToken: func() string {
				b := make([]byte, 16)
				rand.Read(b)
				return hex.EncodeToString(b)
			}(),
		}
		if err := tx.Create(&newTenant).Error; err != nil {
			return fmt.Errorf("tenant: %w", err)
		}

		// 2. Company
		newCompany = company_models.Company{
			LegalName:   req.CompanyName,
			TradeName:   req.CompanyName,
			PlanCode:    "TRIAL",
			TenantID:    newTenant.ID,
			OwnerUserID: user.ID,
		}
		if err := tx.Create(&newCompany).Error; err != nil {
			return fmt.Errorf("company: %w", err)
		}

		// 3. Canonical admin role
		var adminRole user_models.Role
		if err := tx.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&adminRole).Error; err != nil {
			return fmt.Errorf("canonical admin role not found: %w", err)
		}

		// 4. (skipped) User — already exists.

		// 5. Environment (link existing user → new company)
		newEnv = user_models.Environment{
			UserID:    user.ID,
			Name:      req.CompanyName,
			RoleID:    adminRole.ID,
			CompanyID: newCompany.ID,
		}
		if err := tx.Create(&newEnv).Error; err != nil {
			return fmt.Errorf("environment: %w", err)
		}

		// 6. TRIAL subscription (14 days)
		subscription := company_models.Subscription{
			Status:    "active",
			PlanCode:  "TRIAL",
			ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
			CompanyID: newCompany.ID,
		}
		if err := tx.Create(&subscription).Error; err != nil {
			return fmt.Errorf("subscription: %w", err)
		}

		newEnv.Role = adminRole
		newCompany.Tenant = newTenant
		return nil
	})

	if txErr != nil {
		h.logger.Error("CreateAdditionalCompany transaction failed", zap.Int64("user_id", userID), zap.Error(txErr))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// Live-compute enabled_features the same way GetEnvironmentsFromUser does,
	// so the frontend's setEnvironment(response.data) sees a consistent shape.
	if ef, err := company_services.EnabledFeaturesForCompany(h.db, newCompany.ID); err == nil {
		if raw, err := json.Marshal(ef); err == nil {
			newCompany.Tenant.EnabledFeatures = string(raw)
		}
	}

	h.logger.Info("Additional company created",
		zap.Int64("user_id", userID),
		zap.String("company", req.CompanyName),
		zap.String("slug", slug),
		zap.Uint("company_id", newCompany.ID))

	return envelope.New(http.StatusCreated, "company.create_additional.success", user_dto.EnvironmentWithCompany{
		Environment: newEnv,
		Company:     newCompany,
	})
}
