package agenda_services

import (
	"errors"
	"net/http"

	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	agenda_models "pengi-med-saas/features/agenda/models"
)

// Errors of the appointment type of an appointment.
var (
	ErrTypeNotFound         = errors.New("agenda: appointment type not found")
	ErrTypeInactive         = errors.New("agenda: appointment type inactive")
	ErrTypeDoctorNotAllowed = errors.New("agenda: doctor does not attend this type")
)

// ValidateAppointmentType checks the type of an appointment on a
// tenant-bound db: it must exist in the tenant, be active unless it is the
// appointment's current type (current), and allow doctorID when set.
func ValidateAppointmentType(db *gorm.DB, typeID uint, doctorID *uint, current *uint) error {
	var t agenda_models.AppointmentType
	if err := db.Preload("Doctors").Limit(1).Find(&t, typeID).Error; err != nil {
		return err
	}
	if t.ID == 0 {
		return ErrTypeNotFound
	}
	unchanged := current != nil && *current == typeID
	if !t.Active && !unchanged {
		return ErrTypeInactive
	}
	if doctorID != nil && !t.AllowsDoctor(*doctorID) {
		return ErrTypeDoctorNotAllowed
	}
	return nil
}

// ErrorResponse maps the errors above to their response; ok is false for
// any other error (the caller logs it and answers 500).
func ErrorResponse(err error) (resp envelope.Response, ok bool) {
	switch {
	case errors.Is(err, ErrTypeNotFound):
		return envelope.ErrorResponse(http.StatusBadRequest, "agenda.error.type_not_found", core_errors.ErrAgendaTypeNotFound), true
	case errors.Is(err, ErrTypeInactive):
		return envelope.ErrorResponse(http.StatusBadRequest, "agenda.error.type_inactive", core_errors.ErrAgendaTypeInactive), true
	case errors.Is(err, ErrTypeDoctorNotAllowed):
		return envelope.ErrorResponse(http.StatusBadRequest, "agenda.error.type_doctor_not_allowed", core_errors.ErrAgendaTypeDoctorNotAllowed), true
	}
	return envelope.Response{}, false
}
