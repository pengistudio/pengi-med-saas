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

const (
	addClinicSettingsPermissionID = "DB20261028_1"
	clinicSettingsPermission      = "MANAGE_CLINIC_SETTINGS"
)

func init() {
	// Creates MANAGE_CLINIC_SETTINGS (change the clinic-wide UI settings) and
	// gives it to the admin role only. Before, PUT /tenants/settings had no
	// permission check and any role could change what every user sees. It
	// sits in no Feature: the route checks the role alone.
	database.GlobalDBMap[addClinicSettingsPermissionID] = database.DBExecute{
		ID:      addClinicSettingsPermissionID,
		Execute: addClinicSettingsPermission,
	}
}

func addClinicSettingsPermission(db *gorm.DB) error {
	var perm permission_models.Permission
	for _, p := range permission_data.ClinicPermissions {
		if p.ID == clinicSettingsPermission {
			perm = p
		}
	}
	if perm.ID == "" {
		return fmt.Errorf("permission '%s' missing from ClinicPermissions", clinicSettingsPermission)
	}
	if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
		return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
	}

	var adminRole user_models.Role
	switch err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&adminRole).Error; {
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Printf("⚠️  admin role not found, skipping %s role assignment.\n", perm.ID)
		return nil
	case err != nil:
		return fmt.Errorf("failed to find admin role: %w", err)
	}
	if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
		return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
	}
	fmt.Printf("✅ Permission '%s' created and assigned to admin.\n", perm.ID)
	return nil
}
