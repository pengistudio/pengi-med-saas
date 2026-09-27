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

// Creates SIGN_MEDICAL_DOCUMENT, gives it to admin, and gives the doctor role
// what it needs to generate and sign medical documents (it never had
// CREATE_MEDICAL_REPORT / CREATE_MEDICAL_CERTIFICATE).
func init() {
	database.GlobalDBMap["DB20260926_3"] = database.DBExecute{
		ID: "DB20260926_3",
		Execute: func(db *gorm.DB) error {
			var signPerm permission_models.Permission
			for _, perm := range permission_data.ClinicalPermissions {
				if perm.ID == "SIGN_MEDICAL_DOCUMENT" {
					signPerm = perm
				}
			}
			if err := db.Where(permission_models.Permission{BaseStringID: signPerm.BaseStringID}).FirstOrCreate(&signPerm).Error; err != nil {
				return fmt.Errorf("failed to create permission 'SIGN_MEDICAL_DOCUMENT': %w", err)
			}
			fmt.Println("✅ Permission 'SIGN_MEDICAL_DOCUMENT' created/found.")

			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}
			if err := db.Model(&adminRole).Association("Permissions").Append(&signPerm); err != nil {
				return fmt.Errorf("failed to assign 'SIGN_MEDICAL_DOCUMENT' to admin role: %w", err)
			}
			fmt.Println("✅ Assigned permission 'SIGN_MEDICAL_DOCUMENT' to admin role.")

			var doctorRole user_models.Role
			if err := db.Where(user_models.Role{Role: role_data.RoleDoctor}).First(&doctorRole).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("⚠️  doctor role not found, skipping doctor permissions.")
					return nil
				}
				return fmt.Errorf("failed to find doctor role: %w", err)
			}
			for _, id := range []string{"CREATE_MEDICAL_REPORT", "CREATE_MEDICAL_CERTIFICATE", "SIGN_MEDICAL_DOCUMENT"} {
				var perm permission_models.Permission
				if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}}).First(&perm).Error; err != nil {
					return fmt.Errorf("failed to find permission '%s': %w", id, err)
				}
				if err := db.Model(&doctorRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign '%s' to doctor role: %w", id, err)
				}
				fmt.Printf("✅ Assigned permission '%s' to doctor role.\n", id)
			}
			return nil
		},
	}
}
