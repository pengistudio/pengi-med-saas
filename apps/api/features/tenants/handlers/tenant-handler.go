package tenant_handlers

import (
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
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
		"is_configured":   isConfigured,
		"expiration_date": tenantRecord.SriCertExpiration,
	}, "Configuración recuperada")
}
