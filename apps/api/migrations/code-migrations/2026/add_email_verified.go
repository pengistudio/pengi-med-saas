package y2026

import (
	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260519_3"] = database.DBExecute{
		ID: "DB20260519_3",
		Execute: func(db *gorm.DB) error {
			if err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at timestamptz`).Error; err != nil {
				return err
			}
			// Mark all existing users as already verified so they keep access
			return db.Exec(`UPDATE users SET email_verified = true WHERE email_verified = false`).Error
		},
	}
}
