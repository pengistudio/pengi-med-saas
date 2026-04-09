package y2026

import (
	"encoding/json"
	"fmt"
	"pengi-med-saas/core/database"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260409_1"] = database.DBExecute{
		ID: "DB20260409_1",
		Execute: func(db *gorm.DB) error {
			// Add enabled_features column if it doesn't exist
			if !db.Migrator().HasColumn(&tenant_models.Tenant{}, "enabled_features") {
				if err := db.Migrator().AddColumn(&tenant_models.Tenant{}, "enabled_features"); err != nil {
					return fmt.Errorf("failed to add enabled_features column: %w", err)
				}
				fmt.Println("✅ Added enabled_features column to tenants table")

				// Get all permission categories to auto-enable features
				type permissionResult struct {
					Category string
				}
				var permissions []permissionResult
				if err := db.Distinct("category").Table("permissions").Scan(&permissions).Error; err != nil {
					return fmt.Errorf("failed to get permission categories: %w", err)
				}

				// Build enabled features based on available permission categories
				enabledFeatures := tenant_models.EnabledFeatures{
					Clinical: false,
					Billing:  false,
					Team:     false,
				}

				for _, perm := range permissions {
					switch perm.Category {
					case "CLINICAL":
						enabledFeatures.Clinical = true
					case "BILLING":
						enabledFeatures.Billing = true
					case "TEAM":
						enabledFeatures.Team = true
					}
				}

				featuresJSON, err := json.Marshal(enabledFeatures)
				if err != nil {
					return fmt.Errorf("failed to marshal enabled features: %w", err)
				}

				if err := db.Model(&tenant_models.Tenant{}).
					Where("enabled_features = '' OR enabled_features IS NULL").
					Update("enabled_features", string(featuresJSON)).Error; err != nil {
					return fmt.Errorf("failed to set enabled features: %w", err)
				}
				fmt.Printf("✅ Set enabled features for existing tenants: %s\n", string(featuresJSON))
			}

			return nil
		},
	}
}
