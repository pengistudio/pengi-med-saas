package sri_document

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	billing_models "pengi-med-saas/features/billing/models"

	"gorm.io/gorm"
)

// Enqueue requests processing of a document on behalf of a user. tenantDB must be
// scoped to the requesting tenant, so another tenant's document is ErrNotFound.
// A draft, failed or rejected (NO AUTORIZADO) document goes back to pending and
// is resent with the same access key once its cause was corrected (ficha técnica
// SRI offline §5.10, §5.12); a received one keeps its status and
// only has its authorization queried again. A document that would be signed is
// refused with ErrSignatureExpired while the tenant's certificate is expired.
func (l *Lifecycle) Enqueue(tenantDB *gorm.DB, kind Kind, id uint64) error {
	doc, err := l.load(tenantDB.Session(&gorm.Session{}), kind, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	switch doc.Status {
	case billing_models.InvoiceStatusAuthorized:
		return ErrAlreadyAuthorized
	case billing_models.InvoiceStatusValidated:
		// Already received by the SRI: only its authorization is queried, which
		// needs no signature.
	default:
		// Anything else will be signed: refuse up front rather than queue an
		// attempt that is bound to fail.
		tenant, err := l.tenant(doc.TenantID)
		if err != nil {
			return err
		}
		if signatureExpired(tenant, time.Now()) {
			return ErrSignatureExpired
		}
	}

	switch doc.Status {
	case billing_models.InvoiceStatusDraft, billing_models.InvoiceStatusFailed,
		billing_models.InvoiceStatusConnectionError, billing_models.InvoiceStatusRejected:
		if err := tenantDB.Session(&gorm.Session{}).Model(kind.Model).Where("id = ?", doc.ID).Updates(map[string]any{
			"status":        billing_models.InvoiceStatusPending,
			"error_code":    nil,
			"error_message": nil,
		}).Error; err != nil {
			return fmt.Errorf("mark %s %d pending: %w", kind.Name, doc.ID, err)
		}
	}

	return l.publish(kind, uint64(doc.ID))
}

func (l *Lifecycle) publish(kind Kind, id uint64) error {
	body, err := json.Marshal(map[string]uint64{kind.MessageIDField: id})
	if err != nil {
		return err
	}
	if err := l.publisher.Publish(kind.Queue, body); err != nil {
		return fmt.Errorf("publish %s %d: %w", kind.Name, id, err)
	}
	return nil
}
