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

func TestAddAppointmentRequestPermissions_RolesAndPortalFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	roles := map[string]*user_models.Role{}
	for _, name := range []string{role_data.RoleAdmin, role_data.RoleDoctor, role_data.RoleRecepcionista, role_data.RoleContador} {
		r := user_models.Role{Role: name}
		if err := db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
		roles[name] = &r
	}
	// The portal's Feature exists already (DB20261026_1).
	if err := database.GlobalDBMap[addPatientPortalPermissionsID].Execute(db); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // idempotent
		if err := database.GlobalDBMap[addAppointmentRequestPermissionsID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var feature company_models.Feature
	if err := db.Where("code = ?", "PATIENT_PORTAL").First(&feature).Error; err != nil {
		t.Fatal(err)
	}
	has := func(model any) bool {
		var perms []permission_models.Permission
		if err := db.Model(model).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		for _, p := range perms {
			if p.ID == "MANAGE_APPOINTMENT_REQUESTS" && p.Category == "PATIENT_PORTAL" {
				return true
			}
		}
		return false
	}
	if !has(&feature) {
		t.Fatal("MANAGE_APPOINTMENT_REQUESTS must be on the PATIENT_PORTAL feature")
	}
	for _, name := range []string{role_data.RoleAdmin, role_data.RoleDoctor, role_data.RoleRecepcionista} {
		if !has(roles[name]) {
			t.Fatalf("%s must have MANAGE_APPOINTMENT_REQUESTS", name)
		}
	}
	if has(roles[role_data.RoleContador]) {
		t.Fatal("contador must not have MANAGE_APPOINTMENT_REQUESTS")
	}
	var count int64
	db.Table("feature_permissions").Where("permission_id = ?", "MANAGE_APPOINTMENT_REQUESTS").Count(&count)
	if count > 1 {
		t.Fatalf("feature association duplicated: %d", count)
	}
}
