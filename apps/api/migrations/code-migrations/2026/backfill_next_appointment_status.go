package y2026

import (
	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260812_1"] = database.DBExecute{
		ID: "DB20260812_1",
		Execute: func(db *gorm.DB) error {
			return db.Exec(`
				UPDATE medical_records
				SET next_appointment_status = 'scheduled'
				WHERE next_appointment_date IS NOT NULL
				AND (next_appointment_status IS NULL OR next_appointment_status = 'pending')
			`).Error
		},
	}
}
