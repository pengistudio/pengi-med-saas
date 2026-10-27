package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/tenantfiles"
	sri_document "pengi-med-saas/features/billing/sri-document"
	portal_handlers "pengi-med-saas/features/portal/handlers"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// The real /portal wiring: every patient endpoint needs a portal access token.
func TestPortalRoutes_PatientEndpointsNeedAPortalToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	files := tenantfiles.Memory()
	delivery := portal_services.NewDelivery(db, zap.NewNop(), nil, nil)
	r := gin.New()
	registerPortalRoutes(r.Group("/portal"), db,
		portal_handlers.NewAuthHandler(db, zap.NewNop(), files, delivery),
		portal_handlers.NewPatientHandler(db, zap.NewNop(), files, nil, sri_document.Documents{Files: files}),
		portal_handlers.NewBookingHandler(db, zap.NewNop(), nil, nil))

	staff, _ := auth.GenerateToken("admin", 1)
	backoffice, _ := auth.GenerateBackofficeToken("root", 1)
	paths := []string{"/portal/me", "/portal/appointments", "/portal/results", "/portal/results/files/1", "/portal/documents",
		"/portal/documents/prescription/1/file", "/portal/invoices", "/portal/invoices/1/ride", "/portal/invoices/1/xml",
		"/portal/booking/options", "/portal/booking/slots", "/portal/booking/requests"}
	for _, path := range paths {
		for _, token := range []string{"", staff, backoffice} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
			}
		}
	}
}
