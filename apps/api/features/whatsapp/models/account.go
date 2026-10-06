package whatsapp_models

import (
	"errors"
	"os"
	"time"

	"pengi-med-saas/core/secretbox"

	"gorm.io/gorm"
)

// Connection modes.
const (
	ModeManual   = "manual"   // WABA id + phone number id + token typed in by the clinic
	ModeEmbedded = "embedded" // Meta Embedded Signup (OAuth code exchange)
)

// Account statuses.
const (
	// AccountStatusConnected: credentials worked last time they were used.
	AccountStatusConnected = "connected"
	// AccountStatusTokenInvalid: Meta rejected the token (expired / revoked);
	// reminders stop until the clinic connects again.
	AccountStatusTokenInvalid = "token_invalid"
)

// Template statuses are Meta's own values (APPROVED, PENDING, REJECTED,
// PAUSED, DISABLED, ...); TemplateStatusNone means the template could not be
// created or looked up yet.
const (
	TemplateStatusNone     = ""
	TemplateStatusApproved = "APPROVED"
	TemplateStatusPending  = "PENDING"
)

// EncryptionKeyEnv holds the key that seals WhatsApp access tokens.
const EncryptionKeyEnv = "WHATSAPP_ENCRYPTION_KEY"

// ErrNoToken means the account has no access token stored.
var ErrNoToken = errors.New("whatsapp account has no access token")

// WhatsAppAccount is the WhatsApp Business number a tenant connected. One per
// tenant; the webhook finds the tenant by PhoneNumberID.
type WhatsAppAccount struct {
	gorm.Model
	TenantID             uint       `json:"tenant_id" gorm:"uniqueIndex;not null"`
	WabaID               string     `json:"waba_id" gorm:"index;not null"`
	PhoneNumberID        string     `json:"phone_number_id" gorm:"uniqueIndex;not null"`
	DisplayPhone         string     `json:"display_phone"`
	VerifiedName         string     `json:"verified_name"`
	AccessTokenEncrypted string     `json:"-" gorm:"type:text"`
	PinEncrypted         string     `json:"-" gorm:"type:text"`
	Mode                 string     `json:"mode"`
	Status               string     `json:"status"`
	TemplateStatus       string     `json:"template_status"`
	TemplateReason       string     `json:"template_reason"`
	RemindersEnabled     bool       `json:"reminders_enabled"`
	ReminderOffsets      []int      `json:"reminder_offsets" gorm:"serializer:json;type:text"`
	ConnectedAt          *time.Time `json:"connected_at"`
}

func (WhatsAppAccount) TableName() string { return "whatsapp_accounts" }

// Box returns the cipher for WhatsApp secrets: WHATSAPP_ENCRYPTION_KEY when
// set, otherwise the same fallback as secretbox.FromEnv
// (SIGNATURE_ENCRYPTION_KEY, or a dev key outside release).
func Box() (*secretbox.Box, error) {
	if os.Getenv(EncryptionKeyEnv) != "" {
		return secretbox.FromEnvVar(EncryptionKeyEnv)
	}
	return secretbox.FromEnv()
}

// SealAccessToken stores token sealed with box.
func (a *WhatsAppAccount) SealAccessToken(box *secretbox.Box, token string) error {
	sealed, err := box.Seal(token)
	if err != nil {
		return err
	}
	a.AccessTokenEncrypted = sealed
	return nil
}

// SealPin stores the phone's two-step verification pin sealed with box.
func (a *WhatsAppAccount) SealPin(box *secretbox.Box, pin string) error {
	sealed, err := box.Seal(pin)
	if err != nil {
		return err
	}
	a.PinEncrypted = sealed
	return nil
}

// OpenAccessToken returns the plaintext token, opening it with Box().
func (a *WhatsAppAccount) OpenAccessToken() (string, error) {
	if a.AccessTokenEncrypted == "" {
		return "", ErrNoToken
	}
	box, err := Box()
	if err != nil {
		return "", err
	}
	return box.Open(a.AccessTokenEncrypted)
}

// CanSendReminders reports whether the scheduler should consider the account.
func (a *WhatsAppAccount) CanSendReminders() bool {
	return a.Status == AccountStatusConnected && a.RemindersEnabled &&
		a.TemplateStatus == TemplateStatusApproved && len(a.ReminderOffsets) > 0
}
