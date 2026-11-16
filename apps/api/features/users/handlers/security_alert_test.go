package user_handlers

import (
	"net/http"
	"net/http/httptest"
	"pengi-med-saas/core/auth"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// alertRecorder replaces the security alert seam and records what fired.
type alertRecorder struct {
	events []securityEvent
	users  []uint
}

func recordAlerts(h *UserHandler) *alertRecorder {
	r := &alertRecorder{}
	h.notifySecurity = func(user user_models.User, event securityEvent, c *gin.Context) {
		r.events = append(r.events, event)
		r.users = append(r.users, user.ID)
	}
	return r
}

// expect asserts the events fired since the last call, then resets.
func (r *alertRecorder) expect(t *testing.T, userID uint, want ...securityEvent) {
	t.Helper()
	if len(r.events) != len(want) {
		t.Fatalf("expected alerts %v, got %v", want, r.events)
	}
	for i := range want {
		if r.events[i] != want[i] || r.users[i] != userID {
			t.Fatalf("expected alerts %v for user %d, got %v for %v", want, userID, r.events, r.users)
		}
	}
	r.events, r.users = nil, nil
}

func TestSecurityAlert_PasswordChange(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	rec := recordAlerts(h)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	change := func(current, next string) int {
		c := postJSON(t, map[string]string{"current_password": current, "new_password": next})
		c.Set("user_id", int64(user.ID))
		return h.ChangePassword(c).Code
	}
	if code := change("wrong-password", "new-password-123"); code != 422 {
		t.Fatalf("expected 422, got %d", code)
	}
	rec.expect(t, user.ID)
	if code := change("old-password-1", "new-password-123"); code != 200 {
		t.Fatalf("expected 200, got %d", code)
	}
	rec.expect(t, user.ID, eventPasswordChanged)
}

func TestSecurityAlert_PasswordReset(t *testing.T) {
	captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	rec := recordAlerts(h)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password")

	c := postJSON(t, map[string]string{"token": "garbage", "new_password": "new-password-123"})
	if resp := h.ResetPassword(c); resp.Code == 200 {
		t.Fatal("garbage token accepted")
	}
	rec.expect(t, user.ID)

	token, err := auth.GeneratePasswordResetToken(user.ID, user.Password, auth.PasswordResetTTL)
	if err != nil {
		t.Fatal(err)
	}
	c = postJSON(t, map[string]string{"token": token, "new_password": "new-password-123"})
	if resp := h.ResetPassword(c); resp.Code != 200 {
		t.Fatalf("reset: %d", resp.Code)
	}
	rec.expect(t, user.ID, eventPasswordChanged)
}

func TestSecurityAlert_TwoFactor(t *testing.T) {
	db, h := setupTwoFactor(t)
	rec := recordAlerts(h)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")
	if err := IssueRefreshFamily(db, user.ID, user.UserName, postJSON(t, nil)); err != nil {
		t.Fatal(err)
	}

	resp, _ := call(t, h.SetupTwoFactor, user.ID, map[string]string{"current_password": "password123"})
	secret := dataMap(t, resp)["secret"].(string)
	rec.expect(t, user.ID) // setup alone is not an event

	resp, _ = call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": "000000"})
	if resp.Code == http.StatusOK {
		t.Fatal("wrong code enabled 2FA")
	}
	rec.expect(t, user.ID)
	resp, _ = call(t, h.EnableTwoFactor, user.ID, map[string]string{"code": totpAt(t, secret, tfNow)})
	if resp.Code != http.StatusOK {
		t.Fatalf("enable: %d %q", resp.Code, resp.Message)
	}
	rec.expect(t, user.ID, eventTwoFactorEnabled)

	resp, _ = call(t, h.RegenerateRecoveryCodes, user.ID, map[string]string{"current_password": "wrong", "code": totpAt(t, secret, tfNow)})
	if resp.Code == http.StatusOK {
		t.Fatal("wrong password regenerated codes")
	}
	rec.expect(t, user.ID)
	resp, _ = call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": "000000"})
	if resp.Code == http.StatusOK {
		t.Fatal("wrong code disabled 2FA")
	}
	rec.expect(t, user.ID)

	expireLock(t, db, user.ID)
	clearLockout(db, user.ID)
	resp, _ = call(t, h.RegenerateRecoveryCodes, user.ID, map[string]string{"current_password": "password123", "code": totpAt(t, secret, tfNow.Add(30*time.Second))})
	if resp.Code != http.StatusOK {
		t.Fatalf("regenerate: %d %q", resp.Code, resp.Message)
	}
	rec.expect(t, user.ID, eventRecoveryCodesRegenerated)
	fresh, _ := dataMap(t, resp)["recovery_codes"].([]any)

	resp, _ = call(t, h.DisableTwoFactor, user.ID, map[string]string{"current_password": "password123", "code": fresh[0].(string)})
	if resp.Code != http.StatusOK {
		t.Fatalf("disable: %d %q", resp.Code, resp.Message)
	}
	rec.expect(t, user.ID, eventTwoFactorDisabled)
}

func TestSecurityAlert_Passkeys(t *testing.T) {
	db, h := setupPasskeys(t)
	rec := recordAlerts(h)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", testPasskeyPass)

	// Beginning a registration, even with a wrong password, is not an event.
	call(t, h.BeginPasskeyRegistration, user.ID, map[string]string{"current_password": "wrong-password"})
	rec.expect(t, user.ID)

	registerPasskey(t, h, user.ID, "laptop")
	rec.expect(t, user.ID, eventPasskeyAdded)

	var row user_models.Passkey
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	del := func(id string) int {
		c, _ := testutils.NewGinContext(1, int64(user.ID))
		c.Request = httptest.NewRequest("DELETE", "/", nil)
		c.Set("user_id", int64(user.ID))
		c.Params = gin.Params{{Key: "id", Value: id}}
		return h.DeletePasskey(c).Code
	}
	if code := del("9999"); code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
	rec.expect(t, user.ID)
	if code := del(strconv.Itoa(int(row.ID))); code != http.StatusOK {
		t.Fatalf("delete: %d", code)
	}
	rec.expect(t, user.ID, eventPasskeyDeleted)
}

// The alert texts are fixed per event: nothing the user typed or a secret
// can reach the email through them.
func TestSecurityAlert_TextsCarryNoSecrets(t *testing.T) {
	events := []securityEvent{eventPasswordChanged, eventTwoFactorEnabled, eventTwoFactorDisabled,
		eventRecoveryCodesRegenerated, eventPasskeyAdded, eventPasskeyDeleted}
	for _, e := range events {
		text, ok := securityAlertText[e]
		if !ok || text[0] == "" || text[1] == "" {
			t.Fatalf("event %q has no text", e)
		}
		for _, s := range text {
			if strings.ContainsAny(s, "0123456789%{}<>") {
				t.Fatalf("event %q text looks interpolated: %q", e, s)
			}
		}
	}
}
