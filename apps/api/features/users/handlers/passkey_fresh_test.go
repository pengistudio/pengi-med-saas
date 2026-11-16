package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	user_dto "pengi-med-saas/features/users/dto"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
)

// Registering a passkey without the password, right after a login.

// session is what a sign-in or a refresh hands the client.
type session struct {
	access  string
	refresh string
}

func refreshCookieValue(t *testing.T, cookies []string) string {
	t.Helper()
	for _, c := range cookies {
		if v, ok := strings.CutPrefix(c, "refresh_token="); ok && !strings.HasPrefix(v, ";") {
			value, _, _ := strings.Cut(v, ";")
			return value
		}
	}
	t.Fatalf("no refresh cookie in %v", cookies)
	return ""
}

func sessionFrom(t *testing.T, resp envelope.Response, cookies []string) session {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("sign-in: %d %q", resp.Code, resp.Message)
	}
	access, _ := dataMap(t, resp)["token"].(string)
	if access == "" {
		t.Fatalf("no access token in %v", resp.Data)
	}
	return session{access: access, refresh: refreshCookieValue(t, cookies)}
}

func passwordLogin(t *testing.T, h *UserHandler) session {
	t.Helper()
	resp, cookies := call(t, h.Login, 0, user_dto.LoginDTO{UserName: "dra.lopez", Password: testPasskeyPass})
	return sessionFrom(t, resp, cookies)
}

// refreshSession runs POST /auth/refresh with the session's cookie.
func refreshSession(t *testing.T, h *UserHandler, s session) session {
	t.Helper()
	c, w := testutils.NewGinContext(1, 0)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: s.refresh})
	resp := h.RefreshAuthToken(c)
	return sessionFrom(t, resp, w.Header().Values("Set-Cookie"))
}

// callAs is call with the access token AuthMiddleware leaves in the context.
func callAs(t *testing.T, handler func(*gin.Context) envelope.Response, userID uint, accessToken string, payload any) envelope.Response {
	t.Helper()
	body, _ := json.Marshal(payload)
	c, _ := testutils.NewGinContext(1, int64(userID))
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(userID))
	c.Set("auth_token", accessToken)
	return handler(c)
}

func at(h *UserHandler, d time.Duration) { h.now = func() time.Time { return tfNow.Add(d) } }

func authTimeOf(t *testing.T, token string) (time.Time, bool) {
	t.Helper()
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	return auth.AuthTime(claims)
}

func TestPasskeyFresh_RegistersWithoutPassword(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	s := passwordLogin(t, h)

	at(h, freshSessionWindow-time.Second)
	begin := callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{})
	if begin.Code != http.StatusOK {
		t.Fatalf("begin without password on a fresh session: %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	authn := newSoftAuthenticator(t)
	fin, _ := call(t, h.FinishPasskeyRegistration, user.ID, map[string]any{
		"challenge_id": id, "name": "MacBook", "credential": authn.attest(t, challenge),
	})
	if fin.Code != http.StatusOK {
		t.Fatalf("finish: %d %q", fin.Code, fin.Message)
	}

	// A passkey sign-in is a login too: its session is fresh. An explicit
	// empty password counts as none.
	at(h, time.Minute)
	resp, cookies := loginWithPasskey(t, h, authn, user.ID)
	pk := sessionFrom(t, resp, cookies)
	at(h, 2*time.Minute)
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, pk.access, map[string]string{"current_password": ""})
	if resp.Code != http.StatusOK {
		t.Fatalf("begin after a passkey login: %d %q", resp.Code, resp.Message)
	}
}

func TestPasskeyFresh_StaleSessionNeedsPassword(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	s := passwordLogin(t, h)

	at(h, freshSessionWindow)
	resp := callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)
	if resp.Message != "auth.passkey.reauth_required" {
		t.Fatalf("message: %q", resp.Message)
	}

	// The password still works on any session, and is still checked.
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{"current_password": "wrong"})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthWrongCurrentPassword)
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{"current_password": testPasskeyPass})
	if resp.Code != http.StatusOK {
		t.Fatalf("begin with password on a stale session: %d %q", resp.Code, resp.Message)
	}
}

func TestPasskeyFresh_RefreshKeepsLoginTime(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	s := passwordLogin(t, h)

	// The refreshed token carries the login time, so it is still fresh...
	at(h, time.Minute)
	s = refreshSession(t, h, s)
	if got, ok := authTimeOf(t, s.access); !ok || !got.Equal(tfNow) {
		t.Fatalf("refreshed auth_time = %v (%v), want the login %v", got, ok, tfNow)
	}
	at(h, 2*time.Minute)
	if resp := callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{}); resp.Code != http.StatusOK {
		t.Fatalf("begin with a refreshed token inside the window: %d %q", resp.Code, resp.Message)
	}

	// ...but rotating again past the window doesn't renew it.
	at(h, freshSessionWindow+time.Minute)
	s = refreshSession(t, h, s)
	if got, ok := authTimeOf(t, s.access); !ok || !got.Equal(tfNow) {
		t.Fatalf("second refresh auth_time = %v (%v), want the login %v", got, ok, tfNow)
	}
	resp := callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)
}

func TestPasskeyFresh_TokenWithoutLoginTimeIsNotFresh(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)

	old, err := auth.GenerateToken(user.UserName, int64(user.ID))
	if err != nil {
		t.Fatal(err)
	}
	resp := callAs(t, h.BeginPasskeyRegistration, user.ID, old, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)

	// No token in the context at all (e.g. another auth path).
	resp, _ = call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)

	// A fresh-looking claim on another token type doesn't count either.
	exchange, err := auth.GenerateExchangeToken(user.UserName, int64(user.ID))
	if err != nil {
		t.Fatal(err)
	}
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, exchange, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)
}

func TestPasskeyFresh_TwoFactor(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	secret, _ := enableTwoFactor(t, db, user)

	mfa := mfaTokenFor(t, h, "dra.lopez", testPasskeyPass)
	resp, cookies := login2FA(t, h, mfa, totpAt(t, secret, tfNow))
	s := sessionFrom(t, resp, cookies)

	// Fresh: the login just passed password and code, no code asked again.
	at(h, time.Minute)
	if resp := callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{}); resp.Code != http.StatusOK {
		t.Fatalf("begin on a fresh 2FA session: %d %q", resp.Code, resp.Message)
	}

	// Stale: password alone is not enough, the code is still required.
	stale := freshSessionWindow + time.Minute
	at(h, stale)
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthPasskeyReauthRequired)
	// (Success first: for a 2FA user each password attempt counts toward the
	// lockout until a correct code clears it.)
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{
		"current_password": testPasskeyPass, "code": totpAt(t, secret, tfNow.Add(stale)),
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("begin with password + code on a stale session: %d %q", resp.Code, resp.Message)
	}
	resp = callAs(t, h.BeginPasskeyRegistration, user.ID, s.access, map[string]string{"current_password": testPasskeyPass})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthInvalidTwoFactorCode)
}
