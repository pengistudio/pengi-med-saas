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
