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

func examOrderPermissions() []permission_models.Permission {
	wanted := map[string]bool{}
	for _, id := range permission_data.ExamOrderPermissionIDs {
		wanted[id] = true
	}
	var out []permission_models.Permission
	for _, perm := range permission_data.ClinicalPermissions {
		if wanted[perm.ID] {
			out = append(out, perm)
		}
	}
	return out
}

func init() {
	// Creates the exam order permissions and gives them to admin (all), doctor
	// (all) and recepcionista (read + upload results), per the exam orders spec.
	database.GlobalDBMap["DB20261002_10"] = database.DBExecute{
		ID: "DB20261002_10",
		Execute: func(db *gorm.DB) error {
			perms := map[string]permission_models.Permission{}
			for _, perm := range examOrderPermissions() {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				perms[perm.ID] = perm
				fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)
			}

			grants := map[string][]string{role_data.RoleAdmin: permission_data.ExamOrderPermissionIDs}
			for _, role := range []string{role_data.RoleDoctor, role_data.RoleRecepcionista} {
				for _, id := range role_data.RolePermissionMatrix[role] {
					if _, ok := perms[id]; ok {
						grants[role] = append(grants[role], id)
					}
				}
			}

			for _, roleName := range []string{role_data.RoleAdmin, role_data.RoleDoctor, role_data.RoleRecepcionista} {
				var role user_models.Role
				if err := db.Where(user_models.Role{Role: roleName}).First(&role).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) && roleName != role_data.RoleAdmin {
						fmt.Printf("⚠️  %s role not found, skipping its exam order permissions.\n", roleName)
						continue
					}
					return fmt.Errorf("failed to find %s role: %w", roleName, err)
				}
				for _, id := range grants[roleName] {
					perm := perms[id]
					if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
						return fmt.Errorf("failed to assign '%s' to %s role: %w", id, roleName, err)
					}
					fmt.Printf("✅ Assigned permission '%s' to %s role.\n", id, roleName)
				}
			}
			return nil
		},
	}

	// Associates the exam order permissions with the CLINICAL Feature, so every
	// plan with the clinical module includes exam orders.
	database.GlobalDBMap["DB20261002_11"] = database.DBExecute{
		ID: "DB20261002_11",
		Execute: func(db *gorm.DB) error {
			var feature company_models.Feature
			if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("⚠️  CLINICAL feature not found, skipping exam order permissions wiring.")
					return nil
				}
				return fmt.Errorf("failed to find CLINICAL feature: %w", err)
			}
			for _, id := range permission_data.ExamOrderPermissionIDs {
				var perm permission_models.Permission
				if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}}).First(&perm).Error; err != nil {
					return fmt.Errorf("failed to find permission '%s': %w", id, err)
				}
				if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to associate '%s' with CLINICAL feature: %w", id, err)
				}
				fmt.Printf("✅ Associated permission '%s' with CLINICAL feature.\n", id)
			}
			return nil
		},
	}
}
