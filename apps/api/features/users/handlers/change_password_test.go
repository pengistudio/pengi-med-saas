package user_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"pengi-med-saas/core/auth"
	core_errors "pengi-med-saas/core/errors"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestChangePassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	call := func(current, next string) int {
		c := postJSON(t, map[string]string{"current_password": current, "new_password": next})
		c.Set("user_id", int64(user.ID))
		return handler.ChangePassword(c).Code
	}

	if code := call("wrong-password", "new-password-123"); code != 401 {
		t.Fatalf("wrong current: expected 401, got %d", code)
	}
	if code := call("old-password-1", "old-password-1"); code != 400 {
		t.Fatalf("same password: expected 400, got %d", code)
	}
	if code := call("old-password-1", "short"); code != 400 {
		t.Fatalf("too short: expected 400, got %d", code)
	}
	if code := call("old-password-1", "new-password-123"); code != 200 {
		t.Fatalf("change: expected 200, got %d", code)
	}

	var got user_models.User
	db.First(&got, user.ID)
	if !auth.CompareHashAndPassword(got.Password, "new-password-123") {
		t.Fatal("new password not stored")
	}
	if got.PasswordChangedAt == nil {
		t.Fatal("password_changed_at not stamped")
	}
}

// serveWithAuth runs one request through AuthMiddleware and returns the
// status and the error code of the envelope (empty on success).
func serveWithAuth(t *testing.T, token string) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", auth_middleware.AuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body struct {
		Data struct {
			ErrorCode string `json:"error_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body.Data.ErrorCode
}

// After a password change or reset, access tokens issued before it stop
// working at once; a token from a login right after (same second) works.
func TestPasswordChange_RevokesOlderAccessTokens(t *testing.T) {
	if os.Getenv("AUTH_KEY") == "" {
		t.Setenv("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	}
	if os.Getenv("AUTH_EXP") == "" {
		t.Setenv("AUTH_EXP", "60")
	}
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	auth_middleware.UseUserDB(db)
	t.Cleanup(func() { auth_middleware.UseUserDB(nil) })
	handler := NewUserHandler(db, zap.NewNop())
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	oldToken, _ := auth.GenerateToken(user.UserName, int64(user.ID))
	if code, _ := serveWithAuth(t, oldToken); code != http.StatusOK {
		t.Fatalf("token before change: %d, want 200", code)
	}

	// iat has second granularity: let the old token fall a full second behind.
	time.Sleep(1100 * time.Millisecond)
	c := postJSON(t, map[string]string{"current_password": "old-password-1", "new_password": "new-password-123"})
	c.Set("user_id", int64(user.ID))
	if code := handler.ChangePassword(c).Code; code != 200 {
		t.Fatalf("change: %d", code)
	}

	code, errCode := serveWithAuth(t, oldToken)
	if code != http.StatusUnauthorized || errCode != core_errors.ErrAuthSessionRevoked.ErrorCode {
		t.Fatalf("token before change: %d %q, want 401 %s", code, errCode, core_errors.ErrAuthSessionRevoked.ErrorCode)
	}

	newToken, _ := auth.GenerateToken(user.UserName, int64(user.ID))
	if code, _ := serveWithAuth(t, newToken); code != http.StatusOK {
		t.Fatalf("token issued right after change: %d, want 200", code)
	}
}
