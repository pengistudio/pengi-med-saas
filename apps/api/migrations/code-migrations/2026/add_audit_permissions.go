package y2026

import (
	"fmt"
	"pengi-med-saas/core/database"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260705_1"] = database.DBExecute{
		ID: "DB20260705_1",
		Execute: func(db *gorm.DB) error {
			// Retrieve the Admin Role to attach the new permissions
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			for _, perm := range permission_data.AuditPermissions {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)

				if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
				}
				fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
			}

			// Composite indexes for the audit_logs query patterns (tenant+patient+date, tenant+entity+date).
			// Added as raw SQL rather than GORM struct tags since the table is already migrated in prod.
			if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_patient_created ON audit_logs (tenant_id, patient_id, created_at)`).Error; err != nil {
				return fmt.Errorf("failed to create idx_audit_logs_tenant_patient_created: %w", err)
			}
			if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_entity_created ON audit_logs (tenant_id, entity_type, entity_id, created_at)`).Error; err != nil {
				return fmt.Errorf("failed to create idx_audit_logs_tenant_entity_created: %w", err)
			}
			fmt.Println("✅ Audit log composite indexes created/verified.")

			return nil
		},
	}
}
