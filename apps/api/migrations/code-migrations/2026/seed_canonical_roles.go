package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

// Seeds the fixed, global role catalog (admin, doctor, recepcionista,
// contador). admin keeps the existing convention of dynamically holding
// every permission that exists; the other roles get the explicit set from
// role_data.RolePermissionMatrix. Uses FirstOrCreate + Association.Append,
// both idempotent, so this is safe to re-run.
func init() {
	database.GlobalDBMap["DB20260708_2"] = database.DBExecute{
		ID: "DB20260708_2",
		Execute: func(db *gorm.DB) error {
			permsByID := make(map[string]permission_models.Permission)
			for _, catalog := range [][]permission_models.Permission{
				permission_data.ClinicalPermissions,
				permission_data.BillingPermissions,
				permission_data.TeamPermissions,
				permission_data.AuditPermissions,
				permission_data.KanbanPermissions,
			} {
				for _, perm := range catalog {
					permsByID[perm.ID] = perm
				}
			}

			for _, roleName := range role_data.CanonicalRoles {
				role := user_models.Role{Role: roleName}
				if err := db.Where(user_models.Role{Role: roleName}).FirstOrCreate(&role).Error; err != nil {
					return fmt.Errorf("failed to create role '%s': %w", roleName, err)
				}

				if roleName == role_data.RoleAdmin {
					var allPerms []permission_models.Permission
					if err := db.Find(&allPerms).Error; err != nil {
						return fmt.Errorf("failed to list permissions: %w", err)
					}
					if len(allPerms) > 0 {
						if err := db.Model(&role).Association("Permissions").Append(allPerms); err != nil {
							return fmt.Errorf("failed to assign permissions to role '%s': %w", roleName, err)
						}
					}
					fmt.Printf("✅ Role '%s' created/found with %d permissions.\n", roleName, len(allPerms))
					continue
				}

				permIDs := role_data.RolePermissionMatrix[roleName]
				perms := make([]permission_models.Permission, 0, len(permIDs))
				for _, id := range permIDs {
					perm, ok := permsByID[id]
					if !ok {
						return fmt.Errorf("unknown permission ID '%s' referenced by role '%s'", id, roleName)
					}
					if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
						return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
					}
					perms = append(perms, perm)
				}
				if len(perms) > 0 {
					if err := db.Model(&role).Association("Permissions").Append(perms); err != nil {
						return fmt.Errorf("failed to assign permissions to role '%s': %w", roleName, err)
					}
				}
				fmt.Printf("✅ Role '%s' created/found with %d permissions.\n", roleName, len(perms))
			}

			return nil
		},
	}
}
