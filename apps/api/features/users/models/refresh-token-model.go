package user_models

import "time"

// RefreshToken tracks one link in a refresh-token rotation chain ("family").
// FamilyID is the durable session identity: it survives rotation and is
// what gets revoked wholesale on logout or reuse detection. Each row is a
// single issued token; UsedAt marks it as rotated-out (still valid history,
// no longer presentable), RevokedAt marks the whole family as killed.
type RefreshToken struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	FamilyID     string     `gorm:"type:uuid;index;not null" json:"family_id"`
	TokenHash    string     `gorm:"uniqueIndex;not null" json:"-"`
	IssuedAt     time.Time  `gorm:"not null" json:"issued_at"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	RevokedAt    *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	ReplacedByID *uint      `json:"replaced_by_id,omitempty"`
	UserAgent    string     `json:"user_agent"`
	IPAddress    string     `json:"ip_address"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
