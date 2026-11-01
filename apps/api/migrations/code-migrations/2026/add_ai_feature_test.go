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

func TestAddAIFeature_AdminAndDoctorAndOwnFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	admin := user_models.Role{Role: role_data.RoleAdmin}
	doctor := user_models.Role{Role: role_data.RoleDoctor}
	recep := user_models.Role{Role: role_data.RoleRecepcionista}
	for _, v := range []any{&admin, &doctor, &recep} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ { // idempotent
		if err := database.GlobalDBMap[addAIFeatureID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var feature company_models.Feature
	if err := db.Where("code = ?", "AI").First(&feature).Error; err != nil {
		t.Fatalf("AI feature: %v", err)
	}
	ids := func(model any) map[string]bool {
		var perms []permission_models.Permission
		if err := db.Model(model).Association("Permissions").Find(&perms); err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, p := range perms {
			if p.Category == "AI" {
				out[p.ID] = true
			}
		}
		return out
	}
	for name, model := range map[string]any{"admin": &admin, "doctor": &doctor, "feature": &feature} {
		got := ids(model)
		if len(got) != 2 || !got["USE_AI_SUMMARY"] || !got["USE_AI_SCRIBE"] {
			t.Errorf("%s AI permissions = %v", name, got)
		}
	}
	if got := ids(&recep); len(got) != 0 {
		t.Errorf("recepcionista must not get AI permissions: %v", got)
	}
}
