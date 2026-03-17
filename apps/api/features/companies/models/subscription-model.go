package company_models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	Status    string    `gorm:"not null" json:"status"`
	PlanCode  string    `gorm:"not null" json:"plan_code"`
	Plan      Plan      `gorm:"foreignKey:PlanCode;references:Code" json:"plan"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CompanyID uint
	Company   Company `gorm:"foreignKey:CompanyID" json:"company"`
}

func (s *Subscription) Save(db *gorm.DB) error {
	return db.Save(s).Error
}
