package clinical_handlers

import (
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"

	"go.uber.org/zap"
)

// The body only carries the measurements: an ID or soft-delete timestamp sent
// by the client is ignored on create and on update.
func TestUpsertVitalSigns_IgnoresClientModelFields(t *testing.T) {
	s := newIsolation(t)
	h := NewVitalSignsHandler(s.db, zap.NewNop())
	record := clinical_models.MedicalRecord{TenantID: s.own.ID, PatientID: s.ownPatient.ID}
	if err := s.db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	stored := func() clinical_models.VitalSigns {
		var vs []clinical_models.VitalSigns
		s.db.Unscoped().Where("medical_record_id = ?", record.ID).Find(&vs)
		if len(vs) != 1 {
			t.Fatalf("vital signs rows = %d, want 1", len(vs))
		}
		return vs[0]
	}

	for _, weight := range []float64{80, 81} { // create, then update
		body := map[string]any{"ID": 9999, "DeletedAt": "2020-01-01T00:00:00Z", "weight": weight}
		if resp := h.UpsertVitalSigns(s.ctx(record.ID, body)); resp.Code != 200 {
			t.Fatalf("upsert weight %v = %d %q", weight, resp.Code, resp.Message)
		}
		vs := stored()
		if vs.ID == 9999 || vs.DeletedAt.Valid {
			t.Fatalf("client-set fields were stored: id=%d deleted=%v", vs.ID, vs.DeletedAt.Valid)
		}
		if vs.Weight == nil || *vs.Weight != weight {
			t.Fatalf("weight = %v, want %v", vs.Weight, weight)
		}
	}
}

// Creating the record from an appointment links the vital signs taken at
// triage (one row), with the doctor's corrections when the form sends them.
func TestCreateMedicalRecord_LinksTriageVitalSigns(t *testing.T) {
	for _, tc := range []struct {
		name       string
		sent       map[string]any // vital_signs in the record form; nil = none
		wantWeight float64
	}{
		{"kept as taken", nil, 70},
		{"corrected by the doctor", map[string]any{"weight": 72.5}, 72.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newIsolation(t)
			appt := clinical_models.Appointment{TenantID: s.own.ID, PatientID: s.ownPatient.ID, Title: "Control", Date: time.Now(), StartTime: "09:00", EndTime: "09:30", Status: "arrived"}
			if err := s.db.Create(&appt).Error; err != nil {
				t.Fatal(err)
			}
			if resp := NewVitalSignsHandler(s.db, zap.NewNop()).UpsertAppointmentVitalSigns(s.ctx(appt.ID, map[string]any{"weight": 70.0, "blood_pressure": "120/80"})); resp.Code != 200 {
				t.Fatalf("triage = %d %q", resp.Code, resp.Message)
			}

			body := map[string]any{"date": "2026-09-26", "motive": "Control", "observation": "", "patient_id": s.ownPatient.ID, "visit_type": "followup", "appointment_id": appt.ID}
			if tc.sent != nil {
				body["vital_signs"] = tc.sent
			}
			resp := NewMedicalRecordHandler(s.db, zap.NewNop()).CreateMedicalRecord(s.ctx(0, body))
			if resp.Code != 200 {
				t.Fatalf("create record = %d %q", resp.Code, resp.Message)
			}
			record := resp.Data.(*clinical_models.MedicalRecord)

			var rows []clinical_models.VitalSigns
			s.db.Where("appointment_id = ? OR medical_record_id = ?", appt.ID, record.ID).Find(&rows)
			if len(rows) != 1 {
				t.Fatalf("vital signs rows = %d, want the triage row only", len(rows))
			}
			vs := rows[0]
			if vs.MedicalRecordID == nil || *vs.MedicalRecordID != record.ID || vs.AppointmentID == nil || *vs.AppointmentID != appt.ID {
				t.Fatalf("triage row not linked: record=%v appointment=%v", vs.MedicalRecordID, vs.AppointmentID)
			}
			if vs.Weight == nil || *vs.Weight != tc.wantWeight || vs.BloodPressure != "120/80" {
				t.Fatalf("measurements = weight %v bp %q, want %v / 120/80", vs.Weight, vs.BloodPressure, tc.wantWeight)
			}
		})
	}
}
