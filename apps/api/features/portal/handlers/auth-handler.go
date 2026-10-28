// Package portal_handlers serves the patient portal (/portal) and its
// clinic-side settings (/portal-settings).
package portal_handlers

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantfiles"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_services "pengi-med-saas/features/portal/services"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthHandler serves the public portal endpoints: the clinic's card, its
// logo, and signing in with a one-time code.
type AuthHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	files    tenantfiles.Store
	delivery *portal_services.Delivery
	// run starts the code delivery; in production a goroutine, so the login
	// answer takes the same time whether or not the patient exists.
	run func(func())
	// captcha checks the sign-in captcha; nil when TURNSTILE_SECRET_KEY is
	// unset (no captcha).
	captcha portal_services.CaptchaVerifier
}

func NewAuthHandler(db *gorm.DB, logger *zap.Logger, files tenantfiles.Store, delivery *portal_services.Delivery) *AuthHandler {
	return &AuthHandler{
		db: db, logger: logger, files: files, delivery: delivery,
		run:     func(f func()) { go f() },
		captcha: portal_services.TurnstileFromConfig(),
	}
}

// WithCaptcha replaces the captcha verifier (tests); nil disables it.
func (h *AuthHandler) WithCaptcha(v portal_services.CaptchaVerifier) *AuthHandler {
	h.captcha = v
	return h
}

// WithSyncDelivery delivers codes before answering (tests).
func (h *AuthHandler) WithSyncDelivery() *AuthHandler {
	h.run = func(f func()) { f() }
	return h
}

func internalError() envelope.Response {
	return envelope.ErrorResponse(http.StatusInternalServerError, "portal.error.internal", core_errors.ErrPortalInternal)
}

func invalidCode() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.invalid_code", core_errors.ErrPortalInvalidCode)
}

// clinic resolves the :slug clinic; a closed or unknown portal answers 404.
func (h *AuthHandler) clinic(c *gin.Context) (portal_services.Clinic, envelope.Response, bool) {
	clinic, err := portal_services.ClinicBySlug(h.db, c.Param("slug"))
	if err != nil {
		if errors.Is(err, portal_services.ErrUnavailable) {
			return clinic, portal_middleware.NotAvailable(), false
		}
		h.logger.Error("portal: failed to resolve clinic", zap.Error(err))
		return clinic, internalError(), false
	}
	return clinic, envelope.Response{}, true
}

// ClinicInfo is what the portal shows of a clinic.
func ClinicInfo(clinic portal_services.Clinic) portal_dto.ClinicInfo {
	return portal_dto.ClinicInfo{
		Name:     clinic.Tenant.DisplayName(),
		Slug:     clinic.Tenant.Slug,
		HasLogo:  clinic.HasLogo(),
		Sections: clinic.Settings.Sections,
		Booking: portal_dto.BookingInfo{
			Enabled: clinic.Settings.BookingEnabled(), ChooseBy: clinic.Settings.Booking.ChooseBy,
			NewPatients: clinic.Settings.NewPatientsEnabled(),
		},
	}
}

// GetClinic is GET /portal/clinics/:slug.
func (h *AuthHandler) GetClinic(c *gin.Context) envelope.Response {
	clinic, resp, ok := h.clinic(c)
	if !ok {
		return resp
	}
	return envelope.SuccessResponse(portal_dto.PublicClinicResponse{
		ClinicInfo: ClinicInfo(clinic),
		Channels:   portal_dto.Channels{WhatsApp: h.delivery.WhatsAppReady(clinic.Tenant.ID), Email: true},
	}, "portal.clinic.get.success")
}

// GetLogo is GET /portal/clinics/:slug/logo: the image bytes, 404 if none.
func (h *AuthHandler) GetLogo(c *gin.Context) {
	clinic, resp, ok := h.clinic(c)
	if !ok {
		envelope.Write(c, resp)
		return
	}
	if !clinic.HasLogo() {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "portal.error.not_found", core_errors.ErrPortalNotFound))
		return
	}
	data, err := h.files.Read(clinic.Tenant.ID, filepath.Base(*clinic.Tenant.LogoPath))
	if err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "portal.error.not_found", core_errors.ErrPortalNotFound))
		return
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		contentType = "application/octet-stream"
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, contentType, data)
}

// Login is POST /portal/clinics/:slug/login. With a captcha configured, a
// missing or rejected captcha_token answers 400 E-PORTAL-008 (Cloudflare
// unreachable too: fail closed), before the document is looked at. Otherwise
// it always answers the same (portal.login.code_sent, expires_in) for an open
// portal: unknown document, patient without contact, a spent code budget
// (patient, clinic or global) or a filled honeypot included. The code is
// issued and sent in the background.
func (h *AuthHandler) Login(c *gin.Context) envelope.Response {
	clinic, resp, ok := h.clinic(c)
	if !ok {
		return resp
	}
	var req portal_dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_request", core_errors.ErrPortalInvalidData)
	}
	if h.captcha != nil {
		ok, err := h.captcha.Verify(c.Request.Context(), req.CaptchaToken, c.ClientIP())
		if err != nil {
			h.logger.Warn("portal: captcha verification failed", zap.Uint("tenant_id", clinic.Tenant.ID), zap.Error(err))
		}
		if err != nil || !ok {
			return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.captcha_failed", core_errors.ErrPortalCaptchaFailed)
		}
	}
	answer := envelope.SuccessResponse(portal_dto.LoginResponse{ExpiresIn: int(portal_services.CodeTTL.Seconds())}, "portal.login.code_sent")
	if strings.TrimSpace(req.Website) != "" {
		return answer // honeypot: a bot
	}
	tr := translator(c)
	document := req.Document
	h.run(func() {
		patient, found, err := portal_services.PatientByDocument(h.db, clinic.Tenant.ID, document)
		if err != nil {
			h.logger.Error("portal: failed to look up patient", zap.Error(err))
			return
		}
		if !found {
			return
		}
		channel, err := h.delivery.SendLoginCode(clinic, patient, tr)
		var limit *portal_services.LimitError
		switch {
		case errors.As(err, &limit) && limit.PatientBudget():
			h.logger.Info("portal: patient code limit reached", zap.Uint("tenant_id", clinic.Tenant.ID), zap.Uint("patient_id", patient.ID), zap.String("budget", limit.Budget))
		case errors.As(err, &limit):
			h.logger.Warn("portal: code budget reached, nothing sent", zap.Uint("tenant_id", clinic.Tenant.ID), zap.String("budget", limit.Budget))
			if limit.TenantBudget() {
				portal_services.NotifyCodeBudget(h.db, h.logger, clinic.Tenant.ID, limit.Budget, timeNow())
			}
		case err != nil:
			h.logger.Error("portal: failed to send sign-in code", zap.Uint("tenant_id", clinic.Tenant.ID), zap.Uint("patient_id", patient.ID), zap.Error(err))
		case channel == "":
			h.logger.Info("portal: patient has no contact for a sign-in code", zap.Uint("tenant_id", clinic.Tenant.ID), zap.Uint("patient_id", patient.ID))
		}
	})
	return answer
}

// Verify is POST /portal/clinics/:slug/verify: a valid code opens a session
// (access token in the body, refresh token in the portal_refresh cookie).
// Every failure answers 401 E-PORTAL-002.
func (h *AuthHandler) Verify(c *gin.Context) envelope.Response {
	clinic, resp, ok := h.clinic(c)
	if !ok {
		return resp
	}
	var req portal_dto.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidCode()
	}
	patient, found, err := portal_services.PatientByDocument(h.db, clinic.Tenant.ID, req.Document)
	if err != nil {
		h.logger.Error("portal: failed to look up patient", zap.Error(err))
		return internalError()
	}
	if !found {
		return invalidCode()
	}
	if err := portal_services.VerifyCode(h.db, clinic.Tenant.ID, patient.ID, req.Code, timeNow()); err != nil {
		if !errors.Is(err, portal_services.ErrInvalidCode) {
			h.logger.Error("portal: failed to verify code", zap.Error(err))
			return internalError()
		}
		return invalidCode()
	}
	return h.openSession(c, clinic.Tenant.ID, patient.ID)
}

func client(c *gin.Context) portal_services.Client {
	return portal_services.Client{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
}

func (h *AuthHandler) openSession(c *gin.Context, tenantID, patientID uint) envelope.Response {
	access, err := auth.GeneratePortalAccessToken(tenantID, patientID)
	if err != nil {
		h.logger.Error("portal: failed to sign access token", zap.Error(err))
		return internalError()
	}
	refresh, err := portal_services.OpenSession(h.db, tenantID, patientID, client(c), timeNow())
	if err != nil {
		h.logger.Error("portal: failed to open session", zap.Error(err))
		return internalError()
	}
	auth.SetPortalRefreshCookie(c, refresh)
	return envelope.SuccessResponse(portal_dto.TokenResponse{AccessToken: access}, "portal.verify.success")
}

// Refresh is POST /portal/refresh: a new access token from the
// portal_refresh cookie, while the session is live, the clinic's portal is
// open and the patient exists. The cookie rotates on every refresh; a
// rotated-out one presented again revokes the session.
func (h *AuthHandler) Refresh(c *gin.Context) envelope.Response {
	cookie, err := c.Cookie(auth.PortalRefreshCookie)
	if err != nil || cookie == "" {
		return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
	}
	claims, err := auth.ParsePortalRefreshToken(cookie)
	if err != nil {
		auth.ClearPortalRefreshCookie(c)
		return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
	}
	clinic, err := portal_services.ClinicByID(h.db, claims.TenantID)
	if err != nil {
		if errors.Is(err, portal_services.ErrUnavailable) {
			auth.ClearPortalRefreshCookie(c)
			return portal_middleware.NotAvailable()
		}
		h.logger.Error("portal: failed to resolve clinic", zap.Error(err))
		return internalError()
	}
	if _, found, err := portal_services.PatientByID(h.db, clinic.Tenant.ID, claims.PatientID); err != nil || !found {
		if err != nil {
			h.logger.Error("portal: failed to load patient", zap.Error(err))
			return internalError()
		}
		auth.ClearPortalRefreshCookie(c)
		return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
	}
	next, err := portal_services.RotateSession(h.db, claims, client(c), timeNow())
	if err != nil {
		if errors.Is(err, portal_services.ErrSessionReused) {
			h.logger.Warn("portal: refresh token reused, session revoked",
				zap.Uint("tenant_id", claims.TenantID), zap.Uint("patient_id", claims.PatientID),
				zap.String("family_id", claims.FamilyID), zap.String("ip", c.ClientIP()))
		} else if !errors.Is(err, portal_services.ErrSessionInvalid) {
			h.logger.Error("portal: failed to rotate session", zap.Error(err))
			return internalError()
		}
		auth.ClearPortalRefreshCookie(c)
		return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
	}
	access, err := auth.GeneratePortalAccessToken(claims.TenantID, claims.PatientID)
	if err != nil {
		h.logger.Error("portal: failed to sign access token", zap.Error(err))
		return internalError()
	}
	if next != "" { // "" within the rotation grace: the browser has the new cookie
		auth.SetPortalRefreshCookie(c, next)
	}
	return envelope.SuccessResponse(portal_dto.TokenResponse{AccessToken: access}, "portal.refresh.success")
}

// Logout is POST /portal/logout: revokes the session of the refresh cookie
// (if any) and clears it.
func (h *AuthHandler) Logout(c *gin.Context) envelope.Response {
	if cookie, err := c.Cookie(auth.PortalRefreshCookie); err == nil && cookie != "" {
		if claims, err := auth.ParsePortalRefreshToken(cookie); err == nil {
			if err := portal_services.RevokeSession(h.db, claims.TenantID, claims.FamilyID, timeNow()); err != nil {
				h.logger.Error("portal: failed to revoke session", zap.Error(err))
			}
		}
	}
	auth.ClearPortalRefreshCookie(c)
	return envelope.SuccessResponse(nil, "portal.logout.success")
}

// translator is the request's i18n translator (I18nMiddleware), resolved now
// so the background delivery never touches the gin context; Spanish catalog
// text when the middleware did not run.
func translator(c *gin.Context) portal_services.Translator {
	if v, ok := c.Get("translator"); ok {
		if tr, ok := v.(func(string) string); ok {
			return tr
		}
	}
	return whatsapp_services.PatientText
}
