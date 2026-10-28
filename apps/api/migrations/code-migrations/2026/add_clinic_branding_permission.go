package y2026

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/database"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const addClinicBrandingPermissionID = "DB20261027_1"

func init() {
	// Creates MANAGE_CLINIC_BRANDING (upload the clinic's logo) and gives it to
	// the admin role only. It sits in no Feature: the route checks the role
	// alone, so a plan without billing can still set its logo (before, the
	// upload needed MANAGE_SRI_SETTINGS, a billing permission).
	database.GlobalDBMap[addClinicBrandingPermissionID] = database.DBExecute{
		ID:      addClinicBrandingPermissionID,
		Execute: addClinicBrandingPermission,
	}
}

func addClinicBrandingPermission(db *gorm.DB) error {
	var adminRole *user_models.Role
	var role user_models.Role
	switch err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&role).Error; {
	case err == nil:
		adminRole = &role
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Println("⚠️  admin role not found, skipping MANAGE_CLINIC_BRANDING role assignment.")
	default:
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	for _, perm := range permission_data.ClinicPermissions {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		if adminRole != nil {
			if err := db.Model(adminRole).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
			}
		}
		fmt.Printf("✅ Permission '%s' created and assigned to admin.\n", perm.ID)
	}
	return nil
}
