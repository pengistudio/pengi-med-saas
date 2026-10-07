package whatsapp_models

import (
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/gorm"
)

// Message kinds.
const (
	KindReminder = "reminder"
	KindTest     = "test"
	KindReply    = "reply"   // free text sent from the inbox
	KindInbound  = "inbound" // a message the patient sent
	// KindTemplate: a catalog template sent from the inbox ("Nuevo mensaje" or
	// a thread whose window is closed). Counts toward the monthly plan cap.
	KindTemplate = "template"
	// KindSystem: free text Pengi sends on its own inside the window (asking
	// for WhatsApp consent and confirming it). Free, so it is not counted.
	KindSystem = "system"
)

// CountedKinds are the kinds Meta charges for (templates): the ones the
// monthly plan cap (max_whatsapp_messages) counts once sent.
var CountedKinds = []string{KindReminder, KindTest, KindTemplate}

// Message directions.
const (
	DirectionOutbound = "outbound"
	DirectionInbound  = "inbound"
)

// Content types of a message. Attachments are only labelled in v1 (the UI
// shows "[Imagen]", "[Audio]"...); they are not downloaded.
const (
	ContentText     = "text"
	ContentButton   = "button"
	ContentImage    = "image"
	ContentAudio    = "audio"
	ContentDocument = "document"
	ContentOther    = "other"
)

// Message statuses. queued → sending → sent → delivered → read, then replied
// when the patient taps a button; failed and skipped are terminal.
const (
	MessageStatusQueued    = "queued"
	MessageStatusSending   = "sending" // claimed by a consumer; guards against double sends
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
	MessageStatusFailed    = "failed"
	MessageStatusReplied   = "replied"
	// MessageStatusSkipped: the appointment was cancelled, moved past or
	// deleted between queueing and sending, so nothing was sent.
	MessageStatusSkipped = "skipped"
	// MessageStatusReceived: an inbound message (the patient wrote to us).
	MessageStatusReceived = "received"
)

// Error codes stored in WhatsAppMessage.ErrorCode besides Meta's numeric codes
// (e.g. "131026", "190"). The UI shows whatsapp.message.error.<code>.
const (
	ErrorCodeInvalidPhone    = "invalid_phone"
	ErrorCodeNotConnected    = "not_connected"
	ErrorCodeTokenUnusable   = "token_unusable"
	ErrorCodeInternal        = "internal"
	ErrorCodeNoPhone         = "no_phone"
	ErrorCodeAppointmentGone = "appointment_inactive"
	ErrorCodeNoOptIn         = "no_opt_in"
	// ErrorCodeMonthlyLimit: the tenant reached its plan's monthly message cap.
	ErrorCodeMonthlyLimit = "monthly_limit"
)

// Reply actions (payload "<action>:<message id>" of the template buttons).
const (
	ReplyConfirm = "confirm"
	ReplyCancel  = "cancel"
)

// ReminderTemplate is the template every reminder uses.
const ReminderTemplate = "pengi_cita_recordatorio"

// WhatsAppMessage is one message of a conversation: sent (or to be sent) to a
// patient, or received from one (Direction). The unique (appointment_id,
// offset_hours) index makes reminders idempotent: a reminder for the same
// appointment and offset can only ever be queued once. Test messages, replies
// and inbound messages have no appointment (NULLs never collide). Inbound
// messages are deduplicated by WamID before insert (Meta retries webhooks).
type WhatsAppMessage struct {
	gorm.Model
	TenantID      uint                     `json:"tenant_id" gorm:"index;not null"`
	Kind          string                   `json:"kind" gorm:"not null;default:reminder"`
	AppointmentID *uint                    `json:"appointment_id" gorm:"uniqueIndex:idx_whatsapp_message_appointment_offset"`
	OffsetHours   int                      `json:"offset_hours" gorm:"uniqueIndex:idx_whatsapp_message_appointment_offset"`
	PatientID     *uint                    `json:"patient_id" gorm:"index"`
	Patient       *clinical_models.Patient `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	ToPhone       string                   `json:"to_phone"`
	Template      string                   `json:"template"`
	WamID         string                   `json:"wam_id" gorm:"index"`
	Status        string                   `json:"status" gorm:"index;not null"`
	ErrorCode     string                   `json:"error_code"`
	ErrorDetail   string                   `json:"error_detail" gorm:"type:text"`
	Reply         string                   `json:"reply"`
	SentAt        *time.Time               `json:"sent_at"`
	DeliveredAt   *time.Time               `json:"delivered_at"`
	ReadAt        *time.Time               `json:"read_at"`
	RepliedAt     *time.Time               `json:"replied_at"`

	// ConversationID is set once the message is part of a thread: when a
	// reminder or test is delivered, and always for replies and inbound ones.
	ConversationID *uint  `json:"conversation_id" gorm:"index"`
	Direction      string `json:"direction" gorm:"not null;default:outbound"`
	// Body is the text shown in the thread: the rendered template for
	// reminders and tests, the patient's text (or caption / button label) for
	// inbound messages.
	Body         string `json:"body" gorm:"type:text"`
	ContentType  string `json:"content_type" gorm:"not null;default:text"`
	SentByUserID *uint  `json:"sent_by_user_id"`
}

func (WhatsAppMessage) TableName() string { return "whatsapp_messages" }

// statusRank orders the delivery statuses so a late webhook never moves a
// message backwards (e.g. "delivered" arriving after "read").
var statusRank = map[string]int{
	MessageStatusQueued:    0,
	MessageStatusSending:   1,
	MessageStatusSent:      2,
	MessageStatusDelivered: 3,
	MessageStatusRead:      4,
	MessageStatusReplied:   5,
}

// Advances reports whether moving from current to next is forward progress.
// failed only applies to a message not yet delivered.
func Advances(current, next string) bool {
	if current == MessageStatusFailed || current == MessageStatusSkipped || current == MessageStatusReceived {
		return false
	}
	if next == MessageStatusFailed {
		return statusRank[current] <= statusRank[MessageStatusSent]
	}
	n, ok := statusRank[next]
	return ok && n > statusRank[current]
}
