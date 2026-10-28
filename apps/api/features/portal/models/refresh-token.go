package portal_models

import "time"

// PortalRefreshToken is one link of a patient's portal session (a rotation
// chain, "family"). Only the hash of the token's jti is stored. Each refresh
// marks the presented row used and issues the next one in the family;
// RevokedAt kills the whole family (logout, or a used token presented again).
type PortalRefreshToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	TenantID  uint       `json:"tenant_id" gorm:"not null;index:idx_portal_refresh_patient"`
	PatientID uint       `json:"patient_id" gorm:"not null;index:idx_portal_refresh_patient"`
	FamilyID  string     `json:"family_id" gorm:"size:36;not null;index"`
	TokenHash string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	UsedAt    *time.Time `json:"used_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	UserAgent string     `json:"user_agent" gorm:"size:255"`
	IPAddress string     `json:"ip_address" gorm:"size:64"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (PortalRefreshToken) TableName() string { return "portal_refresh_tokens" }
