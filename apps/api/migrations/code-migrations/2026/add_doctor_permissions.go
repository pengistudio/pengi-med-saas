package y2026

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const addDoctorPermissionsID = "DB20261008_1"

func init() {
	// Creates MANAGE_DOCTORS, gives it to the admin role only (doctors edit
	// their own profile through /doctors/me, which needs no permission) and
	// puts it in the CLINICAL Feature so plans that include clinical get it.
	// Self-sufficient and idempotent: a missing admin role or CLINICAL
	// Feature is skipped with a warning.
	database.GlobalDBMap[addDoctorPermissionsID] = database.DBExecute{
		ID:      addDoctorPermissionsID,
		Execute: addDoctorPermissions,
	}
}

func addDoctorPermissions(db *gorm.DB) error {
	var adminRole *user_models.Role
	var role user_models.Role
	switch err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&role).Error; {
	case err == nil:
		adminRole = &role
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Println("⚠️  admin role not found, skipping MANAGE_DOCTORS role assignment.")
	default:
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	var clinical *company_models.Feature
	var feature company_models.Feature
	switch err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; {
	case err == nil:
		clinical = &feature
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Println("⚠️  CLINICAL feature not found, skipping doctor permission wiring.")
	default:
		return fmt.Errorf("failed to find CLINICAL feature: %w", err)
	}

	for _, perm := range permission_data.DoctorPermissions {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)
		if adminRole != nil {
			if err := db.Model(adminRole).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
			}
			fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
		}
		if clinical != nil {
			if err := db.Model(clinical).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to associate permission '%s' with CLINICAL feature: %w", perm.ID, err)
			}
			fmt.Printf("✅ Associated permission '%s' with CLINICAL feature.\n", perm.ID)
		}
	}
	return nil
}
