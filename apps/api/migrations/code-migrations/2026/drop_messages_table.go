package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

// The message catalog is read from the JSON embedded in the binary
// (docs/adr/0003); the messages table it used to be seeded into on every
// startup is no longer read. DROP TABLE IF EXISTS keeps it idempotent.
func init() {
	database.GlobalDBMap["DB20260927_6"] = database.DBExecute{
		ID: "DB20260927_6",
		Execute: func(db *gorm.DB) error {
			if err := db.Migrator().DropTable("messages"); err != nil {
				return fmt.Errorf("failed to drop messages table: %w", err)
			}
			return nil
		},
	}
}
