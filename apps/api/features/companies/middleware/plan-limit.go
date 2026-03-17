package subscription_middleware

import (
	"time"

	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

// GetPlanLimitForCompany returns the configured integer limit for a plan property key.
// Returns -1 if the limit is unlimited (not set, null, or explicitly -1).
func GetPlanLimitForCompany(db *gorm.DB, companyID uint, key string) int64 {
	var subscription company_models.Subscription
	err := db.
		Where("company_id = ? AND status = ? AND expires_at > ?", companyID, "active", time.Now()).
		Preload("Plan").
		First(&subscription).Error
	if err != nil {
		return -1
	}

	props := subscription.Plan.Properties
	if props == nil {
		return -1
	}

	val, ok := props[key]
	if !ok || val == nil {
		return -1
	}

	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}

	return -1
}

// ExceedsPlanLimit checks whether adding one more item would exceed the plan limit.
// Returns false (i.e., allowed) when limit is -1 (unlimited).
func ExceedsPlanLimit(db *gorm.DB, companyID uint, key string, currentCount int64) bool {
	limit := GetPlanLimitForCompany(db, companyID, key)
	if limit == -1 {
		return false
	}
	return currentCount >= limit
}
