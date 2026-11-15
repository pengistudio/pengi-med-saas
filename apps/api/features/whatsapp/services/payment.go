package whatsapp_services

import (
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"gorm.io/gorm"
)

// SetPaymentStatus records the tenant's payment state. db must be bound to the
// tenant (tenantdb.For / ForTenant); the WHERE skips the write when nothing
// changes, so a stream of delivered statuses does not touch the row.
func SetPaymentStatus(db *gorm.DB, status string) error {
	return db.Model(&whatsapp_models.WhatsAppAccount{}).
		Where("payment_status <> ?", status).
		Update("payment_status", status).Error
}
