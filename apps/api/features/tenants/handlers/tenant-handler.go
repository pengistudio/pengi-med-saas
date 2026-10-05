package tenant_handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/secretbox"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	tenant_dto "pengi-med-saas/features/tenants/dto"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TenantHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	// files is the disk store itself (not just a tenantfiles.Store): the P12 and
	// logo locations are persisted on the tenant as disk paths.
	files *tenantfiles.DiskStore
}

func NewTenantHandler(db *gorm.DB, logger *zap.Logger, files *tenantfiles.DiskStore) *TenantHandler {
	return &TenantHandler{db: db, logger: logger, files: files}
}

// signatureFileName is the tenant file holding their P12 certificate.
const signatureFileName = "signature.p12"

func (h *TenantHandler) UploadSignature(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	password := c.PostForm("password")
	if password == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.form.password_required", core_errors.ErrInvalidRequest)
	}

	file, header, err := c.Request.FormFile("signature")
	if err != nil {
		h.logger.Error("Failed to retrieve file from form", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.form.file_required", core_errors.ErrInvalidRequest)
	}
	defer file.Close()

	pfxData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.form.file_required", core_errors.ErrInvalidRequest)
	}

	cert, err := pdfsign.Load(pfxData, password)
	if err != nil {
		h.logger.Warn("Rejected SRI signature upload", zap.Error(err))
		if errors.Is(err, pdfsign.ErrWrongPassword) {
			return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.wrong_password", core_errors.ErrSignatureWrongPassword)
		}
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.sri.invalid_file", core_errors.ErrBillingInvalidSignatureFile)
	}
	// The SRI rejects documents signed outside the certificate's validity.
	switch err := cert.ValidAt(time.Now()); {
	case errors.Is(err, pdfsign.ErrExpiredCertificate):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.expired", core_errors.ErrSignatureExpired)
	case errors.Is(err, pdfsign.ErrCertificateNotYetValid):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.not_yet_valid", core_errors.ErrSignatureNotYetValid)
	}

	// The password is stored sealed; without the key nothing is stored at all
	// (never the password in plaintext).
	box, err := secretbox.FromEnv()
	if err != nil {
		h.logger.Error("Cannot seal the SRI signature password", zap.Error(err))
		if errors.Is(err, secretbox.ErrNoKey) {
			return envelope.ErrorResponse(http.StatusServiceUnavailable, "signature.error.unavailable", core_errors.ErrSignatureKeyUnavailable)
		}
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	// The SRI signature must belong to the company's RUC (not, say, a doctor's
	// personal one). Only reject a clear mismatch: an unrecognised certificate
	// layout or a tenant without RUC yet is accepted.
	certTaxIDs := cert.TaxIDs()
	switch {
	case tenantRecord.TaxID == "" || len(certTaxIDs) == 0:
		h.logger.Warn("SRI signature RUC not verified",
			zap.Uint("tenant_id", tenantRecord.ID), zap.Bool("tenant_has_ruc", tenantRecord.TaxID != ""),
			zap.String("issuer", cert.Issuer()))
	case !pdfsign.MatchesTaxID(certTaxIDs, tenantRecord.TaxID):
		h.logger.Warn("Rejected SRI signature of another taxpayer",
			zap.Uint("tenant_id", tenantRecord.ID), zap.Strings("certificate_ids", certTaxIDs), zap.String("issuer", cert.Issuer()))
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.ruc_mismatch", core_errors.ErrSignatureRucMismatch)
	}

	if err := tenantRecord.SealSriPassword(box, password); err != nil {
		h.logger.Error("Failed to seal the SRI signature password", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// Only a certificate that decoded is stored.
	if err := h.files.Write(tenantdb.TenantID(c), signatureFileName, pfxData); err != nil {
		h.logger.Error("Failed to save signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	signaturePath := h.files.Path(tenantdb.TenantID(c), signatureFileName)

	tenantRecord.SriP12Path = signaturePath
	notAfter := cert.NotAfter()
	tenantRecord.SriCertExpiration = &notAfter

	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{
		"path":            signaturePath,
		"filename":        header.Filename,
		"size":            header.Size,
		"expiration_date": notAfter,
	}, "tenant.signature.upload.success")
}

// UploadLogo uploads the tenant's company logo (PNG/JPG), used on the RIDE.
func (h *TenantHandler) UploadLogo(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	file, _, err := c.Request.FormFile("logo")
	if err != nil {
		h.logger.Error("Failed to retrieve logo file from form", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.sri.logo.file_required", core_errors.ErrTenantInvalidLogoFile)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read logo file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	var ext string
	switch contentType := http.DetectContentType(data); contentType {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	default:
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.sri.logo.invalid_file", core_errors.ErrTenantInvalidLogoFile)
	}

	logoName := "logo" + ext
	logoPath := h.files.Path(tenantdb.TenantID(c), logoName)
	if err := h.files.Write(tenantdb.TenantID(c), logoName, data); err != nil {
		h.logger.Error("Failed to save logo file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	// Remove a previously uploaded logo with a different extension, if any.
	if tenantRecord.LogoPath != nil && *tenantRecord.LogoPath != logoPath {
		_ = h.files.Remove(tenantdb.TenantID(c), filepath.Base(*tenantRecord.LogoPath))
	}

	tenantRecord.LogoPath = &logoPath
	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant logo", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"path": logoPath}, "tenant.logo.upload.success")
}

// DownloadLogo serves the tenant's uploaded logo file so the frontend can preview it.
func (h *TenantHandler) DownloadLogo(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		envelope.Write(c, envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound))
		return
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound))
		return
	}

	if tenantRecord.LogoPath == nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "billing.sri.logo.not_found", core_errors.ErrTenantLogoNotFound))
		return
	}

	data, err := h.files.Read(tenantdb.TenantID(c), filepath.Base(*tenantRecord.LogoPath))
	if err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "billing.sri.logo.not_found", core_errors.ErrTenantLogoNotFound))
		return
	}

	contentType := http.DetectContentType(data)
	c.Data(http.StatusOK, contentType, data)
}

// GetSriStatus retrieves the current status of the SRI signature configuration
func (h *TenantHandler) GetSriStatus(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	// An expired certificate can no longer sign documents, so it does not count
	// as configured; expiration_date still says when it expired.
	expired := tenantRecord.SriCertExpiration != nil && !time.Now().Before(*tenantRecord.SriCertExpiration)
	isConfigured := tenantRecord.SriP12Path != "" && tenantRecord.HasSriPassword() && !expired

	return envelope.SuccessResponse(gin.H{
		"is_configured":              isConfigured,
		"expiration_date":            tenantRecord.SriCertExpiration,
		"tax_id":                     tenantRecord.TaxID,
		"trade_name":                 tenantRecord.TradeName,
		"corporate_name":             tenantRecord.CorporateName,
		"address":                    tenantRecord.Address,
		"accounting_obliged":         tenantRecord.AccountingObliged,
		"special_contributor_number": derefOrEmpty(tenantRecord.SpecialContributorNumber),
		"microenterprise_regime":     tenantRecord.MicroenterpriseRegime != nil,
		"withholding_agent":          derefOrEmpty(tenantRecord.WithholdingAgent),
		"rimpe_taxpayer":             derefOrEmpty(tenantRecord.RimpeTaxpayer),
		"has_logo":                   tenantRecord.LogoPath != nil,
	}, "tenant.sri.status.fetch.success")
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// UpdateSriInfo updates the tenant's SRI information
func (h *TenantHandler) UpdateSriInfo(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	var dto tenant_dto.UpdateSriInfoDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind UpdateSriInfo DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	tenantRecord.TaxID = dto.TaxID
	tenantRecord.TradeName = dto.TradeName
	tenantRecord.CorporateName = dto.CorporateName
	tenantRecord.Address = dto.Address
	tenantRecord.AccountingObliged = dto.AccountingObliged
	tenantRecord.SpecialContributorNumber = nilIfEmpty(dto.SpecialContributorNumber)
	tenantRecord.WithholdingAgent = nilIfEmpty(dto.WithholdingAgent)
	tenantRecord.RimpeTaxpayer = nilIfEmpty(dto.RimpeTaxpayer)
	if dto.MicroenterpriseRegime {
		si := "SI"
		tenantRecord.MicroenterpriseRegime = &si
	} else {
		tenantRecord.MicroenterpriseRegime = nil
	}

	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant SRI info", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(tenantRecord, "tenant.sri_info.update.success")
}

// GetUISettings returns the current UI settings for the tenant.
func (h *TenantHandler) GetUISettings(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	settings := tenant_models.DefaultUISettings()
	if tenantRecord.UISettings != "" && tenantRecord.UISettings != "{}" {
		if err := json.Unmarshal([]byte(tenantRecord.UISettings), &settings); err != nil {
			h.logger.Warn("Failed to parse UISettings, using defaults", zap.Error(err))
		}
	}

	return envelope.SuccessResponse(settings, "tenant.settings.fetch.success")
}

// GenerateDisplayToken replaces the tenant's TV display token, which unlinks
// the TV using the previous link.
func (h *TenantHandler) GenerateDisplayToken(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	// Check the link base first: rotating and then failing would unlink the TV
	// without giving the new link.
	base, errResp := h.displayBase()
	if errResp != nil {
		return *errResp
	}

	token := tenant_models.NewDisplayToken()

	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("display_token", token).Error; err != nil {
		h.logger.Error("Failed to generate display token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(displayTokenData(base, token), "tenant.display_token.generate.success")
}

// GetTodayAppointmentsPublic is the public endpoint behind the waiting-room TV:
// today's appointments of the tenant whose display token is the token query
// parameter. It needs no login, so it answers with tenant_dto.PublicAppointment
// only, and every rejected token (missing, malformed or unknown) gets the same 404.
func (h *TenantHandler) GetTodayAppointmentsPublic(c *gin.Context) envelope.Response {
	invalid := envelope.ErrorResponse(http.StatusNotFound, "tenant.display_token.error.invalid", core_errors.ErrTenantInvalidDisplayToken)

	token := c.Query("token")
	if !tenant_models.IsDisplayToken(token) {
		return invalid
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.Select("id").Where("display_token = ?", token).First(&tenantRecord).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("Failed to resolve display token", zap.Error(err))
		}
		return invalid
	}

	today := time.Now().Format("2006-01-02")
	var appointments []clinical_models.Appointment
	if err := tenantdb.ForTenant(h.db, tenantRecord.ID).
		Select("id", "patient_id", "start_time", "end_time", "status").
		Where("DATE(date) = ?", today).
		Preload("Patient", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).
		Order("start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get today's appointments for display", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(toPublicAppointments(appointments), "appointments.get.success")
}

// toPublicAppointments keeps only what the public TV renders.
func toPublicAppointments(appointments []clinical_models.Appointment) []tenant_dto.PublicAppointment {
	out := make([]tenant_dto.PublicAppointment, 0, len(appointments))
	for _, a := range appointments {
		out = append(out, tenant_dto.PublicAppointment{
			ID:          a.ID,
			StartTime:   a.StartTime,
			EndTime:     a.EndTime,
			Status:      a.Status,
			PatientName: tenant_dto.PatientDisplayName(a.Patient.FirstName, a.Patient.LastName),
		})
	}
	return out
}

// UpdateUISettings saves new UI settings for the tenant.
func (h *TenantHandler) UpdateUISettings(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	var settings tenant_models.UISettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("ui_settings", string(raw)).Error; err != nil {
		h.logger.Error("Failed to save UISettings", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(settings, "tenant.settings.update.success")
}

// GetEnabledFeatures returns the enabled features for the tenant, computed live from the
// company's current subscription plan.
func (h *TenantHandler) GetEnabledFeatures(c *gin.Context) envelope.Response {
	_, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}

	var company company_models.Company
	if err := tenantdb.For(c, h.db).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound)
	}

	features, err := company_services.EnabledFeaturesForCompany(h.db, company.ID)
	if err != nil {
		h.logger.Error("Failed to compute enabled features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(features, "tenant.features.fetch.success")
}

// GetDisplayToken returns the tenant's current TV display token without
// changing it, so viewing or sharing the link never unlinks the TV. A token is
// created only when the tenant has none in the current format; rotating is
// GenerateDisplayToken.
func (h *TenantHandler) GetDisplayToken(c *gin.Context) envelope.Response {
	base, errResp := h.displayBase()
	if errResp != nil {
		return *errResp
	}
	token, errResp := h.currentDisplayToken(c)
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(displayTokenData(base, token), "tenant.display_token.fetch.success")
}

// displayTokenData is the body of the display-token responses: the token and
// the TV link the web app shows and copies (the same link the QR encodes).
func displayTokenData(base, token string) gin.H {
	return gin.H{"token": token, "display_url": DisplayURL(base, token)}
}

// displayBase is FRONTEND_URL, the web app base of the TV link (never the
// request Host). Missing or not an absolute http(s) URL is a server error.
func (h *TenantHandler) displayBase() (string, *envelope.Response) {
	base := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if parsed, err := url.Parse(base); base == "" || err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		h.logger.Error("FRONTEND_URL is missing or invalid; cannot build the TV link", zap.String("frontend_url", base))
		r := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return "", &r
	}
	return base, nil
}

// currentDisplayToken is GetDisplayToken's logic: the tenant's token, created
// when it has none in the current format. On failure it returns the response.
func (h *TenantHandler) currentDisplayToken(c *gin.Context) (string, *envelope.Response) {
	fail := func(r envelope.Response) (string, *envelope.Response) { return "", &r }

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return fail(envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound))
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.Select("id", "display_token").First(&tenantRecord, tenantID).Error; err != nil {
		return fail(envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrTenantNotFound))
	}
	if tenant_models.IsDisplayToken(tenantRecord.DisplayToken) {
		return tenantRecord.DisplayToken, nil
	}

	// Only replace the token we read: if a concurrent request already created a
	// token, keep that one instead of overwriting it.
	if err := h.db.Model(&tenant_models.Tenant{}).
		Where("id = ? AND display_token = ?", tenantRecord.ID, tenantRecord.DisplayToken).
		Update("display_token", tenant_models.NewDisplayToken()).Error; err != nil {
		h.logger.Error("Failed to create display token", zap.Error(err))
		return fail(envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal))
	}
	if err := h.db.Select("id", "display_token").First(&tenantRecord, tenantRecord.ID).Error; err != nil {
		h.logger.Error("Failed to reload display token", zap.Error(err))
		return fail(envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal))
	}
	return tenantRecord.DisplayToken, nil
}

// DisplayURL is the waiting-room TV link for token on the web app at base
// (FRONTEND_URL): what the display-token responses return and the QR encodes.
func DisplayURL(base, token string) string {
	return strings.TrimRight(base, "/") + "/display/waiting-room?token=" + url.QueryEscape(token)
}

// displayQRSize is the side of the QR PNG in pixels: readable from a phone
// held to the screen and from a TV camera.
const displayQRSize = 320

// GetDisplayTokenQR streams a PNG QR code of the waiting-room TV link, so the
// TV can be paired by scanning instead of typing. The link base is
// FRONTEND_URL, never the request Host. The QR is a credential (anyone holding
// it sees today's appointments), so it is not cached.
func (h *TenantHandler) GetDisplayTokenQR(c *gin.Context) {
	base, errResp := h.displayBase()
	if errResp != nil {
		envelope.Write(c, *errResp)
		return
	}

	token, errResp := h.currentDisplayToken(c)
	if errResp != nil {
		envelope.Write(c, *errResp)
		return
	}

	png, err := qrcode.Encode(DisplayURL(base, token), qrcode.Medium, displayQRSize)
	if err != nil {
		h.logger.Error("Failed to encode display QR", zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal))
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}
