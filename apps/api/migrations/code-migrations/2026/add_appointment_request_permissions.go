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

const addAppointmentRequestPermissionsID = "DB20261026_3"

func init() {
	// Creates MANAGE_APPOINTMENT_REQUESTS (the clinic's inbox of appointment
	// requests made from the patient portal), gives it to admin, recepcionista
	// and doctor (as role_data.RolePermissionMatrix does for new installs) and
	// adds it to the PATIENT_PORTAL Feature, so every plan that has the portal
	// gets it.
	database.GlobalDBMap[addAppointmentRequestPermissionsID] = database.DBExecute{
		ID:      addAppointmentRequestPermissionsID,
		Execute: addAppointmentRequestPermissions,
	}
}

func addAppointmentRequestPermissions(db *gorm.DB) error {
	feature := company_models.Feature{Code: "PATIENT_PORTAL", Name: "Portal de pacientes"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to find PATIENT_PORTAL feature: %w", err)
	}
	for _, perm := range permission_data.AppointmentRequestPermissions {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("failed to associate '%s' with PATIENT_PORTAL feature: %w", perm.ID, err)
		}
		for _, roleName := range []string{role_data.RoleAdmin, role_data.RoleRecepcionista, role_data.RoleDoctor} {
			var role user_models.Role
			if err := db.Where(user_models.Role{Role: roleName}).First(&role).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Printf("⚠️  %s role not found, skipping %s.\n", roleName, perm.ID)
					continue
				}
				return fmt.Errorf("failed to find %s role: %w", roleName, err)
			}
			if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to assign '%s' to %s role: %w", perm.ID, roleName, err)
			}
		}
		fmt.Printf("✅ Permission '%s' created, assigned to admin, recepcionista, doctor and the PATIENT_PORTAL feature.\n", perm.ID)
	}
	return nil
}
