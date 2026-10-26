package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

func TestAddPatientPortalPermissions_AdminOnlyAndOwnFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	admin := user_models.Role{Role: role_data.RoleAdmin}
	doctor := user_models.Role{Role: role_data.RoleDoctor}
	for _, v := range []any{&admin, &doctor} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ { // idempotent
		if err := database.GlobalDBMap[addPatientPortalPermissionsID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var feature company_models.Feature
	if err := db.Where("code = ?", "PATIENT_PORTAL").First(&feature).Error; err != nil {
		t.Fatalf("PATIENT_PORTAL feature: %v", err)
	}
	has := func(model any) bool {
		var perms []permission_models.Permission
		if err := db.Model(model).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		for _, p := range perms {
			if p.ID == "MANAGE_PATIENT_PORTAL" && p.Category == "PATIENT_PORTAL" {
				return true
			}
		}
		return false
	}
	if !has(&admin) || !has(&feature) {
		t.Fatal("MANAGE_PATIENT_PORTAL must be on the admin role and the PATIENT_PORTAL feature")
	}
	if has(&doctor) {
		t.Fatal("MANAGE_PATIENT_PORTAL is admin-only")
	}
}

func TestBackfillPatientPortalFlag(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	tenant := tenant_models.Tenant{Name: "A", Slug: "pp-backfill", DisplayToken: "pp-backfill", EnabledFeatures: `{"clinical":true}`}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GlobalDBMap[backfillPatientPortalFeatureFlag].Execute(db); err != nil {
		t.Fatal(err)
	}
	var got tenant_models.Tenant
	db.First(&got, tenant.ID)
	if got.EnabledFeatures != `{"clinical":true,"patient_portal":false}` {
		t.Fatalf("enabled_features = %s", got.EnabledFeatures)
	}
}
