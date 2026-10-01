package y2026

import (
	"fmt"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

func TestMarkEstimatedBirthDates_MarksAgeDerivedDatesOnlyInTenantsWithAgeInput(t *testing.T) {
	db := tenantdb.System(testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}))
	withAge := createBirthDateTenant(t, db, `{"clinical":{"patient_age_input":true}}`)
	withoutAge := createBirthDateTenant(t, db, `{"clinical":{"patient_age_input":false}}`)

	// Saved on 2026-03-18 at 20:00 in Ecuador, which is already the 19th in UTC.
	saved := time.Date(2026, 3, 19, 1, 0, 0, 0, time.UTC)
	fromAge := time.Date(1969, 3, 18, 5, 0, 0, 0, time.UTC) // 1969-03-18 00:00 in Ecuador
	realDate := time.Date(1968, 12, 15, 5, 0, 0, 0, time.UTC)

	estimated := createBirthDatePatient(t, db, withAge.ID, fromAge, saved)
	exact := createBirthDatePatient(t, db, withAge.ID, realDate, saved)
	noDate := createBirthDatePatient(t, db, withAge.ID, time.Time{}, saved)
	otherTenant := createBirthDatePatient(t, db, withoutAge.ID, fromAge, saved)

	m, ok := database.GlobalDBMap["DB20260929_1"]
	if !ok {
		t.Fatal("migration DB20260929_1 not registered")
	}
	if err := m.Execute(db); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		id   uint
		want bool
	}{
		{"age-derived date", estimated, true},
		{"real date", exact, false},
		{"no date", noDate, false},
		{"tenant without age input", otherTenant, false},
	} {
		var p clinical_models.Patient
		if err := db.Unscoped().First(&p, tc.id).Error; err != nil {
			t.Fatal(err)
		}
		if p.BirthDateEstimated != tc.want {
			t.Errorf("%s: birth_date_estimated = %v, want %v", tc.name, p.BirthDateEstimated, tc.want)
		}
	}
}

func createBirthDateTenant(t *testing.T, db *gorm.DB, uiSettings string) tenant_models.Tenant {
	t.Helper()
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{
		Name:         "Clinic",
		Slug:         fmt.Sprintf("birth-%d", now),
		DisplayToken: fmt.Sprintf("tok-birth-%d", now),
		UISettings:   uiSettings,
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant
}

func createBirthDatePatient(t *testing.T, db *gorm.DB, tenantID uint, birth, saved time.Time) uint {
	t.Helper()
	p := clinical_models.Patient{
		TenantID:  tenantID,
		Document:  fmt.Sprintf("%d", time.Now().UnixNano()),
		FirstName: "Ana",
		LastName:  "Paz",
		BirthDate: birth,
	}
	p.CreatedAt = saved
	p.UpdatedAt = saved
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}
	return p.ID
}
