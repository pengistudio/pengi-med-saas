package tenant_handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"pengi-med-saas/core/tenantfiles"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

var displayTokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]{32}$`)

// createDisplayTenant creates a tenant whose display token is token (a unique
// placeholder is used when token is empty, then cleared, to respect the unique index).
func createDisplayTenant(t *testing.T, h *TenantHandler, token string) tenant_models.Tenant {
	t.Helper()
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("display-%d", now), DisplayToken: token}
	if token == "" {
		tenant.DisplayToken = fmt.Sprintf("tmp-%d", now)
	}
	if err := h.db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if token == "" {
		if err := h.db.Model(&tenant).Update("display_token", "").Error; err != nil {
			t.Fatalf("clear token: %v", err)
		}
	}
	return tenant
}

// testFrontendURL is the FRONTEND_URL every display handler test runs with.
const testFrontendURL = "https://app.example.com"

func newDisplayHandler(t *testing.T) *TenantHandler {
	t.Helper()
	t.Setenv("FRONTEND_URL", testFrontendURL)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	return NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))
}

func getDisplayToken(t *testing.T, h *TenantHandler, tenantID uint) string {
	t.Helper()
	c, _ := testutils.NewGinContext(tenantID, 1)
	resp := h.GetDisplayToken(c)
	if resp.Code != http.StatusOK || resp.Message != "tenant.display_token.fetch.success" {
		t.Fatalf("got %d %q, want 200 tenant.display_token.fetch.success", resp.Code, resp.Message)
	}
	data, ok := resp.Data.(gin.H)
	if !ok {
		t.Fatalf("unexpected data %T", resp.Data)
	}
	token, _ := data["token"].(string)
	if want := testFrontendURL + "/display/waiting-room?token=" + token; data["display_url"] != want {
		t.Fatalf("display_url = %v, want %q", data["display_url"], want)
	}
	return token
}

func storedDisplayToken(t *testing.T, h *TenantHandler, tenantID uint) string {
	t.Helper()
	var stored tenant_models.Tenant
	if err := h.db.First(&stored, tenantID).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	return stored.DisplayToken
}

func uniqueCode() string {
	return tenant_models.NewDisplayToken()
}

func TestGetDisplayToken_ReturnsExistingCodeUnchanged(t *testing.T) {
	h := newDisplayHandler(t)
	code := uniqueCode()
	tenant := createDisplayTenant(t, h, code)

	if got := getDisplayToken(t, h, tenant.ID); got != code {
		t.Fatalf("token = %q, want existing %q", got, code)
	}
	if got := getDisplayToken(t, h, tenant.ID); got != code {
		t.Fatalf("second read token = %q, want %q", got, code)
	}
	if stored := storedDisplayToken(t, h, tenant.ID); stored != code {
		t.Fatalf("stored token changed to %q", stored)
	}
}

func TestGetDisplayToken_CreatesCodeWhenMissing(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		// Old 8-digit pairing codes were guessable: they are replaced, not served.
		{"legacy 8-digit code", fmt.Sprintf("%08d", time.Now().UnixNano()%100_000_000)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newDisplayHandler(t)
			tenant := createDisplayTenant(t, h, tc.token)

			got := getDisplayToken(t, h, tenant.ID)
			if !displayTokenRe.MatchString(got) {
				t.Fatalf("token = %q, want a 32-char base64url token", got)
			}
			if stored := storedDisplayToken(t, h, tenant.ID); stored != got {
				t.Fatalf("stored token = %q, want %q", stored, got)
			}
			if again := getDisplayToken(t, h, tenant.ID); again != got {
				t.Fatalf("second read token = %q, want the created %q", again, got)
			}
		})
	}
}

func TestGetDisplayToken_TenantIsolation(t *testing.T) {
	h := newDisplayHandler(t)
	codeA := uniqueCode()
	tenantA := createDisplayTenant(t, h, codeA)
	tenantB := createDisplayTenant(t, h, "")

	if got := getDisplayToken(t, h, tenantA.ID); got != codeA {
		t.Fatalf("tenant A token = %q, want %q", got, codeA)
	}
	gotB := getDisplayToken(t, h, tenantB.ID)
	if gotB == codeA || !displayTokenRe.MatchString(gotB) {
		t.Fatalf("tenant B token = %q, want its own token", gotB)
	}
	if stored := storedDisplayToken(t, h, tenantA.ID); stored != codeA {
		t.Fatalf("tenant A token changed to %q after tenant B read", stored)
	}
}

func TestGetDisplayToken_WithoutTenant(t *testing.T) {
	h := newDisplayHandler(t)
	c, _ := testutils.NewGinContext(0, 1)
	delete(c.Keys, "tenant_id")
	if resp := h.GetDisplayToken(c); resp.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", resp.Code)
	}
}

func TestGetDisplayToken_FailsWithoutFrontendURL(t *testing.T) {
	h := newDisplayHandler(t)
	t.Setenv("FRONTEND_URL", "not a url")
	tenant := createDisplayTenant(t, h, uniqueCode())

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	if resp := h.GetDisplayToken(c); resp.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", resp.Code)
	}
}
