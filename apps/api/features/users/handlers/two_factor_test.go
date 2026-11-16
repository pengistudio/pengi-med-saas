package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// tfNow is the fixed clock of these tests: the middle of a TOTP step.
var tfNow = time.Unix(1_800_000_015, 0)

func setupTwoFactor(t *testing.T) (*gorm.DB, *UserHandler) {
	t.Helper()
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{}, &user_models.RecoveryCode{})
	h := newLockoutUserHandler(db)
	h.now = func() time.Time { return tfNow }
	return db, h
}

// enableTwoFactor turns 2FA on for user directly and returns the secret and
// a recovery-code set.
func enableTwoFactor(t *testing.T, db *gorm.DB, user user_models.User) (string, []string) {
	t.Helper()
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := sealTOTPSecret(user.ID, secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumns(map[string]any{
		"two_factor_enabled": true,
		"two_factor_secret":  sealed,
	}).Error; err != nil {
		t.Fatal(err)
	}
	codes, err := replaceRecoveryCodes(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return secret, codes
}

func totpAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := auth.TOTPCode(secret, auth.TOTPStep(at))
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// call runs a handler with a JSON body, as user userID when non-zero, and
// returns the response and the Set-Cookie headers written.
func call(t *testing.T, handler func(*gin.Context) envelope.Response, userID uint, payload any) (envelope.Response, []string) {
	t.Helper()
	body, _ := json.Marshal(payload)
	c, w := testutils.NewGinContext(1, int64(userID))
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if userID != 0 {
		c.Set("user_id", int64(userID))
	}
	resp := handler(c)
	return resp, w.Header().Values("Set-Cookie")
}

func dataMap(t *testing.T, resp envelope.Response) map[string]any {
	t.Helper()
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("data is not an object: %s", raw)
	}
	return out
}

func hasRefreshCookie(cookies []string) bool {
	for _, c := range cookies {
		if strings.HasPrefix(c, "refresh_token=") && !strings.HasPrefix(c, "refresh_token=;") {
			return true
		}
	}
	return false
}

func mfaTokenFor(t *testing.T, h *UserHandler, userName, password string) string {
	t.Helper()
	resp, cookies := call(t, h.Login, 0, user_dto.LoginDTO{UserName: userName, Password: password})
	if resp.Code != http.StatusOK {
		t.Fatalf("login: %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	if data["two_factor_required"] != true {
		t.Fatalf("expected a 2FA challenge, got %v", data)
	}
	if hasRefreshCookie(cookies) {
		t.Fatal("refresh cookie set before the second factor")
	}
	token, _ := data["mfa_token"].(string)
	return token
}

func login2FA(t *testing.T, h *UserHandler, mfaToken, code string) (envelope.Response, []string) {
	t.Helper()
	return call(t, h.LoginTwoFactor, 0, map[string]string{"mfa_token": mfaToken, "code": code})
}

func assertCode(t *testing.T, resp envelope.Response, status int, code core_errors.AppError) {
	t.Helper()
	if resp.Code != status || errorCode(resp) != code.ErrorCode {
		t.Fatalf("expected %d %s, got %d %q (%s)", status, code.ErrorCode, resp.Code, errorCode(resp), resp.Message)
	}
}

// A user without 2FA gets exactly today's answer: three keys and the cookie.
func TestLogin_WithoutTwoFactorUnchanged(t *testing.T) {
	db, h := setupTwoFactor(t)
	createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")

	resp, cookies := call(t, h.Login, 0, user_dto.LoginDTO{UserName: "dra.lopez", Password: "password123"})
	if resp.Code != http.StatusOK || resp.Message != "login.successful" {
		t.Fatalf("login: %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	if len(data) != 3 || data["token"] == nil || data["exchange_token"] == nil || data["user_id"] == nil {
		t.Fatalf("unexpected login data: %v", data)
	}
	if !hasRefreshCookie(cookies) {
		t.Fatal("expected a refresh cookie")
	}
}

func TestLogin_TwoFactorChallengeThenCode(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)

	resp, cookies := call(t, h.Login, 0, user_dto.LoginDTO{UserName: "dra.lopez", Password: "password123"})
	data := dataMap(t, resp)
	if resp.Code != http.StatusOK || len(data) != 2 || data["two_factor_required"] != true || data["mfa_token"] == "" {
		t.Fatalf("expected only the 2FA challenge, got %d %v", resp.Code, data)
	}
	if hasRefreshCookie(cookies) {
		t.Fatal("refresh cookie set before the second factor")
	}
	var rows int64
	db.Model(&user_models.RefreshToken{}).Count(&rows)
	if rows != 0 {
		t.Fatalf("a refresh family was issued before the second factor: %d rows", rows)
	}

	code := totpAt(t, secret, tfNow)
	resp, cookies = login2FA(t, h, data["mfa_token"].(string), code)
	if resp.Code != http.StatusOK || resp.Message != "login.successful" {
		t.Fatalf("2FA login: %d %q", resp.Code, resp.Message)
	}
	got := dataMap(t, resp)
	if len(got) != 3 || got["token"] == nil || got["exchange_token"] == nil {
		t.Fatalf("2FA login data: %v", got)
	}
	if !hasRefreshCookie(cookies) {
		t.Fatal("expected the refresh cookie after the second factor")
	}

	// The same code doesn't work twice (replay), even with a fresh mfa token.
	resp, _ = login2FA(t, h, mfaTokenFor(t, h, "dra.lopez", "password123"), code)
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidTwoFactorCode)
}

// Only mfa tokens open /auth/login/2fa: not access, exchange or refresh tokens.
func TestLoginTwoFactor_RejectsOtherTokens(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)

	access, _ := auth.GenerateToken(user.UserName, int64(user.ID))
	exchange, _ := auth.GenerateExchangeToken(user.UserName, int64(user.ID))
	refresh, _, _ := auth.GenerateRefreshToken(user.UserName, int64(user.ID), "fam")
	reset, _ := auth.GeneratePasswordResetToken(user.ID, user.Password, time.Hour)
	for name, token := range map[string]string{"access": access, "exchange": exchange, "refresh": refresh, "reset": reset, "garbage": "x.y.z"} {
		resp, cookies := login2FA(t, h, token, totpAt(t, secret, tfNow))
		assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidMFAToken)
		if hasRefreshCookie(cookies) {
			t.Fatalf("%s token: refresh cookie set", name)
		}
	}

	// Expired mfa token.
	token := mfaTokenFor(t, h, "dra.lopez", "password123")
	h.now = func() time.Time { return tfNow.Add(auth.MFATokenTTL + time.Second) }
	resp, _ := login2FA(t, h, token, totpAt(t, secret, tfNow.Add(auth.MFATokenTTL+time.Second)))
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidMFAToken)
}

// The refresh endpoint doesn't mint a session from an mfa token (or an
// access token) presented as the refresh cookie.
func TestRefresh_RejectsMFAToken(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	enableTwoFactor(t, db, user)
	mfa := mfaTokenFor(t, h, "dra.lopez", "password123")
	access, _ := auth.GenerateToken(user.UserName, int64(user.ID))

	for name, token := range map[string]string{"mfa": mfa, "access": access} {
		c, w := testutils.NewGinContext(1, 0)
		c.Request = httptest.NewRequest("POST", "/auth/refresh", nil)
		c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: token})
		resp := h.RefreshAuthToken(c)
		if resp.Code == http.StatusOK || hasRefreshCookie(w.Header().Values("Set-Cookie")) {
			t.Fatalf("%s token as refresh cookie: %d", name, resp.Code)
		}
	}
}

// A password change after the password step voids the mfa token.
func TestLoginTwoFactor_PasswordChangeVoidsToken(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)
	token := mfaTokenFor(t, h, "dra.lopez", "password123")

	hashed, _ := auth.HashPassword("another-password-1")
	if err := storeNewPassword(db, user.ID, hashed); err != nil {
		t.Fatal(err)
	}
	resp, _ := login2FA(t, h, token, totpAt(t, secret, tfNow))
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidMFAToken)
}

// The password step of a 2FA user doesn't clear the counter: an attacker who
// knows the password can't get fresh code guesses by logging in again.
func TestLockout_PasswordDoesNotResetCounterWithTwoFactor(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)

	// maxAttempts = 3: password (1), wrong code (2), password again (3) -> lock.
	token := mfaTokenFor(t, h, "dra.lopez", "password123")
	if attempts, _ := lockState(t, db, user.ID); attempts != 1 {
		t.Fatalf("after the password step: attempts=%d, want 1", attempts)
	}
	resp, _ := login2FA(t, h, token, "000000")
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidTwoFactorCode)
	mfaTokenFor(t, h, "dra.lopez", "password123")

	// Locked: neither the right password nor the right code goes through.
	assertLocked(t, login(t, h, "dra.lopez", "password123"))
	resp, _ = login2FA(t, h, token, totpAt(t, secret, tfNow))
	assertLocked(t, resp)

	// Once the lock expires, a correct code clears the counter.
	expireLock(t, db, user.ID)
	token = mfaTokenFor(t, h, "dra.lopez", "password123")
	if resp, _ := login2FA(t, h, token, totpAt(t, secret, tfNow)); resp.Code != http.StatusOK {
		t.Fatalf("2FA login after the lock: %d %q", resp.Code, resp.Message)
	}
	if attempts, lockedUntil := lockState(t, db, user.ID); attempts != 0 || lockedUntil != nil {
		t.Fatalf("a correct code must clear the counter: attempts=%d locked_until=%v", attempts, lockedUntil)
	}
}

// A password change or reset (mailbox) doesn't clear the code-guess counter
// of a 2FA user either.
func TestLockout_PasswordResetKeepsCounterWithTwoFactor(t *testing.T) {
	db, _ := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	enableTwoFactor(t, db, user)
	db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumn("failed_login_attempts", 2)

	hashed, _ := auth.HashPassword("another-password-1")
	if err := storeNewPassword(db, user.ID, hashed); err != nil {
		t.Fatal(err)
	}
	if attempts, _ := lockState(t, db, user.ID); attempts != 2 {
		t.Fatalf("password reset cleared the counter of a 2FA user: attempts=%d", attempts)
	}
}

func TestTOTP_ReplayAndSkew(t *testing.T) {
	db, _ := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)
	reload := func() user_models.User {
		var u user_models.User
		db.First(&u, user.ID)
		return u
	}
	accept := func(code string) bool {
		t.Helper()
		ok, err := acceptTOTP(db, reload(), code, tfNow)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	step := time.Duration(auth.TOTPPeriod) * time.Second

	if accept(totpAt(t, secret, tfNow.Add(-2*step))) {
		t.Fatal("code two steps old accepted")
	}
	if !accept(totpAt(t, secret, tfNow.Add(-step))) {
		t.Fatal("code one step old (skew) refused")
	}
	if accept(totpAt(t, secret, tfNow.Add(-step))) {
		t.Fatal("same code accepted twice")
	}
	if !accept(totpAt(t, secret, tfNow)) {
		t.Fatal("current code refused after an older one")
	}
	if accept(totpAt(t, secret, tfNow.Add(-step))) {
		t.Fatal("code older than the last accepted one accepted")
	}
	if !accept(totpAt(t, secret, tfNow.Add(step))) {
		t.Fatal("code one step ahead (skew) refused")
	}
}

// Two concurrent requests with the same TOTP code: exactly one wins.
func TestTOTP_ConcurrentReplay(t *testing.T) {
	db, _ := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)
	var u user_models.User
	db.First(&u, user.ID)
	code := totpAt(t, secret, tfNow)

	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if accepted, err := acceptTOTP(db, u, code, tfNow); err == nil && accepted {
				atomic.AddInt64(&ok, 1)
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("expected exactly one acceptance, got %d", ok)
	}
}

func TestRecoveryCode_LoginSingleUseAndConcurrent(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	_, codes := enableTwoFactor(t, db, user)
	if len(codes) != recoveryCodeCount {
		t.Fatalf("expected %d codes, got %d", recoveryCodeCount, len(codes))
	}

	// Stored hashed, never in clear.
	var rows []user_models.RecoveryCode
	db.Where("user_id = ?", user.ID).Find(&rows)
	for _, r := range rows {
		for _, c := range codes {
			if strings.Contains(r.CodeHash, normalizeRecoveryCode(c)) {
				t.Fatal("recovery code stored in clear")
			}
		}
	}

	// Lowercase and without dashes still works; then it's spent.
	typed := strings.ToLower(strings.ReplaceAll(codes[0], "-", ""))
	if resp, cookies := login2FA(t, h, mfaTokenFor(t, h, "dra.lopez", "password123"), typed); resp.Code != http.StatusOK || !hasRefreshCookie(cookies) {
		t.Fatalf("recovery code login: %d %q", resp.Code, resp.Message)
	}
	resp, _ := login2FA(t, h, mfaTokenFor(t, h, "dra.lopez", "password123"), codes[0])
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthInvalidTwoFactorCode)

	// Concurrent use of one code: exactly one succeeds.
	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if used, err := consumeRecoveryCode(db, user.ID, codes[1], tfNow); err == nil && used {
				atomic.AddInt64(&ok, 1)
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("expected exactly one use of the recovery code, got %d", ok)
	}

	// Another user's code doesn't work.
	other := createResetUser(t, db, "dr.perez", "perez@example.com", "password123")
	if used, _ := consumeRecoveryCode(db, other.ID, codes[2], tfNow); used {
		t.Fatal("recovery code used by another user")
	}
}

// Setup -> enable -> status, with wrong password/code answered 422.
func TestTwoFactor_Enrollment(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	if err := IssueRefreshFamily(db, user.ID, user.UserName, tfNow, postJSON(t, nil)); err != nil {
		t.Fatal(err)
	}

	// Enable before setup.
	resp, _ := call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": "123456"})
	assertCode(t, resp, http.StatusConflict, core_errors.ErrAuthTwoFactorNotSetUp)

	resp, _ = call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "wrong"})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthWrongCurrentPassword)

	// Setup twice: the second secret replaces the first.
	resp, _ = call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "password123"})
	first := dataMap(t, resp)["secret"].(string)
	resp, _ = call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "password123"})
	if resp.Code != http.StatusOK {
		t.Fatalf("setup: %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	secret := data["secret"].(string)
	if secret == first || !strings.HasPrefix(data["otpauth_uri"].(string), "otpauth://totp/Gentoo:dra.lopez?") {
		t.Fatalf("unexpected setup data: %v", data)
	}

	// Pending: still off, login still password-only, secret sealed with the
	// user's ID as AAD.
	var u user_models.User
	db.First(&u, user.ID)
	if u.TwoFactorEnabled || bytes.Contains(u.TwoFactorSecret, []byte(secret)) {
		t.Fatal("2FA on before confirming, or secret stored in clear")
	}
	if _, err := openTOTPSecret(user.ID+1, u.TwoFactorSecret); err == nil {
		t.Fatal("secret opens with another user's AAD")
	}

	resp, _ = call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": totpAt(t, first, tfNow)})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthInvalidTwoFactorCode)

	resp, cookies := call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": totpAt(t, secret, tfNow)})
	if resp.Code != http.StatusOK {
		t.Fatalf("enable: %d %q", resp.Code, resp.Message)
	}
	data = dataMap(t, resp)
	if codes, _ := data["recovery_codes"].([]any); len(codes) != recoveryCodeCount {
		t.Fatalf("expected %d recovery codes, got %v", recoveryCodeCount, data["recovery_codes"])
	}
	newAccess, _ := data["token"].(string)
	if newAccess == "" || !hasRefreshCookie(cookies) {
		t.Fatal("the enabling session must get a fresh session")
	}

	// Other sessions end: refresh families revoked, older access tokens
	// rejected; the fresh one is accepted.
	db.First(&u, user.ID)
	if !u.TwoFactorEnabled || u.PasswordChangedAt == nil {
		t.Fatalf("2FA not enabled or sessions not invalidated: %+v", u)
	}
	var live int64
	db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&live)
	if live != 1 {
		t.Fatalf("expected only the new refresh family alive, got %d", live)
	}
	// AuthMiddleware rejects access tokens issued before PasswordChangedAt
	// (covered by the middleware tests); it was stamped now, and the fresh
	// token must not fall before it.
	if time.Since(*u.PasswordChangedAt) > time.Minute {
		t.Fatalf("password_changed_at not stamped on enable: %v", u.PasswordChangedAt)
	}
	newClaims, _ := auth.ParseToken(newAccess)
	if auth.IssuedBefore(newClaims, *u.PasswordChangedAt) {
		t.Fatal("the fresh access token is rejected")
	}

	resp, _ = call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "password123"})
	assertCode(t, resp, http.StatusConflict, core_errors.ErrAuthTwoFactorAlreadyEnabled)

	resp, _ = call(t, h.GetTwoFactorStatus, user.ID, nil)
	status := dataMap(t, resp)
	if status["enabled"] != true || status["recovery_codes_remaining"] != float64(recoveryCodeCount) {
		t.Fatalf("status: %v", status)
	}
}

func TestTwoFactor_DisableAndRegenerate(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, codes := enableTwoFactor(t, db, user)

	resp, _ := call(t, h.RegenerateRecoveryCodes, user.ID, map[string]string{"current_password": "wrong", "code": totpAt(t, secret, tfNow)})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthWrongCurrentPassword)
	resp, _ = call(t, h.RegenerateRecoveryCodes, user.ID, map[string]string{"current_password": "password123", "code": "000000"})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthInvalidTwoFactorCode)

	expireLock(t, db, user.ID)
	clearLockout(db, user.ID)
	resp, _ = call(t, h.RegenerateRecoveryCodes, user.ID, map[string]string{"current_password": "password123", "code": totpAt(t, secret, tfNow)})
	if resp.Code != http.StatusOK {
		t.Fatalf("regenerate: %d %q", resp.Code, resp.Message)
	}
	fresh, _ := dataMap(t, resp)["recovery_codes"].([]any)
	if len(fresh) != recoveryCodeCount {
		t.Fatalf("expected %d new codes, got %d", recoveryCodeCount, len(fresh))
	}
	if used, _ := consumeRecoveryCode(db, user.ID, codes[0], tfNow); used {
		t.Fatal("an old recovery code still works after regenerating")
	}

	// Disable with a recovery code.
	resp, _ = call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": fresh[0].(string)})
	if resp.Code != http.StatusOK {
		t.Fatalf("disable: %d %q", resp.Code, resp.Message)
	}
	var u user_models.User
	db.First(&u, user.ID)
	var left int64
	db.Model(&user_models.RecoveryCode{}).Where("user_id = ?", user.ID).Count(&left)
	if u.TwoFactorEnabled || len(u.TwoFactorSecret) != 0 || left != 0 {
		t.Fatalf("2FA not fully disabled: enabled=%v secret=%d codes=%d", u.TwoFactorEnabled, len(u.TwoFactorSecret), left)
	}
	resp, _ = call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": "123456"})
	assertCode(t, resp, http.StatusConflict, core_errors.ErrAuthTwoFactorNotSetUp)

	// Back to password-only login.
	if resp, cookies := call(t, h.Login, 0, user_dto.LoginDTO{UserName: "dra.lopez", Password: "password123"}); resp.Code != http.StatusOK || !hasRefreshCookie(cookies) {
		t.Fatalf("login after disabling: %d", resp.Code)
	}
}

// Wrong codes on the signed-in endpoints count toward the same lockout.
func TestTwoFactor_SignedInCodeGuessesLock(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	enableTwoFactor(t, db, user)

	// Each request: correct password (+1, not cleared) and a wrong code (+1).
	resp, _ := call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": "000000"})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthInvalidTwoFactorCode)
	resp, _ = call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": "000000"})
	assertLocked(t, resp)
}

// Joining a company with an existing 2FA account never hands out a session
// on the password alone.
func TestJoinCompany_TwoFactorUserGetsChallenge(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{}, &user_models.RecoveryCode{},
		&user_models.Role{}, &user_models.Environment{}, &company_models.Company{})
	h := newLockoutUserHandler(db)
	h.now = func() time.Time { return tfNow }
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	secret, _ := enableTwoFactor(t, db, user)
	db.Create(&user_models.Role{Role: "admin"})
	company := company_models.Company{LegalName: "Clinica", TradeName: "Clinica", TenantID: 1}
	db.Create(&company)
	signup, _ := auth.GenerateCompanySignupToken(company.ID, 0)

	resp, cookies := call(t, h.JoinCompanyWithExistingAccount, 0, map[string]string{
		"token": signup, "email": "lopez@example.com", "password": "password123",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("join: %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	if data["two_factor_required"] != true || data["token"] != nil || hasRefreshCookie(cookies) {
		t.Fatalf("join handed out a session without the second factor: %v", data)
	}
	if resp, cookies := login2FA(t, h, data["mfa_token"].(string), totpAt(t, secret, tfNow)); resp.Code != http.StatusOK || !hasRefreshCookie(cookies) {
		t.Fatalf("2FA after join: %d %q", resp.Code, resp.Message)
	}
}

func TestResetTwoFactor_ClearsEverything(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	enableTwoFactor(t, db, user)
	if err := IssueRefreshFamily(db, user.ID, user.UserName, tfNow, postJSON(t, nil)); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumns(map[string]any{"failed_login_attempts": 2, "locked_until": future})

	if err := ResetTwoFactor(db, user.ID); err != nil {
		t.Fatal(err)
	}
	var u user_models.User
	db.First(&u, user.ID)
	var codes, live int64
	db.Model(&user_models.RecoveryCode{}).Where("user_id = ?", user.ID).Count(&codes)
	db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&live)
	if u.TwoFactorEnabled || len(u.TwoFactorSecret) != 0 || codes != 0 || live != 0 ||
		u.PasswordChangedAt == nil || u.FailedLoginAttempts != 0 || u.LockedUntil != nil {
		t.Fatalf("reset incomplete: %+v codes=%d live=%d", u, codes, live)
	}
	if resp := login(t, h, "dra.lopez", "password123"); resp.Code != http.StatusOK || dataMap(t, resp)["token"] == nil {
		t.Fatalf("password-only login after reset: %d", resp.Code)
	}
}

// startSetup runs SetupTwoFactor for user and returns the pending secret.
func startSetup(t *testing.T, h *UserHandler, user user_models.User) string {
	t.Helper()
	resp, _ := call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "password123"})
	if resp.Code != http.StatusOK {
		t.Fatalf("setup: %d %q", resp.Code, resp.Message)
	}
	return dataMap(t, resp)["secret"].(string)
}

// A refresh token issued before 2FA was enabled never rotates afterwards,
// even if its row escaped the revoke (as a row inserted by a racing rotation
// would).
func TestRotate_RefusesTokenIssuedBeforeEnable(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	before := loginAndGetCookie(t, h, "dra.lopez")
	secret := startSetup(t, h, user)

	if resp, _ := call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": totpAt(t, secret, tfNow)}); resp.Code != http.StatusOK {
		t.Fatalf("enable: %d %q", resp.Code, resp.Message)
	}
	// Un-revoke every row: only the issued-at check is left.
	db.Model(&user_models.RefreshToken{}).Where("user_id = ?", user.ID).Update("revoked_at", nil)

	c, w := ginContextWithRefreshCookie(before)
	if _, err := RotateRefreshToken(db, before, c, nil); err != ErrRefreshTokenRevoked {
		t.Fatalf("pre-2FA refresh token rotated: err=%v", err)
	}
	if hasRefreshCookie(w.Header().Values("Set-Cookie")) {
		t.Fatal("refresh cookie set for a pre-2FA token")
	}
}

// Rotations racing EnableTwoFactor: once everything settles, the only live
// refresh session is the one Enable handed to its caller, and no pre-2FA
// token rotates.
func TestRotate_ConcurrentWithEnableLeavesNoPre2FASession(t *testing.T) {
	db, h := setupTwoFactor(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	const n = 6
	cookies := make([]string, n)
	for i := range cookies {
		cookies[i] = loginAndGetCookie(t, h, "dra.lopez")
	}
	secret := startSetup(t, h, user)

	var mu sync.Mutex
	latest := map[int]string{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, cookie := range cookies {
		wg.Add(1)
		go func(i int, cookie string) {
			defer wg.Done()
			<-start
			// Each session keeps rotating while Enable runs.
			for k := 0; k < 5; k++ {
				c, w := ginContextWithRefreshCookie(cookie)
				if _, err := RotateRefreshToken(db, cookie, c, nil); err != nil {
					break
				}
				for _, ck := range w.Result().Cookies() {
					if ck.Name == "refresh_token" && ck.Value != "" {
						cookie = ck.Value
					}
				}
			}
			mu.Lock()
			latest[i] = cookie
			mu.Unlock()
		}(i, cookie)
	}
	var enableCode int
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		resp, _ := call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": totpAt(t, secret, tfNow)})
		enableCode = resp.Code
	}()
	close(start)
	wg.Wait()

	if enableCode != http.StatusOK {
		t.Fatalf("enable: %d", enableCode)
	}
	var live int64
	db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&live)
	if live != 1 {
		t.Fatalf("expected only the session Enable issued alive, got %d", live)
	}
	for _, cookie := range latest {
		c, _ := ginContextWithRefreshCookie(cookie)
		if _, err := RotateRefreshToken(db, cookie, c, nil); err == nil {
			t.Fatal("a pre-2FA refresh token still rotates")
		}
	}
}
