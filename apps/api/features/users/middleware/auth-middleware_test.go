package auth_middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// testUser is the slice of the users table the existence check reads (this
// package can't import user_models: import cycle through core/audit).
type testUser struct {
	gorm.Model
	UserName string
}

func (testUser) TableName() string { return "users" }

func serveStaff(t *testing.T, mw gin.HandlerFunc, token string) (int, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	authenticated := false
	r := gin.New()
	r.GET("/x", mw, func(c *gin.Context) {
		authenticated = c.GetBool("authenticated")
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, authenticated
}

func setAuthEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("AUTH_KEY") == "" {
		t.Setenv("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	}
	if os.Getenv("AUTH_EXP") == "" {
		t.Setenv("AUTH_EXP", "60")
	}
}

func TestAuthMiddleware_RejectsPortalAndBackofficeTokens(t *testing.T) {
	setAuthEnv(t)
	staff, _ := auth.GenerateToken("admin", 1)
	if code, _ := serveStaff(t, AuthMiddleware(), staff); code != http.StatusOK {
		t.Fatalf("staff token: %d", code)
	}
	portalAccess, _ := auth.GeneratePortalAccessToken(1, 2)
	portalRefresh, _, _ := auth.GeneratePortalRefreshToken(1, 2, "family")
	backoffice, _ := auth.GenerateBackofficeToken("root", 1)
	for name, token := range map[string]string{"portal access": portalAccess, "portal refresh": portalRefresh, "backoffice": backoffice} {
		if code, _ := serveStaff(t, AuthMiddleware(), token); code != http.StatusUnauthorized {
			t.Fatalf("%s token on a staff route: %d", name, code)
		}
		if _, ok := serveStaff(t, OptionalAuthMiddleware(), token); ok {
			t.Fatalf("%s token must not authenticate optionally", name)
		}
	}
}

// A signed token of a user deleted since it was issued no longer opens the
// app, without waiting for it to expire.
func TestAuthMiddleware_RejectsDeletedUser(t *testing.T) {
	setAuthEnv(t)
	db := testutils.SetupTestDB(t, &testUser{})
	UseUserDB(db)
	t.Cleanup(func() { UseUserDB(nil) })

	user := testUser{UserName: "ana"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	token, _ := auth.GenerateToken("ana", int64(user.ID))
	if code, _ := serveStaff(t, AuthMiddleware(), token); code != http.StatusOK {
		t.Fatalf("live user: %d, want 200", code)
	}

	if err := db.Delete(&user).Error; err != nil {
		t.Fatal(err)
	}
	ForgetUser(user.ID)
	if code, _ := serveStaff(t, AuthMiddleware(), token); code != http.StatusUnauthorized {
		t.Fatalf("deleted user: %d, want 401", code)
	}
	if _, ok := serveStaff(t, OptionalAuthMiddleware(), token); ok {
		t.Fatal("deleted user must not authenticate optionally")
	}
}
