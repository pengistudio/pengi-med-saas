package whatsapp_services

import (
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// noShowRepublishAfter is how long a published no-show message may stay
// queued before PublishDue publishes it again (broker down, retries given
// up). The consumer's claim makes a duplicate task harmless.
const noShowRepublishAfter = 15 * time.Minute

// NoShowMessages queues the reschedule invitation (NoShowTemplate) when an
// appointment is marked no_show, by reception or by the no-show scheduler,
// and withdraws it when the mark is corrected before it goes out.
//
// One message per appointment, ever: the (appointment_id, offset_hours)
// unique index with NoShowOffsetHours. A message withdrawn because the mark
// was corrected was never sent, so marking the appointment again revives it;
// one sent (or failed, or skipped for any other reason) is never queued again.
//
// It is queued only when the clinic turned it on, the account is connected,
// the template is approved on the clinic's WABA, the patient consented and
// has a valid phone, and the monthly cap has room; the sender checks all of
// that again when it sends. Outside the send hours (NextSendTime) it waits
// in SendAfter; PublishDue (every reminder tick) publishes it when due.
type NoShowMessages struct {
	db        *gorm.DB
	logger    *zap.Logger
	publisher Publisher
	loc       *time.Location
	now       func() time.Time
}

func NewNoShowMessages(db *gorm.DB, logger *zap.Logger, publisher Publisher) *NoShowMessages {
	return &NoShowMessages{db: db, logger: logger, publisher: publisher, loc: ClinicLocation(), now: time.Now}
}

// WithClock overrides the clock and time zone (tests).
func (n *NoShowMessages) WithClock(now func() time.Time, loc *time.Location) *NoShowMessages {
	n.now, n.loc = now, loc
	return n
}

// Marked queues the no-show message of appointmentID, just marked no_show.
// Never fails the caller: problems are logged.
func (n *NoShowMessages) Marked(tenantID, appointmentID uint) {
	db := tenantdb.ForTenant(n.db, tenantID)
	var account whatsapp_models.WhatsAppAccount
	if err := db.Limit(1).Find(&account).Error; err != nil || account.ID == 0 || !account.CanSendNoShowMessages() {
		return
	}
	if !templateApproved(db, whatsapp_models.TemplateNoShow) {
		return
	}
	var appt clinical_models.Appointment
	if err := db.Preload("Patient").Where("id = ?", appointmentID).Limit(1).Find(&appt).Error; err != nil || appt.ID == 0 {
		if err != nil {
			n.logger.Error("whatsapp no-show: failed to load appointment", zap.Uint("appointment_id", appointmentID), zap.Error(err))
		}
		return
	}
	if appt.Status != clinical_models.AppointmentStatusNoShow || !appt.Patient.WhatsAppOptIn {
		return
	}
	to, err := NormalizePhone(appt.Patient.Phone)
	if err != nil {
		return
	}
	now := n.now()
	allowed, _, err := canSendTemplate(n.db, tenantID, now, n.loc)
	if err != nil {
		// Fail open like the sender: it checks the cap again before sending.
		n.logger.Error("whatsapp no-show: failed to check monthly limit", zap.Uint("tenant_id", tenantID), zap.Error(err))
	} else if !allowed {
		return
	}

	var sendAfter *time.Time
	if at := NextSendTime(now, n.loc); at.After(now) {
		utc := at.UTC() // UTC: compared as text on SQLite (tests)
		sendAfter = &utc
	}
	id, ok := n.queue(db, appt, to, sendAfter, now)
	if !ok || sendAfter != nil {
		return
	}
	if err := EnqueueSend(n.publisher, id); err != nil {
		// Left queued; PublishDue picks it up.
		n.logger.Warn("whatsapp no-show: failed to publish", zap.Uint("message_id", id), zap.Error(err))
	}
}

// queue inserts the queued message, or revives the one withdrawn by a
// correction; ok is false when there is nothing to send.
func (n *NoShowMessages) queue(db *gorm.DB, appt clinical_models.Appointment, to string, sendAfter *time.Time, now time.Time) (uint, bool) {
	apptID, patientID := appt.ID, appt.PatientID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:      appt.TenantID,
		Kind:          whatsapp_models.KindNoShow,
		AppointmentID: &apptID,
		PatientID:     &patientID,
		OffsetHours:   whatsapp_models.NoShowOffsetHours,
		ToPhone:       to,
		Template:      whatsapp_models.TemplateNoShow,
		Status:        whatsapp_models.MessageStatusQueued,
		SendAfter:     sendAfter,
	}
	msg.CreatedAt, msg.UpdatedAt = now, now
	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "appointment_id"}, {Name: "offset_hours"}},
		DoNothing: true,
	}).Create(&msg)
	if res.Error != nil {
		n.logger.Error("whatsapp no-show: failed to queue", zap.Uint("appointment_id", apptID), zap.Error(res.Error))
		return 0, false
	}
	if res.RowsAffected == 1 {
		return msg.ID, true
	}
	revived := db.Model(&whatsapp_models.WhatsAppMessage{}).
		Where("appointment_id = ? AND offset_hours = ? AND kind = ? AND status = ? AND error_code = ?",
			apptID, whatsapp_models.NoShowOffsetHours, whatsapp_models.KindNoShow,
			whatsapp_models.MessageStatusSkipped, whatsapp_models.ErrorCodeNoShowCorrected).
		Updates(map[string]any{
			"status": whatsapp_models.MessageStatusQueued, "error_code": "", "error_detail": "",
			"to_phone": to, "send_after": sendAfter, "updated_at": now,
		})
	if revived.Error != nil {
		n.logger.Error("whatsapp no-show: failed to revive", zap.Uint("appointment_id", apptID), zap.Error(revived.Error))
		return 0, false
	}
	if revived.RowsAffected == 0 {
		return 0, false // already sent, or ended for another reason
	}
	var existing whatsapp_models.WhatsAppMessage
	if err := db.Select("id").Where("appointment_id = ? AND offset_hours = ?", apptID, whatsapp_models.NoShowOffsetHours).
		Limit(1).Find(&existing).Error; err != nil || existing.ID == 0 {
		return 0, false
	}
	return existing.ID, true
}

// Corrected withdraws the queued no-show message of appointmentID, whose
// no_show mark was corrected. One already being sent goes out.
func (n *NoShowMessages) Corrected(tenantID, appointmentID uint) {
	err := tenantdb.ForTenant(n.db, tenantID).Model(&whatsapp_models.WhatsAppMessage{}).
		Where("appointment_id = ? AND kind = ? AND status = ?", appointmentID, whatsapp_models.KindNoShow, whatsapp_models.MessageStatusQueued).
		Updates(map[string]any{
			"status": whatsapp_models.MessageStatusSkipped, "error_code": whatsapp_models.ErrorCodeNoShowCorrected,
			"error_detail": "no-show mark corrected before sending",
		}).Error
	if err != nil {
		n.logger.Error("whatsapp no-show: failed to withdraw", zap.Uint("appointment_id", appointmentID), zap.Error(err))
	}
}

// PublishDue publishes the queued no-show messages whose time came (deferred
// to the send hours) or that were published long ago and are still queued.
func (n *NoShowMessages) PublishDue() {
	now := n.now()
	var rows []whatsapp_models.WhatsAppMessage
	err := tenantdb.System(n.db).Select("id", "tenant_id", "send_after", "updated_at").
		Where("status = ? AND kind = ?", whatsapp_models.MessageStatusQueued, whatsapp_models.KindNoShow).
		Where("send_after IS NULL OR send_after <= ?", now.UTC()).
		Order("id").Limit(100).Find(&rows).Error
	if err != nil {
		n.logger.Error("whatsapp no-show: failed to load due messages", zap.Error(err))
		return
	}
	for _, m := range rows {
		published := m.UpdatedAt.After(now.Add(-noShowRepublishAfter)) && (m.SendAfter == nil || !m.UpdatedAt.Before(*m.SendAfter))
		if published {
			continue
		}
		if err := EnqueueSend(n.publisher, m.ID); err != nil {
			n.logger.Warn("whatsapp no-show: failed to publish", zap.Uint("message_id", m.ID), zap.Error(err))
			return
		}
		tenantdb.ForTenant(n.db, m.TenantID).Model(&whatsapp_models.WhatsAppMessage{}).Where("id = ?", m.ID).Update("updated_at", now)
	}
}

// templateApproved reports whether name is approved on the tenant's WABA (db
// bound to the tenant).
func templateApproved(db *gorm.DB, name string) bool {
	rows, err := TemplateRows(db)
	return err == nil && rows[name].Status == whatsapp_models.TemplateStatusApproved
}

// NoShowMessage builds the no-show template message for message msgID; its
// button carries "reschedule:<id>" so the webhook knows what was answered.
func NoShowMessage(to string, msgID uint, params []string) whatsapp.TemplateMessage {
	return whatsapp.TemplateMessage{
		To:         to,
		Name:       NoShowTemplate.Definition.Name,
		Language:   NoShowTemplate.Definition.Language,
		BodyParams: params,
		Buttons:    []whatsapp.QuickReplyButton{{Payload: fmt.Sprintf("%s:%d", whatsapp_models.ReplyReschedule, msgID)}},
	}
}

// sendNoShow sends a claimed no-show message, after checking again that it
// is still wanted and possible.
func (s *Sender) sendNoShow(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, token string) error {
	skip := func(code, detail string) error {
		return s.finish(db, msg, whatsapp_models.MessageStatusSkipped, code, detail)
	}
	if !account.NoShowMessageEnabled {
		return skip(whatsapp_models.ErrorCodeNoShowDisabled, "no-show message turned off")
	}
	if !templateApproved(db, whatsapp_models.TemplateNoShow) {
		return skip(whatsapp_models.ErrorCodeTemplateNotApproved, "template not approved")
	}
	if msg.AppointmentID == nil {
		return s.finish(db, msg, whatsapp_models.MessageStatusFailed, whatsapp_models.ErrorCodeInternal, "no-show message without appointment")
	}
	var appt clinical_models.Appointment
	if err := db.Preload("Patient").Where("id = ?", *msg.AppointmentID).Limit(1).Find(&appt).Error; err != nil {
		return s.finish(db, msg, whatsapp_models.MessageStatusFailed, whatsapp_models.ErrorCodeInternal, err.Error())
	}
	if appt.ID == 0 || appt.Status != clinical_models.AppointmentStatusNoShow {
		return skip(whatsapp_models.ErrorCodeNoShowCorrected, "appointment no longer no_show")
	}
	if !appt.Patient.WhatsAppOptIn {
		return skip(whatsapp_models.ErrorCodeNoOptIn, "patient has not opted in")
	}
	if strings.TrimSpace(appt.Patient.Phone) == "" {
		return s.finish(db, msg, whatsapp_models.MessageStatusFailed, whatsapp_models.ErrorCodeNoPhone, "patient has no phone")
	}
	to, err := NormalizePhone(appt.Patient.Phone)
	if err != nil {
		return s.finish(db, msg, whatsapp_models.MessageStatusFailed, whatsapp_models.ErrorCodeInvalidPhone, appt.Patient.Phone)
	}
	msg.ToPhone = to
	if ok, _, err := canSendTemplate(s.db, msg.TenantID, s.now(), s.loc); err != nil {
		s.logger.Error("failed to check whatsapp monthly limit", zap.Uint("message_id", msg.ID), zap.Error(err))
	} else if !ok {
		if err := skip(whatsapp_models.ErrorCodeMonthlyLimit, "monthly message limit reached"); err != nil {
			return err
		}
		notifyUsage(s.db, s.logger, msg.TenantID, s.now(), s.loc)
		return nil
	}
	start, err := AppointmentStart(appt, s.loc)
	if err != nil {
		return s.finish(db, msg, whatsapp_models.MessageStatusFailed, whatsapp_models.ErrorCodeInternal, err.Error())
	}
	data := TemplateData{
		PatientName: strings.TrimSpace(appt.Patient.FirstName + " " + appt.Patient.LastName),
		ClinicName:  ClinicName(s.db, msg.TenantID),
		Start:       start,
		DoctorName:  doctorName(db, appt.DoctorID),
	}
	params := TemplateParams(NoShowTemplate, data)
	msg.Body = RenderTemplateBody(NoShowTemplate, params)
	s.thread(db, msg)
	tpl := NoShowMessage(msg.ToPhone, msg.ID, params)
	return s.deliver(db, msg, account, func() (string, error) {
		return s.client.SendTemplate(token, account.PhoneNumberID, tpl)
	})
}

// doctorName is the appointment doctor's name ("" when none or not found,
// deleted ones included).
func doctorName(db *gorm.DB, doctorID *uint) string {
	if doctorID == nil {
		return ""
	}
	var doctor doctor_models.Doctor
	if err := db.Unscoped().Select("id", "full_name").Where("id = ?", *doctorID).Limit(1).Find(&doctor).Error; err != nil {
		return ""
	}
	return doctor.FullName
}
