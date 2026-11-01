package ai_handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	ai_models "pengi-med-saas/features/ai/models"
	ai_services "pengi-med-saas/features/ai/services"
	settings_models "pengi-med-saas/features/settings/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestBackofficeAI_UsageGroupsTenantsAndMonth(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &settings_models.SystemSetting{}, &ai_models.AIUsageLog{})
	now := time.Now().UnixNano()
	a := tenant_models.Tenant{Name: "A", Slug: fmt.Sprintf("bo-ai-a-%d", now), DisplayToken: fmt.Sprintf("bo-ai-a-%d", now)}
	b := tenant_models.Tenant{Name: "B", Slug: fmt.Sprintf("bo-ai-b-%d", now), DisplayToken: fmt.Sprintf("bo-ai-b-%d", now)}
	for _, tn := range []*tenant_models.Tenant{&a, &b} {
		if err := db.Create(tn).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []struct {
		tenant uint
		at     time.Time
	}{{a.ID, time.Now()}, {a.ID, time.Now()}, {b.ID, time.Now()}, {b.ID, time.Now().AddDate(0, -2, 0)}} {
		if err := ai_services.LogUsage(db, ai_services.UsageEntry{TenantID: u.tenant, UserID: 1, Function: ai_models.FunctionSummary, Model: "m", PromptTokens: 100, CompletionTokens: 50}, u.at); err != nil {
			t.Fatal(err)
		}
	}
	h := NewBackofficeAIHandler(db, zap.NewNop())
	c, _ := testutils.NewGinContext(0, 1)
	c.Request = httptest.NewRequest(http.MethodGet, "/backoffice/ai/usage", nil)
	resp := h.GetUsage(c)
	report, ok := resp.Data.(UsageReport)
	if !ok || resp.Code != http.StatusOK {
		t.Fatalf("resp = %+v", resp)
	}
	uses := map[uint]int64{}
	for _, r := range report.Rows {
		uses[r.TenantID] = r.Uses
	}
	if uses[a.ID] != 2 || uses[b.ID] != 1 {
		t.Fatalf("uses = %v, want A=2 B=1 (last-month row excluded)", uses)
	}

	c, _ = testutils.NewGinContext(0, 1)
	c.Request = httptest.NewRequest(http.MethodGet, "/backoffice/ai/usage?month=bad", nil)
	if resp := h.GetUsage(c); resp.Code != http.StatusBadRequest {
		t.Fatalf("bad month = %d", resp.Code)
	}
}

func TestBackofficeAI_UpdateSettingsValidates(t *testing.T) {
	db := testutils.SetupTestDB(t, &settings_models.SystemSetting{})
	h := NewBackofficeAIHandler(db, zap.NewNop())
	put := func(body string) int {
		c, _ := testutils.NewGinContext(0, 1)
		c.Request = httptest.NewRequest(http.MethodPut, "/backoffice/ai/settings", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return h.UpdateSettings(c).Code
	}
	if code := put(`[{"function":"summary","model":"x-model","price_input":0.1,"price_output":0.2}]`); code != http.StatusOK {
		t.Fatalf("valid = %d", code)
	}
	if got := ai_services.ModelFor(db, ai_models.FunctionSummary); got != "x-model" {
		t.Fatalf("model = %q", got)
	}
	for _, body := range []string{`[{"function":"nope"}]`, `[{"function":"summary","price_input":-1}]`, `{}`} {
		if code := put(body); code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", body, code)
		}
	}
}

func TestBackofficeAI_ProviderSettings(t *testing.T) {
	db := testutils.SetupTestDB(t, &settings_models.SystemSetting{})
	t.Setenv("AI_API_KEY", "env-key-should-not-leak")
	t.Setenv(ai_services.EncryptionKeyEnv, "")
	h := NewBackofficeAIHandler(db, zap.NewNop())
	put := func(body string) envelope.Response {
		c, _ := testutils.NewGinContext(0, 1)
		c.Request = httptest.NewRequest(http.MethodPut, "/backoffice/ai/settings", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return h.UpdateSettings(c)
	}

	resp := put(`[{"function":"summary","base_url":"ftp://x.com"}]`)
	if resp.Code != http.StatusBadRequest || resp.Message != "backoffice.ai.settings.invalid_base_url" {
		t.Fatalf("bad base url = %d %s", resp.Code, resp.Message)
	}
	// A key can't be saved without AI_ENCRYPTION_KEY: clear 400, nothing stored.
	resp = put(`[{"function":"summary","model":"m","api_key":"sk-new-secret"}]`)
	if resp.Code != http.StatusBadRequest || resp.Message != "backoffice.ai.settings.encryption_key_missing" {
		t.Fatalf("no encryption key = %d %s", resp.Code, resp.Message)
	}
	if appErr, ok := resp.Data.(core_errors.AppError); !ok || appErr.ErrorCode != core_errors.ErrAIEncryptionUnavailable.ErrorCode {
		t.Fatalf("error data = %#v", resp.Data)
	}

	t.Setenv(ai_services.EncryptionKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	resp = put(`[{"function":"summary","base_url":"https://api.example.com/v1/","api_key":"sk-new-secret","reasoning_effort":"none"}]`)
	if resp.Code != http.StatusOK {
		t.Fatalf("valid = %d %s", resp.Code, resp.Message)
	}
	c, _ := testutils.NewGinContext(0, 1)
	for _, r := range []envelope.Response{resp, h.GetSettings(c)} {
		out, _ := json.Marshal(r)
		if strings.Contains(string(out), "sk-new-secret") || strings.Contains(string(out), "env-key-should-not-leak") {
			t.Fatalf("a key was returned: %s", out)
		}
		if !strings.Contains(string(out), `"has_api_key":true`) || !strings.Contains(string(out), `"base_url":"https://api.example.com/v1"`) {
			t.Fatalf("settings = %s", out)
		}
	}
}
