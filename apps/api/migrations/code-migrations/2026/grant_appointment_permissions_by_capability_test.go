package y2026

import (
	"sort"
	"testing"

	"pengi-med-saas/core/database"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

func createRoleWith(t *testing.T, db *gorm.DB, name string, permIDs ...string) user_models.Role {
	t.Helper()
	role := user_models.Role{Role: name}
	for _, id := range permIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role %s: %v", name, err)
	}
	return role
}

func rolePermissionIDs(t *testing.T, db *gorm.DB, roleID uint) []string {
	t.Helper()
	var role user_models.Role
	if err := db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		t.Fatalf("reload role: %v", err)
	}
	ids := make([]string, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids
}

func has(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func TestGrantAppointmentPermissionsByCapability(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{})
	if _, ok := database.GlobalDBMap[grantAppointmentPermissionsByCapabilityID]; !ok {
		t.Fatalf("migration %s not registered", grantAppointmentPermissionsByCapabilityID)
	}

	nurse := createRoleWith(t, db, "enfermera", "READ_PATIENT")
	auditor := createRoleWith(t, db, "auditor", "READ_MEDICAL_RECORD")
	editor := createRoleWith(t, db, "medico-externo", "READ_MEDICAL_RECORD", "UPDATE_MEDICAL_RECORD")
	contador := createRoleWith(t, db, "contador", "READ_PATIENT", "READ_BILLING")
	contadorCopy := createRoleWith(t, db, "contador", "READ_PATIENT")
	billingOnly := createRoleWith(t, db, "caja", "READ_BILLING")

	// Twice: the second run must change nothing.
	for run := 1; run <= 2; run++ {
		if err := grantAppointmentPermissionsByCapability(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}

	cases := []struct {
		role               user_models.Role
		agenda, vitalSigns bool
	}{
		{nurse, true, false},
		{auditor, true, false},
		{editor, true, true},
		{contador, false, false},
		{contadorCopy, false, false},
		{billingOnly, false, false},
	}
	for _, tc := range cases {
		ids := rolePermissionIDs(t, db, tc.role.ID)
		if got := has(ids, "READ_APPOINTMENT") && has(ids, "MANAGE_APPOINTMENT"); got != tc.agenda {
			t.Errorf("%s (%d): agenda = %v, want %v (%v)", tc.role.Role, tc.role.ID, got, tc.agenda, ids)
		}
		if has(ids, "READ_APPOINTMENT") != has(ids, "MANAGE_APPOINTMENT") {
			t.Errorf("%s: got only one of the appointment permissions: %v", tc.role.Role, ids)
		}
		if got := has(ids, "RECORD_VITAL_SIGNS"); got != tc.vitalSigns {
			t.Errorf("%s (%d): RECORD_VITAL_SIGNS = %v, want %v (%v)", tc.role.Role, tc.role.ID, got, tc.vitalSigns, ids)
		}
	}

	var joins int64
	db.Table("role_permissions").Where("role_id = ?", editor.ID).Count(&joins)
	if joins != 5 {
		t.Fatalf("editor has %d role_permissions rows after two runs, want 5", joins)
	}
}
