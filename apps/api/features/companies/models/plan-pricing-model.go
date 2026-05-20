package company_models

import "gorm.io/gorm"

// PlanPricing stores the price for a plan at a given billing period (months).
type PlanPricing struct {
	gorm.Model
	PlanID uint    `gorm:"not null;uniqueIndex:idx_plan_month" json:"plan_id"`
	Months int     `gorm:"not null;uniqueIndex:idx_plan_month" json:"months"`
	Price  float64 `gorm:"not null" json:"price"`
}
