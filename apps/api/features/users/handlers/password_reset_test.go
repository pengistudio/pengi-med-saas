package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"pengi-med-saas/core/auth"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

type sentResetEmail struct{ to, userName, url string }

// captureResetEmails replaces the mailer and returns the emails "sent".
func captureResetEmails(t *testing.T) func() []sentResetEmail {
	t.Helper()
	t.Setenv("AUTH_KEY", "test-key")
	t.Setenv("AUTH_EXP", "60")
	var mu sync.Mutex
	var sent []sentResetEmail
	original := sendPasswordResetEmail
	sendPasswordResetEmail = func(to, userName, resetURL string) error {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, sentResetEmail{to, userName, resetURL})
		return nil
	}
	t.Cleanup(func() { sendPasswordResetEmail = original })
	return func() []sentResetEmail {
		// The email goes out in a goroutine.
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			n := len(sent)
			mu.Unlock()
			if n > 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		mu.Lock()
		defer mu.Unlock()
		return append([]sentResetEmail(nil), sent...)
	}
}

func createResetUser(t *testing.T, db *gorm.DB, userName, email, password string) user_models.User {
	t.Helper()
	hashed, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := user_models.User{UserName: userName, Email: email, Password: hashed, EmailVerified: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func postJSON(t *testing.T, payload any) *gin.Context {
	t.Helper()
	body, _ := json.Marshal(payload)
	c, _ := testutils.NewGinContext(1, 0)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestForgotPassword_EmailsSingleUseLink(t *testing.T) {
	sent := captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password")

	c := postJSON(t, map[string]string{"identifier": "LOPEZ@example.com"})
	if resp := handler.ForgotPassword(c); resp.Code != 200 {
		t.Fatalf("forgot password: expected 200, got %d", resp.Code)
	}
	emails := sent()
	if len(emails) != 1 || emails[0].to != "lopez@example.com" || emails[0].userName != "dra.lopez" {
		t.Fatalf("expected one email to lopez@example.com, got %+v", emails)
	}
	link, err := url.Parse(emails[0].url)
	if err != nil || !strings.HasSuffix(link.Path, "/reset-password") {
		t.Fatalf("unexpected reset link %q", emails[0].url)
	}
	token := link.Query().Get("token")

	reset := func() int {
		c := postJSON(t, map[string]string{"token": token, "new_password": "new-password"})
		return handler.ResetPassword(c).Code
	}
	if code := reset(); code != 200 {
		t.Fatalf("first reset: expected 200, got %d", code)
	}
	if code := reset(); code != 401 {
		t.Fatalf("second reset with the same link: expected 401, got %d", code)
	}
}

func TestForgotPassword_SameAnswerForUnknownUser(t *testing.T) {
	sent := captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())

	c := postJSON(t, map[string]string{"identifier": "nobody"})
	resp := handler.ForgotPassword(c)
	if resp.Code != 200 || resp.Message != "auth.forgot_password.sent" {
		t.Fatalf("expected the generic success, got %d %q", resp.Code, resp.Message)
	}
	if emails := sent(); len(emails) != 0 {
		t.Fatalf("expected no email, got %+v", emails)
	}
}

func TestResetPassword_RevokesSessions(t *testing.T) {
	captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password")
	session := user_models.RefreshToken{
		UserID: user.ID, FamilyID: "00000000-0000-0000-0000-000000000001", TokenHash: "h",
		IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}

	token, err := auth.GeneratePasswordResetToken(user.ID, user.Password, auth.PasswordResetTTL)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	c := postJSON(t, map[string]string{"token": token, "new_password": "new-password"})
	if resp := handler.ResetPassword(c); resp.Code != 200 {
		t.Fatalf("reset: expected 200, got %d", resp.Code)
	}

	if err := db.First(&session, session.ID).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatal("expected the session to be revoked after the reset")
	}
}

func TestLogin_WithEmail(t *testing.T) {
	captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	createResetUser(t, db, "dra.lopez", "lopez@example.com", "password123")

	c := postJSON(t, user_dto.LoginDTO{UserName: "Lopez@Example.com", Password: "password123"})
	if resp := handler.Login(c); resp.Code != 200 {
		t.Fatalf("login with email: expected 200, got %d", resp.Code)
	}
}

func TestLogin_WithSharedEmailFails(t *testing.T) {
	captureResetEmails(t)
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	createResetUser(t, db, "user.a", "shared@example.com", "password123")
	createResetUser(t, db, "user.b", "shared@example.com", "password123")

	c := postJSON(t, user_dto.LoginDTO{UserName: "shared@example.com", Password: "password123"})
	if resp := handler.Login(c); resp.Code != 401 {
		t.Fatalf("login with an email shared by two users: expected 401, got %d", resp.Code)
	}
}
