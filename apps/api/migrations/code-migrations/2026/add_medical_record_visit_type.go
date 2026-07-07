package y2026

import (
	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260706_1"] = database.DBExecute{
		ID: "DB20260706_1",
		Execute: func(db *gorm.DB) error {
			if err := db.Exec(`ALTER TABLE medical_records ADD COLUMN IF NOT EXISTS visit_type varchar(20) NOT NULL DEFAULT 'followup'`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE medical_records ADD COLUMN IF NOT EXISTS app text NOT NULL DEFAULT ''`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE medical_records ADD COLUMN IF NOT EXISTS apf text NOT NULL DEFAULT ''`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE medical_records ADD COLUMN IF NOT EXISTS apqx text NOT NULL DEFAULT ''`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE medical_records ADD COLUMN IF NOT EXISTS allergies text NOT NULL DEFAULT ''`).Error; err != nil {
				return err
			}
			// Backfill any historical rows so they are treated as follow-ups.
			return db.Exec(`UPDATE medical_records SET visit_type = 'followup' WHERE visit_type = '' OR visit_type IS NULL`).Error
		},
	}
}
