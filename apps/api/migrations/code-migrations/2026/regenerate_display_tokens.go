package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

const regenerateDisplayTokensID = "DB20261004_4"

// regenerateDisplayTokens gives every tenant (soft-deleted ones too, so no old
// code keeps working anywhere) a new long random display token. The 8-digit
// TV pairing codes were brute-forceable and opened a public endpoint with
// patient data; after this, every TV must be set up again with the new link.
func regenerateDisplayTokens(db *gorm.DB) error {
	db = tenantdb.System(db).Unscoped().Session(&gorm.Session{})
	var ids []uint
	if err := db.Model(&tenant_models.Tenant{}).Pluck("id", &ids).Error; err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}
	for _, id := range ids {
		if err := db.Model(&tenant_models.Tenant{}).Where("id = ?", id).
			Update("display_token", tenant_models.NewDisplayToken()).Error; err != nil {
			return fmt.Errorf("failed to regenerate display token of tenant %d: %w", id, err)
		}
	}
	fmt.Printf("✅ Regenerated the display token of %d tenants.\n", len(ids))
	return nil
}

func init() {
	database.GlobalDBMap[regenerateDisplayTokensID] = database.DBExecute{
		ID:      regenerateDisplayTokensID,
		Execute: regenerateDisplayTokens,
	}
}
