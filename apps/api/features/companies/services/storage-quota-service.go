package company_services

import (
	"errors"

	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

// StorageQuotaBytes is the attachment storage quota of the tenant's current
// plan — the plan of its active subscription, resolved the way
// SubscriptionMiddleware does. A tenant without a company or an active
// subscription has no quota (0): no attachments can be uploaded.
func StorageQuotaBytes(db *gorm.DB, tenantID uint) (int64, error) {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	var subscription company_models.Subscription
	err := ActiveSubscriptionQuery(tenantdb.System(db), company.ID).
		Preload("Plan").
		First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return subscription.Plan.StorageQuotaBytes(), nil
}
