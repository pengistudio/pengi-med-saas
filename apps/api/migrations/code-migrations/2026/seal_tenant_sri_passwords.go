package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantdb"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// Seals every tenant's plaintext SRI P12 password into sri_password_encrypted
// (core/secretbox, SIGNATURE_ENCRYPTION_KEY) and clears sri_password.
//
// Idempotent: only rows with a plaintext password are touched, and each one is
// cleared once sealed. A plaintext value next to a sealed one was written by an
// older instance after the last seal, so it is the current password and
// replaces the sealed one.
//
// Without the key (only possible in release: outside it secretbox falls back to
// a dev key) the migration fails when there is something to seal, blocking
// startup until the key is set. Skipping would record it as executed and leave
// the passwords in plaintext for good; with nothing to seal it succeeds.
func init() {
	database.GlobalDBMap["DB20260927_5"] = database.DBExecute{
		ID: "DB20260927_5",
		Execute: func(db *gorm.DB) error {
			// A new session so each query below starts without the previous one's conditions.
			db = tenantdb.System(db).Unscoped().Session(&gorm.Session{})
			var tenants []tenant_models.Tenant
			if err := db.Where("sri_password <> ''").Find(&tenants).Error; err != nil {
				return fmt.Errorf("failed to list tenants with a plaintext SRI password: %w", err)
			}
			if len(tenants) == 0 {
				fmt.Println("✅ No plaintext SRI passwords to seal.")
				return nil
			}
			box, err := secretbox.FromEnv()
			if err != nil {
				return fmt.Errorf("cannot seal %d plaintext SRI passwords: %w", len(tenants), err)
			}
			for _, tenant := range tenants {
				if err := tenant.SealSriPassword(box, tenant.SriPassword); err != nil {
					return fmt.Errorf("failed to seal SRI password of tenant %d: %w", tenant.ID, err)
				}
				if err := db.Model(&tenant_models.Tenant{}).Where("id = ?", tenant.ID).UpdateColumns(map[string]any{
					"sri_password":           "",
					"sri_password_encrypted": tenant.SriPasswordEncrypted,
				}).Error; err != nil {
					return fmt.Errorf("failed to store sealed SRI password of tenant %d: %w", tenant.ID, err)
				}
			}
			fmt.Printf("✅ %d SRI passwords sealed.\n", len(tenants))
			return nil
		},
	}
}
