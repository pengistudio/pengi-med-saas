package tenant_handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/tenantfiles"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func newSettingsHandler(t *testing.T) (*TenantHandler, tenant_models.Tenant) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	h := NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("settings-%d", now), DisplayToken: fmt.Sprintf("tok-settings-%d", now)}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return h, tenant
}

func putSettings(t *testing.T, h *TenantHandler, tenantID uint, body string) (int, string) {
	t.Helper()
	c, _ := testutils.NewGinContext(tenantID, 1)
	c.Request = httptest.NewRequest("PUT", "/tenants/settings", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	resp := h.UpdateUISettings(c)
	return resp.Code, resp.Message
}

func storedAgenda(t *testing.T, h *TenantHandler, tenantID uint) tenant_models.AgendaSettings {
	t.Helper()
	var tenant tenant_models.Tenant
	if err := h.db.First(&tenant, tenantID).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	return tenant_models.ParseUISettings(tenant.UISettings).Agenda
}

func TestUpdateUISettings_AutoNoShowSinceIsServerSide(t *testing.T) {
	h, tenant := newSettingsHandler(t)

	// Defaults: off, 30 minutes, no since.
	c, _ := testutils.NewGinContext(tenant.ID, 1)
	got := h.GetUISettings(c).Data.(tenant_models.UISettings).Agenda
	if got.AutoNoShow || got.NoShowAfterMinutes != 30 || got.AutoNoShowSince != nil {
		t.Fatalf("default agenda = %+v", got)
	}

	// Turning it on sets since to now, ignoring the value the client sent.
	before := time.Now()
	code, msg := putSettings(t, h, tenant.ID, `{"agenda":{"auto_no_show":true,"no_show_after_minutes":45,"auto_no_show_since":"2020-01-01T00:00:00Z"}}`)
	if code != http.StatusOK {
		t.Fatalf("turn on = %d %s", code, msg)
	}
	agenda := storedAgenda(t, h, tenant.ID)
	if !agenda.AutoNoShow || agenda.NoShowAfterMinutes != 45 || agenda.AutoNoShowSince == nil || agenda.AutoNoShowSince.Before(before.Add(-time.Second)) {
		t.Fatalf("after turning on = %+v", agenda)
	}
	firstSince := *agenda.AutoNoShowSince

	// While it stays on, since is kept (also against a client value), and
	// other sections are saved as before.
	code, _ = putSettings(t, h, tenant.ID, `{"kanban":{"auto_archive_delay":"1_day"},"agenda":{"auto_no_show":true,"no_show_after_minutes":60,"auto_no_show_since":"2030-01-01T00:00:00Z"}}`)
	agenda = storedAgenda(t, h, tenant.ID)
	if code != http.StatusOK || agenda.NoShowAfterMinutes != 60 || agenda.AutoNoShowSince == nil || !agenda.AutoNoShowSince.Equal(firstSince) {
		t.Fatalf("staying on = %d %+v, want since %v", code, agenda, firstSince)
	}

	// A body without the agenda section leaves it as stored.
	code, _ = putSettings(t, h, tenant.ID, `{"kanban":{"auto_archive_delay":"never"}}`)
	agenda = storedAgenda(t, h, tenant.ID)
	if code != http.StatusOK || !agenda.AutoNoShow || !agenda.AutoNoShowSince.Equal(firstSince) {
		t.Fatalf("without agenda = %d %+v", code, agenda)
	}

	// Turning it off clears since, even if the client sends one.
	code, _ = putSettings(t, h, tenant.ID, `{"agenda":{"auto_no_show":false,"no_show_after_minutes":60,"auto_no_show_since":"2026-01-01T00:00:00Z"}}`)
	agenda = storedAgenda(t, h, tenant.ID)
	if code != http.StatusOK || agenda.AutoNoShow || agenda.AutoNoShowSince != nil {
		t.Fatalf("turn off = %d %+v", code, agenda)
	}

	// Turning it on again starts a new period.
	time.Sleep(10 * time.Millisecond)
	putSettings(t, h, tenant.ID, `{"agenda":{"auto_no_show":true,"no_show_after_minutes":60}}`)
	agenda = storedAgenda(t, h, tenant.ID)
	if agenda.AutoNoShowSince == nil || !agenda.AutoNoShowSince.After(firstSince) {
		t.Fatalf("turn on again = %+v, want since after %v", agenda, firstSince)
	}
}

func TestUpdateUISettings_RejectsNoShowMinutesOutOfRange(t *testing.T) {
	h, tenant := newSettingsHandler(t)
	for _, minutes := range []int{0, 4, 241} {
		code, msg := putSettings(t, h, tenant.ID, fmt.Sprintf(`{"agenda":{"auto_no_show":true,"no_show_after_minutes":%d}}`, minutes))
		if code != http.StatusBadRequest || msg != "tenant.settings.agenda.invalid_no_show_minutes" {
			t.Errorf("%d minutes = %d %s, want 400", minutes, code, msg)
		}
	}
	if agenda := storedAgenda(t, h, tenant.ID); agenda.AutoNoShow {
		t.Fatalf("a rejected update was stored: %+v", agenda)
	}
}
