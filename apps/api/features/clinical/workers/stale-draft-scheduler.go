package clinical_workers

import (
	"fmt"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_service "pengi-med-saas/features/notifications/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StaleDraftThreshold is how long a medical record draft can go untouched
// before we remind the doctor about it. Fixed for v1 — a per-tenant
// configurable threshold (mirroring kanban's UISettings.AutoArchiveDelay)
// is a natural follow-up if needed.
const StaleDraftThreshold = 60 * time.Minute

type StaleDraftScheduler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewStaleDraftScheduler(db *gorm.DB, logger *zap.Logger) *StaleDraftScheduler {
	return &StaleDraftScheduler{db: db, logger: logger}
}

// Start begins the scheduler loop that runs every 10 minutes.
func (s *StaleDraftScheduler) Start() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.checkStaleDrafts()
	}
}

// checkStaleDrafts finds drafts whose last save is older than
// StaleDraftThreshold and notifies the owning doctor, deduped by
// notifications_service.CreateIfNotExists so repeated ticks don't spam.
func (s *StaleDraftScheduler) checkStaleDrafts() {
	var drafts []clinical_models.MedicalRecordDraft

	// No TenantScope here — this runs outside any gin.Context, same
	// precedent as kanban's ArchiveScheduler: query cross-tenant, then
	// resolve tenant-specific data (or here, patient data) per row.
	cutoff := time.Now().Add(-StaleDraftThreshold)
	if err := s.db.Where("updated_at < ?", cutoff).Find(&drafts).Error; err != nil {
		s.logger.Error("failed to fetch stale medical record drafts", zap.Error(err))
		return
	}

	notifiedCount := 0

	for _, draft := range drafts {
		var patient clinical_models.Patient
		if err := s.db.First(&patient, draft.PatientID).Error; err != nil {
			s.logger.Warn("patient not found for stale draft", zap.Uint("draft_id", draft.ID), zap.Error(err))
			continue
		}

		input := notifications_service.CreateNotificationInput{
			TenantID:     draft.TenantID,
			UserID:       draft.UserID,
			Type:         "clinical.draft.stale",
			ResourceType: "medical_record_draft",
			ResourceID:   draft.ID,
			MessageKey:   "notification.clinical.draft.stale",
			Params: map[string]string{
				"patient_name":     fmt.Sprintf("%s %s", patient.FirstName, patient.LastName),
				"draft_updated_at": draft.UpdatedAt.Format(time.RFC3339),
			},
			ActionURL: fmt.Sprintf("/clinical/medical-records/%d", draft.PatientID),
		}

		if err := notifications_service.CreateIfNotExists(s.db, s.logger, input); err != nil {
			s.logger.Error("failed to create stale draft notification", zap.Uint("draft_id", draft.ID), zap.Error(err))
			continue
		}
		notifiedCount++
	}

	if notifiedCount > 0 {
		s.logger.Info("notified stale medical record drafts", zap.Int("count", notifiedCount))
	}
}
