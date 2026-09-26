package billing_workers

import (
	"encoding/json"
	"time"

	"pengi-med-saas/core/brokers/rabbitmq"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type stuckDocumentKind struct {
	name  string
	model any
	queue string
	body  func(id uint64) any
}

var stuckDocumentKinds = []stuckDocumentKind{
	{"invoice", &billing_models.Invoice{}, "invoice_tasks", func(id uint64) any { return billing_dto.InvoiceDTO{InvoiceID: id} }},
	{"credit_note", &billing_models.CreditNote{}, "credit_note_tasks", func(id uint64) any { return billing_dto.CreditNoteDTO{CreditNoteID: id} }},
	{"debit_note", &billing_models.DebitNote{}, "debit_note_tasks", func(id uint64) any { return billing_dto.DebitNoteDTO{DebitNoteID: id} }},
}

// StuckDocumentScheduler re-enqueues SRI documents left in "processing" by a worker
// that died mid-run (deploy, OOM, crash). It only publishes; the worker's claim
// decides whether the document is still stale, so a late or duplicate message is
// harmless.
type StuckDocumentScheduler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewStuckDocumentScheduler(db *gorm.DB, logger *zap.Logger) *StuckDocumentScheduler {
	return &StuckDocumentScheduler{db: db, logger: logger}
}

func (s *StuckDocumentScheduler) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.requeueStuckDocuments()
	}
}

func (s *StuckDocumentScheduler) requeueStuckDocuments() {
	channel := rabbitmq.PublishChannel()
	if channel == nil {
		return
	}

	// No TenantScope: runs outside any gin.Context, same precedent as the
	// kanban ArchiveScheduler and clinical StaleDraftScheduler.
	cutoff := time.Now().Add(-StuckProcessingThreshold)
	for _, kind := range stuckDocumentKinds {
		var ids []uint64
		if err := s.db.Model(kind.model).
			Where("status = ? AND updated_at < ?", billing_models.InvoiceStatusProcessing, cutoff).
			Pluck("id", &ids).Error; err != nil {
			s.logger.Error("failed to fetch stuck SRI documents", zap.String("kind", kind.name), zap.Error(err))
			continue
		}

		for _, id := range ids {
			body, err := json.Marshal(kind.body(id))
			if err != nil {
				s.logger.Error("failed to marshal stuck SRI document task", zap.String("kind", kind.name), zap.Uint64("id", id), zap.Error(err))
				continue
			}
			if err := rabbitmq.PublishMessage(channel, kind.queue, body); err != nil {
				s.logger.Error("failed to requeue stuck SRI document", zap.String("kind", kind.name), zap.Uint64("id", id), zap.Error(err))
				continue
			}
			s.logger.Warn("requeued SRI document stuck in processing", zap.String("kind", kind.name), zap.Uint64("id", id))
		}
	}
}
