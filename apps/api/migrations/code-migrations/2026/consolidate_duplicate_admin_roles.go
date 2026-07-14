package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

// Self-registration used to create a brand-new "admin" Role row per company
// instead of reusing the single global one. This migration merges any
// duplicates into the oldest (lowest ID) "admin" row before the canonical
// role catalog is seeded (DB20260708_2). Safe to re-run: it's a no-op once
// at most one "admin" role remains.
func init() {
	database.GlobalDBMap["DB20260708_1"] = database.DBExecute{
		ID: "DB20260708_1",
		Execute: func(db *gorm.DB) error {
			var adminRoles []user_models.Role
			if err := db.Where("role = ?", role_data.RoleAdmin).Order("id ASC").Find(&adminRoles).Error; err != nil {
				return fmt.Errorf("failed to list admin roles: %w", err)
			}

			if len(adminRoles) <= 1 {
				fmt.Println("✅ No duplicate admin roles found, skipping consolidation.")
				return nil
			}

			canonical := adminRoles[0]
			fmt.Printf("🔧 Consolidating %d duplicate admin role(s) into canonical role ID %d.\n", len(adminRoles)-1, canonical.ID)

			for _, dup := range adminRoles[1:] {
				// Repoint environments (including soft-deleted ones) before deleting
				// the duplicate role row, to avoid dangling FK references.
				if err := db.Unscoped().
					Model(&user_models.Environment{}).
					Where("role_id = ?", dup.ID).
					Update("role_id", canonical.ID).Error; err != nil {
					return fmt.Errorf("failed to repoint environments off duplicate admin role %d: %w", dup.ID, err)
				}

				if err := db.Exec("DELETE FROM role_permissions WHERE role_id = ?", dup.ID).Error; err != nil {
					return fmt.Errorf("failed to delete role_permissions for duplicate admin role %d: %w", dup.ID, err)
				}

				if err := db.Unscoped().Delete(&user_models.Role{}, dup.ID).Error; err != nil {
					return fmt.Errorf("failed to delete duplicate admin role %d: %w", dup.ID, err)
				}

				fmt.Printf("✅ Consolidated duplicate admin role %d into %d.\n", dup.ID, canonical.ID)
			}

			return nil
		},
	}
}
