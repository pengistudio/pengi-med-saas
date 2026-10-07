package backoffice_handlers

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

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
