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

// patientAttachmentPermissionIDs are the permissions of patient attachments
// (Adjuntos). Upload works without read: a receptionist can bring in the
// patient's papers without seeing clinical files.
var patientAttachmentPermissionIDs = []string{"READ_PATIENT_ATTACHMENT", "UPLOAD_PATIENT_ATTACHMENT"}

func init() {
	// Creates the attachment permissions and gives them to the existing roles:
	// admin and doctor both, recepcionista upload only (as in
	// role_data.RolePermissionMatrix), contador none.
	database.GlobalDBMap["DB20261002_3"] = database.DBExecute{
		ID: "DB20261002_3",
		Execute: func(db *gorm.DB) error {
			perms := map[string]permission_models.Permission{}
			for _, perm := range permission_data.ClinicalPermissions {
				for _, id := range patientAttachmentPermissionIDs {
					if perm.ID != id {
						continue
					}
					if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
						return fmt.Errorf("failed to create permission '%s': %w", id, err)
					}
					perms[id] = perm
					fmt.Printf("✅ Permission '%s' created/found.\n", id)
				}
			}
			if len(perms) != len(patientAttachmentPermissionIDs) {
				return fmt.Errorf("patient attachment permissions missing from the clinical catalog")
			}

			grants := map[string][]string{
				role_data.RoleAdmin:         patientAttachmentPermissionIDs,
				role_data.RoleDoctor:        patientAttachmentPermissionIDs,
				role_data.RoleRecepcionista: {"UPLOAD_PATIENT_ATTACHMENT"},
			}
			for _, roleName := range []string{role_data.RoleAdmin, role_data.RoleDoctor, role_data.RoleRecepcionista} {
				var role user_models.Role
				if err := db.Where(user_models.Role{Role: roleName}).First(&role).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) && roleName != role_data.RoleAdmin {
						fmt.Printf("⚠️  %s role not found, skipping its attachment permissions.\n", roleName)
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

	// Associates the attachment permissions with the CLINICAL Feature, so every
	// plan with the clinical module gets attachments without re-wiring in the
	// backoffice.
	database.GlobalDBMap["DB20261002_4"] = database.DBExecute{
		ID: "DB20261002_4",
		Execute: func(db *gorm.DB) error {
			var feature company_models.Feature
			if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("⚠️  CLINICAL feature not found, skipping patient attachment permissions wiring.")
					return nil
				}
				return fmt.Errorf("failed to find CLINICAL feature: %w", err)
			}
			for _, id := range patientAttachmentPermissionIDs {
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
