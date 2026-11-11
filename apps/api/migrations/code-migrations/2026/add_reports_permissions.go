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

const addReportsPermissionsID = "DB20261111_1"

func init() {
	// Creates READ_REPORTS, gives it to the admin role (no other role holds it:
	// RolePermissionMatrix lists none) and creates the REPORTS Feature with it
	// so plans that include the Feature (RequirePermission) open the reports. No
	// plan gets the Feature here: backoffice adds it to the top plan.
	database.GlobalDBMap[addReportsPermissionsID] = database.DBExecute{
		ID:      addReportsPermissionsID,
		Execute: addReportsPermissions,
	}
}

func addReportsPermissions(db *gorm.DB) error {
	var perms []*permission_models.Permission
	for _, seed := range permission_data.ReportsPermissions {
		p := seed
		if err := db.Where(permission_models.Permission{BaseStringID: p.BaseStringID}).FirstOrCreate(&p).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", p.ID, err)
		}
		perms = append(perms, &p)
	}

	var admin user_models.Role
	switch err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&admin).Error; {
	case errors.Is(err, gorm.ErrRecordNotFound):
		fmt.Println("⚠️  admin role not found, skipping its reports permissions.")
	case err != nil:
		return fmt.Errorf("failed to find admin role: %w", err)
	default:
		for _, p := range perms {
			if err := db.Model(&admin).Association("Permissions").Append(p); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to admin: %w", p.ID, err)
			}
		}
	}

	feature := company_models.Feature{Code: "REPORTS", Name: "Reportes"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to create REPORTS feature: %w", err)
	}
	for _, p := range perms {
		if err := db.Model(&feature).Association("Permissions").Append(p); err != nil {
			return fmt.Errorf("failed to associate permission '%s' with REPORTS feature: %w", p.ID, err)
		}
	}
	fmt.Println("✅ Reports permission and feature created.")
	return nil
}
