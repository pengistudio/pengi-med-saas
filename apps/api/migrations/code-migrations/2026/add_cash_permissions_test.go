package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

func TestAddCashPermissions_RolesAndFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	roles := map[string]*user_models.Role{}
	for _, name := range role_data.CanonicalRoles {
		role := user_models.Role{Role: name}
		if err := db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		roles[name] = &role
	}

	for i := 0; i < 2; i++ { // idempotent
		for _, id := range []string{addCashPermissionsID, addCashFeaturePermissionsID} {
			if err := database.GlobalDBMap[id].Execute(db); err != nil {
				t.Fatalf("run %d of %s: %v", i+1, id, err)
			}
		}
	}

	// The migration creates the CASH Feature itself, once.
	var feature company_models.Feature
	if err := db.Where(company_models.Feature{Code: "CASH"}).First(&feature).Error; err != nil {
		t.Fatalf("CASH feature not created: %v", err)
	}
	var featureCount int64
	db.Model(&company_models.Feature{}).Where("code = ?", "CASH").Count(&featureCount)
	if featureCount != 1 {
		t.Fatalf("want 1 CASH feature, got %d", featureCount)
	}

	ids := func(model any) map[string]bool {
		var perms []permission_models.Permission
		if err := db.Model(model).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, p := range perms {
			if p.Category == "CASH" {
				out[p.ID] = true
			}
		}
		return out
	}
	all := []string{"READ_CASH", "CREATE_PAYMENT", "MANAGE_CASH_SESSION", "VOID_PAYMENT"}
	want := map[string][]string{
		"feature":                   all,
		role_data.RoleAdmin:         all,
		role_data.RoleRecepcionista: {"READ_CASH", "CREATE_PAYMENT", "MANAGE_CASH_SESSION"},
		role_data.RoleContador:      {"READ_CASH"},
		role_data.RoleDoctor:        nil,
	}
	for name, perms := range want {
		var model any = roles[name]
		if name == "feature" {
			model = &feature
		}
		got := ids(model)
		if len(got) != len(perms) {
			t.Errorf("%s cash permissions = %v, want %v", name, got, perms)
			continue
		}
		for _, id := range perms {
			if !got[id] {
				t.Errorf("%s lacks %s (has %v)", name, id, got)
			}
		}
	}
}
