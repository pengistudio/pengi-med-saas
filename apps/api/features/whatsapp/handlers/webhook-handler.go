package whatsapp_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// maxWebhookBody caps what the public webhook reads.
const maxWebhookBody = 1 << 20

// Notification types created when a patient answers a reminder.
const (
	NotificationTypeConfirmed = "whatsapp.appointment.confirmed"
	NotificationTypeCancelled = "whatsapp.appointment.cancelled"
)

// CalendarSync mirrors an appointment status change to Google Calendar
// (clinical_handlers.AppointmentHandler implements it).
type CalendarSync interface {
	SyncStatusChange(tenantID uint, appointment *clinical_models.Appointment)
}

// WebhookHandler receives Meta's WhatsApp webhooks. It is public: requests
// are authenticated by the X-Hub-Signature-256 HMAC (META_APP_SECRET), and the
// tenant comes from the receiving phone number id, never from the payload's
// message ids alone.
type WebhookHandler struct {
	db          *gorm.DB
	logger      *zap.Logger
	calendar    CalendarSync
	appSecret   string
	verifyToken string
	now         func() time.Time
	// sender sends the system texts of the consent flow; nil disables it.
	sender *whatsapp_services.Sender
}

func NewWebhookHandler(db *gorm.DB, logger *zap.Logger, calendar CalendarSync) *WebhookHandler {
	return &WebhookHandler{
		db: db, logger: logger, calendar: calendar,
		appSecret:   os.Getenv("META_APP_SECRET"),
		verifyToken: os.Getenv("WHATSAPP_WEBHOOK_VERIFY_TOKEN"),
		now:         time.Now,
	}
}

// WithClient enables the free texts the webhook sends on its own (asking for
// WhatsApp consent and confirming it) through client.
func (h *WebhookHandler) WithClient(client *whatsapp.Client) *WebhookHandler {
	if client != nil {
		h.sender = whatsapp_services.NewSender(h.db, h.logger, client)
	}
	return h
}

// WithSecrets overrides the app secret and verify token (tests).
func (h *WebhookHandler) WithSecrets(appSecret, verifyToken string) *WebhookHandler {
	h.appSecret, h.verifyToken = appSecret, verifyToken
	return h
}

// Verify answers Meta's subscription handshake. It must echo hub.challenge as
// plain text, so it writes with c.String instead of the envelope (documented
// exception); refusals still go through envelope.Write.
func (h *WebhookHandler) Verify(c *gin.Context) {
	if h.verifyToken != "" && c.Query("hub.mode") == "subscribe" && c.Query("hub.verify_token") == h.verifyToken {
		c.String(http.StatusOK, c.Query("hub.challenge"))
		return
	}
	envelope.Write(c, envelope.ErrorResponse(http.StatusForbidden, "whatsapp.error.webhook_verify", core_errors.ErrWhatsAppWebhookVerify))
}

// Receive handles an event notification. Once the signature checks out it
// always answers 200, even when an event can't be applied, so Meta doesn't
// retry logical errors; those are logged.
func (h *WebhookHandler) Receive(c *gin.Context) envelope.Response {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBody))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	if !whatsapp.VerifySignature(h.appSecret, body, c.GetHeader(whatsapp.SignatureHeader)) {
		h.logger.Warn("whatsapp webhook: invalid signature", zap.Bool("secret_configured", h.appSecret != ""))
		return envelope.ErrorResponse(http.StatusUnauthorized, "whatsapp.error.webhook_signature", core_errors.ErrWhatsAppWebhookSignature)
	}
	var payload whatsapp.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Warn("whatsapp webhook: undecodable payload", zap.Error(err))
		return envelope.SuccessResponse(nil, "whatsapp.webhook.received")
	}
	h.Process(payload)
	return envelope.SuccessResponse(nil, "whatsapp.webhook.received")
}

// Process applies every change of a verified payload.
func (h *WebhookHandler) Process(payload whatsapp.WebhookPayload) {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			switch change.Field {
			case "messages":
				h.processMessages(change.Value)
			case "message_template_status_update":
				h.processTemplateStatus(entry.ID, change.Value)
			}
		}
	}
}

func (h *WebhookHandler) processMessages(v whatsapp.WebhookValue) {
	phoneID := v.Metadata.PhoneNumberID
	if phoneID == "" {
		return
	}
	var account whatsapp_models.WhatsAppAccount
	if err := tenantdb.System(h.db).Where("phone_number_id = ?", phoneID).Limit(1).Find(&account).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to find account", zap.Error(err))
		return
	}
	if account.ID == 0 {
		h.logger.Info("whatsapp webhook: event for unknown number", zap.String("phone_number_id", phoneID))
		return
	}
	db := tenantdb.ForTenant(h.db, account.TenantID)
	for _, st := range v.Statuses {
		h.applyStatus(db, st)
	}
	for _, m := range v.Messages {
		conv, msg, recorded := h.recordInbound(db, account.TenantID, m)
		if !recorded {
			continue // already processed (Meta retried the webhook)
		}
		if conv != nil {
			whatsapp_services.NotifyInbound(h.db, h.logger, account.TenantID, *conv, msg)
		}
		switch {
		case m.ButtonPayload() != "":
			h.applyReply(db, account.TenantID, m)
		case m.OptOutRequested():
			h.applyOptOut(db, account.TenantID, m.From)
		case m.Type == "text" && m.Text != nil && conv != nil:
			h.handleConsent(db, account, conv, m.Text.Body)
		}
	}
}

// recordInbound stores m in the conversation with its sender, creating it,
// and returns both. recorded is false only for a message already stored, so
// the caller skips it; on a storage error it is still true (with a nil
// conversation), so a button tap or STOP is not lost because the inbox
// couldn't record it.
func (h *WebhookHandler) recordInbound(db *gorm.DB, tenantID uint, m whatsapp.WebhookMessage) (conv *whatsapp_models.WhatsAppConversation, msg whatsapp_models.WhatsAppMessage, recorded bool) {
	phone := inboundPhone(m.From)
	if phone == "" {
		return nil, msg, true
	}
	at := h.now()
	if ts, err := strconv.ParseInt(m.Timestamp, 10, 64); err == nil {
		at = time.Unix(ts, 0)
	}
	c, err := whatsapp_services.EnsureConversation(db, tenantID, phone, nil)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to ensure conversation", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return nil, msg, true
	}
	contentType, body, reply := inboundContent(m)
	msg = whatsapp_models.WhatsAppMessage{
		TenantID: tenantID, WamID: m.ID, ContentType: contentType, Body: body, Reply: reply,
	}
	recorded, err = whatsapp_services.RecordInbound(db, &c, &msg, at)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to record inbound message", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return nil, msg, true
	}
	return &c, msg, recorded
}

// inboundPhone normalizes the sender ("from", digits without "+").
func inboundPhone(from string) string {
	if from == "" {
		return ""
	}
	if n, err := whatsapp_services.NormalizePhone("+" + from); err == nil {
		return n
	}
	var b strings.Builder
	for _, r := range from {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// inboundContent maps an incoming message to its content type, the text the
// thread shows and, for a reminder button, the action (confirm / cancel).
func inboundContent(m whatsapp.WebhookMessage) (contentType, body, reply string) {
	if payload := m.ButtonPayload(); payload != "" {
		action, _, _ := strings.Cut(payload, ":")
		if action != whatsapp_models.ReplyConfirm && action != whatsapp_models.ReplyCancel {
			action = ""
		}
		return whatsapp_models.ContentButton, m.ButtonText(), action
	}
	switch m.Type {
	case "text":
		if m.Text != nil {
			return whatsapp_models.ContentText, m.Text.Body, ""
		}
	case "image":
		return whatsapp_models.ContentImage, m.Caption(), ""
	case "audio", "voice":
		return whatsapp_models.ContentAudio, "", ""
	case "document":
		return whatsapp_models.ContentDocument, m.Caption(), ""
	}
	return whatsapp_models.ContentOther, m.Caption(), ""
}

// applyStatus moves a sent message forward (sent → delivered → read, or failed).
func (h *WebhookHandler) applyStatus(db *gorm.DB, st whatsapp.WebhookStatus) {
	if st.ID == "" {
		return
	}
	var msg whatsapp_models.WhatsAppMessage
	if err := db.Where("wam_id = ? AND direction = ?", st.ID, whatsapp_models.DirectionOutbound).Limit(1).Find(&msg).Error; err != nil || msg.ID == 0 {
		return
	}
	if !whatsapp_models.Advances(msg.Status, st.Status) {
		return
	}
	at := h.now()
	if ts, err := strconv.ParseInt(st.Timestamp, 10, 64); err == nil {
		at = time.Unix(ts, 0)
	}
	updates := map[string]any{"status": st.Status}
	switch st.Status {
	case whatsapp_models.MessageStatusDelivered:
		updates["delivered_at"] = at
	case whatsapp_models.MessageStatusRead:
		updates["read_at"] = at
		if msg.DeliveredAt == nil {
			updates["delivered_at"] = at
		}
	case whatsapp_models.MessageStatusFailed:
		if len(st.Errors) > 0 {
			e := st.Errors[0]
			updates["error_code"] = strconv.Itoa(e.Code)
			detail := strings.TrimSpace(e.Title + ": " + e.ErrorData.Details)
			updates["error_detail"] = strings.TrimSuffix(detail, ":")
		}
	}
	if err := db.Model(&msg).Updates(updates).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to update status", zap.Uint("message_id", msg.ID), zap.Error(err))
	}
}

// applyReply handles a Confirmar/Cancelar tap ("confirm:<id>" / "cancel:<id>").
// The message is looked up in the receiving number's tenant only, so a
// payload can never touch another clinic's appointment, and the reply must
// come from the number the message was sent to (and, when Meta says which
// message was answered, quote that message).
func (h *WebhookHandler) applyReply(db *gorm.DB, tenantID uint, m whatsapp.WebhookMessage) {
	action, rawID, ok := strings.Cut(m.ButtonPayload(), ":")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if !ok || err != nil || (action != whatsapp_models.ReplyConfirm && action != whatsapp_models.ReplyCancel) {
		return
	}
	var msg whatsapp_models.WhatsAppMessage
	if err := db.Where("id = ?", id).Limit(1).Find(&msg).Error; err != nil || msg.ID == 0 {
		h.logger.Info("whatsapp webhook: reply to unknown message", zap.Uint("tenant_id", tenantID), zap.Uint64("message_id", id))
		return
	}
	if msg.Status == whatsapp_models.MessageStatusReplied {
		return // Meta may deliver the same event twice
	}
	if from, err := whatsapp_services.NormalizePhone("+" + m.From); err != nil || from != msg.ToPhone {
		h.logger.Warn("whatsapp webhook: reply from a number the message was not sent to",
			zap.Uint("tenant_id", tenantID), zap.Uint("message_id", msg.ID))
		return
	}
	if m.Context != nil && m.Context.ID != "" && m.Context.ID != msg.WamID {
		h.logger.Warn("whatsapp webhook: reply quotes another message",
			zap.Uint("tenant_id", tenantID), zap.Uint("message_id", msg.ID))
		return
	}
	now := h.now()
	if err := db.Model(&msg).Updates(map[string]any{"status": whatsapp_models.MessageStatusReplied, "reply": action, "replied_at": now}).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to mark reply", zap.Uint("message_id", msg.ID), zap.Error(err))
		return
	}
	if msg.AppointmentID == nil {
		return // test message
	}
	h.applyToAppointment(db, tenantID, *msg.AppointmentID, action)
}

// applyOptOut withdraws WhatsApp consent from every patient of the tenant
// whose phone normalizes to the sender's number. Phones are free text, so the
// match is done in Go over the opted-in patients.
func (h *WebhookHandler) applyOptOut(db *gorm.DB, tenantID uint, rawFrom string) {
	from, err := whatsapp_services.NormalizePhone("+" + rawFrom)
	if err != nil {
		return
	}
	var patients []clinical_models.Patient
	if err := db.Select("id", "phone").Where("whatsapp_opt_in = ? AND phone <> ''", true).Find(&patients).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to load patients for opt-out", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	var ids []uint
	for _, p := range patients {
		if to, err := whatsapp_services.NormalizePhone(p.Phone); err == nil && to == from {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	if err := db.Model(&clinical_models.Patient{}).Where("id IN ?", ids).
		Updates(map[string]any{"whatsapp_opt_in": false, "whatsapp_opt_in_at": h.now(), "whatsapp_opt_in_source": clinical_models.WhatsAppOptInSourceStop}).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to apply opt-out", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	h.logger.Info("whatsapp webhook: patients opted out", zap.Uint("tenant_id", tenantID), zap.Int("count", len(ids)))
}

// applyToAppointment confirms a scheduled appointment, or cancels one that is
// still scheduled or confirmed; any later status (arrived, completed,
// cancelled) is left alone.
func (h *WebhookHandler) applyToAppointment(db *gorm.DB, tenantID uint, apptID uint, action string) {
	from := []string{whatsapp_services.AppointmentScheduled}
	to := whatsapp_services.AppointmentConfirmed
	if action == whatsapp_models.ReplyCancel {
		from = append(from, whatsapp_services.AppointmentConfirmed)
		to = whatsapp_services.AppointmentCancelled
	}
	res := db.Model(&clinical_models.Appointment{}).Where("id = ? AND status IN ?", apptID, from).Update("status", to)
	if res.Error != nil {
		h.logger.Error("whatsapp webhook: failed to update appointment", zap.Uint("appointment_id", apptID), zap.Error(res.Error))
		return
	}
	if res.RowsAffected == 0 {
		return
	}
	var appt clinical_models.Appointment
	if err := db.Preload("Patient").Where("id = ?", apptID).Limit(1).Find(&appt).Error; err != nil || appt.ID == 0 {
		return
	}
	if h.calendar != nil {
		go h.calendar.SyncStatusChange(tenantID, &appt)
	}
	h.notify(tenantID, appt, action)
}

// notify tells every member of the clinic that the patient answered.
func (h *WebhookHandler) notify(tenantID uint, appt clinical_models.Appointment, action string) {
	var userIDs []uint
	err := tenantdb.System(h.db).Table("environments AS e").
		Distinct("e.user_id").
		Joins("JOIN companies AS c ON c.id = e.company_id AND c.deleted_at IS NULL").
		Where("c.tenant_id = ? AND e.deleted_at IS NULL", tenantID).
		Pluck("e.user_id", &userIDs).Error
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to resolve recipients", zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
	}
	notifType, key := NotificationTypeConfirmed, "notification.whatsapp.appointment.confirmed"
	if action == whatsapp_models.ReplyCancel {
		notifType, key = NotificationTypeCancelled, "notification.whatsapp.appointment.cancelled"
	}
	loc := whatsapp_services.ClinicLocation()
	date := appt.Date.In(loc).Format("2006-01-02")
	if start, err := whatsapp_services.AppointmentStart(appt, loc); err == nil {
		date = start.Format("2006-01-02")
	}
	params, _ := json.Marshal(map[string]string{
		"patient_name": strings.TrimSpace(appt.Patient.FirstName + " " + appt.Patient.LastName),
		"date":         date,
		"time":         appt.StartTime,
	})
	level := notifications_models.NotificationLevelInfo
	rows := make([]notifications_models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		rows = append(rows, notifications_models.Notification{
			TenantID: tenantID, UserID: uid, Type: notifType, ResourceType: "appointment", ResourceID: appt.ID,
			MessageKey: key, Params: params, ActionURL: "/clinical/appointments", Level: level,
		})
	}
	// One-off events (the appointment changes status once), so no dedupe:
	// notifications_service.CreateIfNotExists dedupes per resource across
	// users and would notify only the first member.
	if err := tenantdb.ForTenant(h.db, tenantID).Create(&rows).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to create notifications", zap.Error(err))
	}
}

// processTemplateStatus stores a review decision on a catalog template, for
// every tenant connected to the WABA; the reminder's is also mirrored on the
// account.
func (h *WebhookHandler) processTemplateStatus(wabaID string, v whatsapp.WebhookValue) {
	spec, ok := whatsapp_services.FindTemplate(v.MessageTemplateName)
	if wabaID == "" || v.Event == "" || !ok {
		return
	}
	if v.MessageTemplateLanguage != "" && v.MessageTemplateLanguage != spec.Definition.Language {
		return
	}
	var accounts []whatsapp_models.WhatsAppAccount
	if err := tenantdb.System(h.db).Select("id", "tenant_id").Where("waba_id = ?", wabaID).Find(&accounts).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to find accounts of waba", zap.String("waba_id", wabaID), zap.Error(err))
		return
	}
	for _, acc := range accounts {
		db := tenantdb.ForTenant(h.db, acc.TenantID)
		if err := whatsapp_services.SetTemplateStatus(db, acc.TenantID, spec.Definition.Name, v.Event, v.Reason); err != nil {
			h.logger.Error("whatsapp webhook: failed to store template status", zap.Uint("tenant_id", acc.TenantID), zap.Error(err))
		}
		if spec.Definition.Name != whatsapp_models.ReminderTemplate {
			continue
		}
		if err := db.Model(&whatsapp_models.WhatsAppAccount{}).Where("id = ?", acc.ID).
			Updates(map[string]any{"template_status": v.Event, "template_reason": v.Reason}).Error; err != nil {
			h.logger.Error("whatsapp webhook: failed to store reminder template status", zap.Uint("tenant_id", acc.TenantID), zap.Error(err))
		}
	}
}
