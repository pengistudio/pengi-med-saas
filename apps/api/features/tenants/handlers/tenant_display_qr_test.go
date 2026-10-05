package tenant_handlers

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantfiles"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

// qrRouter mounts GET /display-token/qr behind the same role gate as
// routes/tenant-routes.go, for a caller of tenant whose role grants permission.
func qrRouter(t *testing.T, permission string) (*gin.Engine, *TenantHandler, tenant_models.Tenant) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{})
	h := NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))
	tenant := createDisplayTenant(t, h, uniqueCode())

	role := user_models.Role{Role: "Caller", Permissions: []permission_models.Permission{
		{BaseStringID: database.BaseStringID{ID: permission}, Name: permission},
	}}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	if err := db.Create(&env).Error; err != nil {
		t.Fatalf("create environment: %v", err)
	}

	r := gin.New()
	r.GET("/display-token/qr", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
	}, subscription_middleware.RequireRolePermission(db, "MANAGE_TEAM_MEMBERS"), h.GetDisplayTokenQR)
	return r, h, tenant
}

func getQR(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/display-token/qr", nil)
	// The link base must come from FRONTEND_URL, never from the request.
	req.Host = "evil.example"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetDisplayTokenQR_AdminGetsPNGOfTheTVLink(t *testing.T) {
	t.Setenv("FRONTEND_URL", "https://app.example.com/")
	r, _, tenant := qrRouter(t, "MANAGE_TEAM_MEMBERS")

	w := getQR(r)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type %q, want image/png", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control %q, want no-store", cc)
	}
	img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != displayQRSize || b.Dy() != displayQRSize {
		t.Fatalf("size %v, want %dx%d", b, displayQRSize, displayQRSize)
	}

	// go-qrcode cannot decode, but encoding is deterministic: the body must be
	// exactly the QR of the expected link (and not of any other).
	wantURL := "https://app.example.com/display/waiting-room?token=" + tenant.DisplayToken
	if got := DisplayURL("https://app.example.com/", tenant.DisplayToken); got != wantURL {
		t.Fatalf("DisplayURL = %q, want %q", got, wantURL)
	}
	want, err := qrcode.Encode(wantURL, qrcode.Medium, displayQRSize)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(w.Body.Bytes(), want) {
		t.Fatal("PNG does not encode the expected TV link")
	}
	other, _ := qrcode.Encode("https://evil.example/display/waiting-room?token="+tenant.DisplayToken, qrcode.Medium, displayQRSize)
	if bytes.Equal(w.Body.Bytes(), other) {
		t.Fatal("PNG encodes a link built from the request Host")
	}
}

func TestGetDisplayTokenQR_ForbiddenWithoutManageTeamMembers(t *testing.T) {
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	r, _, _ := qrRouter(t, "READ_PATIENT")

	w := getQR(r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct == "image/png" {
		t.Fatal("403 must not carry the QR")
	}
}

func TestGetDisplayTokenQR_FailsWithoutFrontendURL(t *testing.T) {
	t.Setenv("FRONTEND_URL", "")
	r, _, _ := qrRouter(t, "MANAGE_TEAM_MEMBERS")

	if w := getQR(r); w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", w.Code)
	}
}

func TestGetDisplayTokenQR_DoesNotRotateTheToken(t *testing.T) {
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	r, h, tenant := qrRouter(t, "MANAGE_TEAM_MEMBERS")

	if w := getQR(r); w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	if got := storedDisplayToken(t, h, tenant.ID); got != tenant.DisplayToken {
		t.Fatalf("token changed to %q, want %q", got, tenant.DisplayToken)
	}
}
