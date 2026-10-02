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

// Creates MANAGE_DOCUMENT_TEMPLATES and gives it to the admin role only:
// templates are a tenant-wide setting, the other canonical roles don't get it.
func init() {
	database.GlobalDBMap["DB20261002_1"] = database.DBExecute{
		ID: "DB20261002_1",
		Execute: func(db *gorm.DB) error {
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			for _, perm := range permission_data.DocumentTemplatePermissions {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)

				if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
				}
				fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
			}
			return nil
		},
	}

	// Associates MANAGE_DOCUMENT_TEMPLATES with the CLINICAL and BILLING
	// Features, so RequirePermission lets through every plan that has either
	// (prescriptions are clinical; invoice RIDE templates will be billing).
	// Category DOCUMENTS maps to no EnabledFeatures flag, so this doesn't turn
	// on clinical/billing for a plan that lacked it.
	database.GlobalDBMap["DB20261002_2"] = database.DBExecute{
		ID: "DB20261002_2",
		Execute: func(db *gorm.DB) error {
			var perm permission_models.Permission
			if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: "MANAGE_DOCUMENT_TEMPLATES"}}).First(&perm).Error; err != nil {
				return fmt.Errorf("failed to find permission 'MANAGE_DOCUMENT_TEMPLATES': %w", err)
			}

			for _, code := range []string{"CLINICAL", "BILLING"} {
				var feature company_models.Feature
				if err := db.Where(company_models.Feature{Code: code}).First(&feature).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						fmt.Printf("⚠️  %s feature not found, skipping MANAGE_DOCUMENT_TEMPLATES wiring.\n", code)
						continue
					}
					return fmt.Errorf("failed to find %s feature: %w", code, err)
				}
				if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to associate 'MANAGE_DOCUMENT_TEMPLATES' with %s feature: %w", code, err)
				}
				fmt.Printf("✅ Associated permission 'MANAGE_DOCUMENT_TEMPLATES' with %s feature.\n", code)
			}
			return nil
		},
	}
}
