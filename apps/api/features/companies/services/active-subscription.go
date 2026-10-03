package company_services

import (
	"time"

	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

// SubscriptionGracePeriod is how long a subscription keeps working after it
// expires. Single source of truth for SubscriptionMiddleware and quota checks.
const SubscriptionGracePeriod = 3 * 24 * time.Hour

// ActiveSubscriptionQuery narrows db to the company's active subscription
// (status "active", not expired beyond the grace period). The caller adds
// Preload and First/Take.
func ActiveSubscriptionQuery(db *gorm.DB, companyID uint) *gorm.DB {
	return db.Model(&company_models.Subscription{}).
		Where("company_id = ? AND status = ? AND expires_at > ?", companyID, "active", time.Now().Add(-SubscriptionGracePeriod))
}
