package migrations

import (
	"fmt"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

func init() {
	database.Baseline = seedBaseline
}

// baselineMigrations are code migrations that still hold on a fresh database
// (idempotent, written against data the baseline has by then), reused instead
// of copying their data. In order: audit log indexes, trial plan, storage
// quota of the clinical plans, CIE-10 codes, allowed origins, backoffice
// admins and their production password.
var baselineMigrations = []string{
	"DB20260705_1",
	"DB20260518_1",
	"DB20261002_7",
	"DB20260315_CIE10_SEED",
	"DB20260529_1",
	"DB20260304_BACKOFFICE_USERS",
	"DB20260926_1",
}

// trialPlanFeatures are the modules of the plan every signup starts on.
var trialPlanFeatures = []string{"CLINICAL", "KANBAN"}

// seedBaseline gives a fresh database the reference data from the current
// catalogs: permissions, canonical roles, plan features, the trial plan and
// the seeds above. Tenants come from signup; other plans from the backoffice.
func seedBaseline(db *gorm.DB) error {
	permsByID := map[string]permission_models.Permission{}
	for _, perm := range permission_data.All() {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		permsByID[perm.ID] = perm
	}

	for _, roleName := range role_data.CanonicalRoles {
		role := user_models.Role{Role: roleName}
		if err := db.Where(user_models.Role{Role: roleName}).FirstOrCreate(&role).Error; err != nil {
			return fmt.Errorf("failed to create role '%s': %w", roleName, err)
		}
		var perms []permission_models.Permission
		if roleName == role_data.RoleAdmin {
			perms = permission_data.All()
		} else {
			for _, id := range role_data.RolePermissionMatrix[roleName] {
				perm, ok := permsByID[id]
				if !ok {
					return fmt.Errorf("unknown permission ID '%s' referenced by role '%s'", id, roleName)
				}
				perms = append(perms, perm)
			}
		}
		if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
			return fmt.Errorf("failed to assign permissions to role '%s': %w", roleName, err)
		}
	}

	featuresByCode := map[string]company_models.Feature{}
	for _, seed := range permission_data.Features {
		feature := company_models.Feature{Code: seed.Code, Name: seed.Name}
		if err := db.Where(company_models.Feature{Code: seed.Code}).FirstOrCreate(&feature).Error; err != nil {
			return fmt.Errorf("failed to create feature '%s': %w", seed.Code, err)
		}
		if err := db.Model(&feature).Association("Permissions").Replace(seed.Permissions); err != nil {
			return fmt.Errorf("failed to assign permissions to feature '%s': %w", seed.Code, err)
		}
		featuresByCode[seed.Code] = feature
	}

	for _, id := range baselineMigrations[:2] {
		if err := runMigration(db, id); err != nil {
			return err
		}
	}
	var trial company_models.Plan
	if err := db.Where(company_models.Plan{Code: "TRIAL"}).First(&trial).Error; err != nil {
		return fmt.Errorf("failed to find the trial plan: %w", err)
	}
	var trialFeatures []company_models.Feature
	for _, code := range trialPlanFeatures {
		trialFeatures = append(trialFeatures, featuresByCode[code])
	}
	if err := db.Model(&trial).Association("Features").Replace(trialFeatures); err != nil {
		return fmt.Errorf("failed to assign features to the trial plan: %w", err)
	}

	for _, id := range baselineMigrations[2:] {
		if err := runMigration(db, id); err != nil {
			return err
		}
	}
	return nil
}

func runMigration(db *gorm.DB, id string) error {
	migration, ok := database.GlobalDBMap[id]
	if !ok {
		return fmt.Errorf("baseline migration '%s' is not registered", id)
	}
	if err := migration.Execute(db); err != nil {
		return fmt.Errorf("baseline migration '%s': %w", id, err)
	}
	return nil
}
