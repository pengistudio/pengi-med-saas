package portal_models

import (
	"time"

	"gorm.io/gorm"
)

// PortalContactCode is a one-time code a person without a patient record
// asked for to verify a contact (email or WhatsApp phone) before requesting
// a first appointment. Only HMACs are stored: of the normalized contact
// (ContactHash) and of the code. Same rules as PortalLoginCode: valid until
// ExpiresAt, for at most MaxAttempts verifications, and once (UsedAt);
// issuing a new code expires the contact's previous unused ones.
type PortalContactCode struct {
	gorm.Model
	TenantID    uint       `json:"tenant_id" gorm:"not null;index:idx_portal_contact_code_contact"`
	Channel     string     `json:"channel" gorm:"size:20;not null"`
	ContactHash string     `json:"-" gorm:"size:64;not null;index:idx_portal_contact_code_contact"`
	CodeHash    string     `json:"-" gorm:"size:64;not null"`
	ExpiresAt   time.Time  `json:"expires_at" gorm:"not null"`
	Attempts    int        `json:"attempts" gorm:"not null;default:0"`
	UsedAt      *time.Time `json:"used_at"`
}

func (PortalContactCode) TableName() string { return "portal_contact_codes" }
