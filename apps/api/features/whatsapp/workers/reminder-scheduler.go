package whatsapp_workers

import (
	"time"

	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// staleQueuedAfter is how long a queued message may wait before the scheduler
// publishes it again (the broker was down, or the retry ladder gave up). The
// consumer's claim makes a duplicate task harmless.
const staleQueuedAfter = 15 * time.Minute

// ReminderScheduler queues WhatsApp appointment reminders. Every minute it
// looks at each tenant whose account can send reminders, picks the
// appointments whose reminder offset is due (whatsapp_services.DueOffset),
// inserts one queued WhatsAppMessage per (appointment, offset) — the unique
// index makes that idempotent across ticks and API instances — and publishes
// it to whatsapp.send.
type ReminderScheduler struct {
	db        *gorm.DB
	logger    *zap.Logger
	publisher whatsapp_services.Publisher
	loc       *time.Location
	now       func() time.Time
}

func NewReminderScheduler(db *gorm.DB, logger *zap.Logger, publisher whatsapp_services.Publisher) *ReminderScheduler {
	return &ReminderScheduler{db: db, logger: logger, publisher: publisher, loc: whatsapp_services.ClinicLocation(), now: time.Now}
}

// WithClock overrides the clock and time zone (tests).
func (s *ReminderScheduler) WithClock(now func() time.Time, loc *time.Location) *ReminderScheduler {
	s.now, s.loc = now, loc
	return s
}

// Start runs Tick every minute.
func (s *ReminderScheduler) Start() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.Tick()
	}
}

// Tick queues every due reminder and re-publishes stale queued messages.
func (s *ReminderScheduler) Tick() {
	var accounts []whatsapp_models.WhatsAppAccount
	err := tenantdb.System(s.db).
		Where("status = ? AND reminders_enabled = ? AND template_status = ?",
			whatsapp_models.AccountStatusConnected, true, whatsapp_models.TemplateStatusApproved).
		Find(&accounts).Error
	if err != nil {
		s.logger.Error("whatsapp reminders: failed to load accounts", zap.Error(err))
		return
	}
	for _, account := range accounts {
		if account.CanSendReminders() {
			s.queueTenant(account)
		}
	}
	s.recoverStuckSending()
	s.republishStale()
}

func (s *ReminderScheduler) queueTenant(account whatsapp_models.WhatsAppAccount) {
	now := s.now()
	db := tenantdb.ForTenant(s.db, account.TenantID)
	// Coarse window on the stored date (local midnight, see AppointmentStart);
	// the exact start is computed in Go.
	from := now.Add(-48 * time.Hour)
	until := now.Add(time.Duration(whatsapp_services.MaxOffset(account.ReminderOffsets))*time.Hour + 48*time.Hour)

	var appts []clinical_models.Appointment
	err := db.Select("id", "tenant_id", "patient_id", "date", "start_time", "status").
		Where("status IN ? AND date >= ? AND date <= ?",
			[]string{whatsapp_services.AppointmentScheduled, whatsapp_services.AppointmentConfirmed}, from, until).
		// Only patients who agreed to WhatsApp reminders (Meta requires opt-in).
		Where("patient_id IN (?)", db.Model(&clinical_models.Patient{}).Select("id").Where("whatsapp_opt_in = ?", true)).
		Find(&appts).Error
	if err != nil {
		s.logger.Error("whatsapp reminders: failed to load appointments", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return
	}

	for _, appt := range appts {
		start, err := whatsapp_services.AppointmentStart(appt, s.loc)
		if err != nil {
			continue
		}
		offset := whatsapp_services.DueOffset(start, now, account.ReminderOffsets)
		if offset == 0 {
			continue
		}
		id, created, err := QueueReminder(db, appt, offset)
		if err != nil {
			s.logger.Error("whatsapp reminders: failed to queue", zap.Uint("appointment_id", appt.ID), zap.Error(err))
			continue
		}
		if !created {
			continue
		}
		if err := whatsapp_services.EnqueueSend(s.publisher, id); err != nil {
			// Left queued; republishStale picks it up.
			s.logger.Warn("whatsapp reminders: failed to publish", zap.Uint("message_id", id), zap.Error(err))
		}
	}
}

// QueueReminder inserts the queued reminder of appt for offset unless one
// already exists (ON CONFLICT DO NOTHING on the unique index). created says
// whether this call inserted it.
func QueueReminder(db *gorm.DB, appt clinical_models.Appointment, offset int) (id uint, created bool, err error) {
	apptID, patientID := appt.ID, appt.PatientID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:      appt.TenantID,
		Kind:          whatsapp_models.KindReminder,
		AppointmentID: &apptID,
		PatientID:     &patientID,
		OffsetHours:   offset,
		Template:      whatsapp_models.ReminderTemplate,
		Status:        whatsapp_models.MessageStatusQueued,
	}
	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "appointment_id"}, {Name: "offset_hours"}},
		DoNothing: true,
	}).Create(&msg)
	if res.Error != nil {
		return 0, false, res.Error
	}
	return msg.ID, res.RowsAffected == 1, nil
}

// recoverStuckSending puts reminders left in "sending" (the consumer died
// between claiming and recording the result) back to queued so they are
// retried. A send takes seconds (the client times out at 20 s), so after
// staleQueuedAfter the claim is certainly dead; the rare cost is a duplicate
// if the process died right after Meta accepted the message. Test messages
// are sent synchronously by a request, so a stuck one is marked failed.
func (s *ReminderScheduler) recoverStuckSending() {
	cutoff := s.now().Add(-staleQueuedAfter)
	db := tenantdb.System(s.db).Model(&whatsapp_models.WhatsAppMessage{})
	if err := db.Where("status = ? AND kind = ? AND updated_at < ?", whatsapp_models.MessageStatusSending, whatsapp_models.KindReminder, cutoff).
		Updates(map[string]any{"status": whatsapp_models.MessageStatusQueued, "updated_at": cutoff.Add(-time.Second)}).Error; err != nil { // updated_at before cutoff: republished below, this tick
		s.logger.Error("whatsapp reminders: failed to recover stuck messages", zap.Error(err))
	}
	if err := tenantdb.System(s.db).Model(&whatsapp_models.WhatsAppMessage{}).
		Where("status = ? AND kind = ? AND updated_at < ?", whatsapp_models.MessageStatusSending, whatsapp_models.KindTest, cutoff).
		Updates(map[string]any{"status": whatsapp_models.MessageStatusFailed, "error_code": whatsapp_models.ErrorCodeInternal}).Error; err != nil {
		s.logger.Error("whatsapp reminders: failed to fail stuck test messages", zap.Error(err))
	}
}

func (s *ReminderScheduler) republishStale() {
	var stale []whatsapp_models.WhatsAppMessage
	err := tenantdb.System(s.db).Select("id", "tenant_id").
		Where("status = ? AND kind = ? AND updated_at < ?", whatsapp_models.MessageStatusQueued, whatsapp_models.KindReminder, s.now().Add(-staleQueuedAfter)).
		Order("id").Limit(100).Find(&stale).Error
	if err != nil {
		s.logger.Error("whatsapp reminders: failed to load stale messages", zap.Error(err))
		return
	}
	for _, m := range stale {
		if err := whatsapp_services.EnqueueSend(s.publisher, m.ID); err != nil {
			s.logger.Warn("whatsapp reminders: failed to republish", zap.Uint("message_id", m.ID), zap.Error(err))
			return
		}
		tenantdb.ForTenant(s.db, m.TenantID).Model(&whatsapp_models.WhatsAppMessage{}).Where("id = ?", m.ID).Update("updated_at", s.now())
	}
}
