package clinical_handlers

import (
	"net/http"
	"strings"
	"testing"

	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ─── Doctor resolution (spec decision 8) ────────────────────────────────────

func (s *isolation) doctorOf(tenantID uint) doctor_models.Doctor {
	s.t.Helper()
	var d doctor_models.Doctor
	if err := s.db.Where("tenant_id = ?", tenantID).Order("id").First(&d).Error; err != nil {
		s.t.Fatal(err)
	}
	return d
}

func (s *isolation) addDoctor(tenantID uint, name string, userID *uint) doctor_models.Doctor {
	s.t.Helper()
	d := doctor_models.Doctor{TenantID: tenantID, UserID: userID, FullName: name, Specialty: "pediatrics", Active: true}
	if err := s.db.Create(&d).Error; err != nil {
		s.t.Fatal(err)
	}
	return d
}

func recordBody(patientID uint, extra map[string]any) map[string]any {
	body := map[string]any{"date": "2026-10-08", "motive": "Control", "observation": "", "patient_id": patientID, "visit_type": "followup"}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func appointmentBody(patientID uint, start, end string, extra map[string]any) map[string]any {
	body := map[string]any{"patient_id": patientID, "title": "Control", "date": "2026-10-20T00:00:00Z", "start_time": start, "end_time": end}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func wantAppError(t *testing.T, code int, data any, gotCode int, want core_errors.AppError) {
	t.Helper()
	if gotCode != code || errorCode(t, data) != want.ErrorCode {
		t.Fatalf("got %d %+v, want %d %s", gotCode, data, code, want.ErrorCode)
	}
}

func TestDoctorResolution_SingleDoctorIsAutoAssigned(t *testing.T) {
	s := newIsolation(t)
	only := s.doctorOf(s.own.ID)

	resp := NewMedicalRecordHandler(s.db, zap.NewNop()).CreateMedicalRecord(s.ctx(0, recordBody(s.ownPatient.ID, map[string]any{
		"prescription": map[string]any{"content": "Paracetamol", "doctor_id": 999},
	})))
	if resp.Code != http.StatusOK {
		t.Fatalf("create record = %d %v", resp.Code, resp.Data)
	}
	record := resp.Data.(*clinical_models.MedicalRecord)
	if record.DoctorID == nil || *record.DoctorID != only.ID {
		t.Fatalf("record doctor = %v, want %d", record.DoctorID, only.ID)
	}
	var rx clinical_models.Prescription
	s.db.First(&rx, *record.PrescriptionID)
	if rx.DoctorID == nil || *rx.DoctorID != only.ID {
		t.Fatalf("prescription doctor = %v, want the record's %d (client value ignored)", rx.DoctorID, only.ID)
	}

	appt := NewAppointmentHandler(s.db, zap.NewNop()).CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "09:00", "09:30", nil)))
	if appt.Code != http.StatusOK || appt.Data.(*clinical_models.Appointment).DoctorID == nil {
		t.Fatalf("create appointment = %d %+v", appt.Code, appt.Data)
	}
}

func TestDoctorResolution_SeveralDoctorsNeedOneOrADefault(t *testing.T) {
	s := newIsolation(t)
	first := s.doctorOf(s.own.ID)
	userID := uint(42)
	mine := s.addDoctor(s.own.ID, "Dr. Mío", &userID)
	h := NewAppointmentHandler(s.db, zap.NewNop())

	resp := h.CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "08:00", "08:30", nil)))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrDoctorSelectionRequired)

	// The patient's cabecera is the default.
	s.db.Model(&clinical_models.Patient{}).Where("id = ?", s.ownPatient.ID).Update("doctor_id", first.ID)
	resp = h.CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "08:00", "08:30", nil)))
	if resp.Code != http.StatusOK || *resp.Data.(*clinical_models.Appointment).DoctorID != first.ID {
		t.Fatalf("cabecera default: %d %+v", resp.Code, resp.Data)
	}

	// The current user's own doctor wins over the cabecera; different doctors may overlap.
	c := s.ctx(0, appointmentBody(s.ownPatient.ID, "08:00", "08:30", nil))
	c.Set("user_id", int64(userID))
	c.Set("username", "dr")
	resp = h.CreateAppointment(c)
	if resp.Code != http.StatusOK || *resp.Data.(*clinical_models.Appointment).DoctorID != mine.ID {
		t.Fatalf("own doctor default: %d %+v", resp.Code, resp.Data)
	}

	// Same doctor, same time: overlap.
	resp = h.CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "08:15", "08:45", map[string]any{"doctor_id": mine.ID})))
	wantAppError(t, http.StatusConflict, resp.Data, resp.Code, core_errors.ErrClinicalAppointmentOverlap)
}

func TestDoctorResolution_NoActiveDoctorBlocksRecordsNotAppointments(t *testing.T) {
	s := newIsolation(t)
	s.db.Model(&doctor_models.Doctor{}).Where("tenant_id = ?", s.own.ID).Update("active", false)

	resp := NewMedicalRecordHandler(s.db, zap.NewNop()).CreateMedicalRecord(s.ctx(0, recordBody(s.ownPatient.ID, nil)))
	wantAppError(t, http.StatusConflict, resp.Data, resp.Code, core_errors.ErrDoctorRegisterFirst)

	appt := NewAppointmentHandler(s.db, zap.NewNop()).CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "09:00", "09:30", nil)))
	if appt.Code != http.StatusOK || appt.Data.(*clinical_models.Appointment).DoctorID != nil {
		t.Fatalf("appointment without doctors = %d %+v", appt.Code, appt.Data)
	}
}

func TestDoctorResolution_RejectsOtherTenantsAndInactiveDoctors(t *testing.T) {
	s := newIsolation(t)
	foreign := s.doctorOf(s.other.ID)
	records := NewMedicalRecordHandler(s.db, zap.NewNop())

	resp := records.CreateMedicalRecord(s.ctx(0, recordBody(s.ownPatient.ID, map[string]any{"doctor_id": foreign.ID})))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrDoctorNotAvailable)

	// An inactive doctor can be kept on update but not newly chosen.
	h := NewAppointmentHandler(s.db, zap.NewNop())
	own := s.doctorOf(s.own.ID)
	created := h.CreateAppointment(s.ctx(0, appointmentBody(s.ownPatient.ID, "11:00", "11:30", nil))).Data.(*clinical_models.Appointment)
	s.db.Model(&own).Update("active", false)
	retired := s.addDoctor(s.own.ID, "Retirado", nil)
	s.db.Model(&retired).Update("active", false)
	s.addDoctor(s.own.ID, "Activo", nil)

	if resp := h.UpdateAppointment(s.ctx(created.ID, map[string]any{"title": "Otro", "doctor_id": own.ID})); resp.Code != http.StatusOK {
		t.Fatalf("keeping the inactive doctor = %d %+v", resp.Code, resp.Data)
	}
	resp = h.UpdateAppointment(s.ctx(created.ID, map[string]any{"doctor_id": retired.ID}))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrDoctorNotAvailable)
	resp = h.UpdateAppointment(s.ctx(created.ID, map[string]any{"doctor_id": foreign.ID}))
	wantAppError(t, http.StatusBadRequest, resp.Data, resp.Code, core_errors.ErrDoctorNotAvailable)
}

// ─── PDFs (decision 9) ──────────────────────────────────────────────────────

func TestDocumentDoctor_FallbackChain(t *testing.T) {
	s := newIsolation(t)
	cabecera := s.doctorOf(s.own.ID)
	s.db.Model(&cabecera).Update("professional_registry", "REG-CAB")
	docDoctor := s.addDoctor(s.own.ID, "Dra. Documento", nil)
	s.db.Model(&docDoctor).Update("professional_registry", "REG-DOC")
	db := s.db.Where("tenant_id = ?", s.own.ID).Session(&gorm.Session{})

	patient := &clinical_models.Patient{Medic: "Dr. Legado", DoctorID: &cabecera.ID}
	if name, reg := documentDoctor(db, patient, &docDoctor.ID); name != "Dra. Documento" || reg != "REG-DOC" {
		t.Fatalf("document doctor: %q %q", name, reg)
	}
	if name, reg := documentDoctor(db, patient, nil); name != cabecera.FullName || reg != "REG-CAB" {
		t.Fatalf("cabecera: %q %q", name, reg)
	}
	if name, reg := documentDoctor(db, &clinical_models.Patient{Medic: " Dr. Legado "}); name != "Dr. Legado" || reg != "" {
		t.Fatalf("legacy medic: %q %q", name, reg)
	}
	if name, _ := documentDoctor(db, &clinical_models.Patient{}); name != defaultDoctorName {
		t.Fatalf("default: %q", name)
	}
}

// ─── Signing (decision 10) ──────────────────────────────────────────────────

func (f *signingFixture) doctor(name string, userID *uint) doctor_models.Doctor {
	f.t.Helper()
	d := doctor_models.Doctor{TenantID: f.tenant.ID, UserID: userID, FullName: name, Specialty: "cardiology", ProfessionalRegistry: "REG-9", Active: true}
	if err := f.db.Create(&d).Error; err != nil {
		f.t.Fatal(err)
	}
	return d
}

func TestSignMedicalReport_PrintsTheReportsDoctorAndOnlyItsUserSigns(t *testing.T) {
	f := newSigningFixture(t)
	f.giveSignature("DRA ANA PEREZ")
	f.db.Model(&f.patient).Update("medic", "Dr. Legado")
	me := uint(f.userID)
	someoneElse := me + 1
	mine := f.doctor("Ana Pérez", &me)
	theirs := f.doctor("Otro Médico", &someoneElse)
	noAccount := f.doctor("Sin Cuenta", nil)

	for _, tc := range []struct {
		name   string
		doctor doctor_models.Doctor
		code   int
		err    core_errors.AppError
	}{
		{"another user's doctor", theirs, http.StatusForbidden, core_errors.ErrDoctorSignNotOwner},
		{"doctor without account", noAccount, http.StatusConflict, core_errors.ErrDoctorSignNoAccount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := f.report()
			f.db.Model(&report).Update("doctor_id", tc.doctor.ID)
			c, _ := f.ctx(report.ID, nil)
			resp := f.docs.SignMedicalReport(c)
			wantAppError(t, tc.code, resp.Data, resp.Code, tc.err)
		})
	}

	report := f.report()
	f.db.Model(&report).Update("doctor_id", mine.ID)
	c, _ := f.ctx(report.ID, nil)
	if resp := f.docs.SignMedicalReport(c); resp.Code != http.StatusOK {
		t.Fatalf("own doctor sign = %d %+v", resp.Code, resp.Data)
	}
	if !strings.Contains(f.conv.html, "Ana Pérez") || !strings.Contains(f.conv.html, "REG-9") || strings.Contains(f.conv.html, "Dr. Legado") {
		t.Fatal("the PDF must print the report's doctor and registry, not patient.medic")
	}
}

func TestSignPrescription_OnlyTheRecordDoctorsUser(t *testing.T) {
	f := newSigningFixture(t)
	other := uint(f.userID) + 1
	theirs := f.doctor("Otro Médico", &other)
	prescription := clinical_models.Prescription{Content: "Paracetamol", DoctorID: &theirs.ID}
	f.db.Create(&prescription)
	soap := clinical_models.SOAPRecord{}
	f.db.Create(&soap)
	record := clinical_models.MedicalRecord{TenantID: f.tenant.ID, PatientID: f.patient.ID, SOAPRecordID: soap.ID, PrescriptionID: &prescription.ID, DoctorID: &theirs.ID}
	f.db.Create(&record)

	h := NewDownloadRecordHandler(f.db, zap.NewNop(), nil, nil, f.files)
	c, _ := f.ctx(record.ID, nil)
	resp := h.SignPrescription(c)
	wantAppError(t, http.StatusForbidden, resp.Data, resp.Code, core_errors.ErrDoctorSignNotOwner)
}
