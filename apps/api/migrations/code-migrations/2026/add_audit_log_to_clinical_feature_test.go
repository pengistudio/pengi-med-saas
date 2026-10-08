package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	"pengi-med-saas/testutils"
)

// Every plan with the clinical module gets READ_AUDIT_LOG; running it again
// changes nothing.
func TestAddAuditLogToClinicalFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &company_models.Feature{})
	perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: "READ_AUDIT_LOG"}, Name: "Read Audit Log", Category: "AUDIT"}
	clinical := company_models.Feature{Code: "CLINICAL", Name: "Clinical"}
	billing := company_models.Feature{Code: "BILLING", Name: "Billing"}
	for _, row := range []any{&perm, &clinical, &billing} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		if err := addAuditLogToClinicalFeature(db); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		feature company_models.Feature
		want    int64
	}{{clinical, 1}, {billing, 0}} {
		if got := db.Model(&tc.feature).Where("id = ?", "READ_AUDIT_LOG").Association("Permissions").Count(); got != tc.want {
			t.Errorf("%s has READ_AUDIT_LOG %d times, want %d", tc.feature.Code, got, tc.want)
		}
	}
}
