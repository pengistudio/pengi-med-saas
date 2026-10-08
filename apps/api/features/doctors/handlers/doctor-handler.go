package doctor_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	doctor_data "pengi-med-saas/features/doctors/data"
	doctor_dto "pengi-med-saas/features/doctors/dto"
	doctor_models "pengi-med-saas/features/doctors/models"
	doctor_services "pengi-med-saas/features/doctors/services"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DoctorHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDoctorHandler(db *gorm.DB, logger *zap.Logger) *DoctorHandler {
	return &DoctorHandler{db: db, logger: logger}
}

// validationError is a 4xx outcome of validating a request.
type validationError struct {
	status  int
	message string
	code    core_errors.AppError
}

func (e *validationError) response() envelope.Response {
	return envelope.ErrorResponse(e.status, e.message, e.code)
}

func badRequest(message string, code core_errors.AppError) *validationError {
	return &validationError{status: http.StatusBadRequest, message: message, code: code}
}

// isUniqueViolation reports whether err is a unique-index violation: the
// (tenant_id, user_id) index losing a race against the pre-check.
func isUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
		return true
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed") // sqlite (tests)
}

// alreadyLinked is the 409 for a user that already has a doctor profile.
func alreadyLinked() *validationError {
	return &validationError{status: http.StatusConflict, message: "doctors.error.user_already_linked", code: core_errors.ErrDoctorUserAlreadyLinked}
}

// applyProfile validates req and copies it onto doctor. Specialty and color
// are checked against doctor_data; SpecialtyOther only survives for "other".
func applyProfile(doctor *doctor_models.Doctor, req doctor_dto.UpdateDoctorRequest) *validationError {
	if req.FullName != nil {
		name := strings.TrimSpace(*req.FullName)
		if name == "" {
			return badRequest("doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
		}
		doctor.FullName = name
	}
	if req.Specialty != nil {
		doctor.Specialty = strings.TrimSpace(*req.Specialty)
	}
	if req.SpecialtyOther != nil {
		doctor.SpecialtyOther = strings.TrimSpace(*req.SpecialtyOther)
	}
	if !doctor_data.IsValidSpecialty(doctor.Specialty) {
		return badRequest("doctors.error.invalid_specialty", core_errors.ErrDoctorInvalidSpecialty)
	}
	if doctor.Specialty == doctor_data.SpecialtyOther {
		if doctor.SpecialtyOther == "" {
			return badRequest("doctors.error.specialty_other_required", core_errors.ErrDoctorInvalidSpecialty)
		}
	} else {
		doctor.SpecialtyOther = ""
	}
	if req.IDNumber != nil {
		doctor.IDNumber = strings.TrimSpace(*req.IDNumber)
	}
	if req.ProfessionalRegistry != nil {
		doctor.ProfessionalRegistry = strings.TrimSpace(*req.ProfessionalRegistry)
	}
	if req.Phone != nil {
		doctor.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Email != nil {
		doctor.Email = strings.TrimSpace(*req.Email)
	}
	if req.Color != nil {
		color := strings.ToUpper(strings.TrimSpace(*req.Color))
		if !doctor_data.IsValidColor(color) {
			return badRequest("doctors.error.invalid_color", core_errors.ErrDoctorInvalidColor)
		}
		doctor.Color = color
	}
	return nil
}

// profileUpdate turns a full profile request into the partial-update shape,
// leaving Color out when empty so create can auto-assign it.
func profileUpdate(req doctor_dto.DoctorProfileRequest) doctor_dto.UpdateDoctorRequest {
	u := doctor_dto.UpdateDoctorRequest{
		FullName:             &req.FullName,
		Specialty:            &req.Specialty,
		SpecialtyOther:       &req.SpecialtyOther,
		IDNumber:             &req.IDNumber,
		ProfessionalRegistry: &req.ProfessionalRegistry,
		Phone:                &req.Phone,
		Email:                &req.Email,
	}
	if strings.TrimSpace(req.Color) != "" {
		u.Color = &req.Color
	}
	return u
}

// nextColor picks the agenda color of a new doctor in the request's tenant.
func nextColor(db *gorm.DB) (string, error) {
	var used []string
	if err := db.Model(&doctor_models.Doctor{}).Pluck("color", &used).Error; err != nil {
		return "", err
	}
	return doctor_data.NextColor(used), nil
}

// checkUserLink verifies userID can be linked to doctorID (0 for a new
// doctor): the user belongs to the tenant and no other doctor has it.
func (h *DoctorHandler) checkUserLink(c *gin.Context, db *gorm.DB, userID, doctorID uint) (*validationError, error) {
	var members int64
	if err := db.Model(&user_models.Environment{}).
		Joins("JOIN companies ON companies.id = environments.company_id AND companies.deleted_at IS NULL").
		Where("companies.tenant_id = ? AND environments.user_id = ?", tenantdb.TenantID(c), userID).
		Count(&members).Error; err != nil {
		return nil, err
	}
	if members == 0 {
		return badRequest("doctors.error.user_not_in_tenant", core_errors.ErrDoctorUserNotInTenant), nil
	}
	var linked int64
	if err := db.Model(&doctor_models.Doctor{}).Where("user_id = ? AND id <> ?", userID, doctorID).Count(&linked).Error; err != nil {
		return nil, err
	}
	if linked > 0 {
		return alreadyLinked(), nil
	}
	return nil, nil
}

// find loads a doctor of the request's tenant by the :id param.
func (h *DoctorHandler) find(c *gin.Context, db *gorm.DB) (*doctor_models.Doctor, *envelope.Response) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
		return nil, &resp
	}
	var doctor doctor_models.Doctor
	if err := db.First(&doctor, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp := envelope.ErrorResponse(http.StatusNotFound, "doctors.error.not_found", core_errors.ErrDoctorNotFound)
			return nil, &resp
		}
		h.logger.Error("failed to fetch doctor", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return nil, &resp
	}
	return &doctor, nil
}

// GetDoctors lists the tenant's doctors by name. ?active=true|false filters.
func (h *DoctorHandler) GetDoctors(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	query := db.Order("full_name")
	if raw := c.Query("active"); raw != "" {
		active, err := strconv.ParseBool(raw)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
		}
		query = query.Where("active = ?", active)
	}
	doctors := []doctor_models.Doctor{}
	if err := query.Find(&doctors).Error; err != nil {
		h.logger.Error("failed to list doctors", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(doctors, "doctors.list.success")
}

// GetDoctor returns one doctor.
func (h *DoctorHandler) GetDoctor(c *gin.Context) envelope.Response {
	doctor, errResp := h.find(c, tenantdb.For(c, h.db))
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(doctor, "doctors.get.success")
}

// GetSpecialties returns the specialty catalog with the i18n key of each label.
func (h *DoctorHandler) GetSpecialties(c *gin.Context) envelope.Response {
	out := make([]doctor_dto.SpecialtyResponse, 0, len(doctor_data.Specialties))
	for _, code := range doctor_data.Specialties {
		out = append(out, doctor_dto.SpecialtyResponse{Code: code, LabelKey: doctor_data.SpecialtyLabelKey(code)})
	}
	return envelope.SuccessResponse(out, "doctors.specialties.success")
}

// create stores a new active doctor built from req, optionally linked to userID.
// conflict is returned when the user already has a profile in the tenant.
func (h *DoctorHandler) create(c *gin.Context, req doctor_dto.DoctorProfileRequest, userID *uint, successKey string, conflict *validationError) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor := doctor_models.Doctor{TenantID: tenantdb.TenantID(c), Active: true}
	if verr := applyProfile(&doctor, profileUpdate(req)); verr != nil {
		return verr.response()
	}
	if userID != nil {
		verr, err := h.checkUserLink(c, db, *userID, 0)
		if err != nil {
			h.logger.Error("failed to check doctor user link", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		if verr != nil {
			if verr.status == http.StatusConflict {
				return conflict.response()
			}
			return verr.response()
		}
		doctor.UserID = userID
	}
	if doctor.Color == "" {
		color, err := nextColor(db)
		if err != nil {
			h.logger.Error("failed to pick doctor color", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		doctor.Color = color
	}
	if err := db.Create(&doctor).Error; err != nil {
		if userID != nil && isUniqueViolation(err) {
			return conflict.response()
		}
		h.logger.Error("failed to create doctor", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(doctor, successKey)
}

// CreateDoctor creates a doctor, optionally linked to a user of the tenant.
func (h *DoctorHandler) CreateDoctor(c *gin.Context) envelope.Response {
	var req doctor_dto.CreateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
	}
	return h.create(c, req.DoctorProfileRequest, req.UserID, "doctors.create.success", alreadyLinked())
}

// update applies a partial profile update to doctor and clears its review flag.
func (h *DoctorHandler) update(c *gin.Context, db *gorm.DB, doctor *doctor_models.Doctor, successKey string) envelope.Response {
	var req doctor_dto.UpdateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
	}
	if verr := applyProfile(doctor, req); verr != nil {
		return verr.response()
	}
	doctor.NeedsReview = false
	// Only profile columns: Active and UserID are never written here.
	if err := db.Model(doctor).Select("full_name", "specialty", "specialty_other", "id_number",
		"professional_registry", "phone", "email", "color", "needs_review").Updates(doctor).Error; err != nil {
		h.logger.Error("failed to update doctor", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(doctor, successKey)
}

// UpdateDoctor updates a doctor's profile fields.
func (h *DoctorHandler) UpdateDoctor(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.find(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.update(c, db, doctor, "doctors.update.success")
}

// LinkUser links the doctor to a user of the tenant, or unlinks it (null).
func (h *DoctorHandler) LinkUser(c *gin.Context) envelope.Response {
	var req doctor_dto.LinkUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.invalid_request", core_errors.ErrDoctorInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.find(c, db)
	if errResp != nil {
		return *errResp
	}
	if req.UserID != nil {
		verr, err := h.checkUserLink(c, db, *req.UserID, doctor.ID)
		if err != nil {
			h.logger.Error("failed to check doctor user link", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		if verr != nil {
			return verr.response()
		}
	}
	if err := db.Model(doctor).Update("user_id", req.UserID).Error; err != nil {
		if isUniqueViolation(err) {
			return alreadyLinked().response()
		}
		h.logger.Error("failed to link doctor user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	doctor.UserID = req.UserID
	return envelope.SuccessResponse(doctor, "doctors.link.success")
}

func (h *DoctorHandler) setActive(c *gin.Context, active bool, successKey string) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.find(c, db)
	if errResp != nil {
		return *errResp
	}
	if err := db.Model(doctor).Update("active", active).Error; err != nil {
		h.logger.Error("failed to change doctor active state", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	doctor.Active = active
	return envelope.SuccessResponse(doctor, successKey)
}

// DeactivateDoctor hides the doctor from selectors; its records keep it.
func (h *DoctorHandler) DeactivateDoctor(c *gin.Context) envelope.Response {
	return h.setActive(c, false, "doctors.deactivate.success")
}

// ActivateDoctor makes a deactivated doctor selectable again.
func (h *DoctorHandler) ActivateDoctor(c *gin.Context) envelope.Response {
	return h.setActive(c, true, "doctors.activate.success")
}

// DeleteDoctor removes a doctor that no record references; otherwise the
// caller must deactivate it.
func (h *DoctorHandler) DeleteDoctor(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.find(c, db)
	if errResp != nil {
		return *errResp
	}
	referenced, err := doctor_services.IsReferenced(db, doctor.ID)
	if err != nil {
		h.logger.Error("failed to check doctor references", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if referenced {
		return envelope.ErrorResponse(http.StatusConflict, "doctors.error.in_use", core_errors.ErrDoctorInUse)
	}
	// Hard delete: a soft-deleted row would keep holding the (tenant, user) unique index.
	if err := db.Unscoped().Delete(doctor).Error; err != nil {
		h.logger.Error("failed to delete doctor", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(nil, "doctors.delete.success")
}

// MarkReviewed clears the review flag of every doctor of the tenant, which
// dismisses the one-time "review your doctors" notice.
func (h *DoctorHandler) MarkReviewed(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	if err := db.Model(&doctor_models.Doctor{}).Where("needs_review = ?", true).Update("needs_review", false).Error; err != nil {
		h.logger.Error("failed to mark doctors reviewed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(nil, "doctors.review.success")
}
