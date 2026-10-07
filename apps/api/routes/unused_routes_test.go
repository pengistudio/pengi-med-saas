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

// Routes no client called, removed so they stop widening the API surface:
// /auth/validate and /permissions* were public; the others duplicated data the
// clients already get elsewhere (enabled_features, /clinical/patients/follow-up,
// the backoffice subscriptions list).
func TestRemovedUnusedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	db := testutils.SetupTestDB(t, &user_models.User{})
	router := gin.New()
	RegisterRoutes(router.Group(""), db, nil)

	// Control: the list the web uses is still registered.
	kept := httptest.NewRecorder()
	router.ServeHTTP(kept, httptest.NewRequest(http.MethodGet, "/clinical/patients/follow-up", nil))
	if kept.Code == http.StatusNotFound {
		t.Fatal("GET /clinical/patients/follow-up = 404, the router was not built as expected")
	}

	for _, r := range []struct{ method, path string }{
		{http.MethodPost, "/auth/validate"},
		{http.MethodGet, "/permissions"},
		{http.MethodGet, "/permissions/categories"},
		{http.MethodGet, "/tenants/features"},
		{http.MethodGet, "/clinical/patients"},
		{http.MethodDelete, "/clinical/patients/delete-multiple/1"},
		{http.MethodGet, "/backoffice/subscriptions/company/1"},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(r.method, r.path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404 (route must not exist)", r.method, r.path, w.Code)
		}
	}
}
