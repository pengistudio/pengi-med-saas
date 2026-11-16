package user_models

import "time"

// Passkey is a WebAuthn credential a user registered to sign in without a
// password. Credential is the library's webauthn.Credential as JSON (public
// key, sign counter, backup flags); CredentialID is its ID, kept in its own
// column to look a credential up. Users are global (not per tenant), so
// there is no TenantID.
type Passkey struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UserID       uint       `gorm:"not null;index" json:"-"`
	Name         string     `gorm:"size:100;not null" json:"name"`
	CredentialID []byte     `gorm:"not null;uniqueIndex" json:"-"`
	Credential   []byte     `gorm:"not null" json:"-"`
	LastUsedAt   *time.Time `json:"last_used_at"`
}

func (Passkey) TableName() string { return "user_passkeys" }

// WebAuthnChallenge is the server-side state of one registration or login
// ceremony (webauthn.SessionData as JSON). Single use: it is deleted when
// finished, and expired rows are swept on every new ceremony. Kept in the
// database so any API instance can finish what another started.
type WebAuthnChallenge struct {
	ID        string    `gorm:"primarykey;size:64"`
	Kind      string    `gorm:"size:16;not null"`
	UserID    uint      `gorm:"not null;default:0"` // 0 for a login (the user is not known yet)
	Data      []byte    `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (WebAuthnChallenge) TableName() string { return "user_webauthn_challenges" }

const (
	ChallengeRegister = "register"
	ChallengeLogin    = "login"
)
