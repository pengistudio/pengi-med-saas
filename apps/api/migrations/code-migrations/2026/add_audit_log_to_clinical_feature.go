package y2026

import (
	"errors"
	"fmt"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"

	"gorm.io/gorm"
)

const addAuditLogToClinicalFeatureID = "DB20261007_1"

func init() {
	// Associates READ_AUDIT_LOG with the CLINICAL Feature, so every plan with
	// the clinical module lets its admins open the audit log viewer. The audit
	// trail is recorded for every clinic; reviewing it is compliance, not an
	// upsell. Roles are unchanged: only admin holds the permission.
	database.GlobalDBMap[addAuditLogToClinicalFeatureID] = database.DBExecute{
		ID:      addAuditLogToClinicalFeatureID,
		Execute: addAuditLogToClinicalFeature,
	}
}

func addAuditLogToClinicalFeature(db *gorm.DB) error {
	var feature company_models.Feature
	if err := db.Where(company_models.Feature{Code: "CLINICAL"}).First(&feature).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Println("⚠️  CLINICAL feature not found, skipping READ_AUDIT_LOG wiring.")
			return nil
		}
		return fmt.Errorf("failed to find CLINICAL feature: %w", err)
	}
	var perm permission_models.Permission
	if err := db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: "READ_AUDIT_LOG"}}).First(&perm).Error; err != nil {
		return fmt.Errorf("failed to find permission 'READ_AUDIT_LOG': %w", err)
	}
	if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
		return fmt.Errorf("failed to associate 'READ_AUDIT_LOG' with CLINICAL feature: %w", err)
	}
	fmt.Println("✅ Associated permission 'READ_AUDIT_LOG' with CLINICAL feature.")
	return nil
}
