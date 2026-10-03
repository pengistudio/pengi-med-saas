package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	"pengi-med-saas/testutils"
)

func TestSeedPlanStorageQuotas_ClinicalPlansByPriceOthersZero(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &company_models.Feature{}, &company_models.Plan{})
	clinical := company_models.Feature{Code: "CLINICAL", Name: "Clinical"}
	billing := company_models.Feature{Code: "BILLING", Name: "Billing"}
	for _, f := range []*company_models.Feature{&clinical, &billing} {
		if err := db.Create(f).Error; err != nil {
			t.Fatal(err)
		}
	}
	plans := []struct {
		code     string
		price    float64
		features []company_models.Feature
		preset   int64
		want     int64
	}{
		{"PRO", 30, []company_models.Feature{clinical, billing}, 0, 10240},
		{"TRIAL", 0, []company_models.Feature{clinical}, 0, 2048},
		{"BIG", 90, []company_models.Feature{clinical}, 0, 51200},
		{"HUGE", 120, []company_models.Feature{clinical}, 0, 51200},
		{"EDITED", 200, []company_models.Feature{clinical}, 777, 777},
		{"BILLING_ONLY", 10, []company_models.Feature{billing}, 0, 0},
	}
	for _, p := range plans {
		plan := company_models.Plan{Name: p.code, Code: p.code, Price: p.price, StorageQuotaMB: p.preset}
		if err := db.Create(&plan).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&plan).Association("Features").Replace(p.features); err != nil {
			t.Fatal(err)
		}
	}

	m, ok := database.GlobalDBMap["DB20261002_7"]
	if !ok {
		t.Fatal("migration DB20261002_7 not registered")
	}
	if err := m.Execute(db); err != nil {
		t.Fatal(err)
	}

	for _, p := range plans {
		var plan company_models.Plan
		if err := db.Where("code = ?", p.code).First(&plan).Error; err != nil {
			t.Fatal(err)
		}
		if plan.StorageQuotaMB != p.want {
			t.Errorf("%s: storage_quota_mb = %d, want %d", p.code, plan.StorageQuotaMB, p.want)
		}
	}
}
