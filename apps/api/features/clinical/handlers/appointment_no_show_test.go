package clinical_handlers

import (
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"

	"go.uber.org/zap"
)

func (s *isolation) appointmentOn(tenantID, patientID uint, date time.Time, status string) clinical_models.Appointment {
	s.t.Helper()
	a := clinical_models.Appointment{TenantID: tenantID, PatientID: patientID, Title: "Control", Date: date, StartTime: "09:00", EndTime: "09:30", Status: status}
	if err := s.db.Create(&a).Error; err != nil {
		s.t.Fatalf("create appointment: %v", err)
	}
	return a
}

func TestUpdateStatus_NoShowAndCorrections(t *testing.T) {
	s := newIsolation(t)
	if err := s.db.AutoMigrate(&audit.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	audit.RegisterCallbacks(s.db)
	h := NewAppointmentHandler(s.db, zap.NewNop())
	appt := s.appointmentOn(s.own.ID, s.ownPatient.ID, time.Now(), "scheduled")

	// Marked as no-show, then corrected back to each allowed status.
	for _, status := range []string{"no_show", "scheduled", "no_show", "arrived", "no_show", "completed"} {
		resp := h.UpdateStatus(s.ctx(appt.ID, map[string]string{"status": status}))
		if resp.Code != http.StatusOK {
			t.Fatalf("status %s = %d %+v", status, resp.Code, resp.Data)
		}
		if got := s.reloadAppointment(appt.ID).Status; got != status {
			t.Fatalf("stored status = %q, want %q", got, status)
		}
	}

	// Each change is audited (Appointment is auditable).
	var updates int64
	s.db.Model(&audit.AuditLog{}).Where("entity_type = ? AND entity_id = ? AND action = ?", "appointments", appt.ID, "UPDATE").Count(&updates)
	if updates < 6 {
		t.Fatalf("audited updates = %d, want at least 6", updates)
	}

	// Unknown statuses are still rejected.
	if resp := h.UpdateStatus(s.ctx(appt.ID, map[string]string{"status": "absent"})); resp.Code != http.StatusBadRequest {
		t.Fatalf("unknown status = %d, want 400", resp.Code)
	}

	// A no-show can be deleted, like a cancelled or completed appointment.
	gone := s.appointmentOn(s.own.ID, s.ownPatient.ID, time.Now(), "no_show")
	if resp := h.DeleteAppointment(s.ctx(gone.ID, nil)); resp.Code != http.StatusOK {
		t.Fatalf("delete no_show = %d %+v", resp.Code, resp.Data)
	}
}

func TestGetPatientNoShows_WindowAndIsolation(t *testing.T) {
	s := newIsolation(t)
	h := NewAppointmentHandler(s.db, zap.NewNop())
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	s.appointmentOn(s.own.ID, s.ownPatient.ID, now.AddDate(0, -13, 0), "no_show") // outside the window
	s.appointmentOn(s.own.ID, s.ownPatient.ID, now.AddDate(0, -6, 0), "no_show")
	s.appointmentOn(s.own.ID, s.ownPatient.ID, lastMonth, "no_show")
	s.appointmentOn(s.own.ID, s.ownPatient.ID, now.AddDate(0, -2, 0), "cancelled") // not a no-show
	s.appointmentOn(s.other.ID, s.otherPatient.ID, lastMonth, "no_show")           // another clinic

	resp := h.GetPatientNoShows(s.ctx(s.ownPatient.ID, nil))
	if resp.Code != http.StatusOK || resp.Message != "appointments.no_shows.fetch.success" {
		t.Fatalf("got %d %q", resp.Code, resp.Message)
	}
	got := resp.Data.(clinical_dto.PatientNoShowsDTO)
	if got.Count != 2 || got.Months != 12 {
		t.Fatalf("count = %d months = %d, want 2 and 12", got.Count, got.Months)
	}
	if got.LastDate == nil || got.LastDate.Sub(lastMonth).Abs() > time.Second {
		t.Fatalf("last date = %v, want %v", got.LastDate, lastMonth)
	}

	// Another clinic's patient is not found, and nothing of theirs leaks.
	if resp := h.GetPatientNoShows(s.ctx(s.otherPatient.ID, nil)); resp.Code != http.StatusNotFound {
		t.Fatalf("other tenant's patient = %d, want 404", resp.Code)
	}

	// A patient without no-shows: zero, no date.
	clean := s.patient(s.own.ID, "CLEAN")
	resp = h.GetPatientNoShows(s.ctx(clean.ID, nil))
	if got := resp.Data.(clinical_dto.PatientNoShowsDTO); resp.Code != http.StatusOK || got.Count != 0 || got.LastDate != nil {
		t.Fatalf("clean patient = %d %+v", resp.Code, resp.Data)
	}
}
