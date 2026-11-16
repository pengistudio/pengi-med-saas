package user_handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"pengi-med-saas/core/auth"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ErrRefreshTokenInvalid covers malformed/expired tokens and rows that
	// aged out of the sliding window — generic "please log in again".
	ErrRefreshTokenInvalid = errors.New("invalid or expired refresh token")
	// ErrRefreshTokenReused signals a rotated-out token was presented again
	// — a theft indicator. The entire family has been revoked as a result.
	ErrRefreshTokenReused = errors.New("refresh token reuse detected")
	// ErrRefreshTokenRevoked signals the family was already revoked
	// (prior logout, or a prior reuse-detection event).
	ErrRefreshTokenRevoked = errors.New("refresh token family revoked")
)

func hashJTI(jti string) string {
	sum := sha256.Sum256([]byte(jti))
	return hex.EncodeToString(sum[:])
}

// createRefreshTokenRow mints a new refresh JWT + DB row within familyID and
// returns the raw token for the caller to set as a cookie.
func createRefreshTokenRow(db *gorm.DB, userID uint, username, familyID string, c *gin.Context) (string, *user_models.RefreshToken, error) {
	rawToken, jti, err := auth.GenerateRefreshToken(username, int64(userID), familyID)
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	row := &user_models.RefreshToken{
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: hashJTI(jti),
		IssuedAt:  now,
		ExpiresAt: now.Add(auth.RefreshTokenTTL()),
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	}
	if err := db.Create(row).Error; err != nil {
		return "", nil, err
	}
	return rawToken, row, nil
}

// IssueRefreshFamily starts a brand new rotation chain for userID at login
// and sets the refresh_token cookie.
func IssueRefreshFamily(db *gorm.DB, userID uint, username string, c *gin.Context) error {
	familyID := uuid.NewString()
	rawToken, _, err := createRefreshTokenRow(db, userID, username, familyID, c)
	if err != nil {
		return err
	}
	auth.SetRefreshTokenCookie(rawToken, c)
	return nil
}

// RevokeFamily kills every still-live token in a family — used by logout
// and by reuse detection.
func RevokeFamily(db *gorm.DB, familyID string) error {
	return db.Model(&user_models.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
}

// RevokeUserSessions kills every refresh-token family of a user (and the
// legacy single token): after a password reset nobody stays signed in with
// the old password.
func RevokeUserSessions(db *gorm.DB, userID uint) error {
	if err := db.Model(&user_models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error; err != nil {
		return err
	}
	return db.Model(&user_models.User{}).
		Where("id = ?", userID).
		Update("refresh_token", "").Error
}

// RotateRefreshToken validates the presented refresh token and, on success,
// rotates it: the old row is marked used, a new one is issued in the same
// family with the sliding expiry reset, and a fresh access token is
// returned.
//
// Reusing an already-rotated-out token is treated as theft: the whole
// family is revoked immediately (ErrRefreshTokenReused). Legacy tokens
// issued before rotation existed (no familyId/jti claims) are upgraded
// transparently into a brand new family rather than rejected, so sessions
// already active at deploy time survive instead of being force-logged-out.
func RotateRefreshToken(db *gorm.DB, presentedToken string, c *gin.Context, logger *zap.Logger) (accessToken string, err error) {
	claims, err := auth.ValidateRefreshToken(presentedToken)
	if err != nil {
		return "", ErrRefreshTokenInvalid
	}

	if claims.FamilyID == "" || claims.JTI == "" {
		return rotateLegacyToken(db, claims, presentedToken, c, logger)
	}

	tokenHash := hashJTI(claims.JTI)
	var row user_models.RefreshToken
	if err := db.Where("token_hash = ?", tokenHash).First(&row).Error; err != nil {
		return "", ErrRefreshTokenRevoked
	}

	if row.RevokedAt != nil {
		return "", ErrRefreshTokenRevoked
	}

	if row.ExpiresAt.Before(time.Now()) {
		return "", ErrRefreshTokenInvalid
	}

	// A deleted user's sessions end even if a family was left unrevoked, and
	// so do sessions started before the user's sessions were invalidated
	// (password change/reset, 2FA enabled or reset).
	if _, ok := liveUser(db, row.UserID); !ok || sessionsInvalidatedSince(db, row.UserID, row.IssuedAt) {
		return "", ErrRefreshTokenRevoked
	}

	if row.UsedAt != nil {
		revokeAndLogReuse(db, row, c, logger, "replay of an already-rotated token")
		return "", ErrRefreshTokenReused
	}

	// Conditional update: only succeeds if this row is still unused and not
	// revoked. If RowsAffected is 0, either it was revoked since the read
	// above (logout, password change, 2FA enabled...) or a concurrent request
	// rotated it out from under us — the latter is treated as reuse.
	result := db.Model(&user_models.RefreshToken{}).
		Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", row.ID).
		Update("used_at", time.Now())
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		if presentedRowRevoked(db, row.ID) {
			return "", ErrRefreshTokenRevoked
		}
		revokeAndLogReuse(db, row, c, logger, "concurrent rotation race")
		return "", ErrRefreshTokenReused
	}

	rawToken, newRow, err := createRefreshTokenRow(db, row.UserID, claims.Username, row.FamilyID, c)
	if err != nil {
		return "", err
	}
	// The presented row revoked while we rotated (a session invalidation's
	// revoke ran between our checks and our insert): the new row would have
	// escaped it. Either that revoke ran before this read (seen here) or it
	// runs after our insert and revokes the new row itself.
	if presentedRowRevoked(db, row.ID) {
		if err := revokeRow(db, newRow.ID); err != nil {
			return "", err
		}
		return "", ErrRefreshTokenRevoked
	}
	if err := recheckInvalidation(db, row.UserID, row.IssuedAt, newRow); err != nil {
		return "", err
	}
	if err := db.Model(&user_models.RefreshToken{}).Where("id = ?", row.ID).
		Update("replaced_by_id", newRow.ID).Error; err != nil && logger != nil {
		logger.Warn("failed to record refresh token replacement chain", zap.Error(err))
	}

	accessToken, err = auth.GenerateToken(claims.Username, int64(row.UserID))
	if err != nil {
		return "", err
	}
	auth.SetRefreshTokenCookie(rawToken, c)
	return accessToken, nil
}

// liveUser loads a user that still exists (soft-deleted rows are skipped).
func liveUser(db *gorm.DB, userID uint) (user_models.User, bool) {
	var user user_models.User
	err := db.Select("id", "user_name", "refresh_token").Take(&user, userID).Error
	return user, err == nil
}

// rotateLegacyToken upgrades a pre-rotation token only while it is still the
// one stored on the user: RevokeUserSessions clears that column, and a
// deleted user has no row.
func rotateLegacyToken(db *gorm.DB, claims auth.RefreshTokenClaims, presentedToken string, c *gin.Context, logger *zap.Logger) (string, error) {
	// A legacy token has no issue time: the reference is when this rotation
	// read it (an invalidation committed after that clears the column, which
	// a concurrent read may not have seen).
	start := time.Now()
	user, ok := liveUser(db, claims.UserID)
	if !ok || user.RefreshToken == "" || user.RefreshToken != presentedToken {
		return "", ErrRefreshTokenRevoked
	}
	// Single use: the stored legacy token is spent once it joins a family.
	// Conditional, so a concurrent rotation or RevokeUserSessions (which
	// clears the column) wins over this one.
	res := db.Model(&user_models.User{}).Where("id = ? AND refresh_token = ?", user.ID, presentedToken).
		Update("refresh_token", "")
	if res.Error != nil {
		return "", res.Error
	}
	if res.RowsAffected == 0 {
		return "", ErrRefreshTokenRevoked
	}
	newFamilyID := uuid.NewString()
	rawToken, newRow, err := createRefreshTokenRow(db, claims.UserID, claims.Username, newFamilyID, c)
	if err != nil {
		return "", err
	}
	if err := recheckInvalidation(db, claims.UserID, start, newRow); err != nil {
		return "", err
	}
	accessToken, err := auth.GenerateToken(claims.Username, int64(claims.UserID))
	if err != nil {
		return "", err
	}
	auth.SetRefreshTokenCookie(rawToken, c)
	if logger != nil {
		logger.Info("legacy refresh token upgraded to rotation family",
			zap.Uint("user_id", claims.UserID), zap.String("family_id", newFamilyID))
	}
	return accessToken, nil
}

// sessionsInvalidatedSince reports whether the user's sessions were
// invalidated (password_changed_at: password change/reset, 2FA enabled or
// reset) after t. A failed read counts as not invalidated, like liveUser's
// callers treat a live user.
func sessionsInvalidatedSince(db *gorm.DB, userID uint, t time.Time) bool {
	var rows []struct{ PasswordChangedAt *time.Time }
	if err := db.Model(&user_models.User{}).Select("password_changed_at").
		Where("id = ?", userID).Limit(1).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return false
	}
	return rows[0].PasswordChangedAt != nil && rows[0].PasswordChangedAt.After(t)
}

// recheckInvalidation closes the race between a rotation and a concurrent
// invalidation (EnableTwoFactor, ResetTwoFactor, password change), which
// stamp password_changed_at and then revoke every family after committing.
// Checked after the new row is inserted: either the invalidation committed
// before this read (seen here: the new row is revoked and the rotation
// fails) or after it, and then its revoke, which runs after its commit,
// finds the new row already inserted.
func recheckInvalidation(db *gorm.DB, userID uint, issuedAt time.Time, newRow *user_models.RefreshToken) error {
	if !sessionsInvalidatedSince(db, userID, issuedAt) {
		return nil
	}
	if err := revokeRow(db, newRow.ID); err != nil {
		return err
	}
	return ErrRefreshTokenRevoked
}

// presentedRowRevoked re-reads whether a refresh-token row is revoked.
func presentedRowRevoked(db *gorm.DB, rowID uint) bool {
	var rows []struct{ RevokedAt *time.Time }
	if err := db.Model(&user_models.RefreshToken{}).Select("revoked_at").
		Where("id = ?", rowID).Limit(1).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return false
	}
	return rows[0].RevokedAt != nil
}

func revokeRow(db *gorm.DB, rowID uint) error {
	return db.Model(&user_models.RefreshToken{}).Where("id = ?", rowID).
		Update("revoked_at", time.Now()).Error
}

func revokeAndLogReuse(db *gorm.DB, row user_models.RefreshToken, c *gin.Context, logger *zap.Logger, reason string) {
	_ = RevokeFamily(db, row.FamilyID)
	auth.ClearRefreshTokenCookie(c)
	if logger != nil {
		logger.Warn("refresh token reuse detected, family revoked",
			zap.String("reason", reason),
			zap.Uint("user_id", row.UserID),
			zap.String("family_id", row.FamilyID),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}
