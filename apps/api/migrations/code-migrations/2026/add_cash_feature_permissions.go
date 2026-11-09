package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"

	"gorm.io/gorm"
)

const addCashFeaturePermissionsID = "DB20261104_2"

func init() {
	// Creates the CASH Feature and associates the cash permissions with it so
	// plans that include it (RequirePermission) open the cash desk. No plan gets
	// the Feature here: backoffice adds it to the top plan.
	database.GlobalDBMap[addCashFeaturePermissionsID] = database.DBExecute{
		ID:      addCashFeaturePermissionsID,
		Execute: addCashFeaturePermissions,
	}
}

func addCashFeaturePermissions(db *gorm.DB) error {
	feature := company_models.Feature{Code: "CASH", Name: "Caja"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to create CASH feature: %w", err)
	}
	for _, seed := range permission_data.CashPermissions {
		var perm permission_models.Permission
		if err := db.Where(permission_models.Permission{BaseStringID: seed.BaseStringID}).First(&perm).Error; err != nil {
			return fmt.Errorf("failed to find permission '%s': %w", seed.ID, err)
		}
		if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("failed to associate permission '%s' with CASH feature: %w", seed.ID, err)
		}
		fmt.Printf("✅ Associated permission '%s' with CASH feature.\n", seed.ID)
	}
	return nil
}
