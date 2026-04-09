package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260409_2"] = database.DBExecute{
		ID: "DB20260409_2",
		Execute: func(db *gorm.DB) error {
			// Retrieve the Admin Role to attach the new permissions
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			// Create Team Permissions
			for _, perm := range permission_data.TeamPermissions {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)

				// Assign permission to admin role
				if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
				}
				fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
			}
			return nil
		},
	}
}
