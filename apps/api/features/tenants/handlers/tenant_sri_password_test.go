package tenant_handlers

import (
	"net/http"
	"testing"
	"time"

	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
	"pengi-med-saas/core/secretbox"
	tenant_models "pengi-med-saas/features/tenants/models"
)

// The SRI P12 password is stored sealed with SIGNATURE_ENCRYPTION_KEY, never in
// plaintext, and the legacy plaintext column is left empty.
func TestUploadSignature_StoresThePasswordSealed(t *testing.T) {
	resp, tenant, h := uploadSriSignature(t, pdfsigntest.P12(t, "PENGI", "p12-secret", time.Now().Add(24*time.Hour)), "p12-secret")
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%s), want 200", resp.Code, resp.Message)
	}
	var stored tenant_models.Tenant
	h.db.First(&stored, tenant.ID)
	if stored.SriPassword != "" {
		t.Fatalf("plaintext column = %q, want empty", stored.SriPassword)
	}
	if stored.SriPasswordEncrypted == "" || stored.SriPasswordEncrypted == "p12-secret" {
		t.Fatalf("sealed column = %q, want the sealed password", stored.SriPasswordEncrypted)
	}
	box, err := secretbox.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := box.Open(stored.SriPasswordEncrypted); err != nil || got != "p12-secret" {
		t.Fatalf("Open = %q, %v; want p12-secret", got, err)
	}
}

// Without the key (release mode) the upload fails; it never falls back to
// storing the password in plaintext, and the P12 is not written either.
func TestUploadSignature_WithoutEncryptionKey_IsRefused(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv(secretbox.EnvKey, "")

	resp, tenant, h := uploadSriSignature(t, pdfsigntest.P12(t, "PENGI", "ok", time.Now().Add(24*time.Hour)), "ok")
	appErr, _ := resp.Data.(core_errors.AppError)
	if resp.Code != http.StatusServiceUnavailable || resp.Message != "signature.error.unavailable" || appErr.ErrorCode != core_errors.ErrSignatureKeyUnavailable.ErrorCode {
		t.Fatalf("got %d %q %q, want 503 signature.error.unavailable %s", resp.Code, resp.Message, appErr.ErrorCode, core_errors.ErrSignatureKeyUnavailable.ErrorCode)
	}
	var stored tenant_models.Tenant
	h.db.First(&stored, tenant.ID)
	if stored.SriPassword != "" || stored.SriPasswordEncrypted != "" || stored.SriP12Path != "" {
		t.Fatalf("refused upload stored something: plain=%q sealed=%q path=%q", stored.SriPassword, stored.SriPasswordEncrypted, stored.SriP12Path)
	}
	if h.files.Exists(tenant.ID, signatureFileName) {
		t.Fatalf("refused signature file was written to disk")
	}
}

func TestGetSriStatus_IsConfiguredWithSealedPassword(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	cases := []struct {
		name   string
		tenant tenant_models.Tenant
		want   bool
	}{
		{"sealed password", tenant_models.Tenant{SriP12Path: "signature.p12", SriPasswordEncrypted: "sealed", SriCertExpiration: &future}, true},
		{"legacy plaintext password", tenant_models.Tenant{SriP12Path: "signature.p12", SriPassword: "x", SriCertExpiration: &future}, true},
		{"no password", tenant_models.Tenant{SriP12Path: "signature.p12", SriCertExpiration: &future}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sriStatusFor(t, tc.tenant)["is_configured"]; got != tc.want {
				t.Fatalf("is_configured = %v, want %v", got, tc.want)
			}
		})
	}
}
