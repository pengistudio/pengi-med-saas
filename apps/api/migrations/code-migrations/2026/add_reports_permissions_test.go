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

func TestAddReportsPermissions_AdminAndFeatureOnly(t *testing.T) {
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
		if err := database.GlobalDBMap[addReportsPermissionsID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	var feature company_models.Feature
	if err := db.Where(company_models.Feature{Code: "REPORTS"}).First(&feature).Error; err != nil {
		t.Fatalf("REPORTS feature not created: %v", err)
	}
	has := func(model any) int {
		var perms []permission_models.Permission
		if err := db.Model(model).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, p := range perms {
			if p.ID == "READ_REPORTS" {
				n++
			}
		}
		return n
	}
	if has(&feature) != 1 {
		t.Error("REPORTS feature must hold READ_REPORTS once")
	}
	if has(roles[role_data.RoleAdmin]) != 1 {
		t.Error("admin must hold READ_REPORTS once")
	}
	for _, name := range []string{role_data.RoleDoctor, role_data.RoleRecepcionista, role_data.RoleContador} {
		if has(roles[name]) != 0 {
			t.Errorf("%s must not hold READ_REPORTS", name)
		}
	}
}
