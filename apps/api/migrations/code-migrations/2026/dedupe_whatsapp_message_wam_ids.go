package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

const dedupeWhatsAppMessageWamIDsID = "DB20261114_1"

// A Meta message id (wam_id) is stored once per tenant. Inbound messages,
// Coexistence echoes and history were deduplicated with a check before the
// insert, so two concurrent deliveries of one webhook could both insert.
// This removes the duplicates already stored (keeping the first row, the
// lowest id) and adds the partial unique index that makes the inserts'
// ON CONFLICT DO NOTHING the guard. Raw SQL instead of a struct tag:
// AutoMigrate runs before this and would fail on the duplicates. Both
// statements are idempotent; also run on a fresh database (baseline).
func init() {
	database.GlobalDBMap[dedupeWhatsAppMessageWamIDsID] = database.DBExecute{
		ID: dedupeWhatsAppMessageWamIDsID,
		Execute: func(db *gorm.DB) error {
			res := db.Exec(`DELETE FROM whatsapp_messages
				WHERE wam_id <> '' AND EXISTS (
					SELECT 1 FROM whatsapp_messages AS keep
					WHERE keep.tenant_id = whatsapp_messages.tenant_id
					  AND keep.wam_id = whatsapp_messages.wam_id
					  AND keep.id < whatsapp_messages.id)`)
			if res.Error != nil {
				return fmt.Errorf("failed to remove duplicate whatsapp messages: %w", res.Error)
			}
			if res.RowsAffected > 0 {
				fmt.Printf("✅ Removed %d duplicate WhatsApp messages.\n", res.RowsAffected)
			}
			if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_messages_tenant_wam_id
				ON whatsapp_messages (tenant_id, wam_id) WHERE wam_id <> ''`).Error; err != nil {
				return fmt.Errorf("failed to create idx_whatsapp_messages_tenant_wam_id: %w", err)
			}
			return nil
		},
	}
}
