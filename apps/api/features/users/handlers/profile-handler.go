package user_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewProfileHandler(db *gorm.DB, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{db: db, logger: logger}
}

// ─── Response DTO ────────────────────────────────────────────────────────────

type ProfileResponse struct {
	UserID    uint   `json:"user_id"`
	UserName  string `json:"user_name"`
	Email     string `json:"email"`
	EnvID     uint   `json:"environment_id"`
	EnvName   string `json:"environment_name"`
	Role      string `json:"role"`
	LegalName string `json:"legal_name"`
	TradeName string `json:"trade_name"`
}

// ─── Update DTO ──────────────────────────────────────────────────────────────

type UpdateProfileDTO struct {
	Email   *string `json:"email"`
	EnvName *string `json:"environment_name"`
}

// GetProfile returns user info + selected environment details
func (h *ProfileHandler) GetProfile(c *gin.Context) envelope.Response {
	userID := c.GetInt64("user_id")

	envIDParam := c.Query("environment_id")
	envID, err := strconv.ParseUint(envIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid environment_id", core_errors.ErrInvalidRequest)
	}

	var user user_models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.Error("Profile: user not found", zap.Int64("user_id", userID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "User not found", core_errors.ErrUserNotFound)
	}

	var env user_models.Environment
	if err := h.db.Preload("Role").Where("id = ? AND user_id = ?", envID, userID).First(&env).Error; err != nil {
		h.logger.Error("Profile: environment not found", zap.Uint64("env_id", envID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Environment not found", core_errors.ErrInvalidRequest)
	}

	var company company_models.Company
	if err := h.db.First(&company, env.CompanyID).Error; err != nil {
		h.logger.Error("Profile: company not found", zap.Uint("company_id", env.CompanyID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrInvalidRequest)
	}

	profile := ProfileResponse{
		UserID:    user.ID,
		UserName:  user.UserName,
		Email:     user.Email,
		EnvID:     env.ID,
		EnvName:   env.Name,
		Role:      env.Role.Role,
		LegalName: company.LegalName,
		TradeName: company.TradeName,
	}

	return envelope.SuccessResponse(profile, "profile.get.success")
}

// UpdateProfile updates user email and/or environment name
func (h *ProfileHandler) UpdateProfile(c *gin.Context) envelope.Response {
	userID := c.GetInt64("user_id")

	var payload UpdateProfileDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrInvalidRequest)
	}

	envIDParam := c.Query("environment_id")
	envID, err := strconv.ParseUint(envIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid environment_id", core_errors.ErrInvalidRequest)
	}

	// Update user email
	if payload.Email != nil {
		if err := h.db.Model(&user_models.User{}).Where("id = ?", userID).Update("email", *payload.Email).Error; err != nil {
			h.logger.Error("Profile: failed to update email", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrInvalidRequest)
		}
	}

	// Update environment name
	if payload.EnvName != nil {
		if err := h.db.Model(&user_models.Environment{}).Where("id = ? AND user_id = ?", envID, userID).Update("name", *payload.EnvName).Error; err != nil {
			h.logger.Error("Profile: failed to update environment name", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrInvalidRequest)
		}
	}

	// Return updated profile
	return h.GetProfile(c)
}
