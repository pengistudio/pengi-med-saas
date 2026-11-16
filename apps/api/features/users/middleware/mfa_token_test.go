package auth_middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"pengi-med-saas/core/auth"

	"github.com/gin-gonic/gin"
)

func serveExchange(t *testing.T, token string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", ExchangeAuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x?exchange_token="+url.QueryEscape(token), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// The mfa token of a half-done 2FA login is no session anywhere.
func TestAuthMiddleware_RejectsMFAToken(t *testing.T) {
	setAuthEnv(t)
	mfa, err := auth.GenerateMFAToken(1, "hash", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := serveStaff(t, AuthMiddleware(), mfa); code != http.StatusUnauthorized {
		t.Fatalf("AuthMiddleware with an mfa token: %d, want 401", code)
	}
}

func TestExchangeAuthMiddleware_RejectsMFAToken(t *testing.T) {
	setAuthEnv(t)
	mfa, _ := auth.GenerateMFAToken(1, "hash", time.Now())
	if code := serveExchange(t, mfa); code != http.StatusUnauthorized {
		t.Fatalf("ExchangeAuthMiddleware with an mfa token: %d, want 401", code)
	}
	exchange, _ := auth.GenerateExchangeToken("ana", 1)
	if code := serveExchange(t, exchange); code != http.StatusOK {
		t.Fatalf("ExchangeAuthMiddleware with an exchange token: %d, want 200", code)
	}
}

// OptionalAuthMiddleware only takes access tokens: not mfa, exchange or
// refresh tokens.
func TestOptionalAuthMiddleware_RejectsMFAAndNonAccessTokens(t *testing.T) {
	setAuthEnv(t)
	mfa, _ := auth.GenerateMFAToken(1, "hash", time.Now())
	exchange, _ := auth.GenerateExchangeToken("ana", 1)
	refresh, _, _ := auth.GenerateRefreshToken("ana", 1, "fam")
	for name, token := range map[string]string{"mfa": mfa, "exchange": exchange, "refresh": refresh} {
		if _, ok := serveStaff(t, OptionalAuthMiddleware(), token); ok {
			t.Fatalf("%s token authenticated optionally", name)
		}
	}
	access, _ := auth.GenerateToken("ana", 1)
	if _, ok := serveStaff(t, OptionalAuthMiddleware(), access); !ok {
		t.Fatal("access token not authenticated optionally")
	}
}
