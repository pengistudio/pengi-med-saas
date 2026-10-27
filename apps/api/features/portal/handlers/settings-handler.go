package portal_handlers

import (
	"encoding/json"
	"net/http"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	portal_dto "pengi-med-saas/features/portal/dto"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SettingsHandler is the clinic side of the portal: turn it on and choose its
// sections (MANAGE_PATIENT_PORTAL, plan + role).
type SettingsHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	// onEnabled runs (in the background) when settings are saved with the
	// portal enabled; production prepares the WhatsApp code template.
	onEnabled func(tenantID uint)
}

func NewSettingsHandler(db *gorm.DB, logger *zap.Logger, onEnabled func(tenantID uint)) *SettingsHandler {
	return &SettingsHandler{db: db, logger: logger, onEnabled: onEnabled}
}

// PortalURL is the clinic's portal link, or "" when PORTAL_URL is not set.
func PortalURL(slug string) string {
	base := config.Get().PortalURL
	if base == "" {
		return ""
	}
	return base + "/" + slug
}

func (h *SettingsHandler) tenant(c *gin.Context) (tenant_models.Tenant, bool) {
	var tenant tenant_models.Tenant
	if err := tenantdb.System(h.db).Where("id = ?", tenantdb.TenantID(c)).Limit(1).Find(&tenant).Error; err != nil || tenant.ID == 0 {
		if err != nil {
			h.logger.Error("portal settings: failed to load tenant", zap.Error(err))
		}
		return tenant, false
	}
	return tenant, true
}

// GetSettings returns the portal settings and the clinic's portal link.
func (h *SettingsHandler) GetSettings(c *gin.Context) envelope.Response {
	tenant, ok := h.tenant(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}
	return envelope.SuccessResponse(portal_dto.SettingsResponse{
		PortalSettings: tenant.ParsePortalSettings(),
		PortalURL:      PortalURL(tenant.Slug),
	}, "portal.settings.get.success")
}

// UpdateSettings saves the portal settings. The body is read over the current
// settings, so a field it leaves out keeps its value.
func (h *SettingsHandler) UpdateSettings(c *gin.Context) envelope.Response {
	tenant, ok := h.tenant(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}
	req := tenant.ParsePortalSettings()
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_request", core_errors.ErrPortalInvalidData)
	}
	if !req.Booking.Valid() {
		return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_booking_settings", core_errors.ErrPortalInvalidBookingSettings)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_request", core_errors.ErrPortalInvalidData)
	}
	if err := tenantdb.System(h.db).Model(&tenant_models.Tenant{}).Where("id = ?", tenant.ID).Update("portal_settings", string(raw)).Error; err != nil {
		h.logger.Error("portal settings: failed to save", zap.Uint("tenant_id", tenant.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "portal.error.internal", core_errors.ErrPortalInternal)
	}
	if req.Enabled && h.onEnabled != nil {
		go h.onEnabled(tenant.ID)
	}
	return envelope.SuccessResponse(portal_dto.SettingsResponse{
		PortalSettings: req,
		PortalURL:      PortalURL(tenant.Slug),
	}, "portal.settings.update.success")
}
