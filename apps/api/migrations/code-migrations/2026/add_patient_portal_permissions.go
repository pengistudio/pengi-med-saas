package y2026

import (
	"encoding/json"
	"errors"
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const (
	addPatientPortalPermissionsID    = "DB20261026_1"
	backfillPatientPortalFeatureFlag = "DB20261026_2"
)

func init() {
	// Creates MANAGE_PATIENT_PORTAL, gives it to the admin role only (turning
	// the portal on is a clinic-wide decision) and puts it in a PATIENT_PORTAL
	// Feature that backoffice can add to plans. No plan gets the Feature here:
	// the portal stays closed until a plan includes it (same as WHATSAPP).
	database.GlobalDBMap[addPatientPortalPermissionsID] = database.DBExecute{
		ID:      addPatientPortalPermissionsID,
		Execute: addPatientPortalPermissions,
	}

	// Writes "patient_portal": false into every tenant's stored
	// enabled_features, so the flag is explicit (an absent key already reads
	// as false).
	database.GlobalDBMap[backfillPatientPortalFeatureFlag] = database.DBExecute{
		ID:      backfillPatientPortalFeatureFlag,
		Execute: backfillPatientPortalFlag,
	}
}

func addPatientPortalPermissions(db *gorm.DB) error {
	var adminRole *user_models.Role
	var role user_models.Role
	switch err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&role).Error; {
	case err == nil:
		adminRole = &role
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Println("⚠️  admin role not found, skipping MANAGE_PATIENT_PORTAL role assignment.")
	default:
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	feature := company_models.Feature{Code: "PATIENT_PORTAL", Name: "Portal de pacientes"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to create PATIENT_PORTAL feature: %w", err)
	}

	for _, perm := range permission_data.PatientPortalPermissions {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		if adminRole != nil {
			if err := db.Model(adminRole).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
			}
		}
		if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("failed to associate '%s' with PATIENT_PORTAL feature: %w", perm.ID, err)
		}
		fmt.Printf("✅ Permission '%s' created, assigned to admin and the PATIENT_PORTAL feature.\n", perm.ID)
	}
	return nil
}

func backfillPatientPortalFlag(db *gorm.DB) error {
	var tenants []tenant_models.Tenant
	if err := db.Select("id", "enabled_features").Find(&tenants).Error; err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}
	for _, t := range tenants {
		flags := map[string]any{}
		if t.EnabledFeatures != "" {
			if err := json.Unmarshal([]byte(t.EnabledFeatures), &flags); err != nil {
				fmt.Printf("⚠️  Tenant %d has unreadable enabled_features, skipping.\n", t.ID)
				continue
			}
		}
		if _, ok := flags["patient_portal"]; ok {
			continue
		}
		flags["patient_portal"] = false
		raw, err := json.Marshal(flags)
		if err != nil {
			return err
		}
		if err := db.Model(&tenant_models.Tenant{}).Where("id = ?", t.ID).Update("enabled_features", string(raw)).Error; err != nil {
			return fmt.Errorf("failed to backfill enabled_features of tenant %d: %w", t.ID, err)
		}
	}
	fmt.Println("✅ patient_portal flag backfilled on every tenant.")
	return nil
}
