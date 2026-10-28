package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

func TestAddClinicSettingsPermission_AdminOnly(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{})
	admin := user_models.Role{Role: role_data.RoleAdmin}
	recepcionista := user_models.Role{Role: role_data.RoleRecepcionista}
	for _, v := range []any{&admin, &recepcionista} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ { // idempotent
		if err := database.GlobalDBMap[addClinicSettingsPermissionID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	has := func(role *user_models.Role) bool {
		var perms []permission_models.Permission
		if err := db.Model(role).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		for _, p := range perms {
			if p.ID == clinicSettingsPermission && p.Category == "SETTINGS" {
				return true
			}
		}
		return false
	}
	if !has(&admin) {
		t.Fatalf("%s must be on the admin role", clinicSettingsPermission)
	}
	if has(&recepcionista) {
		t.Fatalf("%s is admin-only", clinicSettingsPermission)
	}
}
