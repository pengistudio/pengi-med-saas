package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/logger"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The backoffice and the clinic web app used to issue identical access tokens
// (same key, same claims) and share one AuthMiddleware, so any clinic user's
// token opened the whole backoffice API.

func backofficeRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	db := testutils.SetupTestDB(t, &user_models.User{}, &backoffice_models.BackofficeUser{},
		&company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{})
	router := gin.New()
	RegisterBackofficeRoutes(router.Group(""), db)
	RegisterCompanyRoutes(router.Group(""), db)
	return router, db
}

func createBackofficeAdmin(t *testing.T, db *gorm.DB) {
	t.Helper()
	admin := backoffice_models.BackofficeUser{Name: "Admin", UserName: "admin", Password: "s3cret-pass"}
	if err := admin.Save(db); err != nil {
		t.Fatal(err)
	}
}

// backofficeLogin logs in through the real endpoint and returns the access
// token and the Set-Cookie headers.
func backofficeLogin(t *testing.T, router *gin.Engine) (string, []*http.Cookie) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/backoffice/auth/login",
		strings.NewReader(`{"user_name":"admin","password":"s3cret-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data.Token, w.Result().Cookies()
}

func call(router *gin.Engine, method, path, token string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestBackofficeAuth_AdminTokenOpensBackoffice(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)
	token, _ := backofficeLogin(t, router)

	if w := call(router, http.MethodGet, "/backoffice/companies", token); w.Code != http.StatusOK {
		t.Fatalf("GET /backoffice/companies with an admin token = %d, want 200", w.Code)
	}
}

func TestBackofficeAuth_WebUserTokenIsRejected(t *testing.T) {
	router, _ := backofficeRouter(t)
	webToken, err := auth.GenerateToken("doctora.clinica", 1)
	if err != nil {
		t.Fatal(err)
	}

	if w := call(router, http.MethodGet, "/backoffice/companies", webToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /backoffice/companies with a web user's token = %d, want 401", w.Code)
	}
}

func TestBackofficeAuth_RefreshTokenIsNotAnAccessToken(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)
	_, cookies := backofficeLogin(t, router)
	var refresh string
	for _, c := range cookies {
		if strings.Contains(c.Name, "refresh") {
			refresh = c.Value
		}
	}
	if refresh == "" {
		t.Fatal("login set no refresh cookie")
	}

	if w := call(router, http.MethodGet, "/backoffice/companies", refresh); w.Code != http.StatusUnauthorized {
		t.Fatalf("backoffice refresh token used as access token = %d, want 401", w.Code)
	}
}

func TestBackofficeAuth_TokenOfDeletedAdminIsRejected(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)
	token, _ := backofficeLogin(t, router)
	if err := db.Where("user_name = ?", "admin").Delete(&backoffice_models.BackofficeUser{}).Error; err != nil {
		t.Fatal(err)
	}

	if w := call(router, http.MethodGet, "/backoffice/companies", token); w.Code != http.StatusUnauthorized {
		t.Fatalf("token of a deleted admin = %d, want 401", w.Code)
	}
}

func TestBackofficeAuth_AdminTokenIsRejectedByTheWebApp(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)
	token, _ := backofficeLogin(t, router)

	if w := call(router, http.MethodPost, "/companies", token); w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /companies with a backoffice token = %d, want 401", w.Code)
	}
}

func TestBackofficeAuth_RefreshIssuesAWorkingToken(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)
	_, cookies := backofficeLogin(t, router)

	w := call(router, http.MethodPost, "/backoffice/auth/refresh", "", cookies...)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh with the login cookie = %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w := call(router, http.MethodGet, "/backoffice/companies", body.Data.Token); w.Code != http.StatusOK {
		t.Fatalf("refreshed token on the backoffice = %d, want 200", w.Code)
	}
}

func TestBackofficeAuth_WebRefreshTokenCannotRefreshTheBackoffice(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db) // backoffice user 1, same ID as the web user below
	webRefresh, _, err := auth.GenerateRefreshToken("doctora.clinica", 1, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"refresh_token", "backoffice_refresh_token"} {
		w := call(router, http.MethodPost, "/backoffice/auth/refresh", "", &http.Cookie{Name: name, Value: webRefresh})
		if w.Code == http.StatusOK {
			t.Fatalf("backoffice refresh with a web refresh token in %q = 200, want rejection", name)
		}
	}
}

// Anyone could create a backoffice admin: POST /backoffice/auth/signup had no
// authentication. Admins are created from the backoffice (/backoffice/users).
func TestBackofficeAuth_NoPublicSignup(t *testing.T) {
	router, _ := backofficeRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/backoffice/auth/signup",
		strings.NewReader(`{"name":"x","user_name":"intruso","password":"intruso123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("POST /backoffice/auth/signup = %d, want 404 (route must not exist)", w.Code)
	}
}

// A clinic refresh token (30-day sliding window, revocable) must not work as
// an access token: that would bypass refresh rotation and revocation.
func TestWebAuth_RefreshTokenIsNotAnAccessToken(t *testing.T) {
	router, _ := backofficeRouter(t)
	webRefresh, _, err := auth.GenerateRefreshToken("doctora.clinica", 1, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}

	if w := call(router, http.MethodPost, "/companies", webRefresh); w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /companies with a refresh token = %d, want 401", w.Code)
	}
}
