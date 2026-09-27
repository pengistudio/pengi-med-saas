package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"

	"gorm.io/gorm"
)

// Associates SIGN_MEDICAL_DOCUMENT with the CLINICAL Feature so plans that
// already include it can sign documents without re-wiring in the backoffice.
func init() {
	database.GlobalDBMap["DB20260926_4"] = database.DBExecute{
		ID: "DB20260926_4",
		Execute: func(db *gorm.DB) error {
			var feature company_models.Feature
			if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					fmt.Println("⚠️  CLINICAL feature not found, skipping SIGN_MEDICAL_DOCUMENT wiring.")
					return nil
				}
				return fmt.Errorf("failed to find CLINICAL feature: %w", err)
			}

			var perm permission_models.Permission
			if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: "SIGN_MEDICAL_DOCUMENT"}}).First(&perm).Error; err != nil {
				return fmt.Errorf("failed to find permission 'SIGN_MEDICAL_DOCUMENT': %w", err)
			}
			if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to associate 'SIGN_MEDICAL_DOCUMENT' with CLINICAL feature: %w", err)
			}
			fmt.Println("✅ Associated permission 'SIGN_MEDICAL_DOCUMENT' with CLINICAL feature.")
			return nil
		},
	}
}
