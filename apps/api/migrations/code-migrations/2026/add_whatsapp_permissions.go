package y2026

import (
	"encoding/json"
	"fmt"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const (
	addWhatsAppPermissionsID    = "DB20261005_1"
	backfillWhatsAppFeatureFlag = "DB20261005_2"
)

func init() {
	// Creates MANAGE_WHATSAPP, gives it to the admin role (the connection is a
	// tenant-wide setting) and puts it in a WHATSAPP Feature that backoffice
	// can add to plans. No plan gets the Feature here: RequirePermission keeps
	// the routes closed until a plan includes it.
	database.GlobalDBMap[addWhatsAppPermissionsID] = database.DBExecute{
		ID: addWhatsAppPermissionsID,
		Execute: func(db *gorm.DB) error {
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: role_data.RoleAdmin}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			feature := company_models.Feature{Code: "WHATSAPP", Name: "WhatsApp"}
			if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
				return fmt.Errorf("failed to create WHATSAPP feature: %w", err)
			}

			for _, perm := range permission_data.WhatsAppPermissions {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
				}
				if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to associate '%s' with WHATSAPP feature: %w", perm.ID, err)
				}
				fmt.Printf("✅ Permission '%s' created, assigned to admin and the WHATSAPP feature.\n", perm.ID)
			}
			return nil
		},
	}

	// Writes "whatsapp": false into every tenant's stored enabled_features, so
	// the flag is explicit (an absent key already reads as false).
	database.GlobalDBMap[backfillWhatsAppFeatureFlag] = database.DBExecute{
		ID: backfillWhatsAppFeatureFlag,
		Execute: func(db *gorm.DB) error {
			var tenants []tenant_models.Tenant
			if err := db.Select("id", "enabled_features").Find(&tenants).Error; err != nil {
				return fmt.Errorf("failed to list tenants: %w", err)
			}
			for _, t := range tenants {
				flags := map[string]any{}
				if t.EnabledFeatures != "" {
					if err := json.Unmarshal([]byte(t.EnabledFeatures), &flags); err != nil {
						fmt.Printf("⚠️  Tenant %d has unreadable enabled_features, skipping.\n", t.ID)
						continue
					}
				}
				if _, ok := flags["whatsapp"]; ok {
					continue
				}
				flags["whatsapp"] = false
				raw, err := json.Marshal(flags)
				if err != nil {
					return err
				}
				if err := db.Model(&tenant_models.Tenant{}).Where("id = ?", t.ID).Update("enabled_features", string(raw)).Error; err != nil {
					return fmt.Errorf("failed to backfill enabled_features of tenant %d: %w", t.ID, err)
				}
			}
			return nil
		},
	}
}
