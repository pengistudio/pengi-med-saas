package tenant_handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pengi-med-saas/core/tenantfiles"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func sriStatusFor(t *testing.T, tenant tenant_models.Tenant) gin.H {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	now := time.Now().UnixNano()
	tenant.Name = "Clinic"
	tenant.Slug = fmt.Sprintf("sri-status-%d", now)
	tenant.DisplayToken = fmt.Sprintf("tok-sri-status-%d", now)
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	h := NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	resp := h.GetSriStatus(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", resp.Code)
	}
	data, ok := resp.Data.(gin.H)
	if !ok {
		t.Fatalf("data = %#v, want gin.H", resp.Data)
	}
	return data
}

func TestGetSriStatus_IsConfigured(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(30 * 24 * time.Hour)
	cases := []struct {
		name       string
		tenant     tenant_models.Tenant
		configured bool
	}{
		{"no signature", tenant_models.Tenant{}, false},
		{"valid signature", tenant_models.Tenant{SriP12Path: "signature.p12", SriPassword: "x", SriCertExpiration: &future}, true},
		{"signature without known expiration", tenant_models.Tenant{SriP12Path: "signature.p12", SriPassword: "x"}, true},
		{"expired signature", tenant_models.Tenant{SriP12Path: "signature.p12", SriPassword: "x", SriCertExpiration: &past}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := sriStatusFor(t, tc.tenant)
			if data["is_configured"] != tc.configured {
				t.Fatalf("is_configured = %v, want %v", data["is_configured"], tc.configured)
			}
		})
	}
}

func TestGetSriStatus_ExpiredSignatureStillReportsItsExpirationDate(t *testing.T) {
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	data := sriStatusFor(t, tenant_models.Tenant{SriP12Path: "signature.p12", SriPassword: "x", SriCertExpiration: &past})

	got, ok := data["expiration_date"].(*time.Time)
	if !ok || got == nil || !got.Equal(past) {
		t.Fatalf("expiration_date = %#v, want %v", data["expiration_date"], past)
	}
}
