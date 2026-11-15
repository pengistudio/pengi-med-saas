package user_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// testLockout is the policy the tests run with, independent of the env.
var testLockout = accountLockout{maxAttempts: 3, duration: 15 * time.Minute}

func newLockoutUserHandler(db *gorm.DB) *UserHandler {
	h := NewUserHandler(db, zap.NewNop())
	h.lockout = testLockout
	h.unknownLogins = testLockout.unknownUserLimiter()
	return h
}

func errorCode(resp envelope.Response) string {
	if appErr, ok := resp.Data.(core_errors.AppError); ok {
		return appErr.ErrorCode
	}
	return ""
}

func login(t *testing.T, h *UserHandler, userName, password string) envelope.Response {
	t.Helper()
	return h.Login(postJSON(t, user_dto.LoginDTO{UserName: userName, Password: password}))
}

func lockState(t *testing.T, db *gorm.DB, userID uint) (int, *time.Time) {
	t.Helper()
	var u user_models.User
	if err := db.First(&u, userID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	return u.FailedLoginAttempts, u.LockedUntil
}

func expireLock(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	if err := db.Model(&user_models.User{}).Where("id = ?", userID).
		UpdateColumn("locked_until", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire lock: %v", err)
	}
}

func assertLocked(t *testing.T, resp envelope.Response) {
	t.Helper()
	if resp.Code != http.StatusTooManyRequests || errorCode(resp) != core_errors.ErrAuthAccountLocked.ErrorCode {
		t.Fatalf("expected 429 %s, got %d %q", core_errors.ErrAuthAccountLocked.ErrorCode, resp.Code, errorCode(resp))
	}
}

func TestLockout_LocksAfterRepeatedFailuresAndRefusesCorrectPassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")

	for i := 0; i < testLockout.maxAttempts; i++ {
		if resp := login(t, h, "dra.lopez", "wrong"); resp.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, resp.Code)
		}
	}
	if _, lockedUntil := lockState(t, db, user.ID); lockedUntil == nil || !lockedUntil.After(time.Now()) {
		t.Fatalf("expected the account to be locked, locked_until = %v", lockedUntil)
	}

	// Locked: even the right password (by username or email) is refused.
	assertLocked(t, login(t, h, "dra.lopez", "password123"))
	assertLocked(t, login(t, h, "lopez@example.com", "password123"))

	// After the lock expires the right password works and resets the counter.
	expireLock(t, db, user.ID)
	if resp := login(t, h, "dra.lopez", "password123"); resp.Code != http.StatusOK {
		t.Fatalf("after expiry: expected 200, got %d", resp.Code)
	}
	if attempts, lockedUntil := lockState(t, db, user.ID); attempts != 0 || lockedUntil != nil {
		t.Fatalf("expected a clean state after login, got attempts=%d locked_until=%v", attempts, lockedUntil)
	}
}

func TestLockout_SuccessResetsCounter(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")

	// maxAttempts-1 failures, a success, then maxAttempts-1 failures again:
	// never locked because the success restarted the count.
	for round := 0; round < 2; round++ {
		for i := 0; i < testLockout.maxAttempts-1; i++ {
			if resp := login(t, h, "dra.lopez", "wrong"); resp.Code != http.StatusUnauthorized {
				t.Fatalf("round %d attempt %d: expected 401, got %d", round, i+1, resp.Code)
			}
		}
		if resp := login(t, h, "dra.lopez", "password123"); resp.Code != http.StatusOK {
			t.Fatalf("round %d: expected 200, got %d", round, resp.Code)
		}
	}
}

func TestLockout_UnknownUserGetsSameAnswers(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)

	for i := 0; i < testLockout.maxAttempts; i++ {
		resp := login(t, h, "nobody", "wrong")
		if resp.Code != http.StatusUnauthorized || errorCode(resp) != core_errors.ErrAuthInvalidCredentials.ErrorCode {
			t.Fatalf("attempt %d: expected 401 invalid credentials, got %d %q", i+1, resp.Code, errorCode(resp))
		}
	}
	assertLocked(t, login(t, h, "nobody", "wrong"))
}

func TestLockout_ChangePasswordFailuresCount(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	change := func(current string) envelope.Response {
		c := postJSON(t, map[string]string{"current_password": current, "new_password": "new-password-123"})
		c.Set("user_id", int64(user.ID))
		return h.ChangePassword(c)
	}

	// One failed login plus failed change-password attempts share the counter.
	if resp := login(t, h, "dra.lopez", "wrong"); resp.Code != http.StatusUnauthorized {
		t.Fatalf("login: expected 401, got %d", resp.Code)
	}
	for i := 1; i < testLockout.maxAttempts; i++ {
		if resp := change("wrong"); resp.Code != http.StatusUnprocessableEntity {
			t.Fatalf("change attempt %d: expected 422, got %d", i, resp.Code)
		}
	}
	assertLocked(t, change("old-password-1"))
	assertLocked(t, login(t, h, "dra.lopez", "old-password-1"))

	var got user_models.User
	db.First(&got, user.ID)
	if !auth.CompareHashAndPassword(got.Password, "old-password-1") {
		t.Fatal("password changed while the account was locked")
	}
}

func TestLockout_EmailChangeFailuresCount(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.Role{}, &user_models.Environment{}, &company_models.Company{})
	profile := NewProfileHandler(db, zap.NewNop())
	profile.lockout = testLockout
	h := newLockoutUserHandler(db)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	role := user_models.Role{Role: "ADMIN"}
	db.Create(&role)
	company := company_models.Company{LegalName: "Clinica", TradeName: "Clinica", TenantID: 1}
	db.Create(&company)
	env := user_models.Environment{UserID: user.ID, Name: "Clinica", RoleID: role.ID, CompanyID: company.ID}
	db.Create(&env)

	update := func(password string) envelope.Response {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"email": "new@example.com", "current_password": password})
		c, _ := testutils.NewGinContext(1, int64(user.ID))
		c.Set("user_id", int64(user.ID))
		c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/users/profile?environment_id=%d", env.ID), bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return profile.UpdateProfile(c)
	}

	for i := 0; i < testLockout.maxAttempts; i++ {
		if resp := update("wrong"); resp.Code != http.StatusUnprocessableEntity {
			t.Fatalf("attempt %d: expected 422, got %d", i+1, resp.Code)
		}
	}
	assertLocked(t, update("old-password-1"))
	assertLocked(t, login(t, h, "dra.lopez", "old-password-1"))

	var got user_models.User
	db.First(&got, user.ID)
	if got.Email != "lopez@example.com" {
		t.Fatalf("email changed while locked: %q", got.Email)
	}
}

func TestLockout_ResetPasswordClearsLock(t *testing.T) {
	captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	h := newLockoutUserHandler(db)
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password")

	for i := 0; i < testLockout.maxAttempts; i++ {
		login(t, h, "dra.lopez", "wrong")
	}
	assertLocked(t, login(t, h, "dra.lopez", "old-password"))

	token, err := auth.GeneratePasswordResetToken(user.ID, user.Password, auth.PasswordResetTTL)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if resp := h.ResetPassword(postJSON(t, map[string]string{"token": token, "new_password": "new-password-1"})); resp.Code != http.StatusOK {
		t.Fatalf("reset: expected 200, got %d", resp.Code)
	}
	if attempts, lockedUntil := lockState(t, db, user.ID); attempts != 0 || lockedUntil != nil {
		t.Fatalf("expected the reset to clear the lock, got attempts=%d locked_until=%v", attempts, lockedUntil)
	}
	if resp := login(t, h, "dra.lopez", "new-password-1"); resp.Code != http.StatusOK {
		t.Fatalf("login after reset: expected 200, got %d", resp.Code)
	}
}

// Concurrent failures are all counted, and a burst can't check more than
// maxAttempts passwords before the lock applies.
func TestLockout_ConcurrentFailures(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{})
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")

	run := func(l accountLockout, n int) (checked int64) {
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = l.check(db, zap.NewNop(), user.ID, func() bool {
					atomic.AddInt64(&checked, 1)
					return false
				})
			}()
		}
		wg.Wait()
		return checked
	}

	// No increment lost below the threshold.
	if checked := run(accountLockout{maxAttempts: 1000, duration: time.Minute}, 20); checked != 20 {
		t.Fatalf("expected 20 checks, got %d", checked)
	}
	if attempts, _ := lockState(t, db, user.ID); attempts != 20 {
		t.Fatalf("expected 20 failed attempts, got %d", attempts)
	}

	// Burst past the threshold: exactly maxAttempts guesses are checked.
	if err := clearLockout(db, user.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if checked := run(accountLockout{maxAttempts: 5, duration: time.Minute}, 30); checked != 5 {
		t.Fatalf("expected exactly 5 password checks, got %d", checked)
	}
	if _, lockedUntil := lockState(t, db, user.ID); lockedUntil == nil {
		t.Fatal("expected the account to be locked")
	}
}
