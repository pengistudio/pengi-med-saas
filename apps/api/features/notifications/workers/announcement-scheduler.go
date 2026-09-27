package notifications_workers

import (
	"time"

	"pengi-med-saas/core/tenantdb"
	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_service "pengi-med-saas/features/notifications/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AnnouncementScheduler sends backoffice announcements whose scheduled time
// has come. Announcements without a scheduled time are sent by the handler
// right away and never reach it.
type AnnouncementScheduler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAnnouncementScheduler(db *gorm.DB, logger *zap.Logger) *AnnouncementScheduler {
	return &AnnouncementScheduler{db: tenantdb.System(db), logger: logger} // spans every tenant
}

// Start begins the scheduler loop that runs every minute.
func (s *AnnouncementScheduler) Start() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.SendDue()
	}
}

// SendDue dispatches every due scheduled announcement, one per transaction.
// The row lock (SKIP LOCKED) keeps two API instances from sending the same
// announcement twice.
func (s *AnnouncementScheduler) SendDue() {
	// IDs already handled this run: if even marking one failed didn't stick,
	// it must not be picked again in a tight loop.
	tried := []uint{0}
	for {
		sent := false
		err := s.db.Transaction(func(tx *gorm.DB) error {
			var a notifications_models.Announcement
			err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("status = ? AND scheduled_at <= ?", notifications_models.AnnouncementStatusScheduled, time.Now()).
				Where("id NOT IN ?", tried).
				Order("scheduled_at").
				Limit(1).
				Find(&a).Error
			if err != nil || a.ID == 0 {
				return err
			}
			sent = true
			tried = append(tried, a.ID)
			// Dispatch logs a failure and marks the announcement failed itself
			// (inside this transaction), so the loop moves on instead of retrying.
			_ = notifications_service.Dispatch(tx, s.logger, &a)
			return nil
		})
		if err != nil {
			s.logger.Error("failed to fetch due announcements", zap.Error(err))
			return
		}
		if !sent {
			return
		}
	}
}
