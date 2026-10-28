package clinical_workers

import (
	"time"

	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// noShowInterval is how often the no-show scheduler runs.
const noShowInterval = 5 * time.Minute

// noShowPending are the statuses a no-show can follow: the patient never
// checked in. arrived, in_consultation, completed and cancelled are never touched.
var noShowPending = []string{clinical_models.AppointmentStatusScheduled, clinical_models.AppointmentStatusConfirmed}

// CalendarSync mirrors a status change to Google Calendar
// (clinical_handlers.AppointmentHandler.SyncStatusChange).
type CalendarSync interface {
	SyncStatusChange(tenantID uint, appointment *clinical_models.Appointment)
}

// NoShowScheduler marks no-shows for the clinics that turned it on
// (UISettings.Agenda.AutoNoShow). An appointment becomes no_show when it is
// still scheduled or confirmed NoShowAfterMinutes after its end, its end is
// after AutoNoShowSince (history is never marked), and the patient has no
// medical record that day (see attended). Each change goes through the
// Appointment model, so it lands in the audit log (user 0: the system), and
// its Google Calendar event is removed as when reception marks it.
type NoShowScheduler struct {
	db       *gorm.DB
	logger   *zap.Logger
	calendar CalendarSync // nil: no calendar sync
	loc      *time.Location
	now      func() time.Time
	// messages queues the WhatsApp no-show message of each marked appointment.
	messages *whatsapp_services.NoShowMessages
}

func NewNoShowScheduler(db *gorm.DB, logger *zap.Logger, calendar CalendarSync) *NoShowScheduler {
	return &NoShowScheduler{db: db, logger: logger, calendar: calendar, loc: whatsapp_services.ClinicLocation(), now: time.Now,
		messages: whatsapp_services.NewNoShowMessages(db, logger, whatsapp_services.RabbitPublisher{})}
}

// WithClock overrides the clock and time zone (tests).
func (s *NoShowScheduler) WithClock(now func() time.Time, loc *time.Location) *NoShowScheduler {
	s.now, s.loc = now, loc
	s.messages.WithClock(now, loc)
	return s
}

// WithMessages overrides the no-show messages (tests); its clock is set by
// WithClock, so call this first.
func (s *NoShowScheduler) WithMessages(m *whatsapp_services.NoShowMessages) *NoShowScheduler {
	s.messages = m
	return s
}

// Start runs Tick every noShowInterval.
func (s *NoShowScheduler) Start() {
	ticker := time.NewTicker(noShowInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.Tick()
	}
}

// Tick marks the due no-shows of every tenant with the setting on and returns
// how many it marked. Running it again marks nothing new.
func (s *NoShowScheduler) Tick() int {
	var tenants []tenant_models.Tenant
	if err := tenantdb.System(s.db).Select("id", "ui_settings").Find(&tenants).Error; err != nil {
		s.logger.Error("no-show scheduler: failed to load tenants", zap.Error(err))
		return 0
	}
	marked := 0
	for _, t := range tenants {
		agenda := tenant_models.ParseUISettings(t.UISettings).Agenda
		if !agenda.AutoNoShow || agenda.AutoNoShowSince == nil {
			continue
		}
		minutes := agenda.NoShowAfterMinutes
		if !tenant_models.ValidNoShowAfterMinutes(minutes) {
			minutes = tenant_models.DefaultNoShowAfterMinutes
		}
		marked += s.markTenant(t.ID, *agenda.AutoNoShowSince, time.Duration(minutes)*time.Minute)
	}
	if marked > 0 {
		s.logger.Info("no-show scheduler: appointments marked", zap.Int("count", marked))
	}
	return marked
}

func (s *NoShowScheduler) markTenant(tenantID uint, since time.Time, grace time.Duration) int {
	now := s.now()
	// audit_user_id 0: the change is audited as the system's.
	db := tenantdb.ForTenant(s.db, tenantID).Set("audit_user_id", uint(0)).Session(&gorm.Session{})

	// Coarse window on the stored date (local midnight, see
	// whatsapp_services.AppointmentStart); the exact end is computed in Go.
	var appts []clinical_models.Appointment
	err := db.Select("id", "tenant_id", "patient_id", "doctor_id", "date", "start_time", "end_time", "status", "google_event_id").
		Where("status IN ? AND date >= ? AND date <= ?", noShowPending, since.Add(-48*time.Hour), now).
		Find(&appts).Error
	if err != nil {
		s.logger.Error("no-show scheduler: failed to load appointments", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return 0
	}

	marked := 0
	for i := range appts {
		appt := &appts[i]
		end, err := appointmentEnd(*appt, s.loc)
		if err != nil || !end.After(since) || now.Sub(end) <= grace {
			continue
		}
		attended, err := s.attended(db, *appt)
		if err != nil {
			s.logger.Error("no-show scheduler: failed to look for a record", zap.Uint("appointment_id", appt.ID), zap.Error(err))
			continue
		}
		if attended {
			continue
		}
		// Re-checks the status: reception may have marked the arrival since.
		res := db.Model(appt).Where("status IN ?", noShowPending).Update("status", clinical_models.AppointmentStatusNoShow)
		if res.Error != nil {
			s.logger.Error("no-show scheduler: failed to mark", zap.Uint("appointment_id", appt.ID), zap.Error(res.Error))
			continue
		}
		if res.RowsAffected == 0 {
			continue
		}
		marked++
		appt.Status = clinical_models.AppointmentStatusNoShow
		if s.calendar != nil && appt.GoogleEventID != "" {
			s.calendar.SyncStatusChange(tenantID, appt)
		}
		s.messages.Marked(tenantID, appt.ID)
	}
	return marked
}

// attended reports whether the patient has a medical record for appt: one
// linked to it, or one the same local day for the same patient written by the
// same doctor (or by no doctor, as records from before doctors existed). An
// appointment without a doctor matches any record of the patient that day.
func (s *NoShowScheduler) attended(db *gorm.DB, appt clinical_models.Appointment) (bool, error) {
	d := appt.Date.In(s.loc)
	dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, s.loc)
	dayEnd := dayStart.AddDate(0, 0, 1)

	q := db.Model(&clinical_models.MedicalRecord{})
	if appt.DoctorID != nil {
		q = q.Where("(appointment_id = ? OR (patient_id = ? AND date >= ? AND date < ? AND (doctor_id = ? OR doctor_id IS NULL)))",
			appt.ID, appt.PatientID, dayStart, dayEnd, *appt.DoctorID)
	} else {
		q = q.Where("(appointment_id = ? OR (patient_id = ? AND date >= ? AND date < ?))",
			appt.ID, appt.PatientID, dayStart, dayEnd)
	}
	var count int64
	if err := q.Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// appointmentEnd is the instant an appointment ends: the calendar day of
// Date in loc (the frontend sends local midnight) at EndTime ("HH:MM") in loc.
func appointmentEnd(a clinical_models.Appointment, loc *time.Location) (time.Time, error) {
	clock, err := time.Parse("15:04", a.EndTime)
	if err != nil {
		return time.Time{}, err
	}
	d := a.Date.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), clock.Hour(), clock.Minute(), 0, 0, loc), nil
}
