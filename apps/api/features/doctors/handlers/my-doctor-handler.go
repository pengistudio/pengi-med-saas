package doctor_handlers

import (
	"errors"
	"net/http"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	doctor_dto "pengi-med-saas/features/doctors/dto"
	doctor_models "pengi-med-saas/features/doctors/models"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func currentUserID(c *gin.Context) (uint, bool) {
	uid, _, ok := auth_middleware.GetUserFromContext(c)
	return uint(uid), ok && uid > 0
}

// own returns the doctor linked to the current user in this tenant, or nil.
func (h *DoctorHandler) own(db *gorm.DB, userID uint) (*doctor_models.Doctor, error) {
	var doctor doctor_models.Doctor
	err := db.Where("user_id = ?", userID).First(&doctor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doctor, nil
}

// GetMyDoctor returns the current user's doctor profile.
func (h *DoctorHandler) GetMyDoctor(c *gin.Context) envelope.Response {
	userID, ok := currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	doctor, err := h.own(tenantdb.For(c, h.db), userID)
	if err != nil {
		h.logger.Error("failed to fetch own doctor profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if doctor == nil {
		return envelope.ErrorResponse(http.StatusNotFound, "doctors.error.profile_not_found", core_errors.ErrDoctorProfileNotFound)
	}
	return envelope.SuccessResponse(doctor, "doctors.me.fetch.success")
}

// CreateMyDoctor creates a doctor profile linked to the current user
// (onboarding: "do you attend patients?").
func (h *DoctorHandler) CreateMyDoctor(c *gin.Context) envelope.Response {
	userID, ok := currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	var req doctor_dto.DoctorProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
	}
	existing, err := h.own(tenantdb.For(c, h.db), userID)
	if err != nil {
		h.logger.Error("failed to fetch own doctor profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if existing != nil {
		return envelope.ErrorResponse(http.StatusConflict, "doctors.error.profile_exists", core_errors.ErrDoctorProfileExists)
	}
	profileExists := &validationError{status: http.StatusConflict, message: "doctors.error.profile_exists", code: core_errors.ErrDoctorProfileExists}
	return h.create(c, req, &userID, "doctors.me.create.success", profileExists)
}

// UpdateMyDoctor lets the linked user edit their own profile fields. Active
// state and the account link stay admin-only.
func (h *DoctorHandler) UpdateMyDoctor(c *gin.Context) envelope.Response {
	userID, ok := currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	db := tenantdb.For(c, h.db)
	doctor, err := h.own(db, userID)
	if err != nil {
		h.logger.Error("failed to fetch own doctor profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if doctor == nil {
		return envelope.ErrorResponse(http.StatusNotFound, "doctors.error.profile_not_found", core_errors.ErrDoctorProfileNotFound)
	}
	return h.update(c, db, doctor, "doctors.me.update.success")
}

// GetStatus tells the frontend which doctor onboarding/notices apply: whether
// the current user has a profile, how many doctors are active, and whether
// migrated profiles still wait for review.
func (h *DoctorHandler) GetStatus(c *gin.Context) envelope.Response {
	userID, ok := currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	db := tenantdb.For(c, h.db)
	doctor, err := h.own(db, userID)
	if err != nil {
		h.logger.Error("failed to fetch own doctor profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	status := doctor_dto.DoctorStatusResponse{HasProfile: doctor != nil, Doctor: doctor}
	if err := db.Model(&doctor_models.Doctor{}).Where("active = ?", true).Count(&status.ActiveDoctors).Error; err != nil {
		h.logger.Error("failed to count active doctors", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	var pending int64
	if err := db.Model(&doctor_models.Doctor{}).Where("needs_review = ?", true).Count(&pending).Error; err != nil {
		h.logger.Error("failed to count doctors to review", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	status.NeedsReview = pending > 0
	return envelope.SuccessResponse(status, "doctors.status.success")
}
