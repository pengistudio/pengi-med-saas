package user_handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/secretbox"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Two-factor authentication (TOTP + recovery codes), optional per user.
//
// Login: a correct password for a user with 2FA returns an mfa token instead
// of a session (twoFactorChallenge); POST /auth/login/2fa trades it plus a
// code for the session Login would have returned. Secrets, otpauth URIs,
// codes and recovery codes are never logged.

// TwoFactorEncryptionKeyEnv names the optional key for TOTP secrets.
const TwoFactorEncryptionKeyEnv = "TWO_FACTOR_ENCRYPTION_KEY"

// recoveryCodeCount codes are issued per set; each has recoveryCodeBytes of
// randomness (80 bits), shown as 16 base32 characters in groups of four.
const (
	recoveryCodeCount = 10
	recoveryCodeBytes = 10
)

var recoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// twoFactorBox is the cipher for TOTP secrets: TWO_FACTOR_ENCRYPTION_KEY when
// set, otherwise the same fallback as secretbox.FromEnv
// (SIGNATURE_ENCRYPTION_KEY, or a dev key outside release).
func twoFactorBox() (*secretbox.Box, error) {
	if config.Get().TwoFactorEncryptionKey != "" {
		return secretbox.FromEnvVar(TwoFactorEncryptionKeyEnv)
	}
	return secretbox.FromEnv()
}

// secretAAD binds a sealed secret to its user: copied to another row it
// doesn't open.
func secretAAD(userID uint) []byte {
	return []byte("user-2fa:" + strconv.FormatUint(uint64(userID), 10))
}

func sealTOTPSecret(userID uint, secret string) ([]byte, error) {
	box, err := twoFactorBox()
	if err != nil {
		return nil, err
	}
	return box.SealBytesAAD([]byte(secret), secretAAD(userID))
}

func openTOTPSecret(userID uint, sealed []byte) (string, error) {
	box, err := twoFactorBox()
	if err != nil {
		return "", err
	}
	plain, err := box.OpenBytesAAD(sealed, secretAAD(userID))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// normalizeRecoveryCode drops the separators people type and copy.
func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(code)
	return strings.NewReplacer("-", "", " ", "").Replace(code)
}

func hashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(normalizeRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}

// isTOTPCode reports whether code has the shape of a TOTP code (6 digits);
// anything else is tried as a recovery code.
func isTOTPCode(code string) bool {
	if len(code) != auth.TOTPDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// newRecoveryCodes returns a fresh set: the codes to show once, and their rows.
func newRecoveryCodes(userID uint) ([]string, []user_models.RecoveryCode, error) {
	codes := make([]string, 0, recoveryCodeCount)
	rows := make([]user_models.RecoveryCode, 0, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		raw := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		s := recoveryEncoding.EncodeToString(raw) // 16 chars
		code := s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
		codes = append(codes, code)
		rows = append(rows, user_models.RecoveryCode{UserID: userID, CodeHash: hashRecoveryCode(code)})
	}
	return codes, rows, nil
}

// replaceRecoveryCodes deletes userID's codes and stores a new set.
func replaceRecoveryCodes(tx *gorm.DB, userID uint) ([]string, error) {
	codes, rows, err := newRecoveryCodes(userID)
	if err != nil {
		return nil, err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&user_models.RecoveryCode{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// acceptTOTP checks code against the user's secret at now and, atomically,
// records the matched step: a code at or before the last accepted step is
// refused, so a code works once even under concurrent requests.
func acceptTOTP(db *gorm.DB, user user_models.User, code string, now time.Time) (bool, error) {
	if len(user.TwoFactorSecret) == 0 {
		return false, nil
	}
	secret, err := openTOTPSecret(user.ID, user.TwoFactorSecret)
	if err != nil {
		return false, fmt.Errorf("open totp secret: %w", err)
	}
	step, ok := auth.MatchTOTP(secret, code, now)
	if !ok {
		return false, nil
	}
	res := db.Model(&user_models.User{}).
		Where("id = ? AND two_factor_last_step < ?", user.ID, step).
		UpdateColumn("two_factor_last_step", step)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// consumeRecoveryCode spends one unused recovery code of userID. The lookup
// is by hash of the input and the spend is one conditional UPDATE, so two
// concurrent requests can't both use the same code.
func consumeRecoveryCode(db *gorm.DB, userID uint, code string, now time.Time) (bool, error) {
	if normalizeRecoveryCode(code) == "" {
		return false, nil
	}
	res := db.Model(&user_models.RecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hashRecoveryCode(code)).
		UpdateColumn("used_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// acceptSecondFactor accepts a TOTP code or, in any other shape, a recovery
// code (which needs no secret, so it still works if the key changed).
func acceptSecondFactor(db *gorm.DB, user user_models.User, code string, now time.Time) (bool, error) {
	code = strings.Join(strings.Fields(code), "") // apps show "123 456"
	if isTOTPCode(code) {
		return acceptTOTP(db, user, code, now)
	}
	return consumeRecoveryCode(db, user.ID, code, now)
}

// verifyCode runs the second-factor check of user under the lockout: the
// attempt is counted before the code is compared, and a correct code clears
// the counter. Returns nil, errAccountLocked, errWrongPassword (wrong code)
// or an internal error.
func (h *UserHandler) verifyCode(user user_models.User, verify func() (bool, error)) error {
	var verifyErr error
	err := h.lockout.checkSecondFactor(h.db, h.logger, user.ID, func() bool {
		ok, err := verify()
		verifyErr = err
		return ok
	})
	if verifyErr != nil {
		return verifyErr
	}
	return err
}

// twoFactorChallenge answers a correct password of a 2FA user: an mfa token
// for /auth/login/2fa, no session.
func (h *UserHandler) twoFactorChallenge(user user_models.User) envelope.Response {
	token, err := auth.GenerateMFAToken(user.ID, user.Password, h.now())
	if err != nil {
		h.logger.Error("Failed to generate mfa token", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}
	h.logger.Info("Password accepted, second factor required", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(gin.H{
		"two_factor_required": true,
		"mfa_token":           token,
	}, "auth.login.two_factor_required")
}

func invalidMFATokenResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnauthorized, "auth.login.two_factor_invalid_token", core_errors.ErrAuthInvalidMFAToken)
}

// LoginTwoFactor completes a 2FA login: mfa token + TOTP or recovery code ->
// the same session and answer as a successful Login.
func (h *UserHandler) LoginTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		MFAToken string `json:"mfa_token" binding:"required"`
		Code     string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}

	claims, err := auth.ParseMFAToken(req.MFAToken, h.now())
	if err != nil {
		h.logger.Warn("Invalid mfa token")
		return invalidMFATokenResponse()
	}
	var user user_models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return invalidMFATokenResponse()
	}
	// The password changed, 2FA was turned off or reset, or the email is no
	// longer verified since the password step: start over.
	if !user.TwoFactorEnabled || !user.EmailVerified || !claims.MatchesPassword(user.Password) {
		return invalidMFATokenResponse()
	}

	if err := h.verifyCode(user, func() (bool, error) {
		return acceptSecondFactor(h.db, user, req.Code, h.now())
	}); err != nil {
		switch {
		case errors.Is(err, errAccountLocked):
			h.logger.Warn("2FA login refused: account locked", zap.Uint("user_id", user.ID))
			return accountLockedResponse()
		case errors.Is(err, errWrongPassword):
			h.logger.Warn("Failed 2FA login attempt", zap.Uint("user_id", user.ID))
			return envelope.ErrorResponse(http.StatusUnauthorized, "auth.two_factor.invalid_code", core_errors.ErrAuthInvalidTwoFactorCode)
		default:
			h.logger.Error("2FA login: failed to check code", zap.Uint("user_id", user.ID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
	}

	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	h.logger.Info("User logged in successfully", zap.String("username", user.UserName), zap.Bool("two_factor", true))
	return envelope.SuccessResponse(data, "login.successful")
}

// currentUser loads the signed-in user (AuthMiddleware set user_id).
func (h *UserHandler) currentUser(c *gin.Context) (user_models.User, *envelope.Response) {
	var user user_models.User
	if err := h.db.First(&user, c.GetInt64("user_id")).Error; err != nil {
		resp := envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
		return user, &resp
	}
	return user, nil
}

// Wrong password or code on the signed-in endpoints is 422, never 401: the
// web client reads 401 as an expired session (refresh, retry, sign out).
func wrongTwoFactorPasswordResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnprocessableEntity, "auth.two_factor.wrong_password", core_errors.ErrAuthWrongCurrentPassword)
}

func wrongTwoFactorCodeResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnprocessableEntity, "auth.two_factor.invalid_code", core_errors.ErrAuthInvalidTwoFactorCode)
}

func twoFactorNotSetUpResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "auth.two_factor.not_set_up", core_errors.ErrAuthTwoFactorNotSetUp)
}

func twoFactorAlreadyEnabledResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "auth.two_factor.already_enabled", core_errors.ErrAuthTwoFactorAlreadyEnabled)
}

// checkCurrentPassword verifies the signed-in user's password under the
// lockout; nil when correct, else the response to return.
func (h *UserHandler) checkCurrentPassword(user user_models.User, password string) *envelope.Response {
	err := h.lockout.check(h.db, h.logger, user.ID, func() bool {
		return auth.CompareHashAndPassword(user.Password, password)
	})
	var resp envelope.Response
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errAccountLocked):
		resp = accountLockedResponse()
	case errors.Is(err, errWrongPassword):
		resp = wrongTwoFactorPasswordResponse()
	default:
		h.logger.Error("2FA: failed to check password", zap.Uint("user_id", user.ID), zap.Error(err))
		resp = envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return &resp
}

// checkCode verifies a second factor of the signed-in user under the
// lockout; nil when correct, else the response to return.
func (h *UserHandler) checkCode(user user_models.User, verify func() (bool, error)) *envelope.Response {
	err := h.verifyCode(user, verify)
	var resp envelope.Response
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errAccountLocked):
		resp = accountLockedResponse()
	case errors.Is(err, errWrongPassword):
		resp = wrongTwoFactorCodeResponse()
	default:
		h.logger.Error("2FA: failed to check code", zap.Uint("user_id", user.ID), zap.Error(err))
		resp = envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return &resp
}

// GetTwoFactorStatus: GET /users/me/2fa.
func (h *UserHandler) GetTwoFactorStatus(c *gin.Context) envelope.Response {
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	var remaining int64
	if user.TwoFactorEnabled {
		if err := h.db.Model(&user_models.RecoveryCode{}).
			Where("user_id = ? AND used_at IS NULL", user.ID).Count(&remaining).Error; err != nil {
			h.logger.Error("2FA status: failed to count recovery codes", zap.Uint("user_id", user.ID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
	}
	return envelope.SuccessResponse(gin.H{
		"enabled":                  user.TwoFactorEnabled,
		"recovery_codes_remaining": remaining,
	}, "auth.two_factor.status.success")
}

// SetupTwoFactor: POST /users/me/2fa/setup. Stores a new pending secret
// (replacing any earlier pending one); 2FA stays off until EnableTwoFactor.
func (h *UserHandler) SetupTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	if user.TwoFactorEnabled {
		return twoFactorAlreadyEnabledResponse()
	}
	if errResp := h.checkCurrentPassword(user, req.CurrentPassword); errResp != nil {
		return *errResp
	}

	secret, err := auth.NewTOTPSecret()
	if err != nil {
		h.logger.Error("2FA setup: failed to generate secret", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	sealed, err := sealTOTPSecret(user.ID, secret)
	if err != nil {
		h.logger.Error("2FA setup: failed to seal secret", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	res := h.db.Model(&user_models.User{}).
		Where("id = ? AND two_factor_enabled = ?", user.ID, false).
		UpdateColumn("two_factor_secret", sealed)
	if res.Error != nil {
		h.logger.Error("2FA setup: failed to store secret", zap.Uint("user_id", user.ID), zap.Error(res.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if res.RowsAffected == 0 {
		return twoFactorAlreadyEnabledResponse()
	}

	h.logger.Info("2FA setup started", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(gin.H{
		"secret":      secret,
		"otpauth_uri": auth.TOTPURI(secret, user.UserName),
	}, "auth.two_factor.setup.success")
}

// EnableTwoFactor: POST /users/me/2fa/enable. A valid code for the pending
// secret turns 2FA on and returns the recovery codes (shown once).
//
// Every existing session ends (none of them passed the second factor), as
// on a password change: refresh families are revoked and access tokens
// issued before now are rejected. The caller just proved the second factor,
// so it gets a fresh session in the answer (token + exchange_token + a new
// refresh cookie) instead of being signed out.
func (h *UserHandler) EnableTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		Code string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	if user.TwoFactorEnabled {
		return twoFactorAlreadyEnabledResponse()
	}
	if len(user.TwoFactorSecret) == 0 {
		return twoFactorNotSetUpResponse()
	}
	code := strings.Join(strings.Fields(req.Code), "")
	if errResp := h.checkCode(user, func() (bool, error) {
		if !isTOTPCode(code) {
			return false, nil
		}
		return acceptTOTP(h.db, user, code, h.now())
	}); errResp != nil {
		return *errResp
	}

	var codes []string
	errEnabled := errors.New("already enabled")
	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Model(&user_models.User{}).
			// Only the secret the code was checked against.
			Where("id = ? AND two_factor_enabled = ? AND two_factor_secret = ?", user.ID, false, user.TwoFactorSecret).
			UpdateColumns(map[string]any{
				"two_factor_enabled":    true,
				"two_factor_enabled_at": now,
				"password_changed_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errEnabled
		}
		var err error
		if codes, err = replaceRecoveryCodes(tx, user.ID); err != nil {
			return err
		}
		return RevokeUserSessions(tx, user.ID)
	})
	if err == nil {
		// Again after commit: a rotation that inserted its row while the
		// transaction ran is revoked here, and one that inserts later sees
		// the committed password_changed_at (RotateRefreshToken re-checks it).
		err = RevokeUserSessions(h.db, user.ID)
	}
	auth_middleware.ForgetUser(user.ID)
	if errors.Is(err, errEnabled) {
		return twoFactorAlreadyEnabledResponse()
	}
	if err != nil {
		h.logger.Error("2FA enable failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	data["recovery_codes"] = codes
	h.alertSecurity(user, eventTwoFactorEnabled, c)
	h.logger.Info("2FA enabled", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(data, "auth.two_factor.enable.success")
}

type passwordAndCode struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	Code            string `json:"code" binding:"required,max=64"`
}

// checkPasswordAndCode is the proof disable and regenerate ask for: the
// current password and a TOTP or recovery code, each counted by the lockout.
func (h *UserHandler) checkPasswordAndCode(c *gin.Context) (user_models.User, *envelope.Response) {
	var req passwordAndCode
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
		return user_models.User{}, &resp
	}
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return user, errResp
	}
	if !user.TwoFactorEnabled {
		resp := twoFactorNotSetUpResponse()
		return user, &resp
	}
	if errResp := h.checkCurrentPassword(user, req.CurrentPassword); errResp != nil {
		return user, errResp
	}
	if errResp := h.checkCode(user, func() (bool, error) {
		return acceptSecondFactor(h.db, user, req.Code, h.now())
	}); errResp != nil {
		return user, errResp
	}
	return user, nil
}

// DisableTwoFactor: POST /users/me/2fa/disable.
func (h *UserHandler) DisableTwoFactor(c *gin.Context) envelope.Response {
	user, errResp := h.checkPasswordAndCode(c)
	if errResp != nil {
		return *errResp
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		return clearTwoFactor(tx, user.ID, nil)
	}); err != nil {
		h.logger.Error("2FA disable failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	h.alertSecurity(user, eventTwoFactorDisabled, c)
	h.logger.Info("2FA disabled", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(nil, "auth.two_factor.disable.success")
}

// RegenerateRecoveryCodes: POST /users/me/2fa/recovery-codes. The old set
// stops working.
func (h *UserHandler) RegenerateRecoveryCodes(c *gin.Context) envelope.Response {
	user, errResp := h.checkPasswordAndCode(c)
	if errResp != nil {
		return *errResp
	}
	var codes []string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		codes, err = replaceRecoveryCodes(tx, user.ID)
		return err
	}); err != nil {
		h.logger.Error("2FA recovery codes regeneration failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	h.alertSecurity(user, eventRecoveryCodesRegenerated, c)
	h.logger.Info("2FA recovery codes regenerated", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(gin.H{"recovery_codes": codes}, "auth.two_factor.recovery_codes.success")
}

// clearTwoFactor turns 2FA off for userID and deletes its secret and
// recovery codes. extra columns are written in the same UPDATE.
func clearTwoFactor(tx *gorm.DB, userID uint, extra map[string]any) error {
	cols := map[string]any{
		"two_factor_enabled":    false,
		"two_factor_secret":     nil,
		"two_factor_enabled_at": nil,
	}
	for k, v := range extra {
		cols[k] = v
	}
	if err := tx.Model(&user_models.User{}).Where("id = ?", userID).UpdateColumns(cols).Error; err != nil {
		return err
	}
	return tx.Where("user_id = ?", userID).Delete(&user_models.RecoveryCode{}).Error
}

// ResetTwoFactor is the backoffice reset for a user who lost their
// authenticator and recovery codes: 2FA off, secret and recovery codes
// deleted, lockout lifted, and every session ended (refresh families revoked,
// access tokens issued before now rejected). The user signs in again with
// the password alone and may enroll again.
func ResetTwoFactor(db *gorm.DB, userID uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := clearTwoFactor(tx, userID, map[string]any{
			"password_changed_at":   time.Now(),
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}); err != nil {
			return err
		}
		return RevokeUserSessions(tx, userID)
	})
	if err == nil {
		// Again after commit, for a concurrent rotation (see EnableTwoFactor).
		err = RevokeUserSessions(db, userID)
	}
	auth_middleware.ForgetUser(userID)
	return err
}
