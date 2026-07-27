package y2026

import (
	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260727_1"] = database.DBExecute{
		ID: "DB20260727_1",
		Execute: func(db *gorm.DB) error {
			// Backfill owner_user_id for companies created before this field
			// existed. The company's creator always has the oldest Environment
			// row for that company — no Environment can exist before its Company.
			return db.Exec(`
				UPDATE companies
				SET owner_user_id = (
					SELECT user_id FROM environments
					WHERE environments.company_id = companies.id
					ORDER BY environments.created_at ASC
					LIMIT 1
				)
				WHERE owner_user_id = 0 OR owner_user_id IS NULL
			`).Error
		},
	}
}
