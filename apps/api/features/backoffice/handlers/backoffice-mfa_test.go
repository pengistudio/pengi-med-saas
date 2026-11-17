package backoffice_handlers

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	backoffice_middleware "pengi-med-saas/features/backoffice/middleware"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	"pengi-med-saas/testutils"
)

const (
	mfaPass       = "s3cret-pass"
	testBORPID    = "localhost"
	testBOOrigin  = "http://localhost:3001"
	mfaTestAdmin  = "ana-admin"
	mfaOtherAdmin = "luis-admin"
)

var mfaNow = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

func setupMFA(t *testing.T) (*gorm.DB, *BackofficeUserHandler) {
	t.Helper()
	t.Setenv("BACKOFFICE_URL", testBOOrigin)
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{}, &backoffice_models.BackofficeRecoveryCode{},
		&backoffice_models.BackofficePasskey{}, &backoffice_models.BackofficeWebAuthnChallenge{})
	h := NewBackofficeUserHandler(db, zap.NewNop())
	h.now = func() time.Time { return mfaNow }
	return db, h
}

func createAdmin(t *testing.T, db *gorm.DB, userName string) backoffice_models.BackofficeUser {
	t.Helper()
	admin := backoffice_models.BackofficeUser{Name: userName, UserName: userName, Password: mfaPass}
	if err := admin.Save(db); err != nil {
		t.Fatal(err)
	}
	return admin
}

// enableTOTP turns TOTP on for admin directly in the database.
func enableTOTP(t *testing.T, db *gorm.DB, admin backoffice_models.BackofficeUser) string {
	t.Helper()
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := sealAdminTOTPSecret(admin.ID, secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", admin.ID).
		UpdateColumns(map[string]any{"two_factor_secret": sealed, "two_factor_enabled": true}).Error; err != nil {
		t.Fatal(err)
	}
	return secret
}

func totpAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := auth.TOTPCode(secret, auth.TOTPStep(at))
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// call runs handle with body as JSON, as admin userID (0: anonymous), with
// the given bearer token in the context and route params.
func call(t *testing.T, handle func(*gin.Context) envelope.Response, userID uint, token string, body any, params ...gin.Param) (envelope.Response, *httptest.ResponseRecorder) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	c, w := testutils.NewGinContext(0, 0)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	if userID != 0 {
		c.Set("user_id", int64(userID))
	}
	if token != "" {
		c.Set("auth_token", token)
	}
	c.Params = params
	return handle(c), w
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

func login(t *testing.T, h *BackofficeUserHandler, userName string) map[string]any {
	t.Helper()
	resp, w := call(t, h.Login, 0, "", map[string]string{"user_name": userName, "password": mfaPass})
	if resp.Code != http.StatusOK {
		t.Fatalf("login = %d %q", resp.Code, resp.Message)
	}
	if refreshCookie(w) != "" {
		t.Fatal("the password step set a refresh cookie")
	}
	data := dataMap(t, resp)
	if data["token"] != nil {
		t.Fatalf("the password step returned a session: %v", data)
	}
	return data
}

// signIn opens a full session for an admin with TOTP on (password + code at
// the handler's clock) and returns the refresh cookie.
func signIn(t *testing.T, h *BackofficeUserHandler, userName, secret string) string {
	t.Helper()
	data := login(t, h, userName)
	resp, w := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": data["mfa_token"].(string), "code": totpAt(t, secret, h.now())})
	if resp.Code != http.StatusOK {
		t.Fatalf("login/2fa = %d %q", resp.Code, resp.Message)
	}
	cookie := refreshCookie(w)
	if cookie == "" {
		t.Fatal("login/2fa set no refresh cookie")
	}
	return cookie
}

func advance(h *BackofficeUserHandler, d time.Duration) {
	at := h.now().Add(d)
	h.now = func() time.Time { return at }
}

// The first login of an admin without a second factor gives no session, only
// an enrollment token; enrolling TOTP with it opens the session (with the
// recovery codes), and the token can't enroll twice. The next login asks for
// the second factor.
func TestLogin_ForcesEnrollmentWithoutSecondFactor(t *testing.T) {
	db, h := setupMFA(t)
	createAdmin(t, db, mfaTestAdmin)

	data := login(t, h, mfaTestAdmin)
	token, _ := data["enrollment_token"].(string)
	if data["enrollment_required"] != true || token == "" || data["passkeys_available"] != true {
		t.Fatalf("expected an enrollment answer, got %v", data)
	}

	begin, _ := call(t, h.EnrollTOTPBegin, 0, "", map[string]string{"enrollment_token": token})
	if begin.Code != http.StatusOK {
		t.Fatalf("enroll begin = %d %q", begin.Code, begin.Message)
	}
	secret, _ := dataMap(t, begin)["secret"].(string)

	wrong, _ := call(t, h.EnrollTOTPFinish, 0, "", map[string]string{"enrollment_token": token, "code": "000000"})
	if wrong.Code != http.StatusUnprocessableEntity {
		t.Fatalf("enroll with a wrong code = %d, want 422", wrong.Code)
	}
	fin, w := call(t, h.EnrollTOTPFinish, 0, "", map[string]string{"enrollment_token": token, "code": totpAt(t, secret, mfaNow)})
	if fin.Code != http.StatusOK || fin.Message != "backoffice.auth.enroll.success" {
		t.Fatalf("enroll finish = %d %q", fin.Code, fin.Message)
	}
	got := dataMap(t, fin)
	if got["token"] == nil || refreshCookie(w) == "" {
		t.Fatalf("enrollment opened no session: %v", got)
	}
	if codes, _ := got["recovery_codes"].([]any); len(codes) != adminRecoveryCodeCount {
		t.Fatalf("recovery codes: %v", got["recovery_codes"])
	}

	if again, _ := call(t, h.EnrollTOTPBegin, 0, "", map[string]string{"enrollment_token": token}); again.Code != http.StatusConflict {
		t.Fatalf("second enrollment with the same token = %d, want 409", again.Code)
	}
	if next := login(t, h, mfaTestAdmin); next["two_factor_required"] != true || next["mfa_token"] == nil || next["totp"] != true {
		t.Fatalf("login after enrolling: %v", next)
	}
}

// An enrollment token (or mfa token) opens no backoffice endpoint, and an
// access token of an admin without a second factor neither.
func TestEnrollmentToken_CannotCallBackofficeEndpoints(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	enroll := login(t, h, mfaTestAdmin)["enrollment_token"].(string)

	r := gin.New()
	r.GET("/backoffice/users", backoffice_middleware.BackofficeAuthMiddleware(db), func(c *gin.Context) { c.Status(http.StatusOK) })
	get := func(token string) int {
		req := httptest.NewRequest("GET", "/backoffice/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := get(enroll); code != http.StatusUnauthorized {
		t.Errorf("enrollment token on a backoffice endpoint = %d, want 401", code)
	}
	access, _ := auth.GenerateBackofficeToken(admin.UserName, int64(admin.ID), 0)
	if code := get(access); code != http.StatusUnauthorized {
		t.Errorf("access token of an admin without a second factor = %d, want 401", code)
	}
	// Nor as the login second step.
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": enroll, "code": "123456"}); resp.Code != http.StatusUnauthorized {
		t.Errorf("enrollment token at login/2fa = %d, want 401", resp.Code)
	}

	secret := enableTOTP(t, db, admin)
	mfa := login(t, h, mfaTestAdmin)["mfa_token"].(string)
	if code := get(mfa); code != http.StatusUnauthorized {
		t.Errorf("mfa token on a backoffice endpoint = %d, want 401", code)
	}
	if resp, _ := call(t, h.EnrollTOTPBegin, 0, "", map[string]string{"enrollment_token": mfa}); resp.Code != http.StatusUnauthorized {
		t.Errorf("mfa token as enrollment token = %d, want 401", resp.Code)
	}
	signIn(t, h, mfaTestAdmin, secret)
	if code := get(access); code != http.StatusOK {
		t.Errorf("access token of an enrolled admin = %d, want 200", code)
	}
}

// The second step takes a TOTP code once; wrong codes count towards the
// lockout, which a correct password can't lift.
func TestLoginTwoFactor_TOTPAndLockout(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	secret := enableTOTP(t, db, admin)

	mfa := login(t, h, mfaTestAdmin)["mfa_token"].(string)
	code := totpAt(t, secret, mfaNow)
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": code}); resp.Code != http.StatusOK {
		t.Fatalf("login/2fa = %d %q", resp.Code, resp.Message)
	}
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": code}); resp.Code != http.StatusUnauthorized {
		t.Fatalf("replayed code = %d, want 401", resp.Code)
	}

	for i := 0; i < 4; i++ {
		if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": "000000"}); resp.Code != http.StatusUnauthorized {
			t.Fatalf("wrong code %d = %d, want 401", i+1, resp.Code)
		}
	}
	// The replay above was the first failure: this is the fifth.
	advance(h, time.Minute)
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": totpAt(t, secret, h.now())}); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("correct code while locked = %d, want 429", resp.Code)
	}
	if resp, _ := call(t, h.Login, 0, "", map[string]string{"user_name": mfaTestAdmin, "password": mfaPass}); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("password login while locked = %d, want 429", resp.Code)
	}
}

// A correct current password (clear=false) refunds its own attempt: repeated
// legitimate re-authentications never lock the admin, but earlier wrong
// guesses stay counted.
func TestAdminLockout_CorrectPasswordRefundsItsAttempt(t *testing.T) {
	db, _ := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	l := adminLockout{maxAttempts: 3, duration: time.Minute}
	right := func() (bool, error) { return true, nil }
	wrong := func() (bool, error) { return false, nil }

	for i := 0; i < 10; i++ {
		if err := l.check(db, zap.NewNop(), admin.ID, false, right); err != nil {
			t.Fatalf("correct password %d: %v", i+1, err)
		}
	}
	if err := l.check(db, zap.NewNop(), admin.ID, false, wrong); !errors.Is(err, errAdminWrong) {
		t.Fatalf("wrong password: %v", err)
	}
	// One wrong guess stays counted even after correct checks.
	if err := l.check(db, zap.NewNop(), admin.ID, false, right); err != nil {
		t.Fatalf("correct after wrong: %v", err)
	}
	var got backoffice_models.BackofficeUser
	if err := db.First(&got, admin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.FailedLoginAttempts != 1 {
		t.Fatalf("failed attempts = %d, want 1", got.FailedLoginAttempts)
	}
	// Two more wrong guesses reach the limit and lock.
	_ = l.check(db, zap.NewNop(), admin.ID, false, wrong)
	_ = l.check(db, zap.NewNop(), admin.ID, false, wrong)
	if err := l.check(db, zap.NewNop(), admin.ID, false, right); !errors.Is(err, errAdminLocked) {
		t.Fatalf("after 3 wrong guesses: %v, want locked", err)
	}
}

// A recovery code works once.
func TestLoginTwoFactor_RecoveryCodeSingleUse(t *testing.T) {
	db, h := setupMFA(t)
	createAdmin(t, db, mfaTestAdmin)
	enroll := login(t, h, mfaTestAdmin)["enrollment_token"].(string)
	begin, _ := call(t, h.EnrollTOTPBegin, 0, "", map[string]string{"enrollment_token": enroll})
	secret := dataMap(t, begin)["secret"].(string)
	fin, _ := call(t, h.EnrollTOTPFinish, 0, "", map[string]string{"enrollment_token": enroll, "code": totpAt(t, secret, mfaNow)})
	recovery := dataMap(t, fin)["recovery_codes"].([]any)[0].(string)

	mfa := login(t, h, mfaTestAdmin)["mfa_token"].(string)
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": recovery}); resp.Code != http.StatusOK {
		t.Fatalf("recovery code = %d %q", resp.Code, resp.Message)
	}
	mfa = login(t, h, mfaTestAdmin)["mfa_token"].(string)
	if resp, _ := call(t, h.LoginTwoFactor, 0, "", map[string]string{"mfa_token": mfa, "code": recovery}); resp.Code != http.StatusUnauthorized {
		t.Fatalf("used recovery code again = %d, want 401", resp.Code)
	}
}

// --- passkeys ---

type softAuthenticator struct {
	priv    *ecdsa.PrivateKey
	credID  []byte
	counter uint32
}

func newSoftAuthenticator(t *testing.T) *softAuthenticator {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return &softAuthenticator{priv: priv, credID: id}
}

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rpIDHash() []byte {
	sum := sha256.Sum256([]byte(testBORPID))
	return sum[:]
}

func (a *softAuthenticator) attest(t *testing.T, challenge string) json.RawMessage {
	t.Helper()
	cose, err := cbor.Marshal(map[int]any{
		1: 2, 3: -7, -1: 1,
		-2: a.priv.PublicKey.X.FillBytes(make([]byte, 32)),
		-3: a.priv.PublicKey.Y.FillBytes(make([]byte, 32)),
	})
	if err != nil {
		t.Fatal(err)
	}
	authData := append([]byte{}, rpIDHash()...)
	authData = append(authData, 0x45, 0, 0, 0, 0) // UP | UV | AT, sign count 0
	authData = append(authData, make([]byte, 16)...)
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(a.credID)))
	authData = append(authData, a.credID...)
	authData = append(authData, cose...)
	attObj, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}
	clientData := mustJSON(t, map[string]any{"type": "webauthn.create", "challenge": challenge, "origin": testBOOrigin})
	return mustJSON(t, map[string]any{
		"id": b64u(a.credID), "rawId": b64u(a.credID), "type": "public-key",
		"response": map[string]any{"clientDataJSON": b64u(clientData), "attestationObject": b64u(attObj)},
	})
}

func (a *softAuthenticator) assert(t *testing.T, challenge string, adminID uint) json.RawMessage {
	t.Helper()
	a.counter++
	authData := append([]byte{}, rpIDHash()...)
	authData = append(authData, 0x05) // UP | UV
	authData = binary.BigEndian.AppendUint32(authData, a.counter)
	clientData := mustJSON(t, map[string]any{"type": "webauthn.get", "challenge": challenge, "origin": testBOOrigin})
	cdHash := sha256.Sum256(clientData)
	signed := sha256.Sum256(append(append([]byte{}, authData...), cdHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.priv, signed[:])
	if err != nil {
		t.Fatal(err)
	}
	handle := append([]byte{adminHandlePrefix}, binary.BigEndian.AppendUint64(nil, uint64(adminID))...)
	return mustJSON(t, map[string]any{
		"id": b64u(a.credID), "rawId": b64u(a.credID), "type": "public-key",
		"response": map[string]any{
			"authenticatorData": b64u(authData), "clientDataJSON": b64u(clientData),
			"signature": b64u(sig), "userHandle": b64u(handle),
		},
	})
}

func optionsChallenge(t *testing.T, resp envelope.Response) (string, string) {
	t.Helper()
	data := dataMap(t, resp)
	opts, _ := data["options"].(map[string]any)
	id, _ := data["challenge_id"].(string)
	challenge, _ := opts["challenge"].(string)
	if id == "" || challenge == "" {
		t.Fatalf("missing challenge in %v", data)
	}
	return id, challenge
}

func passkeyLogin(t *testing.T, h *BackofficeUserHandler, a *softAuthenticator, adminID uint) (envelope.Response, *httptest.ResponseRecorder) {
	t.Helper()
	begin, _ := call(t, h.BeginPasskeyLogin, 0, "", map[string]string{})
	if begin.Code != http.StatusOK {
		t.Fatalf("passkey login begin = %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	return call(t, h.FinishPasskeyLogin, 0, "", map[string]any{"challenge_id": id, "credential": a.assert(t, challenge, adminID)})
}

// An admin with TOTP registers a passkey (password + code) and then signs in
// with it alone: no TOTP step.
func TestPasskeyLogin_SkipsTOTP(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	secret := enableTOTP(t, db, admin)

	if resp, _ := call(t, h.BeginPasskeyRegistration, admin.ID, "", map[string]string{"current_password": mfaPass}); resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("registration without the code = %d, want 422", resp.Code)
	}
	advance(h, time.Minute)
	begin, _ := call(t, h.BeginPasskeyRegistration, admin.ID, "", map[string]string{"current_password": mfaPass, "code": totpAt(t, secret, h.now())})
	if begin.Code != http.StatusOK {
		t.Fatalf("registration begin = %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	a := newSoftAuthenticator(t)
	fin, _ := call(t, h.FinishPasskeyRegistration, admin.ID, "", map[string]any{"challenge_id": id, "name": "MacBook", "credential": a.attest(t, challenge)})
	if fin.Code != http.StatusOK {
		t.Fatalf("registration finish = %d %q", fin.Code, fin.Message)
	}

	resp, w := passkeyLogin(t, h, a, admin.ID)
	if resp.Code != http.StatusOK || resp.Message != "login.successful" {
		t.Fatalf("passkey login = %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	if data["token"] == nil || data["two_factor_required"] != nil || refreshCookie(w) == "" {
		t.Fatalf("expected a full session without a TOTP step, got %v", data)
	}
}

// Enrolling a passkey is enough: the session opens, and TOTP can't be
// disabled... there is none; deleting the only passkey is refused.
func TestEnrollPasskey_OpensSessionAndKeepsLastFactor(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	enroll := login(t, h, mfaTestAdmin)["enrollment_token"].(string)

	begin, _ := call(t, h.EnrollPasskeyBegin, 0, "", map[string]string{"enrollment_token": enroll})
	if begin.Code != http.StatusOK {
		t.Fatalf("enroll passkey begin = %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	a := newSoftAuthenticator(t)
	fin, w := call(t, h.EnrollPasskeyFinish, 0, "", map[string]any{
		"enrollment_token": enroll, "challenge_id": id, "name": "YubiKey", "credential": a.attest(t, challenge),
	})
	if fin.Code != http.StatusOK || refreshCookie(w) == "" {
		t.Fatalf("enroll passkey finish = %d %q", fin.Code, fin.Message)
	}
	data := dataMap(t, fin)
	passkeyID := uint(data["passkey"].(map[string]any)["id"].(float64))

	resp, _ := call(t, h.DeletePasskey, admin.ID, "", nil, gin.Param{Key: "id", Value: strconv.Itoa(int(passkeyID))})
	if resp.Code != http.StatusConflict || resp.Message != "backoffice.two_factor.last_factor" {
		t.Fatalf("deleting the only passkey = %d %q, want 409", resp.Code, resp.Message)
	}

	// The enrollment session is fresh: a second passkey needs only the password.
	token := data["token"].(string)
	begin, _ = call(t, h.BeginPasskeyRegistration, admin.ID, token, map[string]string{"current_password": mfaPass})
	if begin.Code != http.StatusOK {
		t.Fatalf("second passkey with a fresh session = %d %q", begin.Code, begin.Message)
	}
	advance(h, 10*time.Minute)
	if resp, _ := call(t, h.BeginPasskeyRegistration, admin.ID, token, map[string]string{"current_password": mfaPass}); resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("second passkey with a stale session = %d, want 422", resp.Code)
	}
}

// Disabling TOTP needs password + code, and is refused while it is the
// admin's only factor.
func TestDisableTwoFactor_RefusedForLastFactor(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	secret := enableTOTP(t, db, admin)
	resp, _ := call(t, h.DisableTwoFactor, admin.ID, "", map[string]string{"current_password": mfaPass, "code": totpAt(t, secret, mfaNow)})
	if resp.Code != http.StatusConflict || resp.Message != "backoffice.two_factor.last_factor" {
		t.Fatalf("disable the only factor = %d %q, want 409", resp.Code, resp.Message)
	}
}

// An admin resets another admin's factors, never their own.
func TestResetFactors_NotOneself(t *testing.T) {
	db, h := setupMFA(t)
	me := createAdmin(t, db, mfaTestAdmin)
	other := createAdmin(t, db, mfaOtherAdmin)
	enableTOTP(t, db, me)
	otherSecret := enableTOTP(t, db, other)
	signIn(t, h, mfaOtherAdmin, otherSecret)

	for _, handle := range []func(*gin.Context) envelope.Response{h.ResetBackofficeUserTwoFactor, h.ResetBackofficeUserPasskeys} {
		resp, _ := call(t, handle, me.ID, "", nil, gin.Param{Key: "id", Value: strconv.Itoa(int(me.ID))})
		if resp.Code != http.StatusForbidden || resp.Message != "backoffice.users.cannot_reset_self" {
			t.Fatalf("self reset = %d %q, want 403", resp.Code, resp.Message)
		}
	}

	resp, _ := call(t, h.ResetBackofficeUserTwoFactor, me.ID, "", nil, gin.Param{Key: "id", Value: strconv.Itoa(int(other.ID))})
	if resp.Code != http.StatusOK {
		t.Fatalf("reset another admin = %d %q", resp.Code, resp.Message)
	}
	var got backoffice_models.BackofficeUser
	db.First(&got, other.ID)
	if got.TwoFactorEnabled || len(got.TwoFactorSecret) != 0 || got.RefreshToken != "" {
		t.Fatalf("reset left 2FA or the session: %+v", got)
	}
	if data := login(t, h, mfaOtherAdmin); data["enrollment_required"] != true {
		t.Fatalf("login after reset should force enrollment: %v", data)
	}
}

// A rotated-out refresh token of the current login presented after the grace
// window revokes the session; a token of an older login is only rejected.
func TestRefresh_ReuseRevokesSession(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	secret := enableTOTP(t, db, admin)

	older := signIn(t, h, mfaTestAdmin, secret)
	advance(h, time.Minute)
	first := signIn(t, h, mfaTestAdmin, secret)

	// A token of the older login: stale, the current session survives.
	if resp, _ := refreshWith(h, older); resp.Code != http.StatusUnauthorized || resp.Message == "auth.refresh.reused" {
		t.Fatalf("older login's token = %d %q, want a plain 401", resp.Code, resp.Message)
	}
	resp, w := refreshWith(h, first)
	second := refreshCookie(w)
	if resp.Code != http.StatusOK || second == "" {
		t.Fatalf("refresh = %d %q", resp.Code, resp.Message)
	}

	h.rotated = newRotationGrace() // past the grace window
	resp, w = refreshWith(h, first)
	if resp.Code != http.StatusUnauthorized || resp.Message != "auth.refresh.reused" {
		t.Fatalf("reused token = %d %q, want 401 auth.refresh.reused", resp.Code, resp.Message)
	}
	var got backoffice_models.BackofficeUser
	db.First(&got, admin.ID)
	if got.RefreshToken != "" {
		t.Fatal("reuse left the session's refresh token stored")
	}
	if resp, _ := refreshWith(h, second); resp.Code != http.StatusUnauthorized {
		t.Fatalf("current token after reuse = %d, want 401", resp.Code)
	}
}

// An admin who lost every factor (reset) gets no session from refresh.
func TestRefresh_RefusedWithoutSecondFactor(t *testing.T) {
	db, h := setupMFA(t)
	admin := createAdmin(t, db, mfaTestAdmin)
	secret := enableTOTP(t, db, admin)
	cookie := signIn(t, h, mfaTestAdmin, secret)
	db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", admin.ID).UpdateColumn("two_factor_enabled", false)
	if resp, _ := refreshWith(h, cookie); resp.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without a second factor = %d, want 401", resp.Code)
	}
}

// Admin passwords: common ones, a repeated character and the user name are
// refused on create and on change.
func TestAdminPassword_RejectsWeakPasswords(t *testing.T) {
	db, h := setupMFA(t)
	for _, pw := range []string{"password123", "1234567890", "QwertyUiop", "aaaaaaaaaaaa", "new-admin-01"} {
		resp, _ := callJSON(t, "POST", "/backoffice/users", `{"user_name": "new-admin-01", "password": "`+pw+`"}`, h.CreateUser)
		if resp.Code != http.StatusBadRequest || resp.Message != "auth.password.too_weak" {
			t.Errorf("CreateUser with %q = %d %q, want 400 auth.password.too_weak", pw, resp.Code, resp.Message)
		}
	}
	admin := createAdmin(t, db, "ana-admin-01")
	c, _ := testutils.NewGinContext(0, 1)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(admin.ID))}}
	c.Request = httptest.NewRequest("PUT", "/backoffice/users", bytes.NewBufferString(`{"password": "ANA-ADMIN-01"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	if resp := h.UpdateUser(c); resp.Code != http.StatusBadRequest || resp.Message != "auth.password.too_weak" {
		t.Errorf("UpdateUser to the user name = %d %q, want 400 auth.password.too_weak", resp.Code, resp.Message)
	}
}
