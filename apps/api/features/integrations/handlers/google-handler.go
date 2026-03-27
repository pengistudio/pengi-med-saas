package integration_handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	google_calendar "pengi-med-saas/core/google"
	integration_models "pengi-med-saas/features/integrations/models"
)

type GoogleIntegrationHandler struct {
	db        *gorm.DB
	logger    *zap.Logger
	googleSvc *google_calendar.CalendarService
}

func NewGoogleIntegrationHandler(db *gorm.DB, logger *zap.Logger) *GoogleIntegrationHandler {
	return &GoogleIntegrationHandler{
		db:        db,
		logger:    logger,
		googleSvc: google_calendar.NewCalendarService(),
	}
}

// GetAuthURL returns the Google OAuth authorization URL for the current tenant.
func (h *GoogleIntegrationHandler) GetAuthURL(c *gin.Context) envelope.Response {
	if !h.googleSvc.IsConfigured() {
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "Google integration not configured", core_errors.ErrIntegrationNotConfigured)
	}

	tenantID, _ := c.Get("tenant_id")
	var tenant struct {
		Slug string
	}
	if err := h.db.Table("tenants").Where("id = ? AND deleted_at IS NULL", tenantID).Select("slug").First(&tenant).Error; err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrInternal)
	}

	authURL := h.googleSvc.GetAuthURL(google_calendar.EncodeState(tenant.Slug))
	return envelope.SuccessResponse(gin.H{"url": authURL}, "integrations.google.auth.url.success")
}

// Callback handles the OAuth redirect from Google.
// This endpoint is public (no auth middleware) — tenant is identified via state param.
func (h *GoogleIntegrationHandler) Callback(c *gin.Context) {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.Redirect(http.StatusFound, frontendURL+"/settings?google=error")
		return
	}

	tenantSlug := google_calendar.DecodeState(state)
	var tenant struct {
		ID uint
	}
	if err := h.db.Table("tenants").Where("slug = ? AND deleted_at IS NULL", tenantSlug).Select("id").First(&tenant).Error; err != nil {
		h.logger.Error("Google callback: tenant not found", zap.String("slug", tenantSlug), zap.Error(err))
		c.Redirect(http.StatusFound, frontendURL+"/settings?google=error")
		return
	}

	token, err := h.googleSvc.ExchangeCode(code)
	if err != nil {
		h.logger.Error("Google callback: token exchange failed", zap.Error(err))
		c.Redirect(http.StatusFound, frontendURL+"/settings?google=error")
		return
	}

	expiry := google_calendar.TokenExpiry(token.ExpiresIn)
	integration := integration_models.TenantIntegration{
		TenantID:           tenant.ID,
		GoogleAccessToken:  token.AccessToken,
		GoogleRefreshToken: token.RefreshToken,
		GoogleTokenExpiry:  &expiry,
		GoogleCalendarID:   "primary",
		GoogleConnected:    true,
	}

	// Upsert by tenant_id
	if err := h.db.Where(integration_models.TenantIntegration{TenantID: tenant.ID}).
		Assign(integration_models.TenantIntegration{
			GoogleAccessToken:  token.AccessToken,
			GoogleRefreshToken: token.RefreshToken,
			GoogleTokenExpiry:  &expiry,
			GoogleConnected:    true,
		}).
		FirstOrCreate(&integration).Error; err != nil {
		h.logger.Error("Google callback: failed to save integration", zap.Error(err))
		c.Redirect(http.StatusFound, frontendURL+"/settings?google=error")
		return
	}

	// Update tokens if record already existed
	h.db.Model(&integration).Updates(map[string]interface{}{
		"google_access_token":  token.AccessToken,
		"google_refresh_token": token.RefreshToken,
		"google_token_expiry":  &expiry,
		"google_connected":     true,
	})

	h.logger.Info("Google Calendar connected", zap.Uint("tenant_id", tenant.ID))
	c.Redirect(http.StatusFound, frontendURL+"/settings?google=connected")
}

// GetStatus returns the current Google Calendar integration status.
func (h *GoogleIntegrationHandler) GetStatus(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")

	var integration integration_models.TenantIntegration
	err := h.db.Where("tenant_id = ?", tenantID).First(&integration).Error
	if err != nil {
		return envelope.SuccessResponse(gin.H{"connected": false}, "integrations.google.status.success")
	}

	return envelope.SuccessResponse(gin.H{
		"connected":          integration.GoogleConnected,
		"google_calendar_id": integration.GoogleCalendarID,
	}, "integrations.google.status.success")
}

// Disconnect removes the Google Calendar integration for the current tenant.
func (h *GoogleIntegrationHandler) Disconnect(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")

	var integration integration_models.TenantIntegration
	if err := h.db.Where("tenant_id = ?", tenantID).First(&integration).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "integration not found", core_errors.ErrIntegrationNotFound)
	}

	expiry := time.Time{}
	if err := h.db.Model(&integration).Updates(map[string]interface{}{
		"google_access_token":  "",
		"google_refresh_token": "",
		"google_token_expiry":  &expiry,
		"google_connected":     false,
	}).Error; err != nil {
		h.logger.Error("Failed to disconnect Google integration", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrInternal)
	}

	h.logger.Info("Google Calendar disconnected", zap.Uint("tenant_id", tenantID.(uint)))
	return envelope.SuccessResponse(nil, "integrations.google.disconnect.success")
}
