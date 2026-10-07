package whatsapp_services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SendQueue is the RabbitMQ queue of messages to send.
const SendQueue = "whatsapp.send"

// SendTask is the body of a SendQueue message.
type SendTask struct {
	MessageID uint `json:"message_id"`
}

// errTransient marks failures on our side (database) worth retrying.
var errTransient = errors.New("transient whatsapp send error")

// ErrNotSent is returned by Deliver when the message ended up failed or
// skipped instead of sent; the message row holds the reason.
var ErrNotSent = errors.New("whatsapp message not sent")

// IsRetryable reports whether a Send error should go through the retry ladder.
func IsRetryable(err error) bool {
	return errors.Is(err, errTransient) || whatsapp.IsRetryable(err)
}

// Publisher puts a task on a queue.
type Publisher interface {
	Publish(queue string, body []byte) error
}

// RabbitPublisher publishes on the process-wide RabbitMQ publish channel.
type RabbitPublisher struct{}

var errBrokerDisconnected = errors.New("rabbitmq disconnected")

func (RabbitPublisher) Publish(queue string, body []byte) error {
	ch := rabbitmq.PublishChannel()
	if ch == nil {
		return errBrokerDisconnected
	}
	return rabbitmq.PublishMessage(ch, queue, body)
}

// EnqueueSend publishes the send task of message id.
func EnqueueSend(p Publisher, id uint) error {
	body, err := json.Marshal(SendTask{MessageID: id})
	if err != nil {
		return err
	}
	return p.Publish(SendQueue, body)
}

// Sender sends queued messages through the Graph API.
type Sender struct {
	db     *gorm.DB
	logger *zap.Logger
	client *whatsapp.Client
	loc    *time.Location
	now    func() time.Time
}

func NewSender(db *gorm.DB, logger *zap.Logger, client *whatsapp.Client) *Sender {
	return &Sender{db: db, logger: logger, client: client, loc: ClinicLocation(), now: time.Now}
}

// WithClock overrides the clock and time zone (tests).
func (s *Sender) WithClock(now func() time.Time, loc *time.Location) *Sender {
	s.now, s.loc = now, loc
	return s
}

// StartConsumer declares SendQueue (with its retry ladder) and consumes it on ch.
func (s *Sender) StartConsumer(ch *amqp.Channel) error {
	q, err := rabbitmq.DeclareQueueWithRetry(ch, SendQueue)
	if err != nil {
		return fmt.Errorf("declare %s: %w", SendQueue, err)
	}
	rabbitmq.StartConsumer(ch, q.Name, s.HandleTask, IsRetryable)
	return nil
}

// HandleTask decodes one SendTask and sends its message.
func (s *Sender) HandleTask(body []byte) error {
	var task SendTask
	if err := json.Unmarshal(body, &task); err != nil || task.MessageID == 0 {
		return fmt.Errorf("decode whatsapp send task: %v", err)
	}
	return s.Send(task.MessageID)
}

// Send sends a queued reminder. A message that is not queued (already sent,
// claimed by another consumer, failed) is left alone, so a duplicate task is
// harmless. Retryable failures put the message back to queued and return an
// error IsRetryable accepts; anything else ends as failed or skipped.
func (s *Sender) Send(id uint) error {
	var msg whatsapp_models.WhatsAppMessage
	if err := tenantdb.System(s.db).Where("id = ?", id).Limit(1).Find(&msg).Error; err != nil {
		return fmt.Errorf("%w: load message %d: %v", errTransient, id, err)
	}
	if msg.ID == 0 || msg.Status != whatsapp_models.MessageStatusQueued {
		return nil
	}
	db := tenantdb.ForTenant(s.db, msg.TenantID)
	claim := db.Model(&whatsapp_models.WhatsAppMessage{}).
		Where("id = ? AND status = ?", msg.ID, whatsapp_models.MessageStatusQueued).
		Update("status", whatsapp_models.MessageStatusSending)
	if claim.Error != nil {
		return fmt.Errorf("%w: claim message %d: %v", errTransient, id, claim.Error)
	}
	if claim.RowsAffected == 0 {
		return nil
	}
	msg.Status = whatsapp_models.MessageStatusSending

	account, token, code, err := s.account(msg.TenantID)
	if err != nil {
		return s.finish(db, &msg, whatsapp_models.MessageStatusFailed, code, err.Error())
	}

	data, to, skip, code, detail := s.reminderData(db, msg)
	if skip {
		return s.finish(db, &msg, whatsapp_models.MessageStatusSkipped, code, detail)
	}
	if code != "" {
		return s.finish(db, &msg, whatsapp_models.MessageStatusFailed, code, detail)
	}
	msg.ToPhone = to
	if ok, _, err := canSendTemplate(s.db, msg.TenantID, s.now(), s.loc); err != nil {
		// Fail open: a counting error must not stop reminders.
		s.logger.Error("failed to check whatsapp monthly limit", zap.Uint("message_id", msg.ID), zap.Error(err))
	} else if !ok {
		if err := s.finish(db, &msg, whatsapp_models.MessageStatusSkipped, whatsapp_models.ErrorCodeMonthlyLimit, "monthly message limit reached"); err != nil {
			return err
		}
		notifyUsage(s.db, s.logger, msg.TenantID, s.now(), s.loc)
		return nil
	}
	return s.deliverTemplate(db, &msg, account, token, data)
}

// Deliver sends msg (already created as sending) right away with data; used
// by the test endpoint. It returns ErrNotSent (or a Graph error) when the
// message did not go out; msg is updated either way.
func (s *Sender) Deliver(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, token string, data ReminderData) error {
	return s.deliverTemplate(db, msg, account, token, data)
}

// DeliverTemplate sends msg (a catalog template already recorded in its
// conversation, as sending, with its rendered Body) as tpl. It returns a
// Graph error when Meta rejected it; the message is then failed.
func (s *Sender) DeliverTemplate(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, token string, tpl whatsapp.TemplateMessage) error {
	return s.deliver(db, msg, account, func() (string, error) {
		return s.client.SendTemplate(token, account.PhoneNumberID, tpl)
	})
}

// DeliverText sends msg (a reply already recorded in its conversation, as
// sending) as free text. It returns a Graph error when Meta rejected it; the
// message is then failed with Meta's code.
func (s *Sender) DeliverText(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, token string) error {
	return s.deliver(db, msg, account, func() (string, error) {
		return s.client.SendText(token, account.PhoneNumberID, msg.ToPhone, msg.Body)
	})
}

// deliverTemplate renders the reminder into msg.Body, puts msg in the thread
// of its phone (once) and sends the template.
func (s *Sender) deliverTemplate(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, token string, data ReminderData) error {
	msg.Body = RenderReminderBody(data)
	s.thread(db, msg)
	tpl := ReminderMessage(msg.ToPhone, msg.ID, data)
	return s.deliver(db, msg, account, func() (string, error) {
		return s.client.SendTemplate(token, account.PhoneNumberID, tpl)
	})
}

// thread attaches msg to the conversation with its phone, creating it, unless
// it is already attached (a retried reminder). A failure here is logged and
// does not stop the send.
func (s *Sender) thread(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage) {
	if msg.ConversationID != nil || msg.ToPhone == "" {
		return
	}
	conv, err := EnsureConversation(db, msg.TenantID, msg.ToPhone, msg.PatientID)
	if err == nil {
		err = RecordOutbound(db, &conv, msg, s.now())
	}
	if err != nil {
		msg.ConversationID = nil
		s.logger.Error("failed to add whatsapp message to its conversation", zap.Uint("message_id", msg.ID), zap.Error(err))
	}
}

func (s *Sender) deliver(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, account whatsapp_models.WhatsAppAccount, send func() (string, error)) error {
	wamID, err := send()
	if err != nil {
		if IsRetryable(err) && msg.Kind == whatsapp_models.KindReminder {
			s.logger.Warn("whatsapp send failed, will retry", zap.Uint("message_id", msg.ID), zap.Error(err))
			if uerr := db.Model(msg).Updates(map[string]any{"status": whatsapp_models.MessageStatusQueued, "error_detail": truncate(err.Error())}).Error; uerr != nil {
				s.logger.Error("failed to re-queue whatsapp message", zap.Uint("message_id", msg.ID), zap.Error(uerr))
			}
			return err
		}
		code := whatsapp_models.ErrorCodeInternal
		var apiErr *whatsapp.APIError
		if errors.As(err, &apiErr) {
			code = strconv.Itoa(apiErr.Code)
			if apiErr.IsAuthError() {
				s.markTokenInvalid(account)
			}
		}
		s.logger.Warn("whatsapp send failed", zap.Uint("message_id", msg.ID), zap.String("code", code), zap.Error(err))
		if ferr := s.finish(db, msg, whatsapp_models.MessageStatusFailed, code, err.Error()); ferr != nil {
			return ferr
		}
		return err
	}
	now := s.now()
	updates := map[string]any{
		"status": whatsapp_models.MessageStatusSent, "wam_id": wamID, "to_phone": msg.ToPhone,
		"sent_at": now, "error_code": "", "error_detail": "",
	}
	if err := db.Model(msg).Updates(updates).Error; err != nil {
		// Sent but not recorded: do not retry (that would send it twice).
		s.logger.Error("whatsapp message sent but not recorded", zap.Uint("message_id", msg.ID), zap.String("wam_id", wamID), zap.Error(err))
		return nil
	}
	msg.Status, msg.WamID, msg.SentAt, msg.ErrorCode, msg.ErrorDetail = whatsapp_models.MessageStatusSent, wamID, &now, "", ""
	if counted(msg.Kind) {
		notifyUsage(s.db, s.logger, msg.TenantID, now, s.loc)
	}
	return nil
}

// counted reports whether messages of kind count toward the monthly cap.
func counted(kind string) bool {
	for _, k := range whatsapp_models.CountedKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// finish marks msg with a terminal status. It returns nil so the queue drops
// the task; the reason lives on the row.
func (s *Sender) finish(db *gorm.DB, msg *whatsapp_models.WhatsAppMessage, status, code, detail string) error {
	msg.Status, msg.ErrorCode, msg.ErrorDetail = status, code, truncate(detail)
	err := db.Model(msg).Updates(map[string]any{"status": status, "error_code": code, "error_detail": msg.ErrorDetail, "to_phone": msg.ToPhone}).Error
	if err != nil {
		return fmt.Errorf("%w: finish message %d: %v", errTransient, msg.ID, err)
	}
	return nil
}

// account loads the tenant's connected account and its token.
func (s *Sender) account(tenantID uint) (whatsapp_models.WhatsAppAccount, string, string, error) {
	var account whatsapp_models.WhatsAppAccount
	err := tenantdb.ForTenant(s.db, tenantID).Limit(1).Find(&account).Error
	if err != nil || account.ID == 0 || account.Status != whatsapp_models.AccountStatusConnected {
		return account, "", whatsapp_models.ErrorCodeNotConnected, errors.New("whatsapp account not connected")
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return account, "", whatsapp_models.ErrorCodeTokenUnusable, err
	}
	return account, token, "", nil
}

// reminderData loads what a reminder needs. skip means the appointment is no
// longer worth reminding; a non-empty code without skip is a failure.
func (s *Sender) reminderData(db *gorm.DB, msg whatsapp_models.WhatsAppMessage) (data ReminderData, to string, skip bool, code, detail string) {
	if msg.AppointmentID == nil {
		return data, "", false, whatsapp_models.ErrorCodeInternal, "reminder without appointment"
	}
	var appt clinical_models.Appointment
	if err := db.Preload("Patient").Where("id = ?", *msg.AppointmentID).Limit(1).Find(&appt).Error; err != nil {
		return data, "", false, whatsapp_models.ErrorCodeInternal, err.Error()
	}
	if appt.ID == 0 || (appt.Status != AppointmentScheduled && appt.Status != AppointmentConfirmed) {
		return data, "", true, whatsapp_models.ErrorCodeAppointmentGone, "appointment no longer active"
	}
	start, err := AppointmentStart(appt, s.loc)
	if err != nil || !start.After(s.now()) {
		return data, "", true, whatsapp_models.ErrorCodeAppointmentGone, "appointment already started"
	}
	if !appt.Patient.WhatsAppOptIn {
		// Opted out (or consent removed) after the reminder was queued.
		return data, "", true, whatsapp_models.ErrorCodeNoOptIn, "patient has not opted in"
	}
	if strings.TrimSpace(appt.Patient.Phone) == "" {
		return data, "", false, whatsapp_models.ErrorCodeNoPhone, "patient has no phone"
	}
	to, err = NormalizePhone(appt.Patient.Phone)
	if err != nil {
		return data, "", false, whatsapp_models.ErrorCodeInvalidPhone, appt.Patient.Phone
	}
	return ReminderData{
		PatientName: strings.TrimSpace(appt.Patient.FirstName + " " + appt.Patient.LastName),
		ClinicName:  ClinicName(s.db, msg.TenantID),
		Start:       start,
	}, to, false, "", ""
}

// ClinicName is the name patients know the tenant by.
func ClinicName(db *gorm.DB, tenantID uint) string {
	var tenant tenant_models.Tenant
	if err := tenantdb.System(db).Select("id", "name", "trade_name").Where("id = ?", tenantID).Limit(1).Find(&tenant).Error; err != nil {
		return ""
	}
	if tenant.TradeName != "" {
		return tenant.TradeName
	}
	return tenant.Name
}

func (s *Sender) markTokenInvalid(account whatsapp_models.WhatsAppAccount) {
	if account.ID == 0 {
		return
	}
	if err := tenantdb.ForTenant(s.db, account.TenantID).Model(&account).Update("status", whatsapp_models.AccountStatusTokenInvalid).Error; err != nil {
		s.logger.Error("failed to mark whatsapp token invalid", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
	}
}

func truncate(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

// Appointment statuses the reminders care about.
const (
	AppointmentScheduled = "scheduled"
	AppointmentConfirmed = "confirmed"
	AppointmentCancelled = "cancelled"
)
