package config

import (
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"DB_HOST": "db", "DB_PORT": "5432", "DB_USER": "u", "DB_PASSWORD": "p", "DB_NAME": "n",
		"AUTH_KEY": "k", "AUTH_EXP": "60", "GIN_MODE": "debug", "APP_URL": "https://app.example.com",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadAcceptsCompleteEnvironment(t *testing.T) {
	setRequired(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthExp != 60 || cfg.DBSSLMode != "disable" || cfg.ContactRateLimit != 3 {
		t.Fatalf("unexpected values or defaults: %+v", cfg)
	}
}

func TestLoadRejectsMissingRequired(t *testing.T) {
	setRequired(t)
	t.Setenv("AUTH_KEY", "")
	t.Setenv("DB_HOST", "")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an environment without AUTH_KEY and DB_HOST")
	}
	for _, name := range []string{"AUTH_KEY", "DB_HOST"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoadRejectsNonPositiveAuthExp(t *testing.T) {
	setRequired(t)
	t.Setenv("AUTH_EXP", "0")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AUTH_EXP") {
		t.Fatalf("Load() error = %v, want an AUTH_EXP error", err)
	}
}

func TestLoadRejectsUnparsableNumber(t *testing.T) {
	setRequired(t)
	t.Setenv("CONTACT_RATE_LIMIT", "many")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CONTACT_RATE_LIMIT") {
		t.Fatalf("Load() error = %v, want a CONTACT_RATE_LIMIT error", err)
	}
}

func TestLoadRequiresSignatureKeyInRelease(t *testing.T) {
	setRequired(t)
	t.Setenv("GIN_MODE", "release")
	t.Setenv("SIGNATURE_ENCRYPTION_KEY", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SIGNATURE_ENCRYPTION_KEY") {
		t.Fatalf("Load() error = %v, want a SIGNATURE_ENCRYPTION_KEY error", err)
	}

	t.Setenv("SIGNATURE_ENCRYPTION_KEY", "c2VjcmV0")
	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v with the key set", err)
	}
}

func TestAppURLFallsBackToOldNames(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("APP_URL", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("PUBLIC_APP_URL", "")
	if got := Get().AppURL; got != devAppURL {
		t.Fatalf("AppURL = %q, want the dev default", got)
	}

	t.Setenv("PUBLIC_APP_URL", "https://public.example.com")
	t.Setenv("FRONTEND_URL", "https://frontend.example.com")
	if got := Get().AppURL; got != "https://frontend.example.com" {
		t.Fatalf("AppURL = %q, want FRONTEND_URL", got)
	}

	t.Setenv("APP_URL", "https://app.example.com")
	if got := Get().AppURL; got != "https://app.example.com" {
		t.Fatalf("AppURL = %q, want APP_URL", got)
	}
}

func TestLoadRequiresAppURLInRelease(t *testing.T) {
	setRequired(t)
	t.Setenv("GIN_MODE", "release")
	t.Setenv("SIGNATURE_ENCRYPTION_KEY", "c2VjcmV0")
	t.Setenv("APP_URL", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("PUBLIC_APP_URL", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "APP_URL") {
		t.Fatalf("Load() error = %v, want an APP_URL error", err)
	}
}

func TestGetUsesDefaultForEmptyValue(t *testing.T) {
	t.Setenv("GOTENBERG_URL", "")
	if got := Get().GotenbergURL; got != "http://gotenberg:3000" {
		t.Fatalf("GotenbergURL = %q, want the default", got)
	}
}
