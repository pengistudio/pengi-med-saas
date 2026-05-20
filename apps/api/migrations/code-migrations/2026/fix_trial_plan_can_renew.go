package y2026

import (
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260519_1"] = database.DBExecute{
		ID: "DB20260519_1",
		Execute: func(db *gorm.DB) error {
			return db.Model(&company_models.Plan{}).
				Where("code = ?", "TRIAL").
				Update("can_renew", false).Error
		},
	}
}
