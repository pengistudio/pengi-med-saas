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
