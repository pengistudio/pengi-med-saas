package backoffice_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

// Company 1 has a locked user and one whose lock already expired; company 2
// has a locked user.
func setupCompanyUsers(t *testing.T) (*BackofficeCompanyHandler, *user_models.User, *user_models.User) {
	t.Helper()
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.Role{}, &user_models.Environment{}, &company_models.Company{})
	h := NewBackofficeCompanyHandler(db, zap.NewNop())

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	locked := &user_models.User{UserName: "locked", Email: "locked@x.com", FailedLoginAttempts: 5, LockedUntil: &future}
	expired := &user_models.User{UserName: "expired", Email: "expired@x.com", FailedLoginAttempts: 5, LockedUntil: &past}
	other := &user_models.User{UserName: "other", Email: "other@x.com", FailedLoginAttempts: 5, LockedUntil: &future}
	for _, u := range []*user_models.User{locked, expired, other} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	envs := []user_models.Environment{
		{UserID: locked.ID, CompanyID: 1, Name: "a"},
		{UserID: expired.ID, CompanyID: 1, Name: "a"},
		{UserID: other.ID, CompanyID: 2, Name: "b"},
	}
	if err := db.Create(&envs).Error; err != nil {
		t.Fatal(err)
	}
	return h, locked, other
}

func companyUserCtx(companyID, userID string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: companyID}, {Key: "user_id", Value: userID}}
	return c
}

func TestUnlockCompanyUser_ClearsLock(t *testing.T) {
	h, locked, _ := setupCompanyUsers(t)

	resp := h.UnlockCompanyUser(companyUserCtx("1", strconv.Itoa(int(locked.ID))))
	if resp.Code != http.StatusOK {
		t.Fatalf("UnlockCompanyUser = %d %q", resp.Code, resp.Message)
	}
	var u user_models.User
	if err := h.db.First(&u, locked.ID).Error; err != nil {
		t.Fatal(err)
	}
	if u.FailedLoginAttempts != 0 || u.LockedUntil != nil {
		t.Errorf("not unlocked: attempts=%d locked_until=%v", u.FailedLoginAttempts, u.LockedUntil)
	}
}

// An admin must not unlock a user of another company by guessing IDs.
func TestUnlockCompanyUser_OtherCompanyUserNotFound(t *testing.T) {
	h, _, other := setupCompanyUsers(t)

	resp := h.UnlockCompanyUser(companyUserCtx("1", strconv.Itoa(int(other.ID))))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("UnlockCompanyUser = %d, want 404", resp.Code)
	}
	var u user_models.User
	if err := h.db.First(&u, other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if u.FailedLoginAttempts != 5 || u.LockedUntil == nil {
		t.Errorf("user of another company was modified: %+v", u)
	}
}

func TestResetCompanyUserTwoFactor(t *testing.T) {
	h, locked, other := setupCompanyUsers(t)
	if err := h.db.AutoMigrate(&user_models.RecoveryCode{}, &user_models.RefreshToken{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []*user_models.User{locked, other} {
		h.db.Model(&user_models.User{}).Where("id = ?", u.ID).UpdateColumns(map[string]any{
			"two_factor_enabled": true, "two_factor_secret": []byte("sealed"),
		})
		h.db.Create(&user_models.RecoveryCode{UserID: u.ID, CodeHash: "h"})
		h.db.Create(&user_models.RefreshToken{UserID: u.ID, FamilyID: "f", TokenHash: strconv.Itoa(int(u.ID)), IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)})
	}

	// A user of another company: 404, untouched.
	if resp := h.ResetCompanyUserTwoFactor(companyUserCtx("1", strconv.Itoa(int(other.ID)))); resp.Code != http.StatusNotFound {
		t.Fatalf("other company user: %d, want 404", resp.Code)
	}
	var o user_models.User
	h.db.First(&o, other.ID)
	if !o.TwoFactorEnabled {
		t.Fatal("2FA of another company's user was reset")
	}

	if resp := h.ResetCompanyUserTwoFactor(companyUserCtx("1", strconv.Itoa(int(locked.ID)))); resp.Code != http.StatusOK {
		t.Fatalf("reset: %d %q", resp.Code, resp.Message)
	}
	var u user_models.User
	h.db.First(&u, locked.ID)
	var codes, live int64
	h.db.Model(&user_models.RecoveryCode{}).Where("user_id = ?", locked.ID).Count(&codes)
	h.db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", locked.ID).Count(&live)
	if u.TwoFactorEnabled || len(u.TwoFactorSecret) != 0 || codes != 0 || live != 0 ||
		u.PasswordChangedAt == nil || u.FailedLoginAttempts != 0 || u.LockedUntil != nil {
		t.Fatalf("reset incomplete: %+v codes=%d live=%d", u, codes, live)
	}
}

func TestResetCompanyUserPasskeys(t *testing.T) {
	h, locked, other := setupCompanyUsers(t)
	if err := h.db.AutoMigrate(&user_models.Passkey{}, &user_models.WebAuthnChallenge{}, &user_models.RefreshToken{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []*user_models.User{locked, other} {
		h.db.Create(&user_models.Passkey{UserID: u.ID, CredentialID: []byte{byte(u.ID)}, Credential: []byte("{}"), Name: "k"})
		h.db.Create(&user_models.RefreshToken{UserID: u.ID, FamilyID: "f", TokenHash: strconv.Itoa(int(u.ID)), IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)})
	}

	if resp := h.ResetCompanyUserPasskeys(companyUserCtx("1", strconv.Itoa(int(other.ID)))); resp.Code != http.StatusNotFound {
		t.Fatalf("other company user: %d, want 404", resp.Code)
	}
	var n int64
	h.db.Model(&user_models.Passkey{}).Where("user_id = ?", other.ID).Count(&n)
	if n != 1 {
		t.Fatal("passkeys of another company's user were removed")
	}

	if resp := h.ResetCompanyUserPasskeys(companyUserCtx("1", strconv.Itoa(int(locked.ID)))); resp.Code != http.StatusOK {
		t.Fatalf("reset: %d %q", resp.Code, resp.Message)
	}
	var keys, live int64
	h.db.Model(&user_models.Passkey{}).Where("user_id = ?", locked.ID).Count(&keys)
	h.db.Model(&user_models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", locked.ID).Count(&live)
	if keys != 0 || live != 0 {
		t.Fatalf("reset incomplete: passkeys=%d live=%d", keys, live)
	}
}

func TestGetCompanyUsers_LockedFlag(t *testing.T) {
	h, _, _ := setupCompanyUsers(t)

	resp := h.GetCompanyUsers(companyUserCtx("1", ""))
	if resp.Code != http.StatusOK {
		t.Fatalf("GetCompanyUsers = %d %q", resp.Code, resp.Message)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		UserName string `json:"user_name"`
		Locked   bool   `json:"locked"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range rows {
		got[r.UserName] = r.Locked
	}
	if len(got) != 2 || !got["locked"] || got["expired"] {
		t.Errorf("locked flags = %v, want locked=true expired=false", got)
	}
}

func TestGetCompanyUsers_TwoFactorFlag(t *testing.T) {
	h, locked, _ := setupCompanyUsers(t)
	if err := h.db.Model(&user_models.User{}).Where("id = ?", locked.ID).
		UpdateColumn("two_factor_enabled", true).Error; err != nil {
		t.Fatal(err)
	}

	resp := h.GetCompanyUsers(companyUserCtx("1", ""))
	if resp.Code != http.StatusOK {
		t.Fatalf("GetCompanyUsers = %d %q", resp.Code, resp.Message)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		UserName         string `json:"user_name"`
		TwoFactorEnabled bool   `json:"two_factor_enabled"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range rows {
		got[r.UserName] = r.TwoFactorEnabled
	}
	if len(got) != 2 || !got["locked"] || got["expired"] {
		t.Errorf("two_factor_enabled flags = %v, want locked=true expired=false", got)
	}
}
