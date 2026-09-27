package signature_handlers

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	signature_models "pengi-med-saas/features/signatures/models"
	signature_services "pengi-med-saas/features/signatures/services"
)

// expiringSoon is how far ahead a certificate is flagged as about to expire.
const expiringSoon = 30 * 24 * time.Hour

type SignatureHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	files  tenantfiles.Store
}

func NewSignatureHandler(db *gorm.DB, logger *zap.Logger, files tenantfiles.Store) *SignatureHandler {
	return &SignatureHandler{db: db, logger: logger, files: files}
}

type signatureStatus struct {
	Configured    bool       `json:"configured"`
	SubjectName   string     `json:"subject_name,omitempty"`
	SubjectSerial string     `json:"subject_serial,omitempty"`
	Issuer        string     `json:"issuer,omitempty"`
	NotAfter      *time.Time `json:"not_after,omitempty"`
	Expired       bool       `json:"expired"`
	ExpiringSoon  bool       `json:"expiring_soon"`
}

func statusOf(sig *signature_models.UserSignature) signatureStatus {
	if sig == nil {
		return signatureStatus{}
	}
	now := time.Now()
	return signatureStatus{
		Configured:    true,
		SubjectName:   sig.SubjectName,
		SubjectSerial: sig.SubjectSerial,
		Issuer:        sig.Issuer,
		NotAfter:      &sig.NotAfter,
		Expired:       now.After(sig.NotAfter),
		ExpiringSoon:  !now.After(sig.NotAfter) && sig.NotAfter.Sub(now) < expiringSoon,
	}
}

// find returns the current user's signature in this tenant, or nil.
func (h *SignatureHandler) find(c *gin.Context, userID uint) (*signature_models.UserSignature, error) {
	var sig signature_models.UserSignature
	err := tenantdb.For(c, h.db).Where("user_id = ?", userID).First(&sig).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

// GetMySignature returns the current user's signature status.
func (h *SignatureHandler) GetMySignature(c *gin.Context) envelope.Response {
	userID, ok := signature_services.CurrentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	sig, err := h.find(c, userID)
	if err != nil {
		h.logger.Error("Failed to fetch user signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(statusOf(sig), "signature.fetch.success")
}

// UploadMySignature stores (or replaces) the current user's P12.
func (h *SignatureHandler) UploadMySignature(c *gin.Context) envelope.Response {
	userID, ok := signature_services.CurrentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}

	password := c.PostForm("password")
	file, _, err := c.Request.FormFile("signature")
	if err != nil || password == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.invalid_file", core_errors.ErrSignatureInvalidFile)
	}
	defer file.Close()
	p12, err := io.ReadAll(file)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.invalid_file", core_errors.ErrSignatureInvalidFile)
	}

	cert, err := pdfsign.Load(p12, password)
	if err != nil {
		h.logger.Warn("Rejected signature upload", zap.Error(err))
		return signature_services.ErrorResponse(err)
	}
	if err := cert.ValidAt(time.Now()); err != nil {
		return signature_services.ErrorResponse(err)
	}

	box, err := secretbox.FromEnv()
	if err != nil {
		h.logger.Error("Signature encryption key unavailable", zap.Error(err))
		return signature_services.ErrorResponse(err)
	}
	encrypted, err := box.Seal(password)
	if err != nil {
		h.logger.Error("Failed to encrypt signature password", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	fileName := signature_services.P12FileName(userID)
	if err := h.files.Write(tenantdb.TenantID(c), fileName, p12); err != nil {
		h.logger.Error("Failed to store signature file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	sig, err := h.find(c, userID)
	if err != nil {
		h.logger.Error("Failed to fetch user signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if sig == nil {
		sig = &signature_models.UserSignature{TenantID: tenantdb.TenantID(c), UserID: userID}
	}
	sig.FileName = fileName
	sig.EncryptedPassword = encrypted
	sig.SubjectName = cert.SubjectName()
	sig.SubjectSerial = cert.SubjectSerial()
	sig.Issuer = cert.Issuer()
	sig.NotAfter = cert.NotAfter()

	if err := tenantdb.For(c, h.db).Save(sig).Error; err != nil {
		h.logger.Error("Failed to save user signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(statusOf(sig), "signature.upload.success")
}

// DeleteMySignature removes the current user's P12 and its record.
func (h *SignatureHandler) DeleteMySignature(c *gin.Context) envelope.Response {
	userID, ok := signature_services.CurrentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrUserNotFound)
	}
	sig, err := h.find(c, userID)
	if err != nil {
		h.logger.Error("Failed to fetch user signature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if sig != nil {
		// Hard delete: the (tenant, user) unique index must be free for a re-upload.
		if err := tenantdb.For(c, h.db).Unscoped().Delete(sig).Error; err != nil {
			h.logger.Error("Failed to delete user signature", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		if err := h.files.Remove(tenantdb.TenantID(c), sig.FileName); err != nil {
			h.logger.Warn("Failed to remove signature file", zap.Error(err))
		}
	}
	return envelope.SuccessResponse(statusOf(nil), "signature.delete.success")
}
