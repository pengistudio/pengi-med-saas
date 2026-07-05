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
		return rotateLegacyToken(db, claims, c, logger)
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

	if row.UsedAt != nil {
		revokeAndLogReuse(db, row, c, logger, "replay of an already-rotated token")
		return "", ErrRefreshTokenReused
	}

	// Conditional update: only succeeds if this row is still unused. If
	// RowsAffected is 0, a concurrent request rotated it out from under us
	// between the read above and here — treat identically to reuse.
	result := db.Model(&user_models.RefreshToken{}).
		Where("id = ? AND used_at IS NULL", row.ID).
		Update("used_at", time.Now())
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		revokeAndLogReuse(db, row, c, logger, "concurrent rotation race")
		return "", ErrRefreshTokenReused
	}

	rawToken, newRow, err := createRefreshTokenRow(db, row.UserID, claims.Username, row.FamilyID, c)
	if err != nil {
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

func rotateLegacyToken(db *gorm.DB, claims auth.RefreshTokenClaims, c *gin.Context, logger *zap.Logger) (string, error) {
	newFamilyID := uuid.NewString()
	rawToken, _, err := createRefreshTokenRow(db, claims.UserID, claims.Username, newFamilyID, c)
	if err != nil {
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
