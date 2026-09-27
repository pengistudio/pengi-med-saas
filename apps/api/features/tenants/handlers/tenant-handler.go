package tenant_handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
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
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	password := c.PostForm("password")
	if password == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "Password is required", core_errors.ErrBillingInvalidRequest)
	}

	file, header, err := c.Request.FormFile("signature")
	if err != nil {
		h.logger.Error("Failed to retrieve file from form", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Signature file is required", core_errors.ErrBillingInvalidRequest)
	}
	defer file.Close()

	pfxData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Signature file is required", core_errors.ErrBillingInvalidRequest)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant signature", core_errors.ErrInternal)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant signature", core_errors.ErrInternal)
	}

	// Only a certificate that decoded is stored.
	if err := h.files.Write(tenantdb.TenantID(c), signatureFileName, pfxData); err != nil {
		h.logger.Error("Failed to save signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save signature file", core_errors.ErrInternal)
	}
	signaturePath := h.files.Path(tenantdb.TenantID(c), signatureFileName)

	tenantRecord.SriP12Path = signaturePath
	notAfter := cert.NotAfter()
	tenantRecord.SriCertExpiration = &notAfter

	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant signature", core_errors.ErrInternal)
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
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	file, _, err := c.Request.FormFile("logo")
	if err != nil {
		h.logger.Error("Failed to retrieve logo file from form", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Logo file is required", core_errors.ErrTenantInvalidLogoFile)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read logo file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to read logo file", core_errors.ErrInternal)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save logo file", core_errors.ErrInternal)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
	}

	// Remove a previously uploaded logo with a different extension, if any.
	if tenantRecord.LogoPath != nil && *tenantRecord.LogoPath != logoPath {
		_ = h.files.Remove(tenantdb.TenantID(c), filepath.Base(*tenantRecord.LogoPath))
	}

	tenantRecord.LogoPath = &logoPath
	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant logo", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant logo", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"path": logoPath}, "tenant.logo.upload.success")
}

// DownloadLogo serves the tenant's uploaded logo file so the frontend can preview it.
func (h *TenantHandler) DownloadLogo(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound))
		return
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound))
		return
	}

	if tenantRecord.LogoPath == nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "billing.sri.logo.not_found", core_errors.ErrTenantLogoNotFound))
		return
	}

	data, err := h.files.Read(tenantdb.TenantID(c), filepath.Base(*tenantRecord.LogoPath))
	if err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "billing.sri.logo.not_found", core_errors.ErrTenantLogoNotFound))
		return
	}

	contentType := http.DetectContentType(data)
	c.Data(http.StatusOK, contentType, data)
}

// GetSriStatus retrieves the current status of the SRI signature configuration
func (h *TenantHandler) GetSriStatus(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
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
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var dto tenant_dto.UpdateSriInfoDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind UpdateSriInfo DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid provided payload", core_errors.ErrBillingInvalidRequest)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant SRI info", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(tenantRecord, "tenant.sri_info.update.success")
}

// GetUISettings returns the current UI settings for the tenant.
func (h *TenantHandler) GetUISettings(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
	}

	settings := tenant_models.DefaultUISettings()
	if tenantRecord.UISettings != "" && tenantRecord.UISettings != "{}" {
		if err := json.Unmarshal([]byte(tenantRecord.UISettings), &settings); err != nil {
			h.logger.Warn("Failed to parse UISettings, using defaults", zap.Error(err))
		}
	}

	return envelope.SuccessResponse(settings, "tenant.settings.fetch.success")
}

// GenerateDisplayToken creates or replaces the 8-digit pairing code for the tenant TV display.
func (h *TenantHandler) GenerateDisplayToken(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	code := fmt.Sprintf("%08d", rand.IntN(100_000_000))

	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("display_token", code).Error; err != nil {
		h.logger.Error("Failed to generate display token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to generate display token", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"token": code}, "tenant.display_token.generate.success")
}

// GetTodayAppointmentsPublic is a public endpoint that returns today's appointments
// for the tenant identified by the display token query parameter.
func (h *TenantHandler) GetTodayAppointmentsPublic(c *gin.Context) envelope.Response {
	token := c.Query("token")
	if token == "" {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Missing display token", core_errors.ErrTenantInvalidDisplayToken)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.Where("display_token = ?", token).First(&tenantRecord).Error; err != nil {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid display token", core_errors.ErrTenantInvalidDisplayToken)
	}

	today := time.Now().Format("2006-01-02")
	var appointments []clinical_models.Appointment
	if err := tenantdb.ForTenant(h.db, tenantRecord.ID).Where("DATE(date) = ?", today).
		Preload("Patient").
		Order("start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get today's appointments for display", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to fetch appointments", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// UpdateUISettings saves new UI settings for the tenant.
func (h *TenantHandler) UpdateUISettings(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var settings tenant_models.UISettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid settings payload", core_errors.ErrTenantNotFound)
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to encode settings", core_errors.ErrInternal)
	}

	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("ui_settings", string(raw)).Error; err != nil {
		h.logger.Error("Failed to save UISettings", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save settings", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(settings, "tenant.settings.update.success")
}

// GetEnabledFeatures returns the enabled features for the tenant, computed live from the
// company's current subscription plan.
func (h *TenantHandler) GetEnabledFeatures(c *gin.Context) envelope.Response {
	_, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var company company_models.Company
	if err := tenantdb.For(c, h.db).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
	}

	features, err := company_services.EnabledFeaturesForCompany(h.db, company.ID)
	if err != nil {
		h.logger.Error("Failed to compute enabled features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining features", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(features, "tenant.features.fetch.success")
}

// isDisplayPairingCode reports whether token is an 8-digit TV pairing code. Signup
// stores a random 32-hex placeholder instead, which can't be typed on the TV.
func isDisplayPairingCode(token string) bool {
	if len(token) != 8 {
		return false
	}
	for _, r := range token {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GetDisplayToken returns the tenant's current TV pairing code without changing it,
// so viewing or sharing the code never unpairs the TV. A code is created only when
// the tenant has none yet; rotating is GenerateDisplayToken.
func (h *TenantHandler) GetDisplayToken(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}

	var tenantRecord tenant_models.Tenant
	if err := h.db.Select("id", "display_token").First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
	}
	if isDisplayPairingCode(tenantRecord.DisplayToken) {
		return envelope.SuccessResponse(gin.H{"token": tenantRecord.DisplayToken}, "tenant.display_token.fetch.success")
	}

	// Only replace the token we read: if a concurrent request already created a
	// code, keep that one instead of overwriting it.
	code := fmt.Sprintf("%08d", rand.IntN(100_000_000))
	if err := h.db.Model(&tenant_models.Tenant{}).
		Where("id = ? AND display_token = ?", tenantRecord.ID, tenantRecord.DisplayToken).
		Update("display_token", code).Error; err != nil {
		h.logger.Error("Failed to create display token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create display token", core_errors.ErrInternal)
	}
	if err := h.db.Select("id", "display_token").First(&tenantRecord, tenantRecord.ID).Error; err != nil {
		h.logger.Error("Failed to reload display token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create display token", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"token": tenantRecord.DisplayToken}, "tenant.display_token.fetch.success")
}
