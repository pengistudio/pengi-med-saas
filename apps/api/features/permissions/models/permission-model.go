package permission_models

import (
	"pengi-med-saas/core/database"

	"gorm.io/gorm"
)

type Permission struct {
	database.BaseStringID
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func (p *Permission) Save(db *gorm.DB) error {
	return db.Create(&p).Error
}
