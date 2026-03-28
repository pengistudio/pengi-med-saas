package tenant_handlers

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_dto "pengi-med-saas/features/tenants/dto"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"software.sslmate.com/src/go-pkcs12"
)

type TenantHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTenantHandler(db *gorm.DB, logger *zap.Logger) *TenantHandler {
	return &TenantHandler{db: db, logger: logger}
}

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

	uploadDir := filepath.Join("storage", "tenants", fmt.Sprint(tenantID))
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		h.logger.Error("Failed to create storage directory", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create storage directory", core_errors.ErrInternal)
	}

	signaturePath := filepath.Join(uploadDir, "signature.p12")
	out, err := os.Create(signaturePath)
	if err != nil {
		h.logger.Error("Failed to create signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create signature file", core_errors.ErrInternal)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		h.logger.Error("Failed to save signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save signature file", core_errors.ErrInternal)
	}

	// Read the uploaded file to parse the certificate
	pfxData, err := os.ReadFile(signaturePath)
	if err != nil {
		h.logger.Error("Failed to read signature file for parsing", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to read signature file", core_errors.ErrInternal)
	}

	// Try to decode to extract the expiration date
	var cert *x509.Certificate
	var decodeErr error

	// 1. Try DecodeChain (handles intermediate CAs & legacy ciphers)
	_, cert, _, decodeErr = pkcs12.DecodeChain(pfxData, password)
	if decodeErr != nil {
		// 2. Fallback to simple Decode (strictly 2 bags)
		_, cert, decodeErr = pkcs12.Decode(pfxData, password)
	}

	if decodeErr != nil {
		h.logger.Warn("Failed to decode PKCS12 file giving up all fallbacks, possible invalid password or corrupted file", zap.Error(decodeErr))
		// Delete the file since it's invalid
		os.Remove(signaturePath)
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.sri.invalid_file", core_errors.ErrBillingInvalidSignatureFile)
	}

	// Update tenant record
	var tenantRecord tenant_models.Tenant
	if err := h.db.First(&tenantRecord, tenantID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound)
	}

	tenantRecord.SriPassword = password
	tenantRecord.SriP12Path = signaturePath
	tenantRecord.SriCertExpiration = &cert.NotAfter

	if err := h.db.Save(&tenantRecord).Error; err != nil {
		h.logger.Error("Failed to update tenant signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update tenant signature", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{
		"path":            signaturePath,
		"filename":        header.Filename,
		"size":            header.Size,
		"expiration_date": cert.NotAfter,
	}, "tenant.signature.upload.success")
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

	isConfigured := tenantRecord.SriP12Path != "" && tenantRecord.SriPassword != ""

	return envelope.SuccessResponse(gin.H{
		"is_configured":      isConfigured,
		"expiration_date":    tenantRecord.SriCertExpiration,
		"tax_id":             tenantRecord.TaxID,
		"trade_name":         tenantRecord.TradeName,
		"corporate_name":     tenantRecord.CorporateName,
		"address":            tenantRecord.Address,
		"accounting_obliged": tenantRecord.AccountingObliged,
	}, "tenant.sri.status.fetch.success")
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
	if err := h.db.Where("tenant_id = ? AND DATE(date) = ?", tenantRecord.ID, today).
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
