package clinical_handlers

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestDeleteMultiplePatients_DeletesOwnTenantOnly(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{},
		&clinical_models.Appointment{}, &clinical_models.MedicalRecord{}) // read by the remaining-list query

	var tenants [2]uint
	for i := range tenants {
		slug := fmt.Sprintf("pat-del-%d-%d", i, time.Now().UnixNano())
		tenant := tenant_models.Tenant{Name: "C", Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		tenants[i] = tenant.ID
	}
	own, other := tenants[0], tenants[1]

	newPatient := func(tenantID uint, doc string) uint {
		p := clinical_models.Patient{TenantID: tenantID, Document: doc, FirstName: "A", LastName: "B", Institution: "H"}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	deleted1, deleted2 := newPatient(own, "DEL-1"), newPatient(own, "DEL-2")
	kept := newPatient(own, "KEEP-1")
	foreign := newPatient(other, "FOREIGN-1")

	c, _ := testutils.NewGinContext(own, 1)
	body := fmt.Sprintf(`{"id_list":[%d,%d,%d]}`, deleted1, deleted2, foreign)
	c.Request = httptest.NewRequest("POST", "/patients/delete-multiple", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	resp := NewPatientHandler(db, zap.NewNop()).DeleteMultiplePatients(c)
	if resp.Code != 200 || resp.Message != "clinical.patient.delete.success" {
		t.Fatalf("response = %d %q", resp.Code, resp.Message)
	}

	var remaining []uint
	if err := db.Model(&clinical_models.Patient{}).Where("id IN ?", []uint{deleted1, deleted2, kept, foreign}).Order("id").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(remaining) != fmt.Sprint([]uint{kept, foreign}) {
		t.Fatalf("remaining = %v, want [%d %d] (own unselected + other tenant's)", remaining, kept, foreign)
	}
}
