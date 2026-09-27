package signature_models

import (
	"time"

	"gorm.io/gorm"
)

// UserSignature is a user's electronic signature certificate (P12) within one
// tenant. The P12 lives in tenantfiles under FileName; the password is stored
// encrypted with core/secretbox.
type UserSignature struct {
	gorm.Model
	TenantID          uint      `json:"tenant_id" gorm:"uniqueIndex:idx_user_signature_tenant_user"`
	UserID            uint      `json:"user_id" gorm:"uniqueIndex:idx_user_signature_tenant_user"`
	FileName          string    `json:"-"`
	EncryptedPassword string    `json:"-"`
	SubjectName       string    `json:"subject_name"`
	SubjectSerial     string    `json:"subject_serial"`
	Issuer            string    `json:"issuer"`
	NotAfter          time.Time `json:"not_after"`
}

func (UserSignature) IsAuditable() bool { return true }
