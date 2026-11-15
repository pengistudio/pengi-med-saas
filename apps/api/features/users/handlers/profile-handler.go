package user_handlers

import (
	"errors"
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewProfileHandler(db *gorm.DB, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{db: tenantdb.System(db), logger: logger}
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
	Email   *string `json:"email" binding:"omitempty,email,max=255"`
	EnvName *string `json:"environment_name"`
	// CurrentPassword is required only when Email changes: a stolen session
	// must not be able to move the account (and its password reset) to
	// another inbox.
	CurrentPassword string `json:"current_password"`
}

// GetProfile returns user info + selected environment details
func (h *ProfileHandler) GetProfile(c *gin.Context) envelope.Response {
	userID := c.GetInt64("user_id")

	envIDParam := c.Query("environment_id")
	envID, err := strconv.ParseUint(envIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	var user user_models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.Error("Profile: user not found", zap.Int64("user_id", userID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
	}

	var env user_models.Environment
	if err := h.db.Preload("Role").Where("id = ? AND user_id = ?", envID, userID).First(&env).Error; err != nil {
		h.logger.Error("Profile: environment not found", zap.Uint64("env_id", envID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserEnvironmentNotFound)
	}

	var company company_models.Company
	if err := h.db.First(&company, env.CompanyID).Error; err != nil {
		h.logger.Error("Profile: company not found", zap.Uint("company_id", env.CompanyID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
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

// errRejected aborts the profile transaction when a request is refused.
var errRejected = errors.New("profile update rejected")

// UpdateProfile updates user email and/or environment name
func (h *ProfileHandler) UpdateProfile(c *gin.Context) envelope.Response {
	userID := c.GetInt64("user_id")

	var payload UpdateProfileDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	envIDParam := c.Query("environment_id")
	envID, err := strconv.ParseUint(envIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	// Email and environment name are saved together or not at all.
	var rejected *envelope.Response
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		// Email change is re-authenticated, see UpdateProfileDTO.
		if payload.Email != nil {
			if resp, ok := h.changeEmail(tx, userID, *payload.Email, payload.CurrentPassword); !ok {
				rejected = &resp
				return errRejected
			}
		}
		if payload.EnvName != nil {
			if err := tx.Model(&user_models.Environment{}).Where("id = ? AND user_id = ?", envID, userID).Update("name", *payload.EnvName).Error; err != nil {
				h.logger.Error("Profile: failed to update environment name", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if rejected != nil {
		return *rejected
	}
	if txErr != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// Return updated profile
	return h.GetProfile(c)
}

// changeEmail stores a new email for userID. Emails are compared and stored
// lowercased (login matches them case-insensitively): the same address in
// another case is no change and needs no password. A real change needs the
// current password and an address no other user has. ok is false with the
// error response to return.
func (h *ProfileHandler) changeEmail(db *gorm.DB, userID int64, email, currentPassword string) (envelope.Response, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest), false
	}

	var user user_models.User
	if err := db.First(&user, userID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound), false
	}
	if strings.EqualFold(user.Email, email) {
		return envelope.Response{}, true
	}

	if currentPassword == "" || !auth.CompareHashAndPassword(user.Password, currentPassword) {
		return envelope.ErrorResponse(http.StatusUnauthorized, "auth.change_password.wrong_current", core_errors.ErrAuthWrongCurrentPassword), false
	}

	var taken int64
	if err := db.Model(&user_models.User{}).Where("LOWER(email) = ? AND id <> ?", email, user.ID).Count(&taken).Error; err != nil {
		h.logger.Error("Profile: failed to check email uniqueness", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal), false
	}
	if taken > 0 {
		return envelope.ErrorResponse(http.StatusConflict, "auth.register.email_taken", core_errors.ErrAuthEmailTaken), false
	}

	if err := db.Model(&user_models.User{}).Where("id = ?", user.ID).Update("email", email).Error; err != nil {
		h.logger.Error("Profile: failed to update email", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal), false
	}
	return envelope.Response{}, true
}
