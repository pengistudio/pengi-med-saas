package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"

	"gorm.io/gorm"
)

// Associates the medical report/certificate permissions with the CLINICAL
// Feature so tenants whose Plan already includes that Feature (checked by
// SubscriptionMiddleware.RequirePermission) get access automatically,
// without needing manual re-wiring in the backoffice Features UI.
func init() {
	database.GlobalDBMap["DB20260922_2"] = database.DBExecute{
		ID: "DB20260922_2",
		Execute: func(db *gorm.DB) error {
			var feature company_models.Feature
			if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					fmt.Println("⚠️  CLINICAL feature not found, skipping medical document permission wiring.")
					return nil
				}
				return fmt.Errorf("failed to find CLINICAL feature: %w", err)
			}

			permissionIDs := []string{"CREATE_MEDICAL_REPORT", "CREATE_MEDICAL_CERTIFICATE"}
			for _, id := range permissionIDs {
				var perm permission_models.Permission
				if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}}).First(&perm).Error; err != nil {
					return fmt.Errorf("failed to find permission '%s': %w", id, err)
				}

				if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to associate permission '%s' with CLINICAL feature: %w", id, err)
				}
				fmt.Printf("✅ Associated permission '%s' with CLINICAL feature.\n", id)
			}

			return nil
		},
	}
}
