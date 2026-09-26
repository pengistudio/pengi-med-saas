package billing_workers

import (
	"errors"
	"fmt"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"

	"gorm.io/gorm"
)

// StuckProcessingThreshold is how long a document may sit in "processing" before we
// assume its worker died mid-run. A run only holds "processing" until the XML is
// signed (bounded by the signer client's 30s timeout), i.e. before anything reaches
// SRI, so reclaiming it afterwards cannot register a duplicate.
const StuckProcessingThreshold = 15 * time.Minute

var errTransient = errors.New("transient worker error")

var claimableStatuses = []string{
	billing_models.InvoiceStatusPending,
	billing_models.InvoiceStatusFailed,
	billing_models.InvoiceStatusConnectionError,
}

// claimForProcessing atomically moves a document to "processing" so two deliveries
// of the same document never run concurrently. It returns false when the document
// is already being processed, already done, or gone.
func claimForProcessing(db *gorm.DB, model any, id uint64) (bool, error) {
	res := db.Unscoped().Model(model).
		Where("id = ? AND (status IN ? OR (status = ? AND updated_at < ?))",
			id, claimableStatuses, billing_models.InvoiceStatusProcessing, time.Now().Add(-StuckProcessingThreshold)).
		Update("status", billing_models.InvoiceStatusProcessing)
	if res.Error != nil {
		return false, fmt.Errorf("%w: %w", errTransient, res.Error)
	}
	return res.RowsAffected == 1, nil
}

func isRetryable(err error) bool {
	return errors.Is(err, errTransient) || errors.Is(err, sri_services.ErrSignerUnreachable)
}
