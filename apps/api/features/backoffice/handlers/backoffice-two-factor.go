package backoffice_handlers

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
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Mandatory second factor for backoffice admins (TOTP + recovery codes, and
// passkeys in backoffice-passkey.go), mirroring the clinic's optional 2FA
// (features/users/handlers/two-factor.go) on the admins' own tables.
//
// Login: the password alone never opens a session. An admin with a factor
// gets an mfa token for POST /backoffice/auth/login/2fa (or signs in with a
// passkey); one without gets an enrollment token and must enroll a factor in
// that same login (/backoffice/auth/enroll/*), which then opens the session.
// There is no way around it: BackofficeAuthMiddleware and refresh refuse
// admins without a factor. Secrets, URIs and codes are never logged.

const (
	adminRecoveryCodeCount = 10
	adminRecoveryCodeBytes = 10 // 80 bits, 16 base32 characters
)

var adminRecoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// adminTwoFactorBox is the cipher for admin TOTP secrets: the same key as the
// clinic users' (TWO_FACTOR_ENCRYPTION_KEY, else secretbox.FromEnv), with an
// AAD of its own.
func adminTwoFactorBox() (*secretbox.Box, error) {
	if config.Get().TwoFactorEncryptionKey != "" {
		return secretbox.FromEnvVar("TWO_FACTOR_ENCRYPTION_KEY")
	}
	return secretbox.FromEnv()
}

// adminSecretAAD binds a sealed secret to its admin: copied to another row,
// or to a clinic user with the same ID, it doesn't open.
func adminSecretAAD(id uint) []byte {
	return []byte("backoffice-2fa:" + strconv.FormatUint(uint64(id), 10))
}

func sealAdminTOTPSecret(id uint, secret string) ([]byte, error) {
	box, err := adminTwoFactorBox()
	if err != nil {
		return nil, err
	}
	return box.SealBytesAAD([]byte(secret), adminSecretAAD(id))
}

func openAdminTOTPSecret(id uint, sealed []byte) (string, error) {
	box, err := adminTwoFactorBox()
	if err != nil {
		return "", err
	}
	plain, err := box.OpenBytesAAD(sealed, adminSecretAAD(id))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func normalizeAdminRecoveryCode(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(code))
}

func hashAdminRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(normalizeAdminRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}

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

// replaceAdminRecoveryCodes deletes admin id's recovery codes and stores a
// new set; returns the codes to show once.
func replaceAdminRecoveryCodes(tx *gorm.DB, id uint) ([]string, error) {
	codes := make([]string, 0, adminRecoveryCodeCount)
	rows := make([]backoffice_models.BackofficeRecoveryCode, 0, adminRecoveryCodeCount)
	for i := 0; i < adminRecoveryCodeCount; i++ {
		raw := make([]byte, adminRecoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		s := adminRecoveryEncoding.EncodeToString(raw)
		code := s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
		codes = append(codes, code)
		rows = append(rows, backoffice_models.BackofficeRecoveryCode{BackofficeUserID: id, CodeHash: hashAdminRecoveryCode(code)})
	}
	if err := tx.Where("backoffice_user_id = ?", id).Delete(&backoffice_models.BackofficeRecoveryCode{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// acceptAdminTOTP checks code against the admin's (pending or active) secret
// and atomically records the matched step: each code works once.
func acceptAdminTOTP(db *gorm.DB, user backoffice_models.BackofficeUser, code string, now time.Time) (bool, error) {
	if len(user.TwoFactorSecret) == 0 {
		return false, nil
	}
	secret, err := openAdminTOTPSecret(user.ID, user.TwoFactorSecret)
	if err != nil {
		return false, fmt.Errorf("open totp secret: %w", err)
	}
	step, ok := auth.MatchTOTP(secret, code, now)
	if !ok {
		return false, nil
	}
	res := db.Model(&backoffice_models.BackofficeUser{}).
		Where("id = ? AND two_factor_last_step < ?", user.ID, step).
		UpdateColumn("two_factor_last_step", step)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// consumeAdminRecoveryCode spends one unused recovery code (one conditional
// UPDATE: two concurrent requests can't both use it).
func consumeAdminRecoveryCode(db *gorm.DB, id uint, code string, now time.Time) (bool, error) {
	if normalizeAdminRecoveryCode(code) == "" {
		return false, nil
	}
	res := db.Model(&backoffice_models.BackofficeRecoveryCode{}).
		Where("backoffice_user_id = ? AND code_hash = ? AND used_at IS NULL", id, hashAdminRecoveryCode(code)).
		UpdateColumn("used_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// acceptAdminSecondFactor accepts a TOTP code (6 digits) of an admin with
// TOTP on, or a recovery code in any other shape.
func acceptAdminSecondFactor(db *gorm.DB, user backoffice_models.BackofficeUser, code string, now time.Time) (bool, error) {
	if !user.TwoFactorEnabled {
		return false, nil
	}
	code = strings.Join(strings.Fields(code), "")
	if isTOTPCode(code) {
		return acceptAdminTOTP(db, user, code, now)
	}
	return consumeAdminRecoveryCode(db, user.ID, code, now)
}

func invalidMFATokenResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnauthorized, "auth.login.two_factor_invalid_token", core_errors.ErrAuthInvalidMFAToken)
}

// Wrong password or code on signed-in and enrollment endpoints is 422, never
// 401: the web reads 401 as an expired session.
func wrongPasswordResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnprocessableEntity, "auth.two_factor.wrong_password", core_errors.ErrAuthWrongCurrentPassword)
}

func wrongCodeResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnprocessableEntity, "auth.two_factor.invalid_code", core_errors.ErrAuthInvalidTwoFactorCode)
}

func twoFactorNotSetUpResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "auth.two_factor.not_set_up", core_errors.ErrAuthTwoFactorNotSetUp)
}

func twoFactorAlreadyEnabledResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "auth.two_factor.already_enabled", core_errors.ErrAuthTwoFactorAlreadyEnabled)
}

func lastFactorResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "backoffice.two_factor.last_factor", core_errors.ErrBackofficeLastFactor)
}

func invalidRequestResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
}

// checkCode verifies a second factor of user under the lockout (a correct one
// clears the counter); wrong is the answer to a wrong code.
func (h *BackofficeUserHandler) checkCode(user backoffice_models.BackofficeUser, wrong envelope.Response, verify func() (bool, error)) *envelope.Response {
	return h.lockoutResponse(h.lockout.check(h.db, h.logger, user.ID, true, verify), user.ID, wrong)
}

// checkCurrentPassword verifies the signed-in admin's password under the
// lockout (a correct one doesn't clear the counter).
func (h *BackofficeUserHandler) checkCurrentPassword(user backoffice_models.BackofficeUser, password string) *envelope.Response {
	err := h.lockout.check(h.db, h.logger, user.ID, false, func() (bool, error) {
		return auth.CompareHashAndPassword(user.Password, password), nil
	})
	return h.lockoutResponse(err, user.ID, wrongPasswordResponse())
}

// LoginTwoFactor: POST /backoffice/auth/login/2fa. mfa token + TOTP or
// recovery code -> session.
func (h *BackofficeUserHandler) LoginTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		MFAToken string `json:"mfa_token" binding:"required"`
		Code     string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	claims, err := auth.ParseBackofficeMFAToken(req.MFAToken, auth.BackofficeMFAPurposeLogin, h.now())
	if err != nil {
		return invalidMFATokenResponse()
	}
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return invalidMFATokenResponse()
	}
	// Password changed or TOTP turned off/reset since the password step.
	if !user.TwoFactorEnabled || !claims.MatchesPassword(user.Password) {
		return invalidMFATokenResponse()
	}
	wrong := envelope.ErrorResponse(http.StatusUnauthorized, "auth.two_factor.invalid_code", core_errors.ErrAuthInvalidTwoFactorCode)
	if errResp := h.checkCode(user, wrong, func() (bool, error) {
		return acceptAdminSecondFactor(h.db, user, req.Code, h.now())
	}); errResp != nil {
		return *errResp
	}
	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	h.logger.Info("Backoffice user logged in successfully", zap.String("username", user.UserName), zap.Bool("two_factor", true))
	return envelope.SuccessResponse(data, "login.successful")
}

// enrollee loads the admin of an enrollment token: valid token, same
// password, and still no second factor (one enrollment per login).
func (h *BackofficeUserHandler) enrollee(token string) (backoffice_models.BackofficeUser, *envelope.Response) {
	var user backoffice_models.BackofficeUser
	claims, err := auth.ParseBackofficeMFAToken(token, auth.BackofficeMFAPurposeEnroll, h.now())
	if err != nil {
		resp := invalidMFATokenResponse()
		return user, &resp
	}
	if err := h.db.First(&user, claims.UserID).Error; err != nil || !claims.MatchesPassword(user.Password) {
		resp := invalidMFATokenResponse()
		return user, &resp
	}
	enrolled, err := backoffice_models.HasSecondFactor(h.db, user.ID)
	if err != nil {
		h.logger.Error("Backoffice enrollment: failed to check factors", zap.Uint("user_id", user.ID), zap.Error(err))
		resp := internalResponse()
		return user, &resp
	}
	if enrolled {
		resp := envelope.ErrorResponse(http.StatusConflict, "backoffice.auth.already_enrolled", core_errors.ErrBackofficeAlreadyEnrolled)
		return user, &resp
	}
	if user.LockedUntil != nil && user.LockedUntil.After(h.now()) {
		resp := accountLockedResponse()
		return user, &resp
	}
	return user, nil
}

// startTOTPSetup stores a new pending secret for user (replacing an earlier
// pending one) and returns what the authenticator app needs.
func (h *BackofficeUserHandler) startTOTPSetup(user backoffice_models.BackofficeUser) (gin.H, *envelope.Response) {
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		h.logger.Error("Backoffice 2FA setup: failed to generate secret", zap.Uint("user_id", user.ID), zap.Error(err))
		resp := internalResponse()
		return nil, &resp
	}
	sealed, err := sealAdminTOTPSecret(user.ID, secret)
	if err != nil {
		h.logger.Error("Backoffice 2FA setup: failed to seal secret", zap.Uint("user_id", user.ID), zap.Error(err))
		resp := internalResponse()
		return nil, &resp
	}
	res := h.db.Model(&backoffice_models.BackofficeUser{}).
		Where("id = ? AND two_factor_enabled = ?", user.ID, false).
		UpdateColumn("two_factor_secret", sealed)
	if res.Error != nil {
		h.logger.Error("Backoffice 2FA setup: failed to store secret", zap.Uint("user_id", user.ID), zap.Error(res.Error))
		resp := internalResponse()
		return nil, &resp
	}
	if res.RowsAffected == 0 {
		resp := twoFactorAlreadyEnabledResponse()
		return nil, &resp
	}
	h.logger.Info("Backoffice 2FA setup started", zap.Uint("user_id", user.ID))
	return gin.H{
		"secret":      secret,
		"otpauth_uri": auth.TOTPURI(secret, "backoffice:"+user.UserName),
	}, nil
}

// confirmTOTP checks a code for user's pending secret and turns TOTP on
// with a fresh set of recovery codes.
func (h *BackofficeUserHandler) confirmTOTP(user backoffice_models.BackofficeUser, code string) ([]string, *envelope.Response) {
	if user.TwoFactorEnabled {
		resp := twoFactorAlreadyEnabledResponse()
		return nil, &resp
	}
	if len(user.TwoFactorSecret) == 0 {
		resp := twoFactorNotSetUpResponse()
		return nil, &resp
	}
	code = strings.Join(strings.Fields(code), "")
	if errResp := h.checkCode(user, wrongCodeResponse(), func() (bool, error) {
		if !isTOTPCode(code) {
			return false, nil
		}
		return acceptAdminTOTP(h.db, user, code, h.now())
	}); errResp != nil {
		return nil, errResp
	}
	var codes []string
	errEnabled := errors.New("already enabled")
	err := h.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&backoffice_models.BackofficeUser{}).
			Where("id = ? AND two_factor_enabled = ? AND two_factor_secret = ?", user.ID, false, user.TwoFactorSecret).
			UpdateColumns(map[string]any{"two_factor_enabled": true, "two_factor_enabled_at": h.now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errEnabled
		}
		var err error
		codes, err = replaceAdminRecoveryCodes(tx, user.ID)
		return err
	})
	if errors.Is(err, errEnabled) {
		resp := twoFactorAlreadyEnabledResponse()
		return nil, &resp
	}
	if err != nil {
		h.logger.Error("Backoffice 2FA enable failed", zap.Uint("user_id", user.ID), zap.Error(err))
		resp := internalResponse()
		return nil, &resp
	}
	h.logger.Info("Backoffice 2FA enabled", zap.Uint("user_id", user.ID))
	return codes, nil
}

type enrollmentRequest struct {
	EnrollmentToken string `json:"enrollment_token" binding:"required"`
}

// EnrollTOTPBegin: POST /backoffice/auth/enroll/totp/begin.
func (h *BackofficeUserHandler) EnrollTOTPBegin(c *gin.Context) envelope.Response {
	var req enrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.enrollee(req.EnrollmentToken)
	if errResp != nil {
		return *errResp
	}
	data, errResp := h.startTOTPSetup(user)
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(data, "auth.two_factor.setup.success")
}

// EnrollTOTPFinish: POST /backoffice/auth/enroll/totp/finish. A valid code
// turns TOTP on and opens the session; the answer carries the recovery
// codes (shown once).
func (h *BackofficeUserHandler) EnrollTOTPFinish(c *gin.Context) envelope.Response {
	var req struct {
		EnrollmentToken string `json:"enrollment_token" binding:"required"`
		Code            string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.enrollee(req.EnrollmentToken)
	if errResp != nil {
		return *errResp
	}
	codes, errResp := h.confirmTOTP(user, req.Code)
	if errResp != nil {
		return *errResp
	}
	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	data["recovery_codes"] = codes
	h.logger.Info("Backoffice user enrolled TOTP and logged in", zap.String("username", user.UserName))
	return envelope.SuccessResponse(data, "backoffice.auth.enroll.success")
}

// currentAdmin loads the signed-in admin (BackofficeAuthMiddleware set user_id).
func (h *BackofficeUserHandler) currentAdmin(c *gin.Context) (backoffice_models.BackofficeUser, *envelope.Response) {
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, c.GetInt64("user_id")).Error; err != nil {
		resp := envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
		return user, &resp
	}
	return user, nil
}

func (h *BackofficeUserHandler) passkeyCount(id uint) (int64, error) {
	var n int64
	err := h.db.Model(&backoffice_models.BackofficePasskey{}).Where("backoffice_user_id = ?", id).Count(&n).Error
	return n, err
}

// GetTwoFactorStatus: GET /backoffice/me/2fa.
func (h *BackofficeUserHandler) GetTwoFactorStatus(c *gin.Context) envelope.Response {
	user, errResp := h.currentAdmin(c)
	if errResp != nil {
		return *errResp
	}
	var remaining int64
	if user.TwoFactorEnabled {
		if err := h.db.Model(&backoffice_models.BackofficeRecoveryCode{}).
			Where("backoffice_user_id = ? AND used_at IS NULL", user.ID).Count(&remaining).Error; err != nil {
			h.logger.Error("Backoffice 2FA status: failed to count recovery codes", zap.Uint("user_id", user.ID), zap.Error(err))
			return internalResponse()
		}
	}
	passkeys, err := h.passkeyCount(user.ID)
	if err != nil {
		h.logger.Error("Backoffice 2FA status: failed to count passkeys", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(gin.H{
		"enabled":                  user.TwoFactorEnabled,
		"recovery_codes_remaining": remaining,
		"passkey_count":            passkeys,
		"passkeys_available":       passkeysAvailable(),
	}, "auth.two_factor.status.success")
}

// SetupTwoFactor: POST /backoffice/me/2fa/setup (an admin enrolled with a
// passkey adding TOTP). Needs the current password.
func (h *BackofficeUserHandler) SetupTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.currentAdmin(c)
	if errResp != nil {
		return *errResp
	}
	if user.TwoFactorEnabled {
		return twoFactorAlreadyEnabledResponse()
	}
	if errResp := h.checkCurrentPassword(user, req.CurrentPassword); errResp != nil {
		return *errResp
	}
	data, errResp := h.startTOTPSetup(user)
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(data, "auth.two_factor.setup.success")
}

// EnableTwoFactor: POST /backoffice/me/2fa/enable. A valid code for the
// pending secret turns TOTP on; returns the recovery codes (shown once).
func (h *BackofficeUserHandler) EnableTwoFactor(c *gin.Context) envelope.Response {
	var req struct {
		Code string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.currentAdmin(c)
	if errResp != nil {
		return *errResp
	}
	codes, errResp := h.confirmTOTP(user, req.Code)
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(gin.H{"recovery_codes": codes}, "auth.two_factor.enable.success")
}

// checkPasswordAndCode is the proof disable and regenerate ask for: the
// current password and a TOTP or recovery code.
func (h *BackofficeUserHandler) checkPasswordAndCode(c *gin.Context) (backoffice_models.BackofficeUser, *envelope.Response) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		Code            string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := invalidRequestResponse()
		return backoffice_models.BackofficeUser{}, &resp
	}
	user, errResp := h.currentAdmin(c)
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
	if errResp := h.checkCode(user, wrongCodeResponse(), func() (bool, error) {
		return acceptAdminSecondFactor(h.db, user, req.Code, h.now())
	}); errResp != nil {
		return user, errResp
	}
	return user, nil
}

// DisableTwoFactor: POST /backoffice/me/2fa/disable. Only while the admin
// keeps a passkey: a second factor is mandatory.
func (h *BackofficeUserHandler) DisableTwoFactor(c *gin.Context) envelope.Response {
	user, errResp := h.checkPasswordAndCode(c)
	if errResp != nil {
		return *errResp
	}
	errLast := errors.New("last factor")
	err := h.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&backoffice_models.BackofficeUser{}).
			Where("id = ? AND EXISTS (SELECT 1 FROM backoffice_user_passkeys p WHERE p.backoffice_user_id = ?)", user.ID, user.ID).
			UpdateColumns(map[string]any{"two_factor_enabled": false, "two_factor_secret": nil, "two_factor_enabled_at": nil})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errLast
		}
		return tx.Where("backoffice_user_id = ?", user.ID).Delete(&backoffice_models.BackofficeRecoveryCode{}).Error
	})
	if errors.Is(err, errLast) {
		return lastFactorResponse()
	}
	if err != nil {
		h.logger.Error("Backoffice 2FA disable failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	h.logger.Info("Backoffice 2FA disabled", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(nil, "auth.two_factor.disable.success")
}

// RegenerateRecoveryCodes: POST /backoffice/me/2fa/recovery-codes. The old
// set stops working.
func (h *BackofficeUserHandler) RegenerateRecoveryCodes(c *gin.Context) envelope.Response {
	user, errResp := h.checkPasswordAndCode(c)
	if errResp != nil {
		return *errResp
	}
	var codes []string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		codes, err = replaceAdminRecoveryCodes(tx, user.ID)
		return err
	}); err != nil {
		h.logger.Error("Backoffice recovery codes regeneration failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	h.logger.Info("Backoffice recovery codes regenerated", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(gin.H{"recovery_codes": codes}, "auth.two_factor.recovery_codes.success")
}

// targetAdmin loads the admin of :id for a reset by another admin; an admin
// can't reset their own factors (that would turn a stolen session into a
// way around the second factor).
func (h *BackofficeUserHandler) targetAdmin(c *gin.Context) (backoffice_models.BackofficeUser, *envelope.Response) {
	var user backoffice_models.BackofficeUser
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp := invalidRequestResponse()
		return user, &resp
	}
	if int64(id) == c.GetInt64("user_id") {
		resp := envelope.ErrorResponse(http.StatusForbidden, "backoffice.users.cannot_reset_self", core_errors.ErrBackofficeCannotResetSelf)
		return user, &resp
	}
	if err := h.db.First(&user, id).Error; err != nil {
		resp := envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
		return user, &resp
	}
	return user, nil
}

// ResetBackofficeUserTwoFactor: POST /backoffice/users/:id/2fa/reset. For an
// admin who lost the authenticator and recovery codes: TOTP off, secret and
// recovery codes deleted, lockout lifted, session ended. Without a passkey
// left, the admin enrolls a factor again at the next login.
func (h *BackofficeUserHandler) ResetBackofficeUserTwoFactor(c *gin.Context) envelope.Response {
	user, errResp := h.targetAdmin(c)
	if errResp != nil {
		return *errResp
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", user.ID).UpdateColumns(map[string]any{
			"two_factor_enabled":    false,
			"two_factor_secret":     nil,
			"two_factor_enabled_at": nil,
			"failed_login_attempts": 0,
			"locked_until":          nil,
			"refresh_token":         "",
			"refresh_family":        "",
			"token_version":         backoffice_models.BumpTokenVersion,
		}).Error; err != nil {
			return err
		}
		return tx.Where("backoffice_user_id = ?", user.ID).Delete(&backoffice_models.BackofficeRecoveryCode{}).Error
	}); err != nil {
		h.logger.Error("Backoffice 2FA reset failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	h.logger.Info("Backoffice 2FA reset by another admin", zap.Uint("user_id", user.ID), zap.Int64("by", c.GetInt64("user_id")))
	return envelope.SuccessResponse(nil, "backoffice.users.two_factor_reset.success")
}

// ResetBackofficeUserPasskeys: POST /backoffice/users/:id/passkeys/reset.
// Deletes every passkey of the admin and their pending ceremonies, lifts the
// lockout and ends the session.
func (h *BackofficeUserHandler) ResetBackofficeUserPasskeys(c *gin.Context) envelope.Response {
	user, errResp := h.targetAdmin(c)
	if errResp != nil {
		return *errResp
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("backoffice_user_id = ?", user.ID).Delete(&backoffice_models.BackofficePasskey{}).Error; err != nil {
			return err
		}
		if err := tx.Where("backoffice_user_id = ?", user.ID).Delete(&backoffice_models.BackofficeWebAuthnChallenge{}).Error; err != nil {
			return err
		}
		return tx.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", user.ID).UpdateColumns(map[string]any{
			"failed_login_attempts": 0,
			"locked_until":          nil,
			"refresh_token":         "",
			"refresh_family":        "",
			"token_version":         backoffice_models.BumpTokenVersion,
		}).Error
	}); err != nil {
		h.logger.Error("Backoffice passkeys reset failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	h.logger.Info("Backoffice passkeys reset by another admin", zap.Uint("user_id", user.ID), zap.Int64("by", c.GetInt64("user_id")))
	return envelope.SuccessResponse(nil, "backoffice.users.passkeys_reset.success")
}
