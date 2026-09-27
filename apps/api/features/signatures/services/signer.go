// Package signature_services signs tenant documents with the requesting
// user's electronic signature (P12).
package signature_services

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	signature_models "pengi-med-saas/features/signatures/models"
	auth_middleware "pengi-med-saas/features/users/middleware"
)

// ErrNotConfigured means the user has no signature in this tenant.
var ErrNotConfigured = errors.New("signatures: user has no signature")

// Location is printed on the stamp and stored in the PDF signature.
const Location = "Ecuador"

// Signer loads users' certificates and signs PDFs with them.
type Signer struct {
	db    *gorm.DB
	files tenantfiles.Store
	now   func() time.Time
}

func NewSigner(db *gorm.DB, files tenantfiles.Store) *Signer {
	return &Signer{db: db, files: files, now: time.Now}
}

// SignedDocument is a signed PDF plus who signed it and when.
type SignedDocument struct {
	PDF        []byte
	SignerID   uint
	SignerName string
	SignedAt   time.Time
}

// P12FileName is where a user's certificate lives inside the tenant folder.
func P12FileName(userID uint) string { return fmt.Sprintf("signatures/user-%d.p12", userID) }

// SignedFileName is where a signed document's PDF lives inside the tenant
// folder; signedAt keeps each signing attempt in its own file.
func SignedFileName(kind string, id uint, signedAt time.Time) string {
	return fmt.Sprintf("signed/%s-%d-%d.pdf", kind, id, signedAt.UnixNano())
}

// CurrentUserID is the authenticated user's ID.
func CurrentUserID(c *gin.Context) (uint, bool) {
	uid, _, ok := auth_middleware.GetUserFromContext(c)
	return uint(uid), ok
}

// Load returns the current user's certificate, decrypting its password.
func (s *Signer) Load(c *gin.Context) (*pdfsign.Certificate, error) {
	userID, ok := CurrentUserID(c)
	if !ok {
		return nil, ErrNotConfigured
	}
	var sig signature_models.UserSignature
	if err := tenantdb.For(c, s.db).Where("user_id = ?", userID).First(&sig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotConfigured
		}
		return nil, err
	}
	box, err := secretbox.FromEnv()
	if err != nil {
		return nil, err
	}
	password, err := box.Open(sig.EncryptedPassword)
	if err != nil {
		return nil, err
	}
	p12, err := s.files.Read(tenantdb.TenantID(c), sig.FileName)
	if err != nil {
		return nil, err
	}
	return pdfsign.Load(p12, password)
}

// Sign renders the document with the current user's stamp and signs it.
// render receives the stamp to draw in the "Firma y sello" box.
func (s *Signer) Sign(c *gin.Context, reason string, render func(stamp *pdfsign.Stamp) ([]byte, error)) (*SignedDocument, error) {
	cert, err := s.Load(c)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := cert.ValidAt(now); err != nil {
		return nil, err
	}
	opts := pdfsign.Options{Reason: reason, Location: Location, SignedAt: now}
	stamp, err := pdfsign.NewStamp(cert, opts)
	if err != nil {
		return nil, err
	}
	pdf, err := render(stamp)
	if err != nil {
		return nil, err
	}
	signed, err := pdfsign.Sign(pdf, cert, opts)
	if err != nil {
		return nil, err
	}
	userID, _ := CurrentUserID(c)
	return &SignedDocument{PDF: signed, SignerID: userID, SignerName: cert.SubjectName(), SignedAt: now}, nil
}

// ErrorResponse maps a signing error to its API response.
func ErrorResponse(err error) envelope.Response {
	switch {
	case errors.Is(err, ErrNotConfigured):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.not_configured", core_errors.ErrSignatureNotConfigured)
	case errors.Is(err, pdfsign.ErrExpiredCertificate):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.expired", core_errors.ErrSignatureExpired)
	case errors.Is(err, pdfsign.ErrCertificateNotYetValid):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.not_yet_valid", core_errors.ErrSignatureNotYetValid)
	case errors.Is(err, pdfsign.ErrWrongPassword):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.wrong_password", core_errors.ErrSignatureWrongPassword)
	case errors.Is(err, pdfsign.ErrInvalidCertificate):
		return envelope.ErrorResponse(http.StatusBadRequest, "signature.error.invalid_file", core_errors.ErrSignatureInvalidFile)
	case errors.Is(err, secretbox.ErrNoKey):
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "signature.error.unavailable", core_errors.ErrSignatureKeyUnavailable)
	}
	return envelope.ErrorResponse(http.StatusInternalServerError, "signature.error.sign_failed", core_errors.ErrSignatureSignFailed)
}

// AlreadySignedResponse is returned when signing a document twice.
func AlreadySignedResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "signature.error.already_signed", core_errors.ErrSignatureAlreadySigned)
}
