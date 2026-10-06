package whatsapp_handlers

import (
	"net/http"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Templates sent from the inbox (USE_WHATSAPP_INBOX): "Nuevo mensaje" to a
// patient, or a template in a thread whose 24 h window is closed. They need
// the patient's WhatsApp consent, an approved template and room under the
// plan's monthly cap, and are sent synchronously like SendReply.

// ListTemplates is the catalog with each template's status on the tenant's
// WABA ("" when not created yet, e.g. not connected).
func (h *WhatsAppHandler) ListTemplates(c *gin.Context) envelope.Response {
	rows, err := whatsapp_services.TemplateRows(tenantdb.For(c, h.db))
	if err != nil {
		h.logger.Error("failed to load whatsapp templates", zap.Error(err))
		return h.internalError()
	}
	items := make([]whatsapp_dto.TemplateResponse, 0, len(whatsapp_services.Catalog))
	for _, spec := range whatsapp_services.Catalog {
		def := spec.Definition
		row := rows[def.Name]
		buttons := make([]string, 0, len(def.Buttons))
		for _, b := range def.Buttons {
			buttons = append(buttons, b.Text)
		}
		items = append(items, whatsapp_dto.TemplateResponse{
			Name: def.Name, Status: row.Status, Reason: row.Reason,
			Variables: spec.Variables, NeedsAppointment: spec.NeedsAppointment, Inbox: spec.Inbox,
			Usable: spec.Inbox && row.Status == whatsapp_models.TemplateStatusApproved,
			Body:   def.Body, Preview: whatsapp_services.RenderTemplateBody(spec, def.Example), Buttons: buttons,
		})
	}
	return envelope.SuccessResponse(items, "whatsapp.templates.list.success")
}

// GetUsage is the tenant's charged messages this month against its cap.
func (h *WhatsAppHandler) GetUsage(c *gin.Context) envelope.Response {
	u, err := whatsapp_services.MonthlyUsage(h.db, tenantdb.TenantID(c), time.Now())
	if err != nil {
		h.logger.Error("failed to compute whatsapp usage", zap.Error(err))
		return h.internalError()
	}
	return envelope.SuccessResponse(whatsapp_dto.UsageResponse{
		Used: u.Used, Limit: u.Limit, PeriodStart: u.PeriodStart, PeriodEnd: u.PeriodEnd,
	}, "whatsapp.usage.get.success")
}

// StartConversation sends a catalog template to a patient of the tenant,
// creating the conversation with their number when needed.
func (h *WhatsAppHandler) StartConversation(c *gin.Context) envelope.Response {
	var req whatsapp_dto.StartConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	spec, ok := inboxTemplate(req.Template)
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.template_unknown", core_errors.ErrWhatsAppTemplateUnknown)
	}
	db := tenantdb.For(c, h.db)
	var patient clinical_models.Patient
	if err := db.Where("id = ?", req.PatientID).Limit(1).Find(&patient).Error; err != nil {
		h.logger.Error("failed to load patient", zap.Error(err))
		return h.internalError()
	}
	if patient.ID == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.patient_not_found", core_errors.ErrWhatsAppPatientNotFound)
	}
	phone, err := whatsapp_services.NormalizePhone(patient.Phone)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_phone", core_errors.ErrWhatsAppInvalidPhone)
	}
	send, resp, ok := h.prepareTemplate(c, spec, &patient, req.AppointmentID)
	if !ok {
		return resp
	}
	conv, err := whatsapp_services.EnsureConversation(db, tenantdb.TenantID(c), phone, &patient.ID)
	if err != nil {
		h.logger.Error("failed to ensure whatsapp conversation", zap.Error(err))
		return h.internalError()
	}
	return h.sendTemplate(c, conv, send, "whatsapp.conversation.start.success")
}

// SendTemplate sends a catalog template in an existing conversation, also
// when the reply window is closed. The conversation must be linked to a
// patient who consented.
func (h *WhatsAppHandler) SendTemplate(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	var req whatsapp_dto.SendTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	spec, ok := inboxTemplate(req.Template)
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.template_unknown", core_errors.ErrWhatsAppTemplateUnknown)
	}
	send, resp, ok := h.prepareTemplate(c, spec, conv.Patient, req.AppointmentID)
	if !ok {
		return resp
	}
	return h.sendTemplate(c, conv, send, "whatsapp.conversation.template.success")
}

// templateSend is a checked inbox template, ready to send.
type templateSend struct {
	spec      whatsapp_services.TemplateSpec
	data      whatsapp_services.TemplateData
	patientID uint
	account   whatsapp_models.WhatsAppAccount
	token     string
}

func inboxTemplate(name string) (whatsapp_services.TemplateSpec, bool) {
	spec, ok := whatsapp_services.FindTemplate(strings.TrimSpace(name))
	return spec, ok && spec.Inbox
}

// prepareTemplate runs the checks shared by both endpoints, in order:
// consent (409), connection, template approved (409), appointment (400) and
// the monthly cap (403).
func (h *WhatsAppHandler) prepareTemplate(c *gin.Context, spec whatsapp_services.TemplateSpec, patient *clinical_models.Patient, appointmentID *uint) (templateSend, envelope.Response, bool) {
	var send templateSend
	if patient == nil || patient.ID == 0 || !patient.WhatsAppOptIn {
		return send, envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.no_opt_in", core_errors.ErrWhatsAppNoOptIn), false
	}
	account, token, resp, ok := h.connected(c)
	if !ok {
		return send, resp, false
	}
	db := tenantdb.For(c, h.db)
	rows, err := whatsapp_services.TemplateRows(db)
	if err != nil {
		h.logger.Error("failed to load whatsapp templates", zap.Error(err))
		return send, h.internalError(), false
	}
	if rows[spec.Definition.Name].Status != whatsapp_models.TemplateStatusApproved {
		return send, envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.template_unavailable", core_errors.ErrWhatsAppTemplateUnavailable), false
	}
	data := whatsapp_services.TemplateData{
		PatientName: strings.TrimSpace(patient.FirstName + " " + patient.LastName),
		ClinicName:  whatsapp_services.ClinicName(h.db, tenantdb.TenantID(c)),
	}
	if spec.NeedsAppointment {
		if appointmentID == nil || *appointmentID == 0 {
			return send, envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.appointment_invalid", core_errors.ErrWhatsAppAppointmentInvalid), false
		}
		var appt clinical_models.Appointment
		if err := db.Where("id = ? AND patient_id = ?", *appointmentID, patient.ID).Limit(1).Find(&appt).Error; err != nil {
			h.logger.Error("failed to load appointment", zap.Error(err))
			return send, h.internalError(), false
		}
		start, err := whatsapp_services.AppointmentStart(appt, whatsapp_services.ClinicLocation())
		if appt.ID == 0 || err != nil {
			return send, envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.appointment_invalid", core_errors.ErrWhatsAppAppointmentInvalid), false
		}
		data.Start = start
	}
	allowed, _, err := whatsapp_services.CanSendTemplate(h.db, tenantdb.TenantID(c), time.Now())
	if err != nil {
		h.logger.Error("failed to check whatsapp monthly limit", zap.Error(err))
		return send, h.internalError(), false
	}
	if !allowed {
		return send, monthlyLimitResponse(), false
	}
	return templateSend{spec: spec, data: data, patientID: patient.ID, account: account, token: token}, envelope.Response{}, true
}

// sendTemplate records the template in conv's thread (kind template, the
// rendered body, the sender) and sends it. Meta rejecting it leaves the
// message failed in the thread and answers 502.
func (h *WhatsAppHandler) sendTemplate(c *gin.Context, conv whatsapp_models.WhatsAppConversation, send templateSend, key string) envelope.Response {
	db := tenantdb.For(c, h.db)
	params := whatsapp_services.TemplateParams(send.spec, send.data)
	patientID := send.patientID
	// No AppointmentID: the (appointment_id, offset_hours) unique index is
	// the reminders' idempotency key; the body already carries date and time.
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    tenantdb.TenantID(c),
		Kind:        whatsapp_models.KindTemplate,
		PatientID:   &patientID,
		ToPhone:     conv.Phone,
		Template:    send.spec.Definition.Name,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        whatsapp_services.RenderTemplateBody(send.spec, params),
		ContentType: whatsapp_models.ContentText,
	}
	if uid := c.GetInt64("user_id"); uid > 0 {
		id := uint(uid)
		msg.SentByUserID = &id
	}
	if err := whatsapp_services.RecordOutbound(db, &conv, &msg, time.Now()); err != nil {
		h.logger.Error("failed to record whatsapp template", zap.Error(err))
		return h.internalError()
	}
	tpl := whatsapp_services.TemplateMessageFor(send.spec, conv.Phone, send.data)
	if err := h.sender.DeliverTemplate(db, &msg, send.account, send.token, tpl); err != nil {
		return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.send_failed", core_errors.ErrWhatsAppSendFailed)
	}
	if err := db.Preload("Patient").Where("id = ?", conv.ID).Limit(1).Find(&conv).Error; err != nil {
		h.logger.Error("failed to reload whatsapp conversation", zap.Error(err))
	}
	return envelope.SuccessResponse(whatsapp_dto.TemplateSendResponse{
		Conversation: whatsapp_dto.NewConversationResponse(conv, time.Now()),
		Message:      whatsapp_dto.NewMessageResponse(msg),
	}, key)
}

func monthlyLimitResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusForbidden, "plan.limit.whatsapp_messages", core_errors.ErrWhatsAppMonthlyLimit)
}
