package y2026

import (
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260518_1"] = database.DBExecute{
		ID: "DB20260518_1",
		Execute: func(db *gorm.DB) error {
			plan := company_models.Plan{
				Name:     "Plan Trial",
				Code:     "TRIAL",
				Tier:     0,
				Price:    0,
				CanRenew: false,
				Properties: datatypes.JSONMap{
					"enabled_features": map[string]interface{}{
						"clinical": true,
						"billing":  false,
						"team":     false,
						"kanban":   true,
					},
					"max_users":    1,
					"max_patients": 15,
					"max_offices":  1,
					"trial_days":   14,
				},
			}
			if err := db.Where(company_models.Plan{Code: "TRIAL"}).FirstOrCreate(&plan).Error; err != nil {
				return err
			}
			return db.Model(&plan).Updates(map[string]interface{}{
				"can_renew":  false,
				"tier":       0,
				"price":      0,
				"properties": plan.Properties,
			}).Error
		},
	}
}
