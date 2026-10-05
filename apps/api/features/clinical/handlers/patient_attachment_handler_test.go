package clinical_handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// attachments drives the attachment routes over HTTP with two clinics. Requests
// act as a member of "own"; "other" holds data that must stay out of reach.
type attachments struct {
	t            *testing.T
	db           *gorm.DB
	disk         *tenantfiles.MemoryStore // what is at rest, below the encryption
	router       *gin.Engine
	own, other   tenant_models.Tenant
	ownPatient   clinical_models.Patient
	otherPatient clinical_models.Patient
	plan         company_models.Plan // the plan of own's active subscription
}

const (
	permRead   = "READ_PATIENT_ATTACHMENT"
	permUpload = "UPLOAD_PATIENT_ATTACHMENT"
	permDelete = "DELETE_PATIENT_ATTACHMENT"
)

var samplePDF = []byte("%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")

func newAttachments(t *testing.T) *attachments {
	t.Helper()
	key := sha256.Sum256([]byte("attachment-test-key"))
	box, err := secretbox.New(key[:])
	if err != nil {
		t.Fatal(err)
	}
	disk := tenantfiles.Memory()
	return newAttachmentsWithStore(t, disk, tenantfiles.Encrypted(disk, box))
}

func newAttachmentsWithStore(t *testing.T, disk *tenantfiles.MemoryStore, files tenantfiles.Store) *attachments {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.PatientAttachment{}, &audit.AuditLog{}, &user_models.User{},
		&company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{},
		&clinical_models.MedicalRecord{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{},
		// Delete and restore check whether the attachment is an exam result.
		&clinical_models.ExamOrder{}, &clinical_models.ExamOrderItem{})
	audit.RegisterCallbacks(db)
	s := &attachments{t: t, db: db, disk: disk}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&s.own, &s.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("att-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-att-%d-%d", i, now)}
		if err := db.Create(tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
	}
	for _, p := range []struct {
		dst    *clinical_models.Patient
		tenant uint
	}{{&s.ownPatient, s.own.ID}, {&s.otherPatient, s.other.ID}} {
		*p.dst = clinical_models.Patient{TenantID: p.tenant, FirstName: "P", LastName: "Paciente", Document: fmt.Sprintf("DOC-%d-%d", p.tenant, now)}
		if err := db.Create(p.dst).Error; err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}

	// "own" is subscribed to a plan with room to spare; quota tests shrink it.
	s.plan = company_models.Plan{Name: "Clinic", Code: fmt.Sprintf("CLINIC-%d", now), StorageQuotaMB: 1024}
	if err := db.Create(&s.plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}
	company := company_models.Company{LegalName: "Clinic 0", TradeName: "Clinic 0", PlanCode: s.plan.Code, TenantID: s.own.ID}
	if err := db.Create(&company).Error; err != nil {
		t.Fatalf("create company: %v", err)
	}
	if err := db.Create(&company_models.Subscription{CompanyID: company.ID, PlanCode: s.plan.Code, Status: "active", ExpiresAt: time.Now().AddDate(0, 1, 0)}).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Stands in for auth + tenant middleware: a member of "own", user 42.
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", s.own.ID)
		c.Set("user_id", int64(42))
		c.Set("username", "doctor")
		c.Next()
	})
	// The permission guard grants what the X-Test-Permissions header lists, so
	// tests see which permission each route requires through Mount itself.
	guard := func(permissionID string) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !strings.Contains(","+c.GetHeader("X-Test-Permissions")+",", ","+permissionID+",") {
				envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "permission.error.insufficient", core_errors.ErrPermissionDenied))
				return
			}
			c.Next()
		}
	}
	NewPatientAttachmentHandler(db, zap.NewNop(), files).Mount(router.Group("/patients"), guard)
	s.router = router
	return s
}

type uploadForm struct {
	fileName, category, takenAt, description, medicalRecordID string
	content                                                   []byte
}

func (s *attachments) do(req *http.Request, perms ...string) *httptest.ResponseRecorder {
	if perms == nil {
		perms = []string{permRead, permUpload}
	}
	req.Header.Set("X-Test-Permissions", strings.Join(perms, ","))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *attachments) upload(patientID uint, f uploadForm, perms ...string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if f.content != nil {
		part, _ := mw.CreateFormFile("file", f.fileName)
		_, _ = part.Write(f.content)
	}
	for name, value := range map[string]string{"category": f.category, "taken_at": f.takenAt, "description": f.description, "medical_record_id": f.medicalRecordID} {
		if value != "" {
			_ = mw.WriteField(name, value)
		}
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/patients/%d/attachments", patientID), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return s.do(req, perms...)
}

func (s *attachments) list(patientID uint, query string, perms ...string) *httptest.ResponseRecorder {
	return s.do(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/patients/%d/attachments%s", patientID, query), nil), perms...)
}

func (s *attachments) download(patientID, attachmentID uint, perms ...string) *httptest.ResponseRecorder {
	return s.do(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/patients/%d/attachments/%d/download", patientID, attachmentID), nil), perms...)
}

func (s *attachments) view(patientID, attachmentID uint, perms ...string) *httptest.ResponseRecorder {
	return s.do(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/patients/%d/attachments/%d/view", patientID, attachmentID), nil), perms...)
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var resp struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return resp.Data
}

// listed is the attachments of a list response.
func listed(t *testing.T, w *httptest.ResponseRecorder) []clinical_models.PatientAttachment {
	t.Helper()
	items := decode[AttachmentList](t, w).Items
	out := make([]clinical_models.PatientAttachment, len(items))
	for i, item := range items {
		out[i] = item.PatientAttachment
	}
	return out
}

func (s *attachments) count() int64 {
	var n int64
	s.db.Model(&clinical_models.PatientAttachment{}).Count(&n)
	return n
}

func pdfForm() uploadForm {
	return uploadForm{fileName: "hemograma.pdf", category: clinical_models.AttachmentCategoryLabResult, content: samplePDF}
}

func TestAttachments_UploadListDownloadRoundTrip(t *testing.T) {
	s := newAttachments(t)

	form := pdfForm()
	form.takenAt, form.description = "2026-03-15", "Hemograma Lab. X"
	w := s.upload(s.ownPatient.ID, form)
	if w.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", w.Code, w.Body.String())
	}
	created := decode[clinical_models.PatientAttachment](t, w)
	if created.MimeType != "application/pdf" || created.Size != int64(len(samplePDF)) || created.FileName != "hemograma.pdf" || created.UploadedByID != 42 {
		t.Fatalf("created = %+v", created)
	}
	if got := created.TakenAt.Format("2006-01-02"); got != "2026-03-15" {
		t.Fatalf("taken_at = %s, want 2026-03-15", got)
	}

	// At rest: a generated name, and not the original bytes.
	var stored clinical_models.PatientAttachment
	s.db.First(&stored, created.ID)
	if !strings.HasPrefix(stored.StoredName, "attachments/") || strings.Contains(stored.StoredName, "hemograma") {
		t.Fatalf("stored name = %q, want a generated one", stored.StoredName)
	}
	raw, err := s.disk.Read(s.own.ID, stored.StoredName)
	if err != nil {
		t.Fatalf("nothing at rest: %v", err)
	}
	if bytes.Equal(raw, samplePDF) || bytes.Contains(raw, []byte("%PDF")) {
		t.Fatal("the file is stored unencrypted")
	}
	if sum := sha256.Sum256(samplePDF); stored.SHA256 != fmt.Sprintf("%x", sum) {
		t.Fatalf("sha256 = %s", stored.SHA256)
	}

	w = s.list(s.ownPatient.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list = %d %s", w.Code, w.Body.String())
	}
	if items := listed(t, w); len(items) != 1 || items[0].ID != created.ID || items[0].Description != "Hemograma Lab. X" {
		t.Fatalf("list = %+v", items)
	}

	w = s.download(s.ownPatient.ID, created.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("download = %d %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(w.Body.Bytes(), samplePDF) {
		t.Fatal("download did not return the original bytes")
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") || !strings.Contains(cd, "hemograma.pdf") {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("download is missing X-Content-Type-Options: nosniff")
	}

	var logs []audit.AuditLog
	s.db.Where("action = ? AND entity_type = ? AND entity_id = ?", "READ", "patient_attachments", created.ID).Find(&logs)
	if len(logs) != 1 || logs[0].UserID != 42 || logs[0].TenantID != s.own.ID || logs[0].PatientID == nil || *logs[0].PatientID != s.ownPatient.ID {
		t.Fatalf("access audit = %+v, want one READ by user 42 on the patient", logs)
	}
}

func TestAttachments_TakenAtDefaultsToUploadDay(t *testing.T) {
	s := newAttachments(t)
	w := s.upload(s.ownPatient.ID, pdfForm())
	if w.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", w.Code, w.Body.String())
	}
	got := decode[clinical_models.PatientAttachment](t, w).TakenAt.Format("2006-01-02")
	if today := time.Now().Format("2006-01-02"); got != today {
		t.Fatalf("taken_at = %s, want today %s", got, today)
	}
}

func TestAttachments_ListIsByExamDateAndFiltersByCategory(t *testing.T) {
	s := newAttachments(t)
	for _, f := range []struct{ category, takenAt string }{
		{clinical_models.AttachmentCategoryLabResult, "2026-01-10"},
		{clinical_models.AttachmentCategoryImaging, "2026-06-01"},
		{clinical_models.AttachmentCategoryLabResult, "2026-03-05"},
	} {
		form := pdfForm()
		form.category, form.takenAt = f.category, f.takenAt
		if w := s.upload(s.ownPatient.ID, form); w.Code != http.StatusOK {
			t.Fatalf("upload = %d %s", w.Code, w.Body.String())
		}
	}

	dates := func(items []clinical_models.PatientAttachment) string {
		out := make([]string, len(items))
		for i, a := range items {
			out[i] = a.TakenAt.Format("2006-01-02")
		}
		return strings.Join(out, ",")
	}
	if got := dates(listed(t, s.list(s.ownPatient.ID, ""))); got != "2026-06-01,2026-03-05,2026-01-10" {
		t.Fatalf("order = %s", got)
	}
	if got := dates(listed(t, s.list(s.ownPatient.ID, "?category=lab_result"))); got != "2026-03-05,2026-01-10" {
		t.Fatalf("lab_result = %s", got)
	}
	if w := s.list(s.ownPatient.ID, "?category=nope"); w.Code != http.StatusBadRequest {
		t.Fatalf("unknown category = %d, want 400", w.Code)
	}
}

func TestAttachments_ListExcludesDeleted(t *testing.T) {
	s := newAttachments(t)
	created := decode[clinical_models.PatientAttachment](t, s.upload(s.ownPatient.ID, pdfForm()))
	s.db.Delete(&clinical_models.PatientAttachment{}, created.ID)
	if items := listed(t, s.list(s.ownPatient.ID, "")); len(items) != 0 {
		t.Fatalf("list = %+v, want deleted attachment hidden", items)
	}
}

func TestAttachments_RejectsFilesOver15MB(t *testing.T) {
	s := newAttachments(t)
	form := pdfForm()
	form.content = append(append([]byte{}, samplePDF...), bytes.Repeat([]byte("0"), MaxAttachmentSize)...)
	if w := s.upload(s.ownPatient.ID, form); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("upload = %d %s, want 413", w.Code, w.Body.String())
	}
	if s.count() != 0 {
		t.Fatal("an oversized file was stored")
	}
}

func TestAttachments_TypeIsDecidedByContent(t *testing.T) {
	s := newAttachments(t)
	rejected := map[string][]byte{
		"html renamed .pdf":       []byte("<!DOCTYPE html><html><script>alert(1)</script></html>"),
		"executable renamed .pdf": append([]byte("MZ\x90\x00\x03\x00\x00\x00"), bytes.Repeat([]byte{0}, 64)...),
	}
	for name, content := range rejected {
		t.Run(name, func(t *testing.T) {
			form := pdfForm()
			form.content = content
			if w := s.upload(s.ownPatient.ID, form); w.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("upload = %d %s, want 415", w.Code, w.Body.String())
			}
		})
	}
	if s.count() != 0 {
		t.Fatal("a rejected file was stored")
	}

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	form := uploadForm{fileName: "foto.pdf", category: clinical_models.AttachmentCategoryClinicalPhoto, content: png}
	w := s.upload(s.ownPatient.ID, form)
	if got := decode[clinical_models.PatientAttachment](t, w).MimeType; w.Code != http.StatusOK || got != "image/png" {
		t.Fatalf("png named .pdf = %d, mime %q; want stored as image/png", w.Code, got)
	}
}

func TestAttachments_ValidatesCategoryAndFile(t *testing.T) {
	s := newAttachments(t)
	form := pdfForm()
	form.category = "x-ray"
	if w := s.upload(s.ownPatient.ID, form); w.Code != http.StatusBadRequest {
		t.Fatalf("bad category = %d, want 400", w.Code)
	}
	form = pdfForm()
	form.content = nil
	if w := s.upload(s.ownPatient.ID, form); w.Code != http.StatusBadRequest {
		t.Fatalf("no file = %d, want 400", w.Code)
	}
	if s.count() != 0 {
		t.Fatal("an invalid upload was stored")
	}
}

func TestAttachments_AnotherTenantIsNotFoundOnEveryRoute(t *testing.T) {
	s := newAttachments(t)
	theirs := clinical_models.PatientAttachment{TenantID: s.other.ID, PatientID: s.otherPatient.ID, Category: "other", TakenAt: time.Now(), FileName: "x.pdf", MimeType: "application/pdf", StoredName: "attachments/theirs"}
	if err := s.db.Create(&theirs).Error; err != nil {
		t.Fatal(err)
	}
	_ = s.disk.Write(s.other.ID, theirs.StoredName, samplePDF)

	cases := map[string]*httptest.ResponseRecorder{
		"upload to their patient":             s.upload(s.otherPatient.ID, pdfForm()),
		"list their patient":                  s.list(s.otherPatient.ID, ""),
		"download via their patient":          s.download(s.otherPatient.ID, theirs.ID),
		"download their file via own patient": s.download(s.ownPatient.ID, theirs.ID),
		"view via their patient":              s.view(s.otherPatient.ID, theirs.ID),
		"view their file via own patient":     s.view(s.ownPatient.ID, theirs.ID),
	}
	for name, w := range cases {
		if w.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", name, w.Code)
		}
		if bytes.Contains(w.Body.Bytes(), samplePDF) {
			t.Errorf("%s leaked the file", name)
		}
	}
	var n int64
	s.db.Model(&clinical_models.PatientAttachment{}).Where("patient_id = ?", s.otherPatient.ID).Count(&n)
	if n != 1 {
		t.Fatalf("their patient has %d attachments, want 1 (nothing uploaded to it)", n)
	}
	var logs int64
	s.db.Model(&audit.AuditLog{}).Count(&logs)
	if logs != 0 {
		t.Fatal("a refused download was recorded as an access")
	}
}

func TestAttachments_PermissionGating(t *testing.T) {
	s := newAttachments(t)

	// Upload without read: the upload works, the files stay out of sight.
	w := s.upload(s.ownPatient.ID, pdfForm(), permUpload)
	if w.Code != http.StatusOK {
		t.Fatalf("upload with %s = %d", permUpload, w.Code)
	}
	id := decode[clinical_models.PatientAttachment](t, w).ID
	if w := s.list(s.ownPatient.ID, "", permUpload); w.Code != http.StatusForbidden {
		t.Errorf("list without %s = %d, want 403", permRead, w.Code)
	}
	if w := s.download(s.ownPatient.ID, id, permUpload); w.Code != http.StatusForbidden {
		t.Errorf("download without %s = %d, want 403", permRead, w.Code)
	}

	if w := s.view(s.ownPatient.ID, id, permUpload); w.Code != http.StatusForbidden {
		t.Errorf("view without %s = %d, want 403", permRead, w.Code)
	}
	if w := s.view(s.ownPatient.ID, id, permRead); w.Code != http.StatusOK {
		t.Errorf("view with %s = %d, want 200", permRead, w.Code)
	}

	if w := s.upload(s.ownPatient.ID, pdfForm(), permRead); w.Code != http.StatusForbidden {
		t.Errorf("upload without %s = %d, want 403", permUpload, w.Code)
	}
	if w := s.list(s.ownPatient.ID, "", permRead); w.Code != http.StatusOK {
		t.Errorf("list with %s = %d, want 200", permRead, w.Code)
	}
	if w := s.download(s.ownPatient.ID, id, permRead); w.Code != http.StatusOK {
		t.Errorf("download with %s = %d, want 200", permRead, w.Code)
	}
}

func TestAttachments_WithoutEncryptionKeyEveryRouteIsUnavailable(t *testing.T) {
	s := newAttachmentsWithStore(t, tenantfiles.Memory(), nil)
	for name, w := range map[string]*httptest.ResponseRecorder{
		"upload":   s.upload(s.ownPatient.ID, pdfForm()),
		"list":     s.list(s.ownPatient.ID, ""),
		"download": s.download(s.ownPatient.ID, 1),
	} {
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s = %d, want 503", name, w.Code)
		}
		if !strings.Contains(w.Body.String(), core_errors.ErrClinicalAttachmentUnavailable.ErrorCode) {
			t.Errorf("%s body = %s, want %s", name, w.Body.String(), core_errors.ErrClinicalAttachmentUnavailable.ErrorCode)
		}
	}
}

func TestAttachments_ViewServesInlineUnderCSPAndIsAudited(t *testing.T) {
	s := newAttachments(t)
	created := decode[clinical_models.PatientAttachment](t, s.upload(s.ownPatient.ID, pdfForm()))

	w := s.view(s.ownPatient.ID, created.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("view = %d %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(w.Body.Bytes(), samplePDF) {
		t.Fatal("view did not return the original bytes")
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") {
		t.Fatalf("Content-Disposition = %q, want inline", cd)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("view is missing nosniff")
	}
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "sandbox") || !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("Content-Security-Policy = %q", csp)
	}

	var n int64
	s.db.Model(&audit.AuditLog{}).Where("action = ? AND entity_type = ? AND entity_id = ?", "READ", "patient_attachments", created.ID).Count(&n)
	if n != 1 {
		t.Fatalf("view audit entries = %d, want 1", n)
	}
}

func TestAttachments_ViewOfDeletedIsNotFound(t *testing.T) {
	s := newAttachments(t)
	created := decode[clinical_models.PatientAttachment](t, s.upload(s.ownPatient.ID, pdfForm()))
	s.db.Delete(&clinical_models.PatientAttachment{}, created.ID)
	if w := s.view(s.ownPatient.ID, created.ID); w.Code != http.StatusNotFound {
		t.Fatalf("view of deleted = %d, want 404", w.Code)
	}
}

func (s *attachments) update(patientID, attachmentID uint, body string, perms ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/patients/%d/attachments/%d", patientID, attachmentID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return s.do(req, perms...)
}

func (s *attachments) uploaded() clinical_models.PatientAttachment {
	s.t.Helper()
	w := s.upload(s.ownPatient.ID, pdfForm())
	if w.Code != http.StatusOK {
		s.t.Fatalf("upload = %d", w.Code)
	}
	return decode[clinical_models.PatientAttachment](s.t, w)
}

func TestAttachments_UpdateMetadataPersistsAndIsAudited(t *testing.T) {
	s := newAttachments(t)
	before := s.uploaded()
	s.db.First(&before, before.ID) // StoredName is not in the JSON

	w := s.update(s.ownPatient.ID, before.ID, `{"category":"imaging","taken_at":"2026-03-15","description":"  Rx de torax  "}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update = %d: %s", w.Code, w.Body.String())
	}
	resp := decode[clinical_models.PatientAttachment](t, w)
	if resp.Category != "imaging" || resp.Description != "Rx de torax" {
		t.Fatalf("response = %+v", resp)
	}

	var got clinical_models.PatientAttachment
	if err := s.db.First(&got, before.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Category != "imaging" || got.Description != "Rx de torax" || got.TakenAt.Format("2006-01-02") != "2026-03-15" {
		t.Fatalf("persisted = %+v", got)
	}
	// The file never changes.
	if got.FileName != before.FileName || got.MimeType != before.MimeType || got.Size != before.Size || got.SHA256 != before.SHA256 || got.StoredName != before.StoredName || got.UploadedByID != before.UploadedByID || got.MedicalRecordID != nil {
		t.Fatalf("file fields changed: before %+v after %+v", before, got)
	}
	if w := s.download(s.ownPatient.ID, before.ID); !bytes.Equal(w.Body.Bytes(), samplePDF) {
		t.Fatal("file bytes changed")
	}

	var logs []audit.AuditLog
	s.db.Where("action = ? AND entity_type = ? AND entity_id = ?", "UPDATE", "patient_attachments", before.ID).Find(&logs)
	if len(logs) != 1 || len(logs[0].NewValues) == 0 {
		t.Fatalf("update audit = %+v, want one UPDATE with new values", logs)
	}
}

func TestAttachments_UpdateIsPartialAndIgnoresFileFields(t *testing.T) {
	s := newAttachments(t)
	before := s.uploaded()

	w := s.update(s.ownPatient.ID, before.ID, `{"description":"nota","file_name":"x.exe","sha256":"abc","patient_id":99}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update = %d", w.Code)
	}
	var got clinical_models.PatientAttachment
	s.db.First(&got, before.ID)
	if got.Description != "nota" || got.Category != before.Category || !got.TakenAt.Equal(before.TakenAt) {
		t.Fatalf("partial update = %+v", got)
	}
	if got.FileName != before.FileName || got.SHA256 != before.SHA256 || got.MedicalRecordID != nil || got.PatientID != s.ownPatient.ID {
		t.Fatalf("non-editable fields changed: %+v", got)
	}
}

func TestAttachments_UpdateValidates(t *testing.T) {
	s := newAttachments(t)
	a := s.uploaded()
	cases := map[string]string{
		"bad category":     `{"category":"nope"}`,
		"empty category":   `{"category":""}`,
		"bad date":         `{"taken_at":"15/03/26 nope"}`,
		"empty date":       `{"taken_at":""}`,
		"long description": fmt.Sprintf(`{"description":%q}`, strings.Repeat("a", 256)),
		"not json":         `nope`,
	}
	for name, body := range cases {
		if w := s.update(s.ownPatient.ID, a.ID, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, w.Code)
		}
	}
	var got clinical_models.PatientAttachment
	s.db.First(&got, a.ID)
	if got.Category != a.Category || got.Description != a.Description {
		t.Fatal("a rejected update changed the row")
	}
}

func TestAttachments_UpdateScopeAndDeleted(t *testing.T) {
	s := newAttachments(t)
	mine := s.uploaded()
	theirs := clinical_models.PatientAttachment{TenantID: s.other.ID, PatientID: s.otherPatient.ID, Category: "other", TakenAt: time.Now(), FileName: "x.pdf", MimeType: "application/pdf", StoredName: "attachments/theirs"}
	if err := s.db.Create(&theirs).Error; err != nil {
		t.Fatal(err)
	}
	otherOwnPatient := clinical_models.Patient{TenantID: s.own.ID, FirstName: "O", LastName: "Otro", Document: fmt.Sprintf("DOC-O-%d", time.Now().UnixNano())}
	if err := s.db.Create(&otherOwnPatient).Error; err != nil {
		t.Fatal(err)
	}
	deleted := s.uploaded()
	s.db.Delete(&clinical_models.PatientAttachment{}, deleted.ID)

	body := `{"description":"hack"}`
	cases := map[string]*httptest.ResponseRecorder{
		"other tenant's patient":           s.update(s.otherPatient.ID, theirs.ID, body),
		"other tenant's file, own patient": s.update(s.ownPatient.ID, theirs.ID, body),
		"file of another patient":          s.update(otherOwnPatient.ID, mine.ID, body),
		"deleted":                          s.update(s.ownPatient.ID, deleted.ID, body),
		"missing":                          s.update(s.ownPatient.ID, 999999, body),
	}
	for name, w := range cases {
		if w.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", name, w.Code)
		}
	}
	var n int64
	s.db.Unscoped().Model(&clinical_models.PatientAttachment{}).Where("description = ?", "hack").Count(&n)
	if n != 0 {
		t.Fatal("a refused update changed a row")
	}
}

func TestAttachments_UpdateRequiresUploadPermission(t *testing.T) {
	s := newAttachments(t)
	a := s.uploaded()
	if w := s.update(s.ownPatient.ID, a.ID, `{"description":"x"}`, permRead); w.Code != http.StatusForbidden {
		t.Errorf("update without %s = %d, want 403", permUpload, w.Code)
	}
	if w := s.update(s.ownPatient.ID, a.ID, `{"description":"x"}`, permUpload); w.Code != http.StatusOK {
		t.Errorf("update with %s = %d, want 200", permUpload, w.Code)
	}
}

func (s *attachments) remove(patientID, attachmentID uint, body string, perms ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/patients/%d/attachments/%d", patientID, attachmentID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return s.do(req, perms...)
}

func (s *attachments) deletedList(patientID uint, perms ...string) *httptest.ResponseRecorder {
	return s.do(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/patients/%d/attachments-deleted", patientID), nil), perms...)
}

func (s *attachments) restore(patientID, attachmentID uint, perms ...string) *httptest.ResponseRecorder {
	return s.do(httptest.NewRequest(http.MethodPost, fmt.Sprintf("/patients/%d/attachments/%d/restore", patientID, attachmentID), nil), perms...)
}

func (s *attachments) auditActions(id uint) []string {
	var logs []audit.AuditLog
	s.db.Where("entity_type = ? AND entity_id = ?", "patient_attachments", id).Order("id").Find(&logs)
	actions := make([]string, 0, len(logs))
	for _, l := range logs {
		actions = append(actions, l.Action)
	}
	return actions
}

func TestAttachments_DeleteRequiresReason(t *testing.T) {
	s := newAttachments(t)
	a := s.uploaded()
	for name, body := range map[string]string{
		"empty":    `{"reason":""}`,
		"blank":    `{"reason":"   "}`,
		"missing":  `{}`,
		"not json": `nope`,
		"too long": fmt.Sprintf(`{"reason":%q}`, strings.Repeat("x", 501)),
	} {
		if w := s.remove(s.ownPatient.ID, a.ID, body, permDelete); w.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, w.Code)
		}
	}
	if items := listed(t, s.list(s.ownPatient.ID, "")); len(items) != 1 {
		t.Fatal("a rejected delete removed the attachment")
	}
}

func TestAttachments_DeleteRestoreRoundTrip(t *testing.T) {
	s := newAttachments(t)
	user := user_models.User{UserName: "dra.perez", Email: "dra@example.com"}
	if err := s.db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	s.db.Model(&user).Update("id", 42) // the user the router acts as
	a := s.uploaded()

	if w := s.remove(s.ownPatient.ID, a.ID, `{"reason":"  Paciente equivocado  "}`, permDelete); w.Code != http.StatusOK {
		t.Fatalf("delete = %d: %s", w.Code, w.Body.String())
	}

	var row clinical_models.PatientAttachment
	if err := s.db.Unscoped().First(&row, a.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !row.DeletedAt.Valid || row.DeletedByID == nil || *row.DeletedByID != 42 || row.DeleteReason != "Paciente equivocado" {
		t.Fatalf("row after delete = %+v", row)
	}
	if items := listed(t, s.list(s.ownPatient.ID, "")); len(items) != 0 {
		t.Fatalf("list = %+v, want deleted hidden", items)
	}
	if got := s.auditActions(a.ID); len(got) != 2 || got[1] != "UPDATE" {
		t.Fatalf("audit after delete = %v, want CREATE then UPDATE", got)
	}

	deleted := decode[[]struct {
		ID            uint   `json:"ID"`
		DeletedAt     string `json:"DeletedAt"`
		DeletedByID   uint   `json:"deleted_by_id"`
		DeletedByName string `json:"deleted_by_name"`
		DeleteReason  string `json:"delete_reason"`
	}](t, s.deletedList(s.ownPatient.ID, permDelete))
	if len(deleted) != 1 || deleted[0].ID != a.ID || deleted[0].DeletedAt == "" || deleted[0].DeletedByID != 42 || deleted[0].DeleteReason != "Paciente equivocado" {
		t.Fatalf("deleted list = %+v", deleted)
	}
	if deleted[0].DeletedByName != "dra.perez" {
		t.Errorf("deleted_by_name = %q, want the user's name", deleted[0].DeletedByName)
	}

	// Once deleted: gone for download, view, update and a second delete.
	if w := s.download(s.ownPatient.ID, a.ID); w.Code != http.StatusNotFound {
		t.Errorf("download of deleted = %d, want 404", w.Code)
	}
	if w := s.view(s.ownPatient.ID, a.ID); w.Code != http.StatusNotFound {
		t.Errorf("view of deleted = %d, want 404", w.Code)
	}
	if w := s.update(s.ownPatient.ID, a.ID, `{"description":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("update of deleted = %d, want 404", w.Code)
	}
	if w := s.remove(s.ownPatient.ID, a.ID, `{"reason":"again"}`, permDelete); w.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", w.Code)
	}

	if w := s.restore(s.ownPatient.ID, a.ID, permDelete); w.Code != http.StatusOK {
		t.Fatalf("restore = %d: %s", w.Code, w.Body.String())
	}
	var back clinical_models.PatientAttachment
	if err := s.db.First(&back, a.ID).Error; err != nil {
		t.Fatalf("restored row is not visible: %v", err)
	}
	if back.DeleteReason != "" || back.DeletedByID != nil || back.DeletedAt.Valid {
		t.Fatalf("restored row keeps deletion data: %+v", back)
	}
	if items := listed(t, s.list(s.ownPatient.ID, "")); len(items) != 1 {
		t.Fatal("restored attachment is not back in the list")
	}
	if w := s.download(s.ownPatient.ID, a.ID); w.Code != http.StatusOK {
		t.Errorf("download after restore = %d, want 200", w.Code)
	}
	if items := decode[[]clinical_models.PatientAttachment](t, s.deletedList(s.ownPatient.ID, permDelete)); len(items) != 0 {
		t.Fatal("restored attachment is still in the deleted list")
	}
	if got := s.auditActions(a.ID); len(got) < 3 || got[2] != "UPDATE" {
		t.Fatalf("audit after restore = %v, want a third row (UPDATE)", got)
	}

	if w := s.restore(s.ownPatient.ID, a.ID, permDelete); w.Code != http.StatusNotFound {
		t.Errorf("restore of a non-deleted attachment = %d, want 404", w.Code)
	}
}

func TestAttachments_DeletedListIsNewestFirst(t *testing.T) {
	s := newAttachments(t)
	first, second := s.uploaded(), s.uploaded()
	s.remove(s.ownPatient.ID, first.ID, `{"reason":"a"}`, permDelete)
	time.Sleep(10 * time.Millisecond)
	s.remove(s.ownPatient.ID, second.ID, `{"reason":"b"}`, permDelete)
	items := decode[[]clinical_models.PatientAttachment](t, s.deletedList(s.ownPatient.ID, permDelete))
	if len(items) != 2 || items[0].ID != second.ID {
		t.Fatalf("deleted list order = %+v, want most recently deleted first", items)
	}
}

func TestAttachments_DeleteAndRestoreRequireDeletePermission(t *testing.T) {
	s := newAttachments(t)
	a := s.uploaded()
	body := `{"reason":"x"}`
	if w := s.remove(s.ownPatient.ID, a.ID, body, permRead, permUpload); w.Code != http.StatusForbidden {
		t.Errorf("delete without permission = %d, want 403", w.Code)
	}
	if w := s.deletedList(s.ownPatient.ID, permRead, permUpload); w.Code != http.StatusForbidden {
		t.Errorf("deleted list without permission = %d, want 403", w.Code)
	}
	if w := s.restore(s.ownPatient.ID, a.ID, permRead, permUpload); w.Code != http.StatusForbidden {
		t.Errorf("restore without permission = %d, want 403", w.Code)
	}
	// Delete works with its own permission alone (no read needed).
	if w := s.remove(s.ownPatient.ID, a.ID, body, permDelete); w.Code != http.StatusOK {
		t.Errorf("delete with permission = %d, want 200", w.Code)
	}
}

func TestAttachments_DeleteAndRestoreAreScoped(t *testing.T) {
	s := newAttachments(t)
	mine := s.uploaded()
	theirs := clinical_models.PatientAttachment{TenantID: s.other.ID, PatientID: s.otherPatient.ID, Category: "other", TakenAt: time.Now(), FileName: "x.pdf", MimeType: "application/pdf", StoredName: "attachments/theirs"}
	if err := s.db.Create(&theirs).Error; err != nil {
		t.Fatal(err)
	}
	otherOwnPatient := clinical_models.Patient{TenantID: s.own.ID, FirstName: "O", LastName: "Otro", Document: fmt.Sprintf("DOC-D-%d", time.Now().UnixNano())}
	if err := s.db.Create(&otherOwnPatient).Error; err != nil {
		t.Fatal(err)
	}
	// The other tenant's attachment, deleted: must not be restorable or listed.
	s.db.Delete(&theirs)

	body := `{"reason":"x"}`
	cases := map[string]*httptest.ResponseRecorder{
		"delete other tenant's patient":      s.remove(s.otherPatient.ID, theirs.ID, body, permDelete),
		"delete other tenant's file":         s.remove(s.ownPatient.ID, theirs.ID, body, permDelete),
		"delete file of another patient":     s.remove(otherOwnPatient.ID, mine.ID, body, permDelete),
		"delete missing":                     s.remove(s.ownPatient.ID, 999999, body, permDelete),
		"restore other tenant's patient":     s.restore(s.otherPatient.ID, theirs.ID, permDelete),
		"restore other tenant's file":        s.restore(s.ownPatient.ID, theirs.ID, permDelete),
		"deleted list, other tenant patient": s.deletedList(s.otherPatient.ID, permDelete),
	}
	for name, w := range cases {
		if w.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", name, w.Code)
		}
	}
	var n int64
	s.db.Unscoped().Model(&clinical_models.PatientAttachment{}).Where("id = ? AND delete_reason <> ''", mine.ID).Count(&n)
	if n != 0 {
		t.Fatal("a refused delete changed a row")
	}
	var still clinical_models.PatientAttachment
	if err := s.db.Unscoped().First(&still, theirs.ID).Error; err != nil || !still.DeletedAt.Valid {
		t.Fatal("a refused restore changed the other tenant's row")
	}
	// The deleted list of my patient does not show the other tenant's row.
	if items := decode[[]clinical_models.PatientAttachment](t, s.deletedList(s.ownPatient.ID, permDelete)); len(items) != 0 {
		t.Fatalf("deleted list = %+v, want empty", items)
	}
}

// A deleted row of another tenant that points at one of my patients (the
// patient filter alone would let it through) stays out of reach: only the
// tenant filter on the Unscoped deleted-list/restore queries keeps it out.
func TestAttachments_DeletedListAndRestoreFilterByTenantNotOnlyPatient(t *testing.T) {
	s := newAttachments(t)
	stray := clinical_models.PatientAttachment{TenantID: s.other.ID, PatientID: s.ownPatient.ID, Category: "other", TakenAt: time.Now(), FileName: "stray.pdf", MimeType: "application/pdf", StoredName: "attachments/stray"}
	system := tenantdb.System(s.db)
	if err := system.Create(&stray).Error; err != nil {
		t.Fatal(err)
	}
	if err := system.Delete(&stray).Error; err != nil {
		t.Fatal(err)
	}

	if items := decode[[]clinical_models.PatientAttachment](t, s.deletedList(s.ownPatient.ID, permDelete)); len(items) != 0 {
		t.Fatalf("deleted list = %+v, want the other tenant's row hidden", items)
	}
	if w := s.restore(s.ownPatient.ID, stray.ID, permDelete); w.Code != http.StatusNotFound {
		t.Fatalf("restore = %d, want 404", w.Code)
	}
	var row clinical_models.PatientAttachment
	if err := system.Unscoped().First(&row, stray.ID).Error; err != nil || !row.DeletedAt.Valid {
		t.Fatal("a refused restore changed the other tenant's row")
	}
}

// ── Storage quota ───────────────────────────────────────────────────────────

const mb = int64(1) << 20

func (s *attachments) setQuotaMB(quota int64) {
	s.t.Helper()
	if err := s.db.Model(&s.plan).Update("storage_quota_mb", quota).Error; err != nil {
		s.t.Fatal(err)
	}
}

// stored adds an attachment row of size bytes for tenant (no file: usage only
// reads the rows), soft-deleted when deleted is set.
func (s *attachments) stored(tenant, patient uint, size int64, deleted bool) {
	s.t.Helper()
	row := clinical_models.PatientAttachment{TenantID: tenant, PatientID: patient, Category: "other", TakenAt: time.Now(), FileName: "old.pdf", MimeType: "application/pdf", Size: size, StoredName: fmt.Sprintf("attachments/old-%d", time.Now().UnixNano())}
	db := tenantdb.System(s.db)
	if err := db.Create(&row).Error; err != nil {
		s.t.Fatal(err)
	}
	if deleted {
		if err := db.Delete(&row).Error; err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s *attachments) usage() AttachmentUsage {
	s.t.Helper()
	w := s.list(s.ownPatient.ID, "")
	if w.Code != http.StatusOK {
		s.t.Fatalf("list = %d %s", w.Code, w.Body.String())
	}
	return decode[AttachmentList](s.t, w).Usage
}

func assertOverQuota(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), core_errors.ErrPlanStorageQuota.ErrorCode) {
		t.Fatalf("upload = %d %s, want 403 %s", w.Code, w.Body.String(), core_errors.ErrPlanStorageQuota.ErrorCode)
	}
}

func TestAttachments_UploadUpToTheQuotaThenRejected(t *testing.T) {
	s := newAttachments(t)
	s.setQuotaMB(1)
	pdf := int64(len(samplePDF))
	s.stored(s.own.ID, s.ownPatient.ID, mb-pdf, false)

	// Exactly filling the quota is allowed.
	if w := s.upload(s.ownPatient.ID, pdfForm()); w.Code != http.StatusOK {
		t.Fatalf("upload to exactly the quota = %d %s", w.Code, w.Body.String())
	}
	before := s.count()
	assertOverQuota(t, s.upload(s.ownPatient.ID, pdfForm()))
	if s.count() != before {
		t.Fatal("a rejected upload created a row")
	}
	if u := s.usage(); u.UsedBytes != mb || u.QuotaBytes != mb || !u.Warning {
		t.Fatalf("usage = %+v, want full", u)
	}
}

func TestAttachments_QuotaCountsDeletedFiles(t *testing.T) {
	s := newAttachments(t)
	s.setQuotaMB(1)
	s.stored(s.own.ID, s.ownPatient.ID, mb-10, true)

	if u := s.usage(); u.UsedBytes != mb-10 {
		t.Fatalf("used = %d, want the deleted file counted (%d)", u.UsedBytes, mb-10)
	}
	assertOverQuota(t, s.upload(s.ownPatient.ID, pdfForm()))
}

func TestAttachments_QuotaIgnoresOtherTenants(t *testing.T) {
	s := newAttachments(t)
	s.setQuotaMB(1)
	s.stored(s.other.ID, s.otherPatient.ID, 10*mb, false)
	s.stored(s.other.ID, s.otherPatient.ID, 10*mb, true)

	if u := s.usage(); u.UsedBytes != 0 || u.Warning {
		t.Fatalf("usage = %+v, want nothing of the other tenant counted", u)
	}
	if w := s.upload(s.ownPatient.ID, pdfForm()); w.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", w.Code, w.Body.String())
	}
}

func TestAttachments_UsageWarnsFrom80Percent(t *testing.T) {
	s := newAttachments(t)
	s.setQuotaMB(10)
	quota := 10 * mb
	s.stored(s.own.ID, s.ownPatient.ID, quota*8/10-1, false)
	if u := s.usage(); u.Warning || u.QuotaBytes != quota || u.UsedBytes != quota*8/10-1 {
		t.Fatalf("usage just under 80%% = %+v, want no warning", u)
	}
	s.stored(s.own.ID, s.ownPatient.ID, 1, false)
	if u := s.usage(); !u.Warning {
		t.Fatalf("usage at 80%% = %+v, want the warning", u)
	}
}

func TestAttachments_QuotaZeroRejectsEveryUpload(t *testing.T) {
	s := newAttachments(t)
	s.setQuotaMB(0)
	assertOverQuota(t, s.upload(s.ownPatient.ID, pdfForm()))
	if u := s.usage(); u.QuotaBytes != 0 || !u.Warning {
		t.Fatalf("usage = %+v, want quota 0 flagged", u)
	}
}

func TestAttachments_NoActiveSubscriptionMeansNoQuota(t *testing.T) {
	s := newAttachments(t)
	if err := s.db.Model(&company_models.Subscription{}).Where("plan_code = ?", s.plan.Code).Update("status", "cancelled").Error; err != nil {
		t.Fatal(err)
	}
	assertOverQuota(t, s.upload(s.ownPatient.ID, pdfForm()))
}

func (s *attachments) firstStored() clinical_models.PatientAttachment {
	s.t.Helper()
	var a clinical_models.PatientAttachment
	if err := s.db.Order("id ASC").First(&a).Error; err != nil {
		s.t.Fatal(err)
	}
	return a
}

func duplicateOf(t *testing.T, w *httptest.ResponseRecorder) *AttachmentDuplicate {
	t.Helper()
	if w.Code < 200 || w.Code > 299 {
		t.Fatalf("upload = %d: %s", w.Code, w.Body.String())
	}
	return decode[uploadedAttachment](t, w).DuplicateOf
}

func TestAttachments_DuplicateUploadIsStoredWithWarning(t *testing.T) {
	s := newAttachments(t)
	if dup := duplicateOf(t, s.upload(s.ownPatient.ID, pdfForm())); dup != nil {
		t.Fatalf("first upload warned: %+v", dup)
	}
	first := s.firstStored()

	again := pdfForm()
	again.fileName = "copia.pdf"
	dup := duplicateOf(t, s.upload(s.ownPatient.ID, again))
	if dup == nil || dup.ID != first.ID || dup.FileName != "hemograma.pdf" || dup.CreatedAt.IsZero() {
		t.Fatalf("duplicate_of = %+v, want the first upload", dup)
	}
	if s.count() != 2 {
		t.Fatalf("%d attachments stored, want 2 (duplicates are not blocked)", s.count())
	}
}

func TestAttachments_DuplicateWarningIsPerPatientAndTenant(t *testing.T) {
	s := newAttachments(t)
	second := clinical_models.Patient{TenantID: s.own.ID, FirstName: "Q", LastName: "Otro", Document: fmt.Sprintf("DOC2-%d", time.Now().UnixNano())}
	if err := s.db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	duplicateOf(t, s.upload(s.ownPatient.ID, pdfForm()))

	if dup := duplicateOf(t, s.upload(second.ID, pdfForm())); dup != nil {
		t.Fatalf("same file for another patient warned: %+v", dup)
	}

	// Same content already stored by another tenant.
	sum := sha256.Sum256(samplePDF)
	theirs := clinical_models.PatientAttachment{TenantID: s.other.ID, PatientID: s.otherPatient.ID, Category: "other", TakenAt: time.Now(), FileName: "x.pdf", MimeType: "application/pdf", SHA256: hex.EncodeToString(sum[:]), StoredName: "attachments/theirs"}
	if err := s.db.Create(&theirs).Error; err != nil {
		t.Fatal(err)
	}
	third := clinical_models.Patient{TenantID: s.own.ID, FirstName: "R", LastName: "Tres", Document: fmt.Sprintf("DOC3-%d", time.Now().UnixNano())}
	if err := s.db.Create(&third).Error; err != nil {
		t.Fatal(err)
	}
	if dup := duplicateOf(t, s.upload(third.ID, pdfForm())); dup != nil {
		t.Fatalf("another tenant's file produced a warning: %+v", dup)
	}
}

func TestAttachments_DeletedOriginalIsNotADuplicate(t *testing.T) {
	s := newAttachments(t)
	duplicateOf(t, s.upload(s.ownPatient.ID, pdfForm()))
	s.db.Delete(&clinical_models.PatientAttachment{}, s.firstStored().ID)
	if dup := duplicateOf(t, s.upload(s.ownPatient.ID, pdfForm())); dup != nil {
		t.Fatalf("deleted original warned: %+v", dup)
	}
}
