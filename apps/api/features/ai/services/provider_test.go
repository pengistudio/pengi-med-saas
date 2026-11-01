package ai_services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"pengi-med-saas/core/ai"
	"pengi-med-saas/core/tenantdb"
	ai_models "pengi-med-saas/features/ai/models"
	settings_models "pengi-med-saas/features/settings/models"
)

const testDBKey = "sk-db-secret-0123456789"

// setEncryptionKey sets a fresh random AI_ENCRYPTION_KEY for the test.
func setEncryptionKey(t *testing.T) {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EncryptionKeyEnv, base64.StdEncoding.EncodeToString(raw))
}

func TestResolveProvider_FallbackOrder(t *testing.T) {
	f := newFixture(t, true)
	sys := tenantdb.System(f.db)
	t.Setenv("AI_BASE_URL", "https://env.example.com/v1/")
	t.Setenv("AI_REASONING_EFFORT", "low")

	p, err := ResolveProvider(sys, ai_models.FunctionSummary)
	if err != nil {
		t.Fatal(err)
	}
	if p.BaseURL != "https://env.example.com/v1" || p.APIKey != "env-test-key" || p.KeySource != KeySourceEnv ||
		p.Model != DefaultModel(ai_models.FunctionSummary) || p.ReasoningEffort != "low" {
		t.Fatalf("env only: %+v", p)
	}

	// Same host as the env (trailing slash aside): the env key still applies.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: "https://env.example.com/v1/", ReasoningEffort: "none", Model: "m-db"}))
	p, _ = ResolveProvider(sys, ai_models.FunctionSummary)
	if p.KeySource != KeySourceEnv || p.ReasoningEffort != "none" || p.Model != "m-db" {
		t.Fatalf("same base: %+v", p)
	}

	// Another host without its own key: the env key is never sent there.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: "https://other.example.com/v1"}))
	p, _ = ResolveProvider(sys, ai_models.FunctionSummary)
	if p.Configured() || p.APIKey != "" || p.KeySource != KeySourceNone || p.BaseURL != "https://other.example.com/v1" {
		t.Fatalf("other base, no key: %+v", p)
	}

	// With its key stored, backoffice wins everywhere.
	setEncryptionKey(t)
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: "https://other.example.com/v1", APIKey: testDBKey}))
	p, _ = ResolveProvider(sys, ai_models.FunctionSummary)
	if p.APIKey != testDBKey || p.KeySource != KeySourceDB || !p.Configured() || p.Host() != "other.example.com" {
		t.Fatalf("db key: %+v", p.KeySource)
	}
	// Model and effort cleared: back to the env defaults.
	if p.Model != DefaultModel(ai_models.FunctionSummary) || p.ReasoningEffort != "low" {
		t.Fatalf("cleared fields must fall back: model %q effort %q", p.Model, p.ReasoningEffort)
	}
}

func TestSaveSettings_KeyIsSealedWriteOnlyAndClearable(t *testing.T) {
	f := newFixture(t, true)
	sys := tenantdb.System(f.db)
	setEncryptionKey(t)
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, APIKey: testDBKey}))

	var rows []settings_models.SystemSetting
	f.must(sys.Find(&rows).Error)
	for _, r := range rows {
		if strings.Contains(r.Value, testDBKey) {
			t.Fatalf("key stored in clear under %s", r.Key)
		}
	}
	out, err := json.Marshal(Settings(sys))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), testDBKey) || strings.Contains(string(out), "env-test-key") {
		t.Fatalf("settings serialize a key: %s", out)
	}
	p, err := ResolveProvider(sys, ai_models.FunctionSummary)
	if err != nil || p.APIKey != testDBKey {
		t.Fatalf("round trip: %v", err)
	}
	if raw, _ := json.Marshal(p); strings.Contains(string(raw), testDBKey) {
		t.Fatalf("Provider serializes its key: %s", raw)
	}
	s := Settings(sys)[0]
	if !s.HasAPIKey || s.APIKeySource != KeySourceDB || !s.EncryptionAvailable || !s.EnvHasAPIKey {
		t.Fatalf("settings flags: %+v", s)
	}

	// Saving again without a key keeps it.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, Model: "m2"}))
	if p, _ := ResolveProvider(sys, ai_models.FunctionSummary); p.APIKey != testDBKey {
		t.Fatal("an empty key must keep the stored one")
	}
	// Clearing it goes back to the env key.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, ClearAPIKey: true}))
	p, _ = ResolveProvider(sys, ai_models.FunctionSummary)
	if p.KeySource != KeySourceEnv || Settings(sys)[0].HasAPIKey {
		t.Fatalf("after clear: %s", p.KeySource)
	}

	// A key sealed with another AI_ENCRYPTION_KEY can't be used.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, APIKey: testDBKey}))
	setEncryptionKey(t)
	if _, err := ResolveProvider(sys, ai_models.FunctionSummary); err == nil {
		t.Fatal("a key sealed with another AI_ENCRYPTION_KEY must not open")
	}
	f.addRecord("Control")
	if _, err := f.generate(NewSummarizer(nil)); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unreadable key: err = %v, want ErrNotConfigured", err)
	}
}

func TestSaveSettings_KeyRefusedWithoutEncryptionKey(t *testing.T) {
	f := newFixture(t, true) // AI_ENCRYPTION_KEY empty
	sys := tenantdb.System(f.db)
	err := SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, Model: "m", APIKey: testDBKey})
	if !errors.Is(err, ErrEncryptionUnavailable) {
		t.Fatalf("err = %v, want ErrEncryptionUnavailable", err)
	}
	if ModelFor(sys, ai_models.FunctionSummary) == "m" {
		t.Fatal("a refused save must store nothing")
	}
	// The env key keeps working, and settings without a key still save.
	if p, _ := ResolveProvider(sys, ai_models.FunctionSummary); p.KeySource != KeySourceEnv {
		t.Fatalf("env fallback: %s", p.KeySource)
	}
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, Model: "m"}))
	if Settings(sys)[0].EncryptionAvailable {
		t.Fatal("encryption must be reported unavailable")
	}
}

func TestValidateUpdate(t *testing.T) {
	for _, base := range []string{"ftp://x.com", "not a url", "http://", "https://u:p@h.com/v1", "https://h.com/v1?x=1", "//h.com"} {
		u := SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: base}
		if err := ValidateUpdate(&u); !errors.Is(err, ErrInvalidBaseURL) {
			t.Errorf("base %q: err = %v", base, err)
		}
	}
	u := SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: "  https://api.example.com/v1/ ", Model: " m ", ReasoningEffort: " high "}
	if err := ValidateUpdate(&u); err != nil || u.BaseURL != "https://api.example.com/v1" || u.Model != "m" || u.ReasoningEffort != "high" {
		t.Fatalf("normalize: %+v %v", u, err)
	}
	for _, bad := range []SettingsUpdate{
		{Function: "nope"},
		{Function: ai_models.FunctionSummary, ReasoningEffort: "High!"},
		{Function: ai_models.FunctionSummary, PriceInput: -1},
	} {
		if err := ValidateUpdate(&bad); err == nil {
			t.Errorf("%+v must fail", bad)
		}
	}
}

// The real dialer uses the backoffice provider (a fake server on another URL
// than the env's), its key, model and reasoning effort, and logs its host.
func TestGenerate_UsesBackofficeProvider(t *testing.T) {
	var (
		mu     sync.Mutex
		gotReq ai.ChatRequest
		gotKey string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotKey = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"registrar_resumen","arguments":"{\"overview\":\"Resumen\",\"active_problems\":[],\"current_medications\":[],\"alerts\":[]}"}}]}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer srv.Close()

	f := newFixture(t, true) // env: http://127.0.0.1:1/v1 + env-test-key (unreachable)
	f.addRecord("Control")
	sys := tenantdb.System(f.db)
	setEncryptionKey(t)
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, BaseURL: srv.URL + "/v1", APIKey: testDBKey, Model: "db-model", ReasoningEffort: "none"}))

	sum, err := f.generate(NewSummarizer(nil))
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotKey != "Bearer "+testDBKey || gotReq.Model != "db-model" || gotReq.ReasoningEffort != "none" || sum.Model != "db-model" {
		t.Fatalf("request: key ok=%v model=%q effort=%q", gotKey == "Bearer "+testDBKey, gotReq.Model, gotReq.ReasoningEffort)
	}
	var row ai_models.AIUsageLog
	f.must(f.tdb().First(&row).Error)
	host, _ := url.Parse(srv.URL)
	if row.ProviderHost != host.Host || row.Model != "db-model" {
		t.Fatalf("usage row host %q model %q", row.ProviderHost, row.Model)
	}
}
