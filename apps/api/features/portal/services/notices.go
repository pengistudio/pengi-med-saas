package portal_services

import (
	"encoding/json"
	"strings"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantdb"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_dto "pengi-med-saas/features/portal/dto"
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

// noticeData is what a request notice says, and to whom.
type noticeData struct {
	confirmed bool
	name      string // first name of the person told
	// patientID is the patient told (its WhatsApp thread); nil for a
	// new-patient request not approved.
	patientID *uint
	phone     string // WhatsApp target ("" none)
	optIn     bool   // WhatsApp consent of the phone's owner
	email     string // email target ("" none)
	// newPatient: a first-appointment request, told at its verified contact;
	// the "not confirmed" action line points to the first-appointment page.
	newPatient bool
	clinic     tenant_models.Tenant
	doctor     string
	date       string // "lunes 6 de octubre"
	time       string // "HH:MM"
	reason     string // rejection reason, if any
}

// SendRequestNotice tells the patient how their portal request ended:
// confirmed (approved) or not confirmed (rejected or expired). By WhatsApp
// when the clinic has a connected account with the template approved, the
// patient has a valid phone and WhatsApp consent and the monthly cap has
// room; otherwise (or when the send fails) by email when the patient has
// one; otherwise nothing. A new-patient request (approved or not) goes to
// the contact the person verified (WhatsApp when that was
// the channel, the verification standing for consent; else, or as the
// fallback, the verified email or the form's email). Returns the channel
// used. Never fails the caller: problems are logged.
func (d *Delivery) SendRequestNotice(tenantID, requestID uint) string {
	data, ok := d.noticeData(tenantID, requestID)
	if !ok {
		return NoticeNone
	}
	if d.sendNoticeWhatsApp(tenantID, requestID, data) {
		return NoticeWhatsApp
	}
	email := strings.TrimSpace(data.email)
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
	if req.NewPatient {
		// Approved or not, a new patient is told at the contact they verified.
		if !data.toContact(req) {
			return noticeData{}, false
		}
	} else {
		patient, found, err := PatientByID(d.db, tenantID, req.PatientID)
		if err != nil || !found {
			return noticeData{}, false
		}
		id := patient.ID
		data.name, data.patientID = strings.TrimSpace(patient.FirstName), &id
		data.phone, data.optIn, data.email = patient.Phone, patient.WhatsAppOptIn, patient.Email
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
	data.doctor = doctor.FullName
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
	if !data.optIn || data.phone == "" {
		return false
	}
	account, token, ok := d.accountWithTemplate(tenantID, spec.Definition.Name)
	if !ok {
		return false
	}
	to, err := whatsapp_services.NormalizePhone(data.phone)
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
		PatientName: data.name,
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
	patientID := data.patientID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    tenantID,
		Kind:        kind,
		PatientID:   patientID,
		ToPhone:     to,
		Template:    spec.Definition.Name,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        whatsapp_services.RenderTemplateBody(spec, params),
		ContentType: whatsapp_models.ContentText,
	}
	// In the patient's thread, so staff see it in the inbox.
	conv, err := whatsapp_services.EnsureConversation(scoped, tenantID, to, patientID)
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
	link := portalLink(data.clinic.Slug)
	if data.newPatient && link != "" {
		link += FirstAppointmentPath
	}
	fill := func(key string) string {
		s := whatsapp_services.PatientText(key)
		return strings.NewReplacer(
			"{patient}", data.name,
			"{clinic}", clinic,
			"{doctor}", data.doctor,
			"{date}", data.date,
			"{time}", data.time,
			"{reason}", data.reason,
			"{url}", link,
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
	if link != "" {
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

// FirstAppointmentPath is the portal page (under the clinic's link) where a
// person without a patient record requests a first appointment.
const FirstAppointmentPath = "/primera-cita"

// toContact aims the notice of a new-patient request at the contact the
// person verified (approved ones too: the record may lack that contact or
// WhatsApp consent); false when there is nowhere to send it.
func (n *noticeData) toContact(req portal_models.AppointmentRequest) bool {
	var form portal_dto.NewPatientForm
	if err := json.Unmarshal([]byte(req.NewPatientData), &form); err != nil {
		return false
	}
	n.name, n.newPatient = strings.TrimSpace(form.FirstName), true
	if req.PatientID != 0 {
		id := req.PatientID // approved: the message joins the patient's thread
		n.patientID = &id
	}
	switch req.ContactChannel {
	case portal_models.ChannelWhatsApp:
		n.phone, n.optIn, n.email = req.ContactValue, true, form.Email
	case portal_models.ChannelEmail:
		n.email = req.ContactValue
	default:
		return false
	}
	return n.phone != "" || n.email != ""
}
