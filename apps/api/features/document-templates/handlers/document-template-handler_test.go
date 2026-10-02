package document_template_handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	billing_templates "pengi-med-saas/features/billing/templates"
	clinical_templates "pengi-med-saas/features/clinical/templates"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

type fakeConverter struct{ html string }

func (f *fakeConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	f.html = html
	return []byte("%PDF-fake"), nil
}

type fixture struct {
	t        *testing.T
	files    *tenantfiles.MemoryStore
	conv     *fakeConverter
	h        *DocumentTemplateHandler
	features tenant_models.EnabledFeatures
}

const tenantID uint = 7

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, files: tenantfiles.Memory(), conv: &fakeConverter{}, features: tenant_models.DefaultEnabledFeatures()}
	renderer := pdfrender.New(f.files, f.conv, clinical_templates.Prescription, clinical_templates.Report, clinical_templates.Certificate, billing_templates.InvoiceRide)
	f.h = NewDocumentTemplateHandler(nil, zap.NewNop(), renderer)
	f.h.enabledFeatures = func(*gin.Context) (tenant_models.EnabledFeatures, error) { return f.features, nil }
	return f
}

// request builds a context for :doc, with an optional multipart "template".
func (f *fixture) request(method, doc string, template []byte) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(tenantID, 1)
	var body bytes.Buffer
	contentType := ""
	if template != nil {
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("template", "plantilla.html")
		_, _ = fw.Write(template)
		_ = mw.Close()
		contentType = mw.FormDataContentType()
	}
	c.Request = httptest.NewRequest(method, "/", &body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	if doc != "" {
		c.Params = gin.Params{{Key: "doc", Value: doc}}
	}
	return c, w
}

func defaultSource(t *testing.T, doc pdfrender.Document) []byte {
	t.Helper()
	src, err := doc.DefaultSource()
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func appError(t *testing.T, resp envelope.Response) core_errors.AppError {
	t.Helper()
	appErr, ok := resp.Data.(core_errors.AppError)
	if !ok {
		t.Fatalf("data = %#v, want an AppError", resp.Data)
	}
	return appErr
}

// jsonError decodes an error written by a streaming handler.
func jsonError(t *testing.T, w *httptest.ResponseRecorder) (string, core_errors.AppError) {
	t.Helper()
	var body struct {
		Message string               `json:"message"`
		Data    core_errors.AppError `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return body.Message, body.Data
}

func TestList_FiltersByEnabledFeaturesAndReportsCustom(t *testing.T) {
	f := newFixture(t)
	_ = f.files.Write(tenantID, "prescription_template.html", []byte("<p>propia</p>"))
	f.features.Billing = false

	c, _ := f.request(http.MethodGet, "", nil)
	resp := f.h.ListDocumentTemplates(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d", resp.Code)
	}
	got := resp.Data.([]DocumentTemplate)
	want := []DocumentTemplate{
		{ID: "prescription", HasCustom: true, Paper: "a5_landscape"},
		{ID: "medical_report", Paper: "a4_portrait"},
		{ID: "medical_certificate", Paper: "a4_portrait"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestUpload_StoresAValidTemplate(t *testing.T) {
	f := newFixture(t)
	src := defaultSource(t, billing_templates.InvoiceRide)

	c, _ := f.request(http.MethodPut, "invoice_ride", src)
	resp := f.h.UploadDocumentTemplate(c)
	if resp.Code != http.StatusOK || resp.Message != "document_templates.upload.success" {
		t.Fatalf("resp = %+v", resp)
	}
	if stored, err := f.files.Read(tenantID, "invoice_ride_template.html"); err != nil || !bytes.Equal(stored, src) {
		t.Fatalf("stored template = %d bytes, err %v", len(stored), err)
	}
	if f.files.Exists(tenantID+1, "invoice_ride_template.html") {
		t.Fatalf("template leaked to another tenant")
	}
}

func TestUpload_RejectsInvalidTemplatesWithTheFailingRule(t *testing.T) {
	cases := []struct {
		name, src, message, code, detail string
	}{
		{"parse", `<p>{{.PatientName</p>`, "document_templates.error.parse", "E-DOCTPL-005", "prescription_template.html:1"},
		{"unknown field", `<p>{{.Pacient}}</p>`, "document_templates.error.execute", "E-DOCTPL-006", ".Pacient"},
		{"no signature", `<p>{{.PatientName}}</p>`, "document_templates.error.required", "E-DOCTPL-007", ".Signature.QR"},
		{"external url", `<img src="https://example.com/logo.png">`, "document_templates.error.external_url", "E-DOCTPL-008", "https://example.com/logo.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			c, _ := f.request(http.MethodPut, "prescription", []byte(tc.src))
			resp := f.h.UploadDocumentTemplate(c)
			appErr := appError(t, resp)
			if resp.Code != http.StatusBadRequest || resp.Message != tc.message || appErr.ErrorCode != tc.code || !strings.Contains(appErr.Detail, tc.detail) {
				t.Fatalf("resp = %d %s %+v", resp.Code, resp.Message, appErr)
			}
			if f.files.Exists(tenantID, "prescription_template.html") {
				t.Fatalf("an invalid template was stored")
			}
		})
	}
}

func TestUpload_RejectsMissingAndTooLargeFiles(t *testing.T) {
	f := newFixture(t)
	c, _ := f.request(http.MethodPut, "prescription", nil)
	if resp := f.h.UploadDocumentTemplate(c); resp.Code != http.StatusBadRequest || appError(t, resp).ErrorCode != core_errors.ErrDocTemplateFileRequired.ErrorCode {
		t.Fatalf("missing file resp = %+v", resp)
	}

	big := append(defaultSource(t, clinical_templates.Prescription), bytes.Repeat([]byte(" "), pdfrender.MaxTemplateSize)...)
	c, _ = f.request(http.MethodPut, "prescription", big)
	if resp := f.h.UploadDocumentTemplate(c); resp.Code != http.StatusRequestEntityTooLarge || appError(t, resp).ErrorCode != core_errors.ErrDocTemplateTooLarge.ErrorCode {
		t.Fatalf("too large resp = %+v", resp)
	}
}

func TestUnknownOrDisabledDocumentIs404(t *testing.T) {
	f := newFixture(t)
	f.features.Billing = false

	for _, doc := range []string{"nope", "invoice_ride"} {
		c, _ := f.request(http.MethodPut, doc, defaultSource(t, billing_templates.InvoiceRide))
		if resp := f.h.UploadDocumentTemplate(c); resp.Code != http.StatusNotFound || resp.Message != "document_templates.error.not_found" {
			t.Fatalf("%s upload resp = %+v", doc, resp)
		}
		c, w := f.request(http.MethodGet, doc, nil)
		f.h.DownloadDefaultTemplate(c)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s download code = %d", doc, w.Code)
		}
	}
}

func TestDownloads(t *testing.T) {
	f := newFixture(t)

	c, w := f.request(http.MethodGet, "medical_report", nil)
	f.h.DownloadDefaultTemplate(c)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), defaultSource(t, clinical_templates.Report)) ||
		!strings.Contains(w.Header().Get("Content-Disposition"), "medical_report_template.html") {
		t.Fatalf("default download = %d %q", w.Code, w.Header().Get("Content-Disposition"))
	}

	c, w = f.request(http.MethodGet, "medical_report", nil)
	f.h.DownloadCustomTemplate(c)
	if msg, appErr := jsonError(t, w); w.Code != http.StatusNotFound || msg != "document_templates.error.no_custom" || appErr.ErrorCode != "E-DOCTPL-002" {
		t.Fatalf("custom download without custom = %d %s", w.Code, w.Body.String())
	}

	_ = f.files.Write(tenantID, "medical_report_template.html", []byte("<p>propia</p>"))
	c, w = f.request(http.MethodGet, "medical_report", nil)
	f.h.DownloadCustomTemplate(c)
	if w.Code != http.StatusOK || w.Body.String() != "<p>propia</p>" {
		t.Fatalf("custom download = %d %q", w.Code, w.Body.String())
	}
}

func TestDelete_RestoresTheDefault(t *testing.T) {
	f := newFixture(t)
	_ = f.files.Write(tenantID, "medical_certificate_template.html", []byte("<p>propia</p>"))

	c, _ := f.request(http.MethodDelete, "medical_certificate", nil)
	if resp := f.h.DeleteDocumentTemplate(c); resp.Code != http.StatusOK {
		t.Fatalf("resp = %+v", resp)
	}
	if f.files.Exists(tenantID, "medical_certificate_template.html") {
		t.Fatalf("custom template still there")
	}
}

func TestPreview(t *testing.T) {
	t.Run("effective template with sample data", func(t *testing.T) {
		f := newFixture(t)
		c, w := f.request(http.MethodPost, "medical_certificate", nil)
		f.h.PreviewDocumentTemplate(c)
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" || w.Body.String() != "%PDF-fake" {
			t.Fatalf("preview = %d %q", w.Code, w.Body.String())
		}
		if !strings.Contains(f.conv.html, "Juan Carlos Pérez Mora") {
			t.Fatalf("preview did not render the sample patient")
		}
	})

	t.Run("uploaded template is validated and not stored", func(t *testing.T) {
		f := newFixture(t)
		src := []byte(`<h1>{{.PatientName}}</h1>{{if .Signature}}<img src="{{.Signature.QR}}">{{.Signature.Name}}{{end}}`)
		c, w := f.request(http.MethodPost, "medical_certificate", src)
		f.h.PreviewDocumentTemplate(c)
		if w.Code != http.StatusOK || !strings.HasPrefix(f.conv.html, "<h1>Juan Carlos Pérez Mora</h1>") {
			t.Fatalf("preview = %d html %.60q", w.Code, f.conv.html)
		}
		if f.files.Exists(tenantID, "medical_certificate_template.html") {
			t.Fatalf("preview stored the template")
		}
	})

	t.Run("invalid uploaded template", func(t *testing.T) {
		f := newFixture(t)
		c, w := f.request(http.MethodPost, "medical_certificate", []byte(`<p>{{.Nope}}</p>`))
		f.h.PreviewDocumentTemplate(c)
		if msg, appErr := jsonError(t, w); w.Code != http.StatusBadRequest || msg != "document_templates.error.execute" || appErr.Detail != ".Nope" {
			t.Fatalf("preview = %d %s", w.Code, w.Body.String())
		}
	})
}
