// Package portal_workers runs the patient portal's background work.
package portal_workers

import (
	"time"

	"pengi-med-saas/core/tenantdb"
	portal_models "pengi-med-saas/features/portal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// expiryBatch bounds the requests one tick expires (the rest wait a tick).
const expiryBatch = 500

// RequestNotifier tells the patient how a request ended
// (*portal_services.Delivery in production).
type RequestNotifier interface {
	SendRequestNotice(tenantID, requestID uint) string
}

// RequestExpiryScheduler expires the portal appointment requests nobody
// decided in time: every 5 minutes, across tenants, each pending request
// whose ExpiresAt passed becomes expired and its patient is told. The update
// is conditional (status still pending), so a request decided meanwhile, or
// expired by another API instance, is left alone and notified only once.
type RequestExpiryScheduler struct {
	db      *gorm.DB
	logger  *zap.Logger
	notices RequestNotifier
	now     func() time.Time
}

func NewRequestExpiryScheduler(db *gorm.DB, logger *zap.Logger, notices RequestNotifier) *RequestExpiryScheduler {
	return &RequestExpiryScheduler{db: db, logger: logger, notices: notices, now: time.Now}
}

// WithClock overrides the clock (tests).
func (s *RequestExpiryScheduler) WithClock(now func() time.Time) *RequestExpiryScheduler {
	s.now = now
	return s
}

// Start runs Tick every 5 minutes.
func (s *RequestExpiryScheduler) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.Tick()
	}
}

// Tick expires the due requests and returns how many it expired.
func (s *RequestExpiryScheduler) Tick() int {
	now := s.now().UTC() // SQLite compares timestamps as text: one zone everywhere
	db := tenantdb.System(s.db)
	var due []portal_models.AppointmentRequest
	if err := db.Select("id", "tenant_id").
		Where("status = ? AND expires_at <= ?", portal_models.RequestPending, now).
		Order("id").Limit(expiryBatch).Find(&due).Error; err != nil {
		s.logger.Error("portal requests: failed to load expired requests", zap.Error(err))
		return 0
	}
	expired := 0
	for _, r := range due {
		res := db.Model(&portal_models.AppointmentRequest{}).
			Where("id = ? AND status = ?", r.ID, portal_models.RequestPending).
			Updates(map[string]any{"status": portal_models.RequestExpired, "updated_at": now})
		if res.Error != nil {
			s.logger.Error("portal requests: failed to expire request", zap.Uint("id", r.ID), zap.Error(res.Error))
			continue
		}
		if res.RowsAffected == 0 {
			continue // decided or expired meanwhile
		}
		expired++
		if s.notices != nil {
			s.notices.SendRequestNotice(r.TenantID, r.ID)
		}
	}
	return expired
}
