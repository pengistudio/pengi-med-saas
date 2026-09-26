package sri_document

import (
	"time"

	billing_models "pengi-med-saas/features/billing/models"

	"go.uber.org/zap"
)

// AuthorizationRecheckAfter is how long a received document waits before a sweep
// asks the SRI for its authorization again.
const AuthorizationRecheckAfter = 2 * time.Minute

// Sweep re-enqueues documents a consumer should look at again: attempts stuck in
// "processing"/"signed" (worker died mid-run) and received documents still
// awaiting the SRI's authorization. It only publishes; Process decides what to
// do, so a late or duplicate task is harmless.
func (l *Lifecycle) Sweep(kinds ...Kind) {
	now := time.Now()
	for _, kind := range kinds {
		var ids []uint64
		// No TenantScope: runs outside any request, across every tenant.
		if err := l.db.Model(kind.Model).
			Where("(status IN ? AND updated_at < ?) OR (status = ? AND updated_at < ?)",
				stuckStatuses, now.Add(-StuckThreshold),
				billing_models.InvoiceStatusValidated, now.Add(-AuthorizationRecheckAfter)).
			Order("id").
			Pluck("id", &ids).Error; err != nil {
			l.logger.Error("failed to fetch SRI documents to sweep", zap.String("kind", kind.Name), zap.Error(err))
			continue
		}
		for _, id := range ids {
			if err := l.publish(kind, id); err != nil {
				l.logger.Error("failed to requeue SRI document", zap.String("kind", kind.Name), zap.Uint64("id", id), zap.Error(err))
			}
		}
	}
}

// RunSweeper sweeps every interval, forever. Run it in a goroutine.
func (l *Lifecycle) RunSweeper(interval time.Duration, kinds ...Kind) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		l.Sweep(kinds...)
	}
}
