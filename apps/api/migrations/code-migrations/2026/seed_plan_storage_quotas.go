package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

// initialStorageQuotasMB are the attachment quotas of the clinical plans, from
// the cheapest up: 2 GB, 10 GB, 50 GB. Any further clinical plan gets the last.
var initialStorageQuotasMB = []int64{2048, 10240, 51200}

func init() {
	// Gives the existing plans with the clinical module their attachment quota
	// (Plan.StorageQuotaMB, added by AutoMigrate with 0 = no attachments). The
	// plans have no fixed codes across environments, so the tiers go by price:
	// the cheapest clinical plan gets 2 GB, the next 10 GB, the rest 50 GB.
	// Plans without CLINICAL stay at 0. A plan already given a quota (edited in
	// the backoffice) is left alone.
	database.GlobalDBMap["DB20261002_7"] = database.DBExecute{
		ID: "DB20261002_7",
		Execute: func(db *gorm.DB) error {
			var plans []company_models.Plan
			if err := db.
				Joins("JOIN plan_features ON plan_features.plan_id = plans.id").
				Joins("JOIN features ON features.id = plan_features.feature_id AND features.deleted_at IS NULL").
				Where("features.code = ?", "CLINICAL").
				Order("plans.price ASC").Order("plans.id ASC").
				Find(&plans).Error; err != nil {
				return fmt.Errorf("failed to list clinical plans: %w", err)
			}
			for i, plan := range plans {
				quota := initialStorageQuotasMB[min(i, len(initialStorageQuotasMB)-1)]
				result := db.Model(&company_models.Plan{}).
					Where("id = ? AND storage_quota_mb = 0", plan.ID).
					Update("storage_quota_mb", quota)
				if result.Error != nil {
					return fmt.Errorf("failed to set the storage quota of plan '%s': %w", plan.Code, result.Error)
				}
				if result.RowsAffected > 0 {
					fmt.Printf("✅ Plan %s: storage quota %d MB\n", plan.Code, quota)
				}
			}
			return nil
		},
	}
}
