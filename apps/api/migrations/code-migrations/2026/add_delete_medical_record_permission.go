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

const deleteMedicalRecordPermissionID = "DELETE_MEDICAL_RECORD"

func init() {
	// Creates DELETE_MEDICAL_RECORD and gives it to the existing admin and
	// doctor roles (not recepcionista or contador), like the attachment delete.
	database.GlobalDBMap["DB20261016_1"] = database.DBExecute{
		ID: "DB20261016_1",
		Execute: func(db *gorm.DB) error {
			var perm permission_models.Permission
			found := false
			for _, p := range permission_data.ClinicalPermissions {
				if p.ID == deleteMedicalRecordPermissionID {
					perm, found = p, true
				}
			}
			if !found {
				return fmt.Errorf("%s missing from the clinical catalog", deleteMedicalRecordPermissionID)
			}
			if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
				return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
			}
			for _, roleName := range []string{role_data.RoleAdmin, role_data.RoleDoctor} {
				var role user_models.Role
				if err := db.Where(user_models.Role{Role: roleName}).First(&role).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) && roleName != role_data.RoleAdmin {
						fmt.Printf("⚠️  %s role not found, skipping.\n", roleName)
						continue
					}
					return fmt.Errorf("failed to find %s role: %w", roleName, err)
				}
				if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign '%s' to %s role: %w", perm.ID, roleName, err)
				}
			}
			return nil
		},
	}

	// Associates the permission with the CLINICAL Feature.
	database.GlobalDBMap["DB20261016_2"] = database.DBExecute{
		ID: "DB20261016_2",
		Execute: func(db *gorm.DB) error {
			var feature company_models.Feature
			if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("⚠️  CLINICAL feature not found, skipping.")
					return nil
				}
				return fmt.Errorf("failed to find CLINICAL feature: %w", err)
			}
			var perm permission_models.Permission
			if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: deleteMedicalRecordPermissionID}}).First(&perm).Error; err != nil {
				return fmt.Errorf("failed to find permission '%s': %w", deleteMedicalRecordPermissionID, err)
			}
			if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to associate with CLINICAL feature: %w", err)
			}
			return nil
		},
	}
}
