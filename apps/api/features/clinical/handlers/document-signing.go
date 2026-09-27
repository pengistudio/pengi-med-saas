package clinical_handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_models "pengi-med-saas/features/clinical/models"
	signature_services "pengi-med-saas/features/signatures/services"
)

// errAlreadySigned means another request signed the document first.
var errAlreadySigned = errors.New("clinical: document already signed")

// signatureColumns are the DocumentSignature columns written when signing.
var signatureColumns = []string{"signed_by_id", "signed_at", "signer_name", "signed_file"}

// clearedSignature resets a document's signature columns (the content changed,
// so its signed PDF no longer matches).
func clearedSignature() map[string]any {
	return map[string]any{"signed_by_id": nil, "signed_at": nil, "signer_name": "", "signed_file": ""}
}

// prescriptionChanged reports whether updates change the prescription's
// printed content (content/indications), which invalidates its signature.
func prescriptionChanged(current *clinical_models.Prescription, updates map[string]any) bool {
	if current == nil {
		return true
	}
	if v, ok := updates["content"]; ok && v != current.Content {
		return true
	}
	if v, ok := updates["indications"]; ok && v != current.Indications {
		return true
	}
	return false
}

// signDocument signs a document with the current user's certificate, stores
// the signed PDF and records the signature on row (a pointer to the model),
// only if it is still unsigned. render draws the document with the stamp.
func signDocument(
	c *gin.Context, db *gorm.DB, signer *signature_services.Signer, files tenantfiles.Store,
	row any, kind string, id uint, reason string,
	render func(stamp *pdfsign.Stamp) ([]byte, error),
) (clinical_models.DocumentSignature, error) {
	signed, err := signer.Sign(c, reason, render)
	if err != nil {
		return clinical_models.DocumentSignature{}, err
	}

	// A fresh name per attempt: a request that loses the race below never
	// overwrites the winner's PDF.
	fileName := signature_services.SignedFileName(kind, id, signed.SignedAt)
	tenantID := tenantdb.TenantID(c)
	if err := files.Write(tenantID, fileName, signed.PDF); err != nil {
		return clinical_models.DocumentSignature{}, err
	}

	sig := clinical_models.DocumentSignature{
		SignedByID: &signed.SignerID,
		SignedAt:   &signed.SignedAt,
		SignerName: signed.SignerName,
		SignedFile: fileName,
	}
	res := tenantdb.For(c, db).Model(row).
		Where("id = ? AND (signed_file = '' OR signed_file IS NULL)", id).
		Select(signatureColumns).
		Updates(map[string]any{
			"signed_by_id": sig.SignedByID,
			"signed_at":    sig.SignedAt,
			"signer_name":  sig.SignerName,
			"signed_file":  sig.SignedFile,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		_ = files.Remove(tenantID, fileName)
		if res.Error != nil {
			return clinical_models.DocumentSignature{}, res.Error
		}
		return clinical_models.DocumentSignature{}, errAlreadySigned
	}
	return sig, nil
}

// storedOrRendered returns the signed PDF when the document is signed,
// otherwise renders it fresh (unsigned).
func storedOrRendered(c *gin.Context, files tenantfiles.Store, sig clinical_models.DocumentSignature, render func() ([]byte, error)) ([]byte, error) {
	if sig.IsSigned() {
		return files.Read(tenantdb.TenantID(c), sig.SignedFile)
	}
	return render()
}
