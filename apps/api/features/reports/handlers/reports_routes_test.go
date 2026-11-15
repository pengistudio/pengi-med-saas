package reports_handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/routes"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
)

// Every report route sits behind authentication: without a token none of them
// reaches its handler, whatever the format.
func TestReportsRoutes_RequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t)
	r := gin.New()
	routes.RegisterReportsRoutes(r.Group("/api/v1"), db)

	for _, kind := range []string{"revenue", "receivables", "appointments", "clinical"} {
		for _, q := range []string{"", "?format=csv", "?format=pdf"} {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+kind+q, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
				t.Errorf("%s%s without a token = %d, want 401/403", kind, q, w.Code)
			}
		}
	}
}
