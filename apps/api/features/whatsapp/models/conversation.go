package whatsapp_models

import (
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/gorm"
)

// ReplyWindow is how long after the patient's last message free text may be
// sent (Meta's customer service window). Outside it only approved templates
// can be sent.
const ReplyWindow = 24 * time.Hour

// PreviewLength caps LastMessagePreview (in runes).
const PreviewLength = 120

// WhatsAppConversation is the thread between a clinic and one WhatsApp number
// (one per tenant + phone). Every message to or from that number belongs to
// it. PatientID is set when exactly one patient of the tenant has that phone,
// or when someone links it from the inbox; otherwise it stays nil.
type WhatsAppConversation struct {
	gorm.Model
	TenantID  uint                     `json:"tenant_id" gorm:"not null;uniqueIndex:idx_whatsapp_conversation_tenant_phone"`
	Phone     string                   `json:"phone" gorm:"not null;uniqueIndex:idx_whatsapp_conversation_tenant_phone"` // digits-only E.164
	PatientID *uint                    `json:"patient_id" gorm:"index"`
	Patient   *clinical_models.Patient `json:"patient,omitempty" gorm:"foreignKey:PatientID"`

	LastMessageAt      *time.Time `json:"last_message_at" gorm:"index"`
	LastMessagePreview string     `json:"last_message_preview"`
	LastDirection      string     `json:"last_direction"`
	LastContentType    string     `json:"last_content_type"`
	// LastInboundAt opens the reply window (ReplyWindow from it).
	LastInboundAt *time.Time `json:"last_inbound_at"`
	UnreadCount   int        `json:"unread_count" gorm:"not null;default:0"`
	// OptInRequestedAt is when Pengi asked this number to reply SÍ to receive
	// reminders. It is asked at most once per conversation, and a SÍ only
	// counts as consent once it is set.
	OptInRequestedAt *time.Time `json:"opt_in_requested_at"`
}

func (WhatsAppConversation) TableName() string { return "whatsapp_conversations" }

// WindowExpiresAt is when the reply window closes, or nil when the patient
// never wrote.
func (c WhatsAppConversation) WindowExpiresAt() *time.Time {
	if c.LastInboundAt == nil {
		return nil
	}
	t := c.LastInboundAt.Add(ReplyWindow)
	return &t
}

// WindowOpen reports whether free text can be sent at now.
func (c WhatsAppConversation) WindowOpen(now time.Time) bool {
	exp := c.WindowExpiresAt()
	return exp != nil && now.Before(*exp)
}
