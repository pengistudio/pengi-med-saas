package clinical_handlers

import (
	"net/http"
	"testing"

	core_errors "pengi-med-saas/core/errors"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"

	"go.uber.org/zap"
)

// appointmentType creates a type in tenantID restricted to doctors (none = all).
func (s *isolation) appointmentType(tenantID uint, name string, active bool, doctors ...doctor_models.Doctor) agenda_models.AppointmentType {
	s.t.Helper()
	t := agenda_models.AppointmentType{TenantID: tenantID, Name: name, DurationMinutes: 30, Active: true, Doctors: doctors}
	if err := s.db.Omit("Doctors.*").Create(&t).Error; err != nil {
		s.t.Fatal(err)
	}
	if !active {
		s.db.Model(&t).Update("active", false)
	}
	return t
}

func TestAppointmentType_ValidatedOnCreateAndUpdate(t *testing.T) {
	s := newIsolation(t)
	if err := s.db.AutoMigrate(&agenda_models.AppointmentType{}); err != nil {
		t.Fatal(err)
	}
	first := s.doctorOf(s.own.ID)
	second := s.addDoctor(s.own.ID, "Dr. Segundo", nil)
	onlyFirst := s.appointmentType(s.own.ID, "Procedimiento", true, first)
	open := s.appointmentType(s.own.ID, "Control", true)
	retired := s.appointmentType(s.own.ID, "Antiguo", false)
	foreign := s.appointmentType(s.other.ID, "Ajeno", true)
	h := NewAppointmentHandler(s.db, zap.NewNop())

	create := func(doctorID uint, typeID uint, start string) (int, any) {
		resp := h.CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, start, start[:3]+"30", map[string]any{
			"doctor_id": doctorID, "appointment_type_id": typeID,
		})))
		return resp.Code, resp.Data
	}

	code, data := create(second.ID, onlyFirst.ID, "08:00")
	wantAppError(t, http.StatusBadRequest, data, code, core_errors.ErrAgendaTypeDoctorNotAllowed)
	code, data = create(first.ID, retired.ID, "08:00")
	wantAppError(t, http.StatusBadRequest, data, code, core_errors.ErrAgendaTypeInactive)
	code, data = create(first.ID, foreign.ID, "08:00")
	wantAppError(t, http.StatusBadRequest, data, code, core_errors.ErrAgendaTypeNotFound)

	code, data = create(first.ID, onlyFirst.ID, "08:00")
	if code != http.StatusOK {
		t.Fatalf("create with allowed type = %d %+v", code, data)
	}
	appt := data.(*clinical_models.Appointment)
	if appt.AppointmentTypeID == nil || *appt.AppointmentTypeID != onlyFirst.ID {
		t.Fatalf("type = %v, want %d", appt.AppointmentTypeID, onlyFirst.ID)
	}

	// Moving it to a doctor the type doesn't allow is rejected…
	resp := h.UpdateAppointment(s.ctx(appt.ID, map[string]any{"doctor_id": second.ID}))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrAgendaTypeDoctorNotAllowed)
	// …unless the type changes too.
	resp = h.UpdateAppointment(s.ctx(appt.ID, map[string]any{"doctor_id": second.ID, "appointment_type_id": open.ID}))
	if resp.Code != http.StatusOK || *s.reloadAppointment(appt.ID).AppointmentTypeID != open.ID {
		t.Fatalf("change doctor and type = %d %+v", resp.Code, resp.Data)
	}

	// A type deactivated after booking can be kept, but not chosen again.
	s.db.Model(&open).Update("active", false)
	resp = h.UpdateAppointment(s.ctx(appt.ID, map[string]any{"title": "Control 2", "appointment_type_id": open.ID}))
	if resp.Code != http.StatusOK {
		t.Fatalf("keep inactive type = %d %+v", resp.Code, resp.Data)
	}
	// 0 removes the type.
	resp = h.UpdateAppointment(s.ctx(appt.ID, map[string]any{"appointment_type_id": 0}))
	if resp.Code != http.StatusOK || s.reloadAppointment(appt.ID).AppointmentTypeID != nil {
		t.Fatalf("clear type = %d, type %v", resp.Code, s.reloadAppointment(appt.ID).AppointmentTypeID)
	}
	resp = h.UpdateAppointment(s.ctx(appt.ID, map[string]any{"appointment_type_id": open.ID}))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrAgendaTypeInactive)
}
