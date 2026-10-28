package portal_services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/tenantdb"
	portal_models "pengi-med-saas/features/portal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RotationGrace is how long a refresh token already rotated out still
// answers with an access token (and no new cookie): two tabs refreshing at
// once present the same cookie, and the browser already holds the new one.
// Presented later it is a replay and revokes the session.
const RotationGrace = 30 * time.Second

var (
	// ErrSessionInvalid: unknown, expired or revoked session; sign in again.
	ErrSessionInvalid = errors.New("invalid portal session")
	// ErrSessionReused: a rotated-out refresh token presented again after
	// RotationGrace. The whole session has been revoked.
	ErrSessionReused = errors.New("portal refresh token reused")
)

// Client is who a session was opened from (stored with each refresh token).
type Client struct {
	UserAgent string
	IP        string
}

func hashJTI(jti string) string {
	sum := sha256.Sum256([]byte(jti))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// issueRefresh stores, on tx (bound to the tenant), the next refresh token of
// familyID and returns it signed.
func issueRefresh(tx *gorm.DB, tenantID, patientID uint, familyID string, client Client, now time.Time) (string, error) {
	token, jti, err := auth.GeneratePortalRefreshToken(tenantID, patientID, familyID)
	if err != nil {
		return "", err
	}
	row := portal_models.PortalRefreshToken{
		TenantID:  tenantID,
		PatientID: patientID,
		FamilyID:  familyID,
		TokenHash: hashJTI(jti),
		// UTC: SQLite (CI) compares timestamps as text.
		ExpiresAt: now.Add(auth.PortalRefreshTTL).UTC(),
		UserAgent: truncate(client.UserAgent, 255),
		IPAddress: truncate(client.IP, 64),
	}
	if err := tx.Create(&row).Error; err != nil {
		return "", err
	}
	return token, nil
}

// OpenSession starts a new session for the patient and returns its first
// refresh token.
func OpenSession(db *gorm.DB, tenantID, patientID uint, client Client, now time.Time) (string, error) {
	return issueRefresh(tenantdb.ForTenant(db, tenantID), tenantID, patientID, uuid.NewString(), client, now)
}

// RotateSession consumes the refresh token of claims and returns the next one
// of its session (sliding expiry). It returns "" and no error for a token
// rotated out less than RotationGrace ago: the caller answers with an access
// token and leaves the cookie alone. ErrSessionReused (session revoked) for
// an older rotated-out token, ErrSessionInvalid for an unknown, expired or
// revoked one.
func RotateSession(db *gorm.DB, claims auth.PortalClaims, client Client, now time.Time) (string, error) {
	var next string
	scoped := tenantdb.ForTenant(db, claims.TenantID)
	err := scoped.Transaction(func(tx *gorm.DB) error {
		var row portal_models.PortalRefreshToken
		if err := tx.Where("token_hash = ?", hashJTI(claims.JTI)).Limit(1).Find(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 || row.PatientID != claims.PatientID || row.FamilyID != claims.FamilyID ||
			row.RevokedAt != nil || !now.Before(row.ExpiresAt) {
			return ErrSessionInvalid
		}
		// Conditional: of two parallel refreshes only one rotates the token.
		res := tx.Model(&portal_models.PortalRefreshToken{}).
			Where("id = ? AND used_at IS NULL", row.ID).
			Update("used_at", now.UTC())
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			if err := tx.Where("id = ?", row.ID).Find(&row).Error; err != nil {
				return err
			}
			if row.UsedAt != nil && now.Sub(*row.UsedAt) < RotationGrace {
				return nil
			}
			return ErrSessionReused
		}
		var err error
		next, err = issueRefresh(tx, claims.TenantID, claims.PatientID, row.FamilyID, client, now)
		return err
	})
	if errors.Is(err, ErrSessionReused) {
		if revokeErr := RevokeSession(db, claims.TenantID, claims.FamilyID, now); revokeErr != nil {
			return "", revokeErr
		}
	}
	return next, err
}

// RevokeSession ends a session: none of its refresh tokens works again.
func RevokeSession(db *gorm.DB, tenantID uint, familyID string, now time.Time) error {
	return tenantdb.ForTenant(db, tenantID).Model(&portal_models.PortalRefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now.UTC()).Error
}

// PurgeSessions deletes, across tenants, refresh tokens expired before now
// (useless after expiry, revoked or not) and returns how many.
func PurgeSessions(db *gorm.DB, now time.Time) (int64, error) {
	res := tenantdb.System(db).Where("expires_at < ?", now.UTC()).Delete(&portal_models.PortalRefreshToken{})
	return res.RowsAffected, res.Error
}
