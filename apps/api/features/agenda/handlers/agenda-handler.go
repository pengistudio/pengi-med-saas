package agenda_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	doctor_models "pengi-med-saas/features/doctors/models"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AgendaHandler serves doctor schedules, schedule blocks, appointment types
// and the availability/agenda queries built on them.
type AgendaHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAgendaHandler(db *gorm.DB, logger *zap.Logger) *AgendaHandler {
	return &AgendaHandler{db: db, logger: logger}
}

func (h *AgendaHandler) internal(msg string, err error) envelope.Response {
	h.logger.Error(msg, zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
}

func badRequest(message string, code core_errors.AppError) envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, message, code)
}

func invalidRequest() envelope.Response {
	return badRequest("agenda.error.invalid_request", core_errors.ErrAgendaInvalidRequest)
}

// uintParam parses the path parameter name as an ID.
func uintParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(id), err == nil && id > 0
}

// doctorParam loads the doctor of path :id from the tenant-bound db.
func (h *AgendaHandler) doctorParam(c *gin.Context, db *gorm.DB) (*doctor_models.Doctor, *envelope.Response) {
	id, ok := uintParam(c, "id")
	if !ok {
		resp := envelope.ErrorResponse(http.StatusNotFound, "doctors.error.not_found", core_errors.ErrDoctorNotFound)
		return nil, &resp
	}
	var doctor doctor_models.Doctor
	if err := db.Limit(1).Find(&doctor, id).Error; err != nil {
		resp := h.internal("failed to fetch doctor", err)
		return nil, &resp
	}
	if doctor.ID == 0 {
		resp := envelope.ErrorResponse(http.StatusNotFound, "doctors.error.not_found", core_errors.ErrDoctorNotFound)
		return nil, &resp
	}
	return &doctor, nil
}

// ownDoctor loads the doctor profile linked to the current user.
func (h *AgendaHandler) ownDoctor(c *gin.Context, db *gorm.DB) (*doctor_models.Doctor, *envelope.Response) {
	uid, _, ok := auth_middleware.GetUserFromContext(c)
	if !ok || uid <= 0 {
		resp := envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
		return nil, &resp
	}
	var doctor doctor_models.Doctor
	if err := db.Where("user_id = ?", uint(uid)).Limit(1).Find(&doctor).Error; err != nil {
		resp := h.internal("failed to fetch own doctor profile", err)
		return nil, &resp
	}
	if doctor.ID == 0 {
		resp := envelope.ErrorResponse(http.StatusNotFound, "doctors.error.profile_not_found", core_errors.ErrDoctorProfileNotFound)
		return nil, &resp
	}
	return &doctor, nil
}
