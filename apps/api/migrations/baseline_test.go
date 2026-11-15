package migrations

import (
	"testing"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

// A fresh database migrates without replaying the code migrations: it gets
// the baseline and every migration recorded as executed, and a second run
// is a no-op.
func TestRunMigrations_FreshDatabaseGetsBaseline(t *testing.T) {
	db := testutils.SetupTestDB(t)
	for i := 0; i < 2; i++ {
		if err := RunMigrations(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	var executed int64
	if err := db.Model(&database.DBExecute{}).Count(&executed).Error; err != nil {
		t.Fatal(err)
	}
	if int(executed) != len(database.GlobalDBMap) {
		t.Errorf("recorded %d migrations, want %d", executed, len(database.GlobalDBMap))
	}
	// Created by a code migration (not a struct tag), so the baseline runs it.
	if !db.Migrator().HasIndex(&whatsapp_models.WhatsAppMessage{}, whatsapp_models.WamIDUniqueIndex) {
		t.Error("fresh database lacks the unique WhatsApp message id index")
	}

	rolePerms := func(name string) map[string]bool {
		var role user_models.Role
		if err := db.Preload("Permissions").Where(user_models.Role{Role: name}).First(&role).Error; err != nil {
			t.Fatalf("role %s: %v", name, err)
		}
		ids := map[string]bool{}
		for _, p := range role.Permissions {
			ids[p.ID] = true
		}
		return ids
	}
	if got, want := len(rolePerms(role_data.RoleAdmin)), len(permission_data.All()); got != want {
		t.Errorf("admin has %d permissions, want every one (%d)", got, want)
	}
	for _, name := range []string{role_data.RoleDoctor, role_data.RoleRecepcionista, role_data.RoleContador} {
		got := rolePerms(name)
		if len(got) != len(role_data.RolePermissionMatrix[name]) {
			t.Errorf("%s has %d permissions, want %d", name, len(got), len(role_data.RolePermissionMatrix[name]))
		}
		if got["MANAGE_CLINIC_SETTINGS"] {
			t.Errorf("%s must not manage the clinic settings", name)
		}
	}

	var features int64
	if err := db.Model(&company_models.Feature{}).Count(&features).Error; err != nil {
		t.Fatal(err)
	}
	if int(features) != len(permission_data.Features) {
		t.Errorf("seeded %d features, want %d", features, len(permission_data.Features))
	}

	var trial company_models.Plan
	if err := db.Preload("Features").Where(company_models.Plan{Code: "TRIAL"}).First(&trial).Error; err != nil {
		t.Fatalf("trial plan: %v", err)
	}
	if len(trial.Features) != len(trialPlanFeatures) || trial.StorageQuotaMB == 0 {
		t.Errorf("trial plan: %d features, quota %d MB; want %v and a quota", len(trial.Features), trial.StorageQuotaMB, trialPlanFeatures)
	}

	// Caja is top-plan only: the trial plan must not reach it by feature or permission.
	for _, f := range trial.Features {
		if f.Code == "CASH" {
			t.Errorf("trial plan includes the CASH feature")
		}
	}
	var withPerms company_models.Plan
	if err := db.Preload("Features.Permissions").First(&withPerms, trial.ID).Error; err != nil {
		t.Fatal(err)
	}
	for _, f := range withPerms.Features {
		for _, p := range f.Permissions {
			if p.Category == "CASH" {
				t.Errorf("trial plan feature %s grants cash permission %s", f.Code, p.ID)
			}
		}
	}
}
