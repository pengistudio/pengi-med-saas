package signature_handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/pdfsign/pdfsigntest"
	"pengi-med-saas/core/tenantfiles"
	signature_models "pengi-med-saas/features/signatures/models"
	signature_services "pengi-med-saas/features/signatures/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

type fixture struct {
	t      *testing.T
	db     *gorm.DB
	files  *tenantfiles.MemoryStore
	h      *SignatureHandler
	own    tenant_models.Tenant
	other  tenant_models.Tenant
	userID int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &signature_models.UserSignature{})
	files := tenantfiles.Memory()
	f := &fixture{t: t, db: db, files: files, h: NewSignatureHandler(db, zap.NewNop(), files), userID: time.Now().UnixNano() % 1_000_000}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&f.own, &f.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("sig-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-sig-%d-%d", i, now)}
		if err := db.Create(tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
	}
	return f
}

func (f *fixture) ctx(tenantID uint, body *bytes.Buffer, contentType string) *gin.Context {
	c, _ := testutils.NewGinContext(tenantID, f.userID)
	c.Set("user_id", f.userID)
	c.Set("username", "doctor")
	if body == nil {
		body = &bytes.Buffer{}
	}
	c.Request = httptest.NewRequest(http.MethodPut, "/", body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	return c
}

func (f *fixture) upload(tenantID uint, p12 []byte, password string) int {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("signature", "firma.p12")
	_, _ = fw.Write(p12)
	_ = w.WriteField("password", password)
	_ = w.Close()
	return f.h.UploadMySignature(f.ctx(tenantID, &body, w.FormDataContentType())).Code
}

func TestUploadMySignature_StoresEncryptedPassword(t *testing.T) {
	f := newFixture(t)
	p12 := pdfsigntest.P12(t, "DRA ANA PEREZ", "clave123", time.Now().Add(365*24*time.Hour))

	if code := f.upload(f.own.ID, p12, "clave123"); code != http.StatusOK {
		t.Fatalf("upload code = %d, want 200", code)
	}

	var sig signature_models.UserSignature
	if err := f.db.Unscoped().Where("tenant_id = ? AND user_id = ?", f.own.ID, f.userID).First(&sig).Error; err != nil {
		t.Fatalf("signature row: %v", err)
	}
	if sig.SubjectName != "DRA ANA PEREZ" || sig.SubjectSerial != "0912345678" {
		t.Fatalf("subject = %q / %q", sig.SubjectName, sig.SubjectSerial)
	}
	if sig.EncryptedPassword == "" || sig.EncryptedPassword == "clave123" {
		t.Fatalf("password not encrypted: %q", sig.EncryptedPassword)
	}
	if !f.files.Exists(f.own.ID, signature_services.P12FileName(uint(f.userID))) {
		t.Fatal("p12 not stored in tenant files")
	}
}

func TestUploadMySignature_RejectsWrongPasswordAndExpired(t *testing.T) {
	f := newFixture(t)
	valid := pdfsigntest.P12(t, "DR X", "ok", time.Now().Add(24*time.Hour))
	if code := f.upload(f.own.ID, valid, "wrong"); code != http.StatusBadRequest {
		t.Fatalf("wrong password code = %d, want 400", code)
	}
	expired := pdfsigntest.P12(t, "DR X", "ok", time.Now().Add(-time.Hour))
	if code := f.upload(f.own.ID, expired, "ok"); code != http.StatusBadRequest {
		t.Fatalf("expired code = %d, want 400", code)
	}
}

func TestGetMySignature_IsScopedToTenant(t *testing.T) {
	f := newFixture(t)
	p12 := pdfsigntest.P12(t, "DR X", "ok", time.Now().Add(24*time.Hour))
	if code := f.upload(f.own.ID, p12, "ok"); code != http.StatusOK {
		t.Fatalf("upload code = %d", code)
	}

	own := f.h.GetMySignature(f.ctx(f.own.ID, nil, "")).Data.(signatureStatus)
	other := f.h.GetMySignature(f.ctx(f.other.ID, nil, "")).Data.(signatureStatus)
	if !own.Configured {
		t.Fatal("own tenant should see the signature")
	}
	if other.Configured {
		t.Fatal("same user in another tenant must not see the signature")
	}
}

func TestDeleteMySignature_AllowsReupload(t *testing.T) {
	f := newFixture(t)
	p12 := pdfsigntest.P12(t, "DR X", "ok", time.Now().Add(24*time.Hour))
	f.upload(f.own.ID, p12, "ok")

	if code := f.h.DeleteMySignature(f.ctx(f.own.ID, nil, "")).Code; code != http.StatusOK {
		t.Fatalf("delete code = %d", code)
	}
	if f.files.Exists(f.own.ID, signature_services.P12FileName(uint(f.userID))) {
		t.Fatal("p12 file not removed")
	}
	if code := f.upload(f.own.ID, p12, "ok"); code != http.StatusOK {
		t.Fatalf("re-upload code = %d, want 200", code)
	}
}
