package tenant_handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
	"pengi-med-saas/core/tenantfiles"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func uploadSriSignature(t *testing.T, p12 []byte, password string) (envelope.Response, tenant_models.Tenant, *TenantHandler) {
	t.Helper()
	return uploadSriSignatureFor(t, "", p12, password)
}

// uploadSriSignatureFor uploads p12 as the SRI signature of a new tenant whose RUC is taxID.
func uploadSriSignatureFor(t *testing.T, taxID string, p12 []byte, password string) (envelope.Response, tenant_models.Tenant, *TenantHandler) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("sri-sig-%d", now), DisplayToken: fmt.Sprintf("tok-sri-sig-%d", now), TaxID: taxID}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	h := NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("signature", "firma.p12")
	_, _ = fw.Write(p12)
	_ = w.WriteField("password", password)
	_ = w.Close()

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest(http.MethodPost, "/", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	return h.UploadSignature(c), tenant, h
}

func TestUploadSignature_RejectsWithSpecificReason(t *testing.T) {
	cases := []struct {
		name     string
		p12      []byte
		password string
		message  string
		code     string
	}{
		{"wrong password", pdfsigntest.P12(t, "PENGI", "ok", time.Now().Add(24*time.Hour)), "wrong", "signature.error.wrong_password", core_errors.ErrSignatureWrongPassword.ErrorCode},
		{"expired", pdfsigntest.P12(t, "PENGI", "ok", time.Now().Add(-time.Hour)), "ok", "signature.error.expired", core_errors.ErrSignatureExpired.ErrorCode},
		{"not a p12", []byte("not a p12 file"), "ok", "billing.sri.invalid_file", core_errors.ErrBillingInvalidSignatureFile.ErrorCode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, tenant, h := uploadSriSignature(t, tc.p12, tc.password)
			appErr, _ := resp.Data.(core_errors.AppError)
			if resp.Code != http.StatusBadRequest || resp.Message != tc.message || appErr.ErrorCode != tc.code {
				t.Fatalf("got %d %q %q, want 400 %q %q", resp.Code, resp.Message, appErr.ErrorCode, tc.message, tc.code)
			}
			var stored tenant_models.Tenant
			h.db.First(&stored, tenant.ID)
			if stored.SriP12Path != "" || stored.SriCertExpiration != nil {
				t.Fatalf("rejected signature was stored: path=%q", stored.SriP12Path)
			}
		})
	}
}

func TestUploadSignature_StoresValidCertificate(t *testing.T) {
	notAfter := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	resp, tenant, h := uploadSriSignature(t, pdfsigntest.P12(t, "PENGI", "ok", notAfter), "ok")
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%s), want 200", resp.Code, resp.Message)
	}
	var stored tenant_models.Tenant
	h.db.First(&stored, tenant.ID)
	if stored.SriCertExpiration == nil || !stored.SriCertExpiration.Equal(notAfter) {
		t.Fatalf("expiration = %v, want %v", stored.SriCertExpiration, notAfter)
	}
}

// bceCert is a Banco Central certificate: holder cédula plus, for a company
// certificate, the company RUC.
func bceCert(t *testing.T, cedula, ruc string) []byte {
	ext := map[string]string{"1.3.6.1.4.1.37947.3.1": cedula}
	if ruc != "" {
		ext["1.3.6.1.4.1.37947.3.11"] = ruc
	}
	return pdfsigntest.P12With(t, pdfsigntest.Cert{CommonName: "PENGI", Extensions: ext}, "ok", time.Now().Add(24*time.Hour))
}

func TestUploadSignature_RejectsCertificateOfAnotherTaxpayer(t *testing.T) {
	cases := []struct {
		name string
		p12  []byte
	}{
		{"doctor's personal certificate", bceCert(t, "1712345675", "")},
		{"another company's certificate", bceCert(t, "1712345675", "0990012345001")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, tenant, h := uploadSriSignatureFor(t, "1790012345001", tc.p12, "ok")
			appErr, _ := resp.Data.(core_errors.AppError)
			if resp.Code != http.StatusBadRequest || resp.Message != "signature.error.ruc_mismatch" || appErr.ErrorCode != core_errors.ErrSignatureRucMismatch.ErrorCode {
				t.Fatalf("got %d %q %q, want 400 signature.error.ruc_mismatch", resp.Code, resp.Message, appErr.ErrorCode)
			}
			var stored tenant_models.Tenant
			h.db.First(&stored, tenant.ID)
			if stored.SriP12Path != "" || stored.SriCertExpiration != nil {
				t.Fatalf("rejected signature was stored: path=%q", stored.SriP12Path)
			}
			if h.files.Exists(tenant.ID, signatureFileName) {
				t.Fatalf("rejected signature file was written to disk")
			}
		})
	}
}

func TestUploadSignature_AcceptsCertificateOfTheTenant(t *testing.T) {
	cases := []struct {
		name  string
		taxID string
		p12   []byte
	}{
		{"company certificate with the tenant RUC", "1790012345001", bceCert(t, "1712345675", "1790012345001")},
		{"persona natural: cédula + 001", "1712345675001", bceCert(t, "1712345675", "")},
		{"certificate without a recognisable id", "1790012345001", pdfsigntest.P12With(t, pdfsigntest.Cert{CommonName: "PENGI"}, "ok", time.Now().Add(24*time.Hour))},
		{"tenant without a RUC yet", "", bceCert(t, "1712345675", "0990012345001")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, tenant, h := uploadSriSignatureFor(t, tc.taxID, tc.p12, "ok")
			if resp.Code != http.StatusOK {
				t.Fatalf("code = %d (%s), want 200", resp.Code, resp.Message)
			}
			var stored tenant_models.Tenant
			h.db.First(&stored, tenant.ID)
			if stored.SriP12Path == "" {
				t.Fatalf("signature was not stored")
			}
		})
	}
}
