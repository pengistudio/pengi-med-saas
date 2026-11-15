package backoffice_handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	"pengi-med-saas/testutils"
)

// CreateUser used to bind the BackofficeUser model, whose Password is
// json:"-": the password sent was dropped and the new admin got the hash of an
// empty password. It must store the password sent, and accept only the
// fields an admin may set.
func TestCreateUser_StoresSentPasswordAndIgnoresOtherFields(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	h := NewBackofficeUserHandler(db, zap.NewNop())

	c, _ := testutils.NewGinContext(0, 1)
	body := `{"ID": 4242, "name": "Ana", "user_name": "ana-admin", "password": "s3cret-pass", "refresh_token": "x"}`
	c.Request = httptest.NewRequest("POST", "/backoffice/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if resp := h.CreateUser(c); resp.Code != 200 {
		t.Fatalf("CreateUser = %d %q", resp.Code, resp.Message)
	}

	var user backoffice_models.BackofficeUser
	if err := db.Where("user_name = ?", "ana-admin").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.ID == 4242 || user.RefreshToken != "" {
		t.Errorf("client-set fields were stored: id=%d refresh_token=%q", user.ID, user.RefreshToken)
	}
	for pw, want := range map[string]bool{"s3cret-pass": true, "": false} {
		attempt := backoffice_models.BackofficeUser{UserName: "ana-admin", Password: pw}
		if ok := attempt.ValidateCredentials(db) == nil; ok != want {
			t.Errorf("login with password %q: ok=%v, want %v", pw, ok, want)
		}
	}
}

// Creating an admin without a user name or password is rejected.
func TestCreateUser_RequiresUserNameAndPassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	h := NewBackofficeUserHandler(db, zap.NewNop())

	for _, body := range []string{
		`{"name": "Ana", "user_name": "ana-admin"}`,
		`{"name": "Ana", "password": "s3cret-pass"}`,
	} {
		c, _ := testutils.NewGinContext(0, 1)
		c.Request = httptest.NewRequest("POST", "/backoffice/users", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if resp := h.CreateUser(c); resp.Code != 400 {
			t.Errorf("CreateUser(%s) = %d, want 400", body, resp.Code)
		}
	}
}

// An empty password is rejected before the credential check, so admins left
// with the hash of an empty password can't be logged into.
func TestLogin_RejectsEmptyPassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	legacy := backoffice_models.BackofficeUser{UserName: "legacy-admin", Password: ""}
	if err := legacy.Save(db); err != nil {
		t.Fatal(err)
	}
	h := NewBackofficeUserHandler(db, zap.NewNop())

	c, _ := testutils.NewGinContext(0, 1)
	c.Request = httptest.NewRequest("POST", "/backoffice/auth/login", bytes.NewBufferString(`{"user_name": "legacy-admin", "password": ""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	if resp := h.Login(c); resp.Code == 200 {
		t.Fatalf("login with an empty password = 200, want it rejected")
	}
}

func callJSON(t *testing.T, method, path, body string, handle func(*gin.Context) envelope.Response) (envelope.Response, *httptest.ResponseRecorder) {
	t.Helper()
	c, w := testutils.NewGinContext(0, 1)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return handle(c), w
}

// Admin passwords need at least 10 characters, on create and on change.
func TestCreateAndUpdateUser_RejectShortPasswords(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	h := NewBackofficeUserHandler(db, zap.NewNop())

	resp, _ := callJSON(t, "POST", "/backoffice/users", `{"user_name": "ana-admin", "password": "123456789"}`, h.CreateUser)
	if resp.Code != 400 || resp.Message != "auth.password.too_short" {
		t.Fatalf("CreateUser with 9 chars = %d %q, want 400 auth.password.too_short", resp.Code, resp.Message)
	}
	if resp, _ := callJSON(t, "POST", "/backoffice/users", `{"user_name": "ana-admin", "password": "1234567890"}`, h.CreateUser); resp.Code != 200 {
		t.Fatalf("CreateUser with 10 chars = %d %q", resp.Code, resp.Message)
	}

	var user backoffice_models.BackofficeUser
	db.Where("user_name = ?", "ana-admin").First(&user)
	c, _ := testutils.NewGinContext(0, 1)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(user.ID))}}
	c.Request = httptest.NewRequest("PUT", "/backoffice/users", bytes.NewBufferString(`{"password": "short"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	if resp := h.UpdateUser(c); resp.Code != 400 || resp.Message != "auth.password.too_short" {
		t.Fatalf("UpdateUser with a short password = %d %q, want 400 auth.password.too_short", resp.Code, resp.Message)
	}
}

// The seed created admins with the password "password", replaced by a
// migration only in release: such an account can't sign in outside local
// development.
func TestLogin_RefusesSeedPasswordOutsideDevelopment(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	seeded := backoffice_models.BackofficeUser{UserName: "admin", Password: "password"}
	if err := seeded.Save(db); err != nil {
		t.Fatal(err)
	}
	login := `{"user_name": "admin", "password": "password"}`

	for _, mode := range []string{"release", "test", ""} {
		t.Setenv("GIN_MODE", mode)
		h := NewBackofficeUserHandler(db, zap.NewNop())
		if resp, _ := callJSON(t, "POST", "/backoffice/auth/login", login, h.Login); resp.Code != 401 {
			t.Errorf("GIN_MODE=%q: seed password login = %d, want 401", mode, resp.Code)
		}
	}
	t.Setenv("GIN_MODE", "development")
	h := NewBackofficeUserHandler(db, zap.NewNop())
	if resp, _ := callJSON(t, "POST", "/backoffice/auth/login", login, h.Login); resp.Code != 200 {
		t.Errorf("GIN_MODE=development: seed password login = %d %q, want 200", resp.Code, resp.Message)
	}
}

// After 5 failed attempts at one admin, even the right password waits.
func TestLogin_LimitsFailedAttemptsPerUsername(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	admin := backoffice_models.BackofficeUser{UserName: "ana-admin", Password: "s3cret-pass"}
	if err := admin.Save(db); err != nil {
		t.Fatal(err)
	}
	h := NewBackofficeUserHandler(db, zap.NewNop())
	for i := 0; i < 5; i++ {
		if resp, _ := callJSON(t, "POST", "/backoffice/auth/login", `{"user_name": "ana-admin", "password": "wrong-pass"}`, h.Login); resp.Code != 401 {
			t.Fatalf("attempt %d = %d, want 401", i+1, resp.Code)
		}
	}
	if resp, _ := callJSON(t, "POST", "/backoffice/auth/login", `{"user_name": "ana-admin", "password": "s3cret-pass"}`, h.Login); resp.Code != 429 {
		t.Fatalf("login after 5 failures = %d, want 429", resp.Code)
	}
}

func refreshCookie(w *httptest.ResponseRecorder) string {
	for _, ck := range w.Result().Cookies() {
		if ck.Name == auth.BackofficeRefreshCookie {
			return ck.Value
		}
	}
	return ""
}

func refreshWith(h *BackofficeUserHandler, token string) (envelope.Response, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(0, 1)
	c.Request = httptest.NewRequest("POST", "/backoffice/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: auth.BackofficeRefreshCookie, Value: token})
	return h.RefreshAuthToken(c), w
}

// Refreshing rotates the refresh token; the old one works only for a moment
// (concurrent tabs) and only while its successor is still current.
func TestRefresh_RotatesRefreshToken(t *testing.T) {
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	admin := backoffice_models.BackofficeUser{UserName: "ana-admin", Password: "s3cret-pass"}
	if err := admin.Save(db); err != nil {
		t.Fatal(err)
	}
	h := NewBackofficeUserHandler(db, zap.NewNop())
	resp, w := callJSON(t, "POST", "/backoffice/auth/login", `{"user_name": "ana-admin", "password": "s3cret-pass"}`, h.Login)
	if resp.Code != 200 {
		t.Fatalf("login = %d", resp.Code)
	}
	first := refreshCookie(w)

	resp, w = refreshWith(h, first)
	second := refreshCookie(w)
	if resp.Code != 200 || second == "" || second == first {
		t.Fatalf("refresh = %d, rotated=%v", resp.Code, second != "" && second != first)
	}

	// A concurrent tab still holding the first token gets the second one.
	resp, w = refreshWith(h, first)
	if resp.Code != 200 || refreshCookie(w) != second {
		t.Fatalf("concurrent refresh with the old token = %d, want 200 with the current token", resp.Code)
	}

	// Once the second is rotated too, the first is dead.
	if resp, _ := refreshWith(h, second); resp.Code != 200 {
		t.Fatalf("refresh with the second token = %d", resp.Code)
	}
	if resp, _ := refreshWith(h, first); resp.Code != 401 {
		t.Fatalf("refresh with a stale token = %d, want 401", resp.Code)
	}

	// Outside the grace window the old token is rejected.
	h.rotated = newRotationGrace()
	if resp, _ := refreshWith(h, second); resp.Code != 401 {
		t.Fatalf("refresh with a rotated-out token after the grace window = %d, want 401", resp.Code)
	}
}
