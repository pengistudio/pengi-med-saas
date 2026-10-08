package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	integration_models "pengi-med-saas/features/integrations/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// isolation holds two clinics. Every test acts as a member of "own" and tries
// to reach data that belongs to "other" by its ID.
type isolation struct {
	t            *testing.T
	db           *gorm.DB
	own, other   tenant_models.Tenant
	ownPatient   clinical_models.Patient
	otherPatient clinical_models.Patient
}

func newIsolation(t *testing.T) *isolation {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.MedicalRecord{}, &clinical_models.VitalSigns{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{}, &integration_models.TenantIntegration{})
	now := time.Now().UnixNano()
	s := &isolation{t: t, db: db}
	for i, tenant := range []*tenant_models.Tenant{&s.own, &s.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("iso-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-iso-%d-%d", i, now)}
		if err := db.Create(tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
	}
	s.ownPatient = s.patient(s.own.ID, "OWN")
	s.otherPatient = s.patient(s.other.ID, "OTHER")
	return s
}

func (s *isolation) patient(tenantID uint, tag string) clinical_models.Patient {
	p := clinical_models.Patient{TenantID: tenantID, FirstName: tag, LastName: "Paciente", Institution: "H", Document: fmt.Sprintf("DOC-%s-%d", tag, time.Now().UnixNano())}
	if err := s.db.Create(&p).Error; err != nil {
		s.t.Fatalf("create patient: %v", err)
	}
	return p
}

func (s *isolation) otherAppointment() clinical_models.Appointment {
	a := clinical_models.Appointment{TenantID: s.other.ID, PatientID: s.otherPatient.ID, Title: "Control", Date: time.Now(), StartTime: "09:00", EndTime: "09:30", Status: "scheduled"}
	if err := s.db.Create(&a).Error; err != nil {
		s.t.Fatalf("create appointment: %v", err)
	}
	return a
}

// ctx is a request context of a member of "own", with :id set and an optional JSON body.
func (s *isolation) ctx(id uint, body any) *gin.Context {
	c, _ := testutils.NewGinContext(s.own.ID, 1)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func (s *isolation) reloadAppointment(id uint) clinical_models.Appointment {
	var a clinical_models.Appointment
	if err := s.db.Unscoped().First(&a, id).Error; err != nil {
		s.t.Fatalf("reload appointment: %v", err)
	}
	return a
}

// ─── Appointments ────────────────────────────────────────────────────────────

func TestAppointments_AnotherTenantsAppointmentIsNotFound(t *testing.T) {
	s := newIsolation(t)
	h := NewAppointmentHandler(s.db, zap.NewNop())
	target := s.otherAppointment()
	newTitle := "hijacked"

	cases := map[string]func() int{
		"get":    func() int { return h.GetAppointment(s.ctx(target.ID, nil)).Code },
		"update": func() int { return h.UpdateAppointment(s.ctx(target.ID, map[string]any{"title": newTitle})).Code },
		"status": func() int { return h.UpdateStatus(s.ctx(target.ID, map[string]any{"status": "cancelled"})).Code },
		"delete": func() int { return h.DeleteAppointment(s.ctx(target.ID, nil)).Code },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			if code := call(); code != 404 {
				t.Fatalf("code = %d, want 404", code)
			}
			got := s.reloadAppointment(target.ID)
			if got.DeletedAt.Valid || got.Status != "scheduled" || got.Title != "Control" {
				t.Fatalf("another tenant's appointment was changed: %+v", got)
			}
		})
	}
}

func TestAppointments_CannotBeLinkedToAnotherTenantsPatient(t *testing.T) {
	s := newIsolation(t)
	h := NewAppointmentHandler(s.db, zap.NewNop())

	create := h.CreateAppointment(s.ctx(0, map[string]any{
		"patient_id": s.otherPatient.ID, "title": "x", "date": time.Now().Add(24 * time.Hour), "start_time": "10:00", "end_time": "10:30",
	}))
	if create.Code != 404 {
		t.Fatalf("create code = %d, want 404", create.Code)
	}

	own := clinical_models.Appointment{TenantID: s.own.ID, PatientID: s.ownPatient.ID, Title: "Mine", Date: time.Now(), StartTime: "11:00", EndTime: "11:30", Status: "scheduled"}
	s.db.Create(&own)
	update := h.UpdateAppointment(s.ctx(own.ID, map[string]any{"patient_id": s.otherPatient.ID}))
	if update.Code != 404 {
		t.Fatalf("update code = %d, want 404", update.Code)
	}
	if got := s.reloadAppointment(own.ID); got.PatientID != s.ownPatient.ID {
		t.Fatalf("appointment was re-linked to patient %d", got.PatientID)
	}
}

// ─── Vital signs ─────────────────────────────────────────────────────────────

func TestVitalSigns_AnotherTenantsRecordIsNotFound(t *testing.T) {
	s := newIsolation(t)
	h := NewVitalSignsHandler(s.db, zap.NewNop())
	record := clinical_models.MedicalRecord{TenantID: s.other.ID, PatientID: s.otherPatient.ID}
	if err := s.db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	weight := 70.0
	s.db.Create(&clinical_models.VitalSigns{MedicalRecordID: &record.ID, Weight: &weight})

	if code := h.GetVitalSigns(s.ctx(record.ID, nil)).Code; code != 404 {
		t.Fatalf("get code = %d, want 404", code)
	}
	if code := h.UpsertVitalSigns(s.ctx(record.ID, map[string]any{"weight": 99.0})).Code; code != 404 {
		t.Fatalf("upsert code = %d, want 404", code)
	}
	var vs clinical_models.VitalSigns
	s.db.Where("medical_record_id = ?", record.ID).First(&vs)
	if vs.Weight == nil || *vs.Weight != 70 {
		t.Fatalf("another tenant's vital signs were changed: %v", vs.Weight)
	}
}

// ─── Medical records ─────────────────────────────────────────────────────────

func TestMedicalRecords_CannotCompleteAnotherTenantsAppointment(t *testing.T) {
	s := newIsolation(t)
	h := NewMedicalRecordHandler(s.db, zap.NewNop())
	record := clinical_models.MedicalRecord{TenantID: s.own.ID, PatientID: s.ownPatient.ID, Motive: "Control"}
	if err := s.db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	target := s.otherAppointment()

	if code := h.UpdateMedicalRecord(s.ctx(record.ID, map[string]any{"appointment_id": target.ID})).Code; code != 404 {
		t.Fatalf("code = %d, want 404", code)
	}
	if got := s.reloadAppointment(target.ID); got.Status != "scheduled" {
		t.Fatalf("another tenant's appointment status = %q, want scheduled", got.Status)
	}
}

func TestMedicalRecords_CannotBeCreatedForAnotherTenantsPatient(t *testing.T) {
	s := newIsolation(t)
	h := NewMedicalRecordHandler(s.db, zap.NewNop())

	response := h.CreateMedicalRecord(s.ctx(0, map[string]any{
		"date": "2026-09-26", "motive": "Primera vez", "patient_id": s.otherPatient.ID, "visit_type": "first", "allergies": "Penicilina",
	}))

	if response.Code != 404 {
		t.Fatalf("code = %d, want 404", response.Code)
	}
	var count int64
	s.db.Model(&clinical_models.MedicalRecord{}).Where("patient_id = ?", s.otherPatient.ID).Count(&count)
	if count != 0 {
		t.Fatalf("a medical record was created for another tenant's patient")
	}
	var patient clinical_models.Patient
	s.db.First(&patient, s.otherPatient.ID)
	if patient.Allergies != "" {
		t.Fatalf("another tenant's patient was modified: allergies = %q", patient.Allergies)
	}
}

func TestMedicalRecords_OwnPatientStillWorks(t *testing.T) {
	s := newIsolation(t)
	h := NewMedicalRecordHandler(s.db, zap.NewNop())

	response := h.CreateMedicalRecord(s.ctx(0, map[string]any{
		"date": "2026-09-26", "motive": "Primera vez", "observation": "", "patient_id": s.ownPatient.ID, "visit_type": "first",
	}))
	if response.Code != 200 {
		t.Fatalf("code = %d (%v), want 200", response.Code, response.Message)
	}
}
