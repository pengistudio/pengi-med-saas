package portal_services

import (
	"pengi-med-saas/core/tenantdb"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// BookingLink is the clinic's portal link when its patients can book online
// there: the plan includes the portal, the clinic enabled it with booking
// (PortalSettings.BookingEnabled) and PORTAL_URL is set. "" otherwise. The
// link is the clinic's portal home: the patient signs in there and books.
func BookingLink(db *gorm.DB, tenantID uint) (string, error) {
	var tenant tenant_models.Tenant
	if err := tenantdb.System(db).Select("id", "slug", "portal_settings").Where("id = ?", tenantID).Limit(1).Find(&tenant).Error; err != nil {
		return "", err
	}
	if tenant.ID == 0 || !tenant.ParsePortalSettings().BookingEnabled() {
		return "", nil
	}
	link := portalLink(tenant.Slug)
	if link == "" {
		return "", nil
	}
	included, err := PlanIncludesPortal(db, tenantID)
	if err != nil || !included {
		return "", err
	}
	return link, nil
}
