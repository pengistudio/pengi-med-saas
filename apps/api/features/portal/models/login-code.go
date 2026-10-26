package portal_models

import (
	"time"

	"gorm.io/gorm"
)

// Login code delivery channels.
const (
	ChannelWhatsApp = "whatsapp"
	ChannelEmail    = "email"
)

// PortalLoginCode is a one-time code a patient asked for to sign in to the
// patient portal. Only its HMAC is stored. A code is valid until ExpiresAt,
// for at most MaxAttempts verifications, and once (UsedAt). Issuing a new code
// expires the patient's previous unused ones.
type PortalLoginCode struct {
	gorm.Model
	TenantID  uint       `json:"tenant_id" gorm:"not null;index:idx_portal_login_code_patient"`
	PatientID uint       `json:"patient_id" gorm:"not null;index:idx_portal_login_code_patient"`
	Channel   string     `json:"channel" gorm:"size:20;not null"`
	CodeHash  string     `json:"-" gorm:"size:64;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	Attempts  int        `json:"attempts" gorm:"not null;default:0"`
	UsedAt    *time.Time `json:"used_at"`
}

func (PortalLoginCode) TableName() string { return "portal_login_codes" }
