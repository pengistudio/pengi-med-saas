package whatsapp_handlers

import (
	"strings"

	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_services "pengi-med-saas/features/portal/services"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The "Reagendar" button of the no-show message ("reschedule:<message id>").
// The tap itself is already in the inbox (recordInbound, NotifyInbound). When
// the clinic's patients can book online, Pengi answers inside the window the
// tap just opened with the portal link (KindSystem: free, not counted);
// otherwise, or when that text could not be sent, the inbox users are told
// that the patient wants a new date. The appointment is not touched.

// isReschedule reports whether payload is a "Reagendar" tap.
func isReschedule(payload string) bool {
	action, _, ok := strings.Cut(payload, ":")
	return ok && action == whatsapp_models.ReplyReschedule
}

// applyReschedule handles a "Reagendar" tap in conv (nil when the inbox could
// not record it).
func (h *WebhookHandler) applyReschedule(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation, m whatsapp.WebhookMessage) {
	tenantID := account.TenantID
	_, rawID, _ := strings.Cut(m.ButtonPayload(), ":")
	msg, ok := h.answer(db, tenantID, m, rawID, whatsapp_models.ReplyReschedule, whatsapp_models.KindNoShow)
	if !ok {
		return
	}
	if conv == nil {
		c, err := whatsapp_services.EnsureConversation(db, tenantID, msg.ToPhone, msg.PatientID)
		if err != nil {
			h.logger.Error("whatsapp webhook: failed to ensure conversation for reschedule", zap.Uint("tenant_id", tenantID), zap.Error(err))
			return
		}
		conv = &c
	}

	link, err := portal_services.BookingLink(h.db, tenantID)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to resolve booking link", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
	if link != "" && h.sender != nil {
		body := strings.ReplaceAll(whatsapp_services.PatientText(whatsapp_services.NoShowBookingLinkKey), "{url}", link)
		if h.sendSystem(db, account, conv, body) {
			return
		}
	}
	date, clock := h.appointmentWhen(db, msg.AppointmentID)
	whatsapp_services.NotifyRescheduleRequest(h.db, h.logger, tenantID, *conv, date, clock)
}

// appointmentWhen is the date (YYYY-MM-DD, clinic time) and start time of
// the appointment, "" when it is gone.
func (h *WebhookHandler) appointmentWhen(db *gorm.DB, apptID *uint) (date, clock string) {
	if apptID == nil {
		return "", ""
	}
	var appt clinical_models.Appointment
	if err := db.Unscoped().Select("id", "date", "start_time").Where("id = ?", *apptID).Limit(1).Find(&appt).Error; err != nil || appt.ID == 0 {
		return "", ""
	}
	loc := whatsapp_services.ClinicLocation()
	if start, err := whatsapp_services.AppointmentStart(appt, loc); err == nil {
		return start.Format("2006-01-02"), appt.StartTime
	}
	return appt.Date.In(loc).Format("2006-01-02"), appt.StartTime
}
