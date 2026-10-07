package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/logger"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GET /users used to list every user of every company — with password hashes —
// to anyone, without authentication. No client uses it.
func TestUserRoutes_NoPublicUserListing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	db := testutils.SetupTestDB(t, &user_models.User{})
	router := gin.New()
	RegisterUserRoutes(router.Group(""), db)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /users = %d, want 404 (route must not exist)", w.Code)
	}
}

// POST /auth/signup bound the whole User model from an unauthenticated body
// and saved it with its associations (environments: company + role with
// permissions), so anyone could create a user inside any company. No client
// uses it; the company signup flow (/auth/signup/company*) must remain.
func TestUserRoutes_NoPublicRawSignup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	db := testutils.SetupTestDB(t, &user_models.User{})
	router := gin.New()
	RegisterUserRoutes(router.Group(""), db)

	raw := httptest.NewRecorder()
	router.ServeHTTP(raw, httptest.NewRequest(http.MethodPost, "/auth/signup", nil))
	if raw.Code != http.StatusNotFound {
		t.Fatalf("POST /auth/signup = %d, want 404 (route must not exist)", raw.Code)
	}

	company := httptest.NewRecorder()
	router.ServeHTTP(company, httptest.NewRequest(http.MethodPost, "/auth/signup/company", nil))
	if company.Code == http.StatusNotFound {
		t.Fatal("POST /auth/signup/company = 404, the company signup must remain")
	}
}

// GET /companies listed every company (legal and trade names, plan, owner) to
// anyone, without authentication; no client uses it. POST /companies (create
// your own company) must remain, behind authentication.
func TestCompanyRoutes_NoPublicCompanyListing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	db := testutils.SetupTestDB(t, &user_models.User{})
	router := gin.New()
	RegisterCompanyRoutes(router.Group(""), db)

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/companies", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET /companies = %d, want 404 (route must not exist)", get.Code)
	}

	post := httptest.NewRecorder()
	router.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/companies", nil))
	if post.Code != http.StatusUnauthorized {
		t.Fatalf("POST /companies without a token = %d, want 401", post.Code)
	}
}
