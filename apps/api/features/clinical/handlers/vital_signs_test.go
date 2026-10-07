package clinical_handlers

import (
	"testing"

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
