package portal_handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/tenantdb"
	portal_dto "pengi-med-saas/features/portal/dto"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestSettings_DefaultsUpdateAndPortalURL(t *testing.T) {
	t.Setenv("PORTAL_URL", "https://portal.example.com/")
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	own := tenant_models.Tenant{Name: "A", Slug: "clinica-a", DisplayToken: "tok-settings-a"}
	other := tenant_models.Tenant{Name: "B", Slug: "clinica-b", DisplayToken: "tok-settings-b"}
	for _, tn := range []*tenant_models.Tenant{&own, &other} {
		if err := tenantdb.System(db).Create(tn).Error; err != nil {
			t.Fatal(err)
		}
	}
	enabledFor := make(chan uint, 1)
	h := NewSettingsHandler(db, zap.NewNop(), func(id uint) { enabledFor <- id })

	c, _ := testutils.NewGinContext(own.ID, 1)
	c.Request = httptest.NewRequest(http.MethodGet, "/portal-settings", nil)
	resp := h.GetSettings(c)
	got := resp.Data.(portal_dto.SettingsResponse)
	want := tenant_models.DefaultPortalSettings()
	if resp.Code != http.StatusOK || got.PortalSettings != want || got.PortalURL != "https://portal.example.com/clinica-a" {
		t.Fatalf("defaults = %+v", got)
	}

	// A partial body keeps the fields it leaves out.
	c, _ = testutils.NewGinContext(own.ID, 1)
	c.Request = httptest.NewRequest(http.MethodPut, "/portal-settings", bytes.NewBufferString(`{"enabled":true,"sections":{"appointments":true,"results":true,"documents":true,"invoices":false}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	resp = h.UpdateSettings(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("update: %+v", resp)
	}
	if id := <-enabledFor; id != own.ID {
		t.Fatalf("onEnabled for %d", id)
	}
	var saved, untouched tenant_models.Tenant
	tenantdb.System(db).First(&saved, own.ID)
	tenantdb.System(db).First(&untouched, other.ID)
	s := saved.ParsePortalSettings()
	if !s.Enabled || s.Sections.Invoices || !s.Sections.Results {
		t.Fatalf("saved = %+v", s)
	}
	if untouched.ParsePortalSettings().Enabled {
		t.Fatal("another clinic's settings changed")
	}

	c, _ = testutils.NewGinContext(own.ID, 1)
	c.Request = httptest.NewRequest(http.MethodPut, "/portal-settings", bytes.NewBufferString(`{"sections":{"documents":false}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	got = h.UpdateSettings(c).Data.(portal_dto.SettingsResponse)
	if !got.Enabled || got.Sections.Invoices || got.Sections.Documents || !got.Sections.Appointments {
		t.Fatalf("partial update = %+v", got)
	}
	<-enabledFor

	c, _ = testutils.NewGinContext(own.ID, 1)
	c.Request = httptest.NewRequest(http.MethodPut, "/portal-settings", bytes.NewBufferString(`{"enabled":`))
	c.Request.Header.Set("Content-Type", "application/json")
	if resp := h.UpdateSettings(c); resp.Code != http.StatusBadRequest {
		t.Fatalf("bad body: %d", resp.Code)
	}
}
