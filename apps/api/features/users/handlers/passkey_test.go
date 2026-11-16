package user_handlers

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/fxamacker/cbor/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	testPasskeyRPID   = "localhost"
	testPasskeyOrigin = "http://localhost:3000"
	testPasskeyPass   = "password123"
)

// softAuthenticator is a software WebAuthn authenticator: ES256, user
// verified, "none" attestation. Enough to run the real ceremonies.
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

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rpIDHash() []byte {
	sum := sha256.Sum256([]byte(testPasskeyRPID))
	return sum[:]
}

func (a *softAuthenticator) coseKey(t *testing.T) []byte {
	t.Helper()
	out, err := cbor.Marshal(map[int]any{
		1:  2,  // kty EC2
		3:  -7, // alg ES256
		-1: 1,  // crv P-256
		-2: a.priv.PublicKey.X.FillBytes(make([]byte, 32)),
		-3: a.priv.PublicKey.Y.FillBytes(make([]byte, 32)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// attest answers navigator.credentials.create for challenge.
func (a *softAuthenticator) attest(t *testing.T, challenge string) json.RawMessage {
	t.Helper()
	authData := append([]byte{}, rpIDHash()...)
	authData = append(authData, 0x45)                // UP | UV | AT
	authData = append(authData, 0, 0, 0, 0)          // sign count
	authData = append(authData, make([]byte, 16)...) // AAGUID
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(a.credID)))
	authData = append(authData, a.credID...)
	authData = append(authData, a.coseKey(t)...)

	attObj, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}
	clientData := mustJSON(t, map[string]any{"type": "webauthn.create", "challenge": challenge, "origin": testPasskeyOrigin})
	return mustJSON(t, map[string]any{
		"id": b64u(a.credID), "rawId": b64u(a.credID), "type": "public-key",
		"response": map[string]any{"clientDataJSON": b64u(clientData), "attestationObject": b64u(attObj)},
	})
}

// assert answers navigator.credentials.get for challenge as userID.
func (a *softAuthenticator) assert(t *testing.T, challenge string, userID uint) json.RawMessage {
	t.Helper()
	a.counter++
	authData := append([]byte{}, rpIDHash()...)
	authData = append(authData, 0x05) // UP | UV
	authData = binary.BigEndian.AppendUint32(authData, a.counter)

	clientData := mustJSON(t, map[string]any{"type": "webauthn.get", "challenge": challenge, "origin": testPasskeyOrigin})
	cdHash := sha256.Sum256(clientData)
	signed := sha256.Sum256(append(append([]byte{}, authData...), cdHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.priv, signed[:])
	if err != nil {
		t.Fatal(err)
	}
	handle := binary.BigEndian.AppendUint64(nil, uint64(userID))
	return mustJSON(t, map[string]any{
		"id": b64u(a.credID), "rawId": b64u(a.credID), "type": "public-key",
		"response": map[string]any{
			"authenticatorData": b64u(authData),
			"clientDataJSON":    b64u(clientData),
			"signature":         b64u(sig),
			"userHandle":        b64u(handle),
		},
	})
}

func setupPasskeys(t *testing.T) (*gorm.DB, *UserHandler) {
	t.Helper()
	t.Setenv("APP_URL", testPasskeyOrigin)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{}, &user_models.RecoveryCode{},
		&user_models.Passkey{}, &user_models.WebAuthnChallenge{})
	h := newLockoutUserHandler(db)
	h.now = func() time.Time { return tfNow }
	return db, h
}

func optionsChallenge(t *testing.T, resp envelope.Response) (challengeID, challenge string) {
	t.Helper()
	data := dataMap(t, resp)
	opts, _ := data["options"].(map[string]any)
	challengeID, _ = data["challenge_id"].(string)
	challenge, _ = opts["challenge"].(string)
	if challengeID == "" || challenge == "" {
		t.Fatalf("missing challenge in %v", data)
	}
	return challengeID, challenge
}

// registerPasskey runs a full registration for userID and returns the
// authenticator that now holds the credential.
func registerPasskey(t *testing.T, h *UserHandler, userID uint, name string) *softAuthenticator {
	t.Helper()
	begin, _ := call(t, h.BeginPasskeyRegistration, userID, map[string]string{"current_password": testPasskeyPass})
	if begin.Code != http.StatusOK {
		t.Fatalf("begin registration: %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	auth := newSoftAuthenticator(t)
	fin, _ := call(t, h.FinishPasskeyRegistration, userID, map[string]any{
		"challenge_id": id, "name": name, "credential": auth.attest(t, challenge),
	})
	if fin.Code != http.StatusOK {
		t.Fatalf("finish registration: %d %q", fin.Code, fin.Message)
	}
	return auth
}

func loginWithPasskey(t *testing.T, h *UserHandler, auth *softAuthenticator, userID uint) (envelope.Response, []string) {
	t.Helper()
	begin, _ := call(t, h.BeginPasskeyLogin, 0, map[string]string{})
	if begin.Code != http.StatusOK {
		t.Fatalf("begin login: %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	return call(t, h.FinishPasskeyLogin, 0, map[string]any{"challenge_id": id, "credential": auth.assert(t, challenge, userID)})
}

func TestPasskey_RegisterAndLoginSkipsTwoFactor(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	secret, _ := enableTwoFactor(t, db, user)

	begin, _ := call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{
		"current_password": testPasskeyPass, "code": totpAt(t, secret, tfNow),
	})
	if begin.Code != http.StatusOK {
		t.Fatalf("begin registration: %d %q", begin.Code, begin.Message)
	}
	id, challenge := optionsChallenge(t, begin)
	auth := newSoftAuthenticator(t)
	fin, _ := call(t, h.FinishPasskeyRegistration, user.ID, map[string]any{
		"challenge_id": id, "name": "  MacBook  ", "credential": auth.attest(t, challenge),
	})
	if fin.Code != http.StatusOK {
		t.Fatalf("finish registration: %d %q", fin.Code, fin.Message)
	}

	resp, cookies := loginWithPasskey(t, h, auth, user.ID)
	if resp.Code != http.StatusOK || resp.Message != "login.successful" {
		t.Fatalf("passkey login: %d %q", resp.Code, resp.Message)
	}
	data := dataMap(t, resp)
	if data["token"] == nil || data["two_factor_required"] != nil || !hasRefreshCookie(cookies) {
		t.Fatalf("expected a full session without a 2FA step, got %v", data)
	}

	var rows []user_models.Passkey
	if err := db.Find(&rows).Error; err != nil || len(rows) != 1 {
		t.Fatalf("rows: %v %v", rows, err)
	}
	if rows[0].Name != "MacBook" || rows[0].LastUsedAt == nil {
		t.Fatalf("unexpected row: %+v", rows[0])
	}

	// The sign counter moved forward and a second login still works.
	resp, _ = loginWithPasskey(t, h, auth, user.ID)
	if resp.Code != http.StatusOK {
		t.Fatalf("second passkey login: %d %q", resp.Code, resp.Message)
	}

	// 2FA is on, so registering another one needs the code too.
	resp, _ = call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{"current_password": testPasskeyPass})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthInvalidTwoFactorCode)
}

func TestPasskey_RegistrationNeedsPassword(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)

	resp, _ := call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{"current_password": "wrong-password"})
	assertCode(t, resp, http.StatusUnprocessableEntity, core_errors.ErrAuthWrongCurrentPassword)
}

func TestPasskey_ChallengeIsSingleUse(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	auth := registerPasskey(t, h, user.ID, "laptop")

	begin, _ := call(t, h.BeginPasskeyLogin, 0, map[string]string{})
	id, challenge := optionsChallenge(t, begin)
	body := map[string]any{"challenge_id": id, "credential": auth.assert(t, challenge, user.ID)}

	if resp, _ := call(t, h.FinishPasskeyLogin, 0, body); resp.Code != http.StatusOK {
		t.Fatalf("first finish: %d %q", resp.Code, resp.Message)
	}
	resp, cookies := call(t, h.FinishPasskeyLogin, 0, body)
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthPasskeyInvalid)
	if hasRefreshCookie(cookies) {
		t.Fatal("replayed assertion got a session")
	}
}

func TestPasskey_ExpiredChallenge(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	auth := registerPasskey(t, h, user.ID, "laptop")

	begin, _ := call(t, h.BeginPasskeyLogin, 0, map[string]string{})
	id, challenge := optionsChallenge(t, begin)
	h.now = func() time.Time { return tfNow.Add(passkeyChallengeTTL + time.Second) }

	resp, _ := call(t, h.FinishPasskeyLogin, 0, map[string]any{"challenge_id": id, "credential": auth.assert(t, challenge, user.ID)})
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthPasskeyInvalid)
}

func TestPasskey_LoginRefusals(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	other := createResetUser(t, db, "dr.perez", "perez@example.com", testPasskeyPass)
	auth := registerPasskey(t, h, user.ID, "laptop")

	t.Run("signed by a key that is not registered", func(t *testing.T) {
		stranger := newSoftAuthenticator(t)
		stranger.credID = auth.credID // same credential ID, different key
		resp, _ := loginWithPasskey(t, h, stranger, user.ID)
		assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthPasskeyInvalid)
	})
	t.Run("another user's handle", func(t *testing.T) {
		resp, _ := loginWithPasskey(t, h, auth, other.ID)
		assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthPasskeyInvalid)
	})
	t.Run("unverified email", func(t *testing.T) {
		db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumn("email_verified", false)
		resp, _ := loginWithPasskey(t, h, auth, user.ID)
		assertCode(t, resp, http.StatusForbidden, core_errors.ErrAuthEmailNotVerified)
		db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumn("email_verified", true)
	})
	t.Run("locked account", func(t *testing.T) {
		db.Model(&user_models.User{}).Where("id = ?", user.ID).UpdateColumn("locked_until", tfNow.Add(time.Hour))
		resp, _ := loginWithPasskey(t, h, auth, user.ID)
		assertCode(t, resp, http.StatusTooManyRequests, core_errors.ErrAuthAccountLocked)
	})
}

func TestPasskey_DeleteIsScopedToOwner(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	other := createResetUser(t, db, "dr.perez", "perez@example.com", testPasskeyPass)
	auth := registerPasskey(t, h, user.ID, "laptop")

	var row user_models.Passkey
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	del := func(userID uint) envelope.Response {
		c, _ := testutils.NewGinContext(1, int64(userID))
		c.Request = httptest.NewRequest("DELETE", "/", bytes.NewReader(nil))
		c.Set("user_id", int64(userID))
		c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(row.ID))}}
		return h.DeletePasskey(c)
	}

	assertCode(t, del(other.ID), http.StatusNotFound, core_errors.ErrAuthPasskeyNotFound)
	if resp := del(user.ID); resp.Code != http.StatusOK {
		t.Fatalf("delete: %d %q", resp.Code, resp.Message)
	}
	resp, _ := loginWithPasskey(t, h, auth, user.ID)
	assertCode(t, resp, http.StatusUnauthorized, core_errors.ErrAuthPasskeyInvalid)
}

func TestPasskey_LimitPerUser(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	for i := 0; i < maxPasskeysPerUser; i++ {
		registerPasskey(t, h, user.ID, "key")
	}
	resp, _ := call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{"current_password": testPasskeyPass})
	assertCode(t, resp, http.StatusConflict, core_errors.ErrAuthPasskeyLimit)
}

func TestResetPasskeys(t *testing.T) {
	db, h := setupPasskeys(t)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)
	auth := registerPasskey(t, h, user.ID, "MacBook")

	if resp, _ := loginWithPasskey(t, h, auth, user.ID); resp.Code != http.StatusOK {
		t.Fatalf("passkey login before reset: %d %q", resp.Code, resp.Message)
	}
	db.Create(&user_models.WebAuthnChallenge{ID: "pending", Kind: "login", UserID: user.ID, Data: []byte("{}"), ExpiresAt: tfNow.Add(time.Minute)})

	if err := ResetPasskeys(db, user.ID); err != nil {
		t.Fatal(err)
	}
	var keys, challenges, live int64
	db.Model(&user_models.Passkey{}).Where("user_id = ?", user.ID).Count(&keys)
	db.Model(&user_models.WebAuthnChallenge{}).Where("user_id = ?", user.ID).Count(&challenges)
	db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&live)
	if keys != 0 || challenges != 0 || live != 0 {
		t.Fatalf("reset incomplete: passkeys=%d challenges=%d live sessions=%d", keys, challenges, live)
	}
	if resp, _ := loginWithPasskey(t, h, auth, user.ID); resp.Code == http.StatusOK {
		t.Fatal("passkey login still works after reset")
	}
}
