package whatsapp_models

import (
	"time"

	"gorm.io/gorm"
)

// FAQ limits of the WhatsApp assistant (in runes).
const (
	MaxBotFAQEntries  = 30
	MaxBotFAQQuestion = 200
	MaxBotFAQAnswer   = 500
)

// BotFAQEntry is one question/answer pair a clinic admin wrote: with the
// clinic's address, office hours and phone, the only material the assistant
// may answer from.
type BotFAQEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// WhatsAppBotSettings is a clinic's WhatsApp assistant (one per tenant). Off
// by default; enabling it requires the plan's WHATSAPP and AI features, the
// clinic's AI opt-in, a provider key for the "bot" AI function, office hours
// and address loaded and at least one FAQ entry. By default it answers only
// outside office hours; DuringOfficeHours makes it answer always.
type WhatsAppBotSettings struct {
	gorm.Model
	TenantID          uint          `json:"tenant_id" gorm:"uniqueIndex;not null"`
	Enabled           bool          `json:"enabled" gorm:"not null;default:false"`
	DuringOfficeHours bool          `json:"during_office_hours" gorm:"not null;default:false"`
	FAQ               []BotFAQEntry `json:"faq" gorm:"serializer:json;type:text"`
	EnabledAt         *time.Time    `json:"enabled_at"`
}

func (WhatsAppBotSettings) TableName() string { return "whatsapp_bot_settings" }

// Assistant actions on one inbound message (WhatsAppBotEvent.Action).
const (
	BotActionAnswer  = "answer"  // replied from the FAQ
	BotActionHandoff = "handoff" // handed the chat to reception
	BotActionSilent  = "silent"  // said nothing (a person is on it, opted out...)
)

// Reasons of a handoff or a silence (WhatsAppBotEvent.Reason,
// WhatsAppConversation.NeedsHumanReason). Codes, never the model's words.
const (
	BotReasonAskedPerson    = "asked_person"       // the patient asked for a person
	BotReasonEmergency      = "emergency"          // emergency words: told to call 911
	BotReasonSymptoms       = "symptoms"           // health question: never answered
	BotReasonNonText        = "non_text"           // audio, image, ...
	BotReasonSuspicious     = "suspicious_text"    // look-alike letters mixed in a word
	BotReasonTooLong        = "too_long"           // message too long to answer
	BotReasonNotInFAQ       = "not_in_faq"         // the model handed off
	BotReasonGuard          = "guard"              // the reply had data not in the FAQ
	BotReasonUnparseable    = "unparseable"        // the model's answer was unusable
	BotReasonQuota          = "quota"              // monthly AI conversations used up
	BotReasonNotConfigured  = "not_configured"     // no provider key
	BotReasonProviderError  = "provider_error"     // the provider failed
	BotReasonDailyLimit     = "daily_limit"        // too many assistant replies today
	BotReasonClinicLimit    = "clinic_daily_limit" // too many assistant messages in the clinic today
	BotReasonOverloaded     = "overloaded"         // too many messages being handled at once
	BotReasonSendFailed     = "send_failed"        // Meta rejected the reply
	BotReasonHumanActive    = "human_active"       // a person answered recently
	BotReasonAwaitingHuman  = "awaiting_human"     // already handed off
	BotReasonOptedOut       = "opted_out"          // the number sent STOP
	BotReasonOutsideWindow  = "outside_window"     // the 24 h window is closed
	BotReasonConsentConsume = "consent_answer"     // the text was a consent "SÍ"
	BotReasonInternal       = "internal"           // storage error
)

// WhatsAppBotEvent records what the assistant did with one inbound message.
// The unique (tenant, wam id) index is the claim that makes it idempotent:
// Meta retries webhooks, and only the delivery that inserts the row acts.
// It holds the action and a reason code, never the patient's text, the
// prompt or the model's answer.
type WhatsAppBotEvent struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	TenantID       uint      `json:"tenant_id" gorm:"not null;uniqueIndex:idx_whatsapp_bot_event_wam"`
	WamID          string    `json:"wam_id" gorm:"type:varchar(255);not null;uniqueIndex:idx_whatsapp_bot_event_wam"`
	ConversationID uint      `json:"conversation_id" gorm:"not null;index"`
	Action         string    `json:"action" gorm:"type:varchar(20);not null;default:''"`
	Reason         string    `json:"reason" gorm:"type:varchar(40);not null;default:''"`
	ModelCalled    bool      `json:"model_called" gorm:"not null;default:false"`
	MessageSent    bool      `json:"message_sent" gorm:"not null;default:false"`
	CreatedAt      time.Time `json:"created_at"`
}

func (WhatsAppBotEvent) TableName() string { return "whatsapp_bot_events" }
