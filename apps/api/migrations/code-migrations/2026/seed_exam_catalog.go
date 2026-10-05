package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_services "pengi-med-saas/features/clinical/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// Loads the initial exam catalog and profiles into every existing tenant
// that has none (new tenants get it when they are created).
func init() {
	database.GlobalDBMap["DB20261002_12"] = database.DBExecute{
		ID: "DB20261002_12",
		Execute: func(db *gorm.DB) error {
			var tenants []tenant_models.Tenant
			if err := tenantdb.System(db).Select("id").Find(&tenants).Error; err != nil {
				return fmt.Errorf("failed to list tenants: %w", err)
			}
			for _, tenant := range tenants {
				err := db.Transaction(func(tx *gorm.DB) error {
					return clinical_services.SeedExamCatalog(tx, tenant.ID)
				})
				if err != nil {
					return fmt.Errorf("failed to seed exam catalog for tenant %d: %w", tenant.ID, err)
				}
			}
			fmt.Printf("✅ Exam catalog ensured for %d tenant(s).\n", len(tenants))
			return nil
		},
	}
}
