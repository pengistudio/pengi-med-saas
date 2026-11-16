package backoffice_models

import (
	"time"

	"gorm.io/gorm"
)

// hasSecondFactorSQL is true for a backoffice_users row with TOTP on or at
// least one passkey.
const hasSecondFactorSQL = "(backoffice_users.two_factor_enabled = ? OR EXISTS (SELECT 1 FROM backoffice_user_passkeys p WHERE p.backoffice_user_id = backoffice_users.id))"

// WithSecondFactor narrows a query on backoffice_users to admins that have a
// second factor. Every session check goes through it: an admin without one
// (not enrolled yet, or reset by another admin) has no usable session.
func WithSecondFactor(db *gorm.DB) *gorm.DB {
	return db.Where(hasSecondFactorSQL, true)
}

// HasSecondFactor reports whether admin id has TOTP on or a passkey.
func HasSecondFactor(db *gorm.DB, id uint) (bool, error) {
	var n int64
	err := db.Model(&BackofficeUser{}).Scopes(WithSecondFactor).Where("backoffice_users.id = ?", id).Count(&n).Error
	return n > 0, err
}

// Second-factor tables of backoffice admins, separate from the clinic users'
// (user_recovery_codes, user_passkeys, user_webauthn_challenges): admin IDs
// and user IDs overlap, and the relying party differs (BACKOFFICE_URL).

// BackofficeRecoveryCode is a single-use second factor for a lost
// authenticator. Only the SHA-256 of the code is stored; it is shown once.
type BackofficeRecoveryCode struct {
	ID               uint `gorm:"primarykey"`
	CreatedAt        time.Time
	BackofficeUserID uint   `gorm:"not null;index"`
	CodeHash         string `gorm:"size:64;not null;index"`
	UsedAt           *time.Time
}

func (BackofficeRecoveryCode) TableName() string { return "backoffice_user_recovery_codes" }

// BackofficePasskey is a WebAuthn credential of an admin (discoverable, user
// verified). Credential is the library's webauthn.Credential as JSON.
type BackofficePasskey struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time  `json:"created_at"`
	BackofficeUserID uint       `gorm:"not null;index" json:"-"`
	Name             string     `gorm:"size:100;not null" json:"name"`
	CredentialID     []byte     `gorm:"not null;uniqueIndex" json:"-"`
	Credential       []byte     `gorm:"not null" json:"-"`
	LastUsedAt       *time.Time `json:"last_used_at"`
}

func (BackofficePasskey) TableName() string { return "backoffice_user_passkeys" }

// BackofficeWebAuthnChallenge is the server-side state of one ceremony
// (webauthn.SessionData as JSON), single use, swept when expired.
type BackofficeWebAuthnChallenge struct {
	ID               string    `gorm:"primarykey;size:64"`
	Kind             string    `gorm:"size:16;not null"`
	BackofficeUserID uint      `gorm:"not null;default:0"` // 0 for a login
	Data             []byte    `gorm:"not null"`
	ExpiresAt        time.Time `gorm:"not null;index"`
}

func (BackofficeWebAuthnChallenge) TableName() string { return "backoffice_webauthn_challenges" }

const (
	BackofficeChallengeRegister = "register"
	BackofficeChallengeLogin    = "login"
)
