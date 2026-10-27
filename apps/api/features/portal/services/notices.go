package portal_services

import (
	"strings"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_models "pengi-med-saas/features/portal/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
)

// Channels of a request notice (RequestNotice's return value).
const (
	NoticeNone     = ""
	NoticeWhatsApp = "whatsapp"
	NoticeEmail    = "email"
)

// noticeData is what a request notice says.
type noticeData struct {
	confirmed bool
	patient   clinical_models.Patient
	clinic    tenant_models.Tenant
	doctor    string
	date      string // "lunes 6 de octubre"
	time      string // "HH:MM"
	reason    string // rejection reason, if any
}

// SendRequestNotice tells the patient how their portal request ended:
// confirmed (approved) or not confirmed (rejected or expired). By WhatsApp
// when the clinic has a connected account with the template approved, the
// patient has a valid phone and WhatsApp consent and the monthly cap has
// room; otherwise (or when the send fails) by email when the patient has
// one; otherwise nothing. Returns the channel used. Never fails the caller:
// problems are logged.
func (d *Delivery) SendRequestNotice(tenantID, requestID uint) string {
	data, ok := d.noticeData(tenantID, requestID)
	if !ok {
		return NoticeNone
	}
	if d.sendNoticeWhatsApp(tenantID, requestID, data) {
		return NoticeWhatsApp
	}
	email := strings.TrimSpace(data.patient.Email)
	if email == "" || d.mailer == nil {
		return NoticeNone
	}
	if err := d.mailer.SendPortalNotice(email, noticeEmail(data)); err != nil {
		d.logger.Warn("portal booking: notice email not sent", zap.Uint("tenant_id", tenantID), zap.Uint("request_id", requestID), zap.Error(err))
		return NoticeNone
	}
	return NoticeEmail
}

func (d *Delivery) noticeData(tenantID, requestID uint) (noticeData, bool) {
	db := tenantdb.ForTenant(d.db, tenantID)
	var req portal_models.AppointmentRequest
	if err := db.Where("id = ?", requestID).Limit(1).Find(&req).Error; err != nil || req.ID == 0 {
		if err != nil {
			d.logger.Error("portal booking: failed to load request for notice", zap.Uint("request_id", requestID), zap.Error(err))
		}
		return noticeData{}, false
	}
	var data noticeData
	switch req.Status {
	case portal_models.RequestApproved:
		data.confirmed = true
	case portal_models.RequestRejected:
		data.reason = strings.TrimSpace(req.RejectReason)
	case portal_models.RequestExpired:
	default:
		return noticeData{}, false
	}
	patient, found, err := PatientByID(d.db, tenantID, req.PatientID)
	if err != nil || !found {
		return noticeData{}, false
	}
	if err := tenantdb.System(d.db).Where("id = ?", tenantID).Limit(1).Find(&data.clinic).Error; err != nil || data.clinic.ID == 0 {
		return noticeData{}, false
	}
	var doctor doctor_models.Doctor
	_ = db.Unscoped().Select("id", "full_name").Where("id = ?", req.DoctorID).Limit(1).Find(&doctor).Error
	start, ok := SlotStart(req.Date, req.StartTime)
	if !ok {
		return noticeData{}, false
	}
	data.patient, data.doctor = patient, doctor.FullName
	data.date, data.time = whatsapp_services.SpanishDate(start), req.StartTime
	return data, true
}

// sendNoticeWhatsApp records and sends the notice's template; false when it
// was not sent (not possible or failed).
func (d *Delivery) sendNoticeWhatsApp(tenantID, requestID uint, data noticeData) bool {
	spec, kind := whatsapp_services.BookingNotConfirmedTemplate, whatsapp_models.KindBookingNotConfirmed
	if data.confirmed {
		spec, kind = whatsapp_services.BookingConfirmedTemplate, whatsapp_models.KindBookingConfirmed
	}
	if !data.patient.WhatsAppOptIn {
		return false
	}
	account, token, ok := d.accountWithTemplate(tenantID, spec.Definition.Name)
	if !ok {
		return false
	}
	to, err := whatsapp_services.NormalizePhone(data.patient.Phone)
	if err != nil {
		return false
	}
	now := d.now()
	allowed, _, err := whatsapp_services.CanSendTemplate(d.db, tenantID, now)
	if err != nil {
		d.logger.Error("portal booking: failed to check whatsapp monthly limit", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return false
	}
	if !allowed {
		return false
	}
	params := whatsapp_services.TemplateParams(spec, whatsapp_services.TemplateData{
		PatientName: strings.TrimSpace(data.patient.FirstName),
		ClinicName:  data.clinic.DisplayName(),
		DoctorName:  data.doctor,
	})
	// Date and time are already formatted for the patient (TemplateParams
	// would format a time.Time): put them in their placeholders.
	for i, v := range spec.Variables {
		switch v {
		case whatsapp_services.VarDate:
			params[i] = data.date
		case whatsapp_services.VarTime:
			params[i] = data.time
		}
	}
	scoped := tenantdb.ForTenant(d.db, tenantID)
	patientID := data.patient.ID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    tenantID,
		Kind:        kind,
		PatientID:   &patientID,
		ToPhone:     to,
		Template:    spec.Definition.Name,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        whatsapp_services.RenderTemplateBody(spec, params),
		ContentType: whatsapp_models.ContentText,
	}
	// In the patient's thread, so staff see it in the inbox.
	conv, err := whatsapp_services.EnsureConversation(scoped, tenantID, to, &patientID)
	if err == nil {
		err = whatsapp_services.RecordOutbound(scoped, &conv, &msg, now)
	}
	if err != nil {
		d.logger.Error("portal booking: failed to record whatsapp notice", zap.Uint("request_id", requestID), zap.Error(err))
		return false
	}
	tpl := whatsapp_services.TemplateMessageFor(spec, to, whatsapp_services.TemplateData{})
	tpl.BodyParams = params
	sender := whatsapp_services.NewSender(d.db, d.logger, d.client)
	if err := sender.DeliverTemplate(scoped, &msg, account, token, tpl); err != nil {
		d.logger.Warn("portal booking: whatsapp notice not sent, falling back to email", zap.Uint("request_id", requestID), zap.Error(err))
		return false
	}
	return true
}

// noticeEmail is the notice's email, in the patients' language.
func noticeEmail(data noticeData) mailer.PortalNoticeEmail {
	clinic := data.clinic.DisplayName()
	fill := func(key string) string {
		s := whatsapp_services.PatientText(key)
		return strings.NewReplacer(
			"{patient}", strings.TrimSpace(data.patient.FirstName),
			"{clinic}", clinic,
			"{doctor}", data.doctor,
			"{date}", data.date,
			"{time}", data.time,
			"{reason}", data.reason,
			"{url}", portalLink(data.clinic.Slug),
		).Replace(s)
	}
	footer := fill("portal.email.booking.footer")
	if data.confirmed {
		return mailer.PortalNoticeEmail{
			Subject: fill("portal.email.booking.confirmed.subject"),
			Heading: fill("portal.email.booking.confirmed.heading"),
			Lines:   []string{fill("portal.email.booking.confirmed.body")},
			Footer:  footer,
		}
	}
	lines := []string{fill("portal.email.booking.not_confirmed.body")}
	if data.reason != "" {
		lines = append(lines, fill("portal.email.booking.not_confirmed.reason"))
	}
	if portalLink(data.clinic.Slug) != "" {
		lines = append(lines, fill("portal.email.booking.not_confirmed.action"))
	} else {
		lines = append(lines, fill("portal.email.booking.not_confirmed.action_no_link"))
	}
	return mailer.PortalNoticeEmail{
		Subject: fill("portal.email.booking.not_confirmed.subject"),
		Heading: fill("portal.email.booking.not_confirmed.heading"),
		Lines:   lines,
		Footer:  footer,
	}
}

// portalLink is the clinic's portal link, or "" when PORTAL_URL is not set.
func portalLink(slug string) string {
	base := strings.TrimRight(config.Get().PortalURL, "/")
	if base == "" || slug == "" {
		return ""
	}
	return base + "/" + slug
}
