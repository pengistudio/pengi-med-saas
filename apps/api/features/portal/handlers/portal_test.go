package portal_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/whatsapp"
	agenda_models "pengi-med-saas/features/agenda/models"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	permission_models "pengi-med-saas/features/permissions/models"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// fakeMailer records the codes it was asked to send.
type fakeMailer struct {
	mu      sync.Mutex
	sent    map[string][]string // email -> codes, in order
	calls   int
	notices map[string][]mailer.PortalNoticeEmail // email -> booking notices
}

func (m *fakeMailer) SendPortalCode(to string, e mailer.PortalCodeEmail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sent == nil {
		m.sent = map[string][]string{}
	}
	m.sent[to] = append(m.sent[to], e.Code)
	m.calls++
	return nil
}

func (m *fakeMailer) SendPortalNotice(to string, e mailer.PortalNoticeEmail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.notices == nil {
		m.notices = map[string][]mailer.PortalNoticeEmail{}
	}
	m.notices[to] = append(m.notices[to], e)
	return nil
}

func (m *fakeMailer) noticesTo(to string) []mailer.PortalNoticeEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mailer.PortalNoticeEmail(nil), m.notices[to]...)
}

func (m *fakeMailer) last(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	codes := m.sent[to]
	if len(codes) == 0 {
		return ""
	}
	return codes[len(codes)-1]
}

// fakeGraph answers the WhatsApp Graph API: every send succeeds, unless fail.
type fakeGraph struct {
	mu     sync.Mutex
	bodies []string
	fail   bool
}

func (g *fakeGraph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	g.mu.Lock()
	g.bodies = append(g.bodies, r.Method+" "+r.URL.Path+" "+string(body))
	fail := g.fail
	g.mu.Unlock()
	if fail {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":131000,"message":"boom"}}`))
		return
	}
	_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.PORTAL"}]}`))
}

func (g *fakeGraph) sends() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := []string{}
	for _, b := range g.bodies {
		if strings.Contains(b, "/messages ") {
			out = append(out, b)
		}
	}
	return out
}

type fixture struct {
	t        *testing.T
	db       *gorm.DB
	router   *gin.Engine
	mail     *fakeMailer
	graph    *fakeGraph
	files    *tenantfiles.MemoryStore
	attach   *tenantfiles.MemoryStore
	plan     company_models.Plan
	own      tenant_models.Tenant // portal open
	other    tenant_models.Tenant // portal open, another clinic
	ownPat   clinical_models.Patient
	ownPat2  clinical_models.Patient // same clinic, another patient
	otherPat clinical_models.Patient // other clinic, same document as ownPat
	doctor   doctor_models.Doctor
	auth     *AuthHandler
}

const sharedDocument = "0102030405"

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &permission_models.Permission{}, &company_models.Feature{}, &company_models.Plan{},
		&company_models.Company{}, &company_models.Subscription{},
		&doctor_models.Doctor{}, &clinical_models.Patient{}, &clinical_models.Appointment{}, &agenda_models.AppointmentType{},
		&clinical_models.MedicalRecord{}, &clinical_models.Prescription{}, &clinical_models.MedicalCertificate{}, &clinical_models.MedicalReport{},
		&clinical_models.PatientAttachment{}, &clinical_models.ExamOrder{}, &clinical_models.ExamOrderItem{},
		&billing_models.Invoice{}, &billing_models.InvoiceItem{},
		&portal_models.PortalLoginCode{}, &portal_models.PortalContactCode{}, &audit.AuditLog{},
		&whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppTemplate{}, &whatsapp_models.WhatsAppMessage{},
	)
	f := &fixture{t: t, db: db, mail: &fakeMailer{}, graph: &fakeGraph{}, files: tenantfiles.Memory(), attach: tenantfiles.Memory()}
	perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: "MANAGE_PATIENT_PORTAL"}, Name: "Manage patient portal", Category: "PATIENT_PORTAL"}
	feature := company_models.Feature{Code: "PATIENT_PORTAL", Name: "Portal", Permissions: []permission_models.Permission{perm}}
	f.mustCreate(&feature)
	f.plan = company_models.Plan{Name: "Pro", Code: "PRO", Price: 10, Features: []company_models.Feature{feature}}
	f.mustCreate(&f.plan)

	f.own = f.clinic("own", true, f.plan.Code)
	f.other = f.clinic("other", true, f.plan.Code)

	f.doctor = doctor_models.Doctor{TenantID: f.own.ID, FullName: "Dra. Ana Ruiz", Specialty: "cardiology", Active: true}
	f.mustCreate(&f.doctor)
	f.ownPat = f.patient(f.own.ID, sharedDocument, "ana@example.com", "")
	f.ownPat2 = f.patient(f.own.ID, "0999999999", "luis@example.com", "")
	f.otherPat = f.patient(f.other.ID, sharedDocument, "otra@example.com", "")

	graphServer := httptest.NewServer(f.graph)
	t.Cleanup(graphServer.Close)
	client := whatsapp.New(graphServer.URL, "v23.0", "", "")
	delivery := portal_services.NewDelivery(db, zap.NewNop(), f.mail, client)
	// No captcha unless a test sets one (TURNSTILE_SECRET_KEY may be in the env).
	a := NewAuthHandler(db, zap.NewNop(), f.files, delivery).WithSyncDelivery().WithCaptcha(nil)
	f.auth = a
	p := NewPatientHandler(db, zap.NewNop(), f.files, f.attach, sri_document.Documents{Files: f.files})

	r := gin.New()
	g := r.Group("/portal")
	g.GET("/clinics/:slug", envelope.Handle(a.GetClinic))
	g.POST("/clinics/:slug/login", envelope.Handle(a.Login))
	g.POST("/clinics/:slug/verify", envelope.Handle(a.Verify))
	g.POST("/refresh", envelope.Handle(a.Refresh))
	authed := g.Group("", portal_middleware.RequirePortalSession(db))
	section := portal_middleware.RequireSection
	authed.GET("/me", envelope.Handle(p.Me))
	authed.GET("/appointments", section(portal_middleware.SectionAppointments), envelope.Handle(p.ListAppointments))
	authed.GET("/results", section(portal_middleware.SectionResults), envelope.Handle(p.ListResults))
	authed.GET("/results/files/:id", section(portal_middleware.SectionResults), p.DownloadResultFile)
	authed.GET("/documents", section(portal_middleware.SectionDocuments), envelope.Handle(p.ListDocuments))
	authed.GET("/documents/:type/:id/file", section(portal_middleware.SectionDocuments), p.DownloadDocument)
	authed.GET("/invoices", section(portal_middleware.SectionInvoices), envelope.Handle(p.ListInvoices))
	authed.GET("/invoices/:id/ride", section(portal_middleware.SectionInvoices), p.DownloadInvoiceRide)
	authed.GET("/invoices/:id/xml", section(portal_middleware.SectionInvoices), p.DownloadInvoiceXML)
	f.router = r
	return f
}

func (f *fixture) mustCreate(v any) {
	f.t.Helper()
	if err := tenantdb.System(f.db).Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

func (f *fixture) clinic(tag string, enabled bool, planCode string) tenant_models.Tenant {
	settings := tenant_models.DefaultPortalSettings()
	settings.Enabled = enabled
	raw, _ := json.Marshal(settings)
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clínica " + tag, Slug: fmt.Sprintf("%s-%d", tag, now), DisplayToken: fmt.Sprintf("tok-%s-%d", tag, now), PortalSettings: string(raw)}
	f.mustCreate(&tenant)
	company := company_models.Company{LegalName: tag, TradeName: tag, PlanCode: planCode, TenantID: tenant.ID}
	f.mustCreate(&company)
	f.mustCreate(&company_models.Subscription{Status: "active", PlanCode: planCode, ExpiresAt: time.Now().Add(30 * 24 * time.Hour), CompanyID: company.ID})
	return tenant
}

func (f *fixture) patient(tenantID uint, document, email, phone string) clinical_models.Patient {
	p := clinical_models.Patient{TenantID: tenantID, Document: document, Email: email, Phone: phone, FirstName: "Pac", LastName: document}
	f.mustCreate(&p)
	return p
}

func (f *fixture) setSettings(tenant tenant_models.Tenant, mutate func(*tenant_models.PortalSettings)) {
	s := tenant.ParsePortalSettings()
	mutate(&s)
	raw, _ := json.Marshal(s)
	if err := tenantdb.System(f.db).Model(&tenant_models.Tenant{}).Where("id = ?", tenant.ID).Update("portal_settings", string(raw)).Error; err != nil {
		f.t.Fatal(err)
	}
}

type result struct {
	Code    int
	Body    map[string]any
	Raw     []byte
	Headers http.Header
}

func (f *fixture) do(method, path, token string, body any) result {
	f.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	res := result{Code: w.Code, Raw: w.Body.Bytes(), Headers: w.Header()}
	_ = json.Unmarshal(w.Body.Bytes(), &res.Body)
	return res
}

func (f *fixture) login(tenant tenant_models.Tenant, document string) result {
	return f.do(http.MethodPost, "/portal/clinics/"+tenant.Slug+"/login", "", map[string]string{"document": document})
}

func (f *fixture) verify(tenant tenant_models.Tenant, document, code string) result {
	return f.do(http.MethodPost, "/portal/clinics/"+tenant.Slug+"/verify", "", map[string]string{"document": document, "code": code})
}

// signIn requests a code by email and exchanges it for an access token.
func (f *fixture) signIn(tenant tenant_models.Tenant, p clinical_models.Patient) string {
	f.t.Helper()
	if r := f.login(tenant, p.Document); r.Code != http.StatusOK {
		f.t.Fatalf("login: %d %s", r.Code, r.Raw)
	}
	r := f.verify(tenant, p.Document, f.mail.last(p.Email))
	if r.Code != http.StatusOK {
		f.t.Fatalf("verify: %d %s", r.Code, r.Raw)
	}
	return r.Body["data"].(map[string]any)["access_token"].(string)
}

func errorCode(r result) string {
	if data, ok := r.Body["data"].(map[string]any); ok {
		code, _ := data["error_code"].(string)
		return code
	}
	return ""
}

func listLen(t *testing.T, r result) int {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("status %d: %s", r.Code, r.Raw)
	}
	items, ok := r.Body["data"].([]any)
	if !ok {
		return 0
	}
	return len(items)
}

func (f *fixture) codes(patientID uint) []portal_models.PortalLoginCode {
	var rows []portal_models.PortalLoginCode
	tenantdb.System(f.db).Where("patient_id = ?", patientID).Order("id").Find(&rows)
	return rows
}

// --- sign-in ---

func TestLogin_AnswersTheSameForUnknownKnownAndHoneypot(t *testing.T) {
	f := newFixture(t)
	unknown := f.login(f.own, "0000000000")
	known := f.login(f.own, f.ownPat.Document)
	honeypot := f.do(http.MethodPost, "/portal/clinics/"+f.own.Slug+"/login", "", map[string]string{"document": f.ownPat2.Document, "website": "http://spam"})
	noContact := f.patient(f.own.ID, "1111111111", "", "")
	silent := f.login(f.own, noContact.Document)

	for name, r := range map[string]result{"unknown": unknown, "known": known, "honeypot": honeypot, "no contact": silent} {
		if r.Code != http.StatusOK || r.Body["message"] != "portal.login.code_sent" || string(mustJSON(r.Body["data"])) != `{"expires_in":600}` {
			t.Fatalf("%s: %d %s", name, r.Code, r.Raw)
		}
	}
	if string(unknown.Raw) != string(known.Raw) || string(known.Raw) != string(honeypot.Raw) {
		t.Fatalf("bodies differ:\n%s\n%s\n%s", unknown.Raw, known.Raw, honeypot.Raw)
	}
	if len(f.codes(f.ownPat.ID)) != 1 || f.mail.last(f.ownPat.Email) == "" {
		t.Fatal("the known patient must get one code by email")
	}
	if len(f.codes(f.ownPat2.ID)) != 0 || f.mail.last(f.ownPat2.Email) != "" {
		t.Fatal("the honeypot must not issue a code")
	}
	if len(f.codes(noContact.ID)) != 0 || f.mail.calls != 1 {
		t.Fatalf("only one email expected, got %d", f.mail.calls)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestVerify_CodeRules(t *testing.T) {
	f := newFixture(t)
	doc, email := f.ownPat.Document, f.ownPat.Email

	t.Run("single use", func(t *testing.T) {
		f.login(f.own, doc)
		code := f.mail.last(email)
		if r := f.verify(f.own, doc, code); r.Code != http.StatusOK {
			t.Fatalf("first use: %d %s", r.Code, r.Raw)
		}
		if r := f.verify(f.own, doc, code); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("second use: %d %s", r.Code, r.Raw)
		}
	})

	t.Run("new code invalidates old", func(t *testing.T) {
		tenantdb.System(f.db).Where("patient_id = ?", f.ownPat.ID).Delete(&portal_models.PortalLoginCode{})
		f.login(f.own, doc)
		first := f.mail.last(email)
		f.login(f.own, doc)
		second := f.mail.last(email)
		if first == second {
			t.Skip("random codes collided")
		}
		if r := f.verify(f.own, doc, first); r.Code != http.StatusUnauthorized {
			t.Fatalf("old code: %d", r.Code)
		}
		if r := f.verify(f.own, doc, second); r.Code != http.StatusOK {
			t.Fatalf("new code: %d %s", r.Code, r.Raw)
		}
	})

	t.Run("max 5 attempts", func(t *testing.T) {
		tenantdb.System(f.db).Unscoped().Where("patient_id = ?", f.ownPat.ID).Delete(&portal_models.PortalLoginCode{})
		f.login(f.own, doc)
		code := f.mail.last(email)
		wrong := "000000"
		if code == wrong {
			wrong = "111111"
		}
		for i := 0; i < portal_services.MaxAttempts; i++ {
			if r := f.verify(f.own, doc, wrong); r.Code != http.StatusUnauthorized {
				t.Fatalf("wrong try %d: %d", i+1, r.Code)
			}
		}
		if r := f.verify(f.own, doc, code); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("right code after 5 tries must fail: %d %s", r.Code, r.Raw)
		}
	})

	t.Run("expires", func(t *testing.T) {
		tenantdb.System(f.db).Unscoped().Where("patient_id = ?", f.ownPat.ID).Delete(&portal_models.PortalLoginCode{})
		f.login(f.own, doc)
		code := f.mail.last(email)
		tenantdb.System(f.db).Model(&portal_models.PortalLoginCode{}).Where("patient_id = ?", f.ownPat.ID).
			Update("expires_at", time.Now().Add(-time.Second).UTC())
		if r := f.verify(f.own, doc, code); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("expired: %d %s", r.Code, r.Raw)
		}
	})

	t.Run("same answer for unknown document", func(t *testing.T) {
		r := f.verify(f.own, "nobody", "123456")
		if r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("unknown: %d %s", r.Code, r.Raw)
		}
	})

	t.Run("a code of one clinic does not open another", func(t *testing.T) {
		tenantdb.System(f.db).Unscoped().Where("patient_id = ?", f.ownPat.ID).Delete(&portal_models.PortalLoginCode{})
		f.login(f.own, doc)
		if r := f.verify(f.other, doc, f.mail.last(email)); r.Code != http.StatusUnauthorized {
			t.Fatalf("cross-clinic code: %d", r.Code)
		}
	})
}

func TestLogin_PerPatientCodeLimit(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < portal_services.MaxCodesPerWindow+2; i++ {
		if r := f.login(f.own, f.ownPat.Document); r.Code != http.StatusOK {
			t.Fatalf("login %d: %d", i, r.Code)
		}
	}
	if n := len(f.codes(f.ownPat.ID)); n != portal_services.MaxCodesPerWindow {
		t.Fatalf("codes issued = %d, want %d", n, portal_services.MaxCodesPerWindow)
	}
}

// --- tokens ---

func TestPortalRoutes_RejectNonPortalTokens(t *testing.T) {
	f := newFixture(t)
	staff, _ := auth.GenerateToken("admin", 1)
	backoffice, _ := auth.GenerateBackofficeToken("root", 1)
	refresh, _ := auth.GeneratePortalRefreshToken(f.own.ID, f.ownPat.ID)
	for name, token := range map[string]string{"none": "", "staff": staff, "backoffice": backoffice, "portal refresh": refresh, "garbage": "x.y.z"} {
		if r := f.do(http.MethodGet, "/portal/me", token, nil); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-003" {
			t.Fatalf("%s token: %d %s", name, r.Code, r.Raw)
		}
	}
	access := f.signIn(f.own, f.ownPat)
	if r := f.do(http.MethodGet, "/portal/me", access, nil); r.Code != http.StatusOK {
		t.Fatalf("portal token: %d %s", r.Code, r.Raw)
	}
}

func TestRefresh_UsesTheCookie(t *testing.T) {
	f := newFixture(t)
	refresh, _ := auth.GeneratePortalRefreshToken(f.own.ID, f.ownPat.ID)
	req := httptest.NewRequest(http.MethodPost, "/portal/refresh", nil)
	req.AddCookie(&http.Cookie{Name: auth.PortalRefreshCookie, Value: refresh})
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "access_token") {
		t.Fatalf("refresh: %d %s", w.Code, w.Body.String())
	}
	access, _ := auth.GeneratePortalAccessToken(f.own.ID, f.ownPat.ID)
	req = httptest.NewRequest(http.MethodPost, "/portal/refresh", nil)
	req.AddCookie(&http.Cookie{Name: auth.PortalRefreshCookie, Value: access}) // wrong type
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("access token as refresh: %d", w.Code)
	}
}

func TestVerify_SetsRefreshCookie(t *testing.T) {
	f := newFixture(t)
	f.login(f.own, f.ownPat.Document)
	r := f.verify(f.own, f.ownPat.Document, f.mail.last(f.ownPat.Email))
	cookie := r.Headers.Get("Set-Cookie")
	if r.Code != http.StatusOK || !strings.Contains(cookie, "portal_refresh=") || !strings.Contains(cookie, "Path=/api/v1/portal") ||
		!strings.Contains(cookie, "HttpOnly") || !strings.Contains(cookie, "SameSite=Lax") {
		t.Fatalf("cookie = %q", cookie)
	}
}

// --- isolation and filters ---

func TestPortal_PatientSeesOnlyOwnData(t *testing.T) {
	f := newFixture(t)
	sys := tenantdb.System(f.db)
	// One appointment, authorized invoice, signed certificate and reviewed
	// result for each of: the patient, another patient of the clinic, and the
	// patient with the same document at another clinic.
	type seeded struct {
		appt    clinical_models.Appointment
		invoice billing_models.Invoice
		cert    clinical_models.MedicalCertificate
		file    clinical_models.PatientAttachment
	}
	seed := func(tenantID, patientID uint, tag string) seeded {
		var s seeded
		s.appt = clinical_models.Appointment{TenantID: tenantID, PatientID: patientID, Date: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC), StartTime: "09:00", EndTime: "09:30", Status: "scheduled", Notes: "internal " + tag}
		f.mustCreate(&s.appt)
		key := fmt.Sprintf("%049d", patientID*1000+tenantID)
		s.invoice = billing_models.Invoice{TenantID: tenantID, PatientID: &patientID, Status: billing_models.InvoiceStatusAuthorized, AccessKey: &key,
			EstablishmentCode: "001", EmissionPointCode: "001", Sequential: "000000001", Total: 11.5, IssueDate: time.Now()}
		f.mustCreate(&s.invoice)
		_ = f.files.Write(tenantID, sri_document.RideName(key), []byte("%PDF-ride-"+tag))
		_ = f.files.Write(tenantID, sri_document.InvoiceXMLName(key), []byte("<factura>"+tag+"</factura>"))
		signedAt := time.Now()
		s.cert = clinical_models.MedicalCertificate{TenantID: tenantID, PatientID: patientID,
			DocumentSignature: clinical_models.DocumentSignature{SignedAt: &signedAt, SignerName: "Dr", SignedFile: "signed/cert-" + tag + ".pdf"}}
		f.mustCreate(&s.cert)
		_ = f.files.Write(tenantID, s.cert.SignedFile, []byte("%PDF-cert-"+tag))
		s.file = clinical_models.PatientAttachment{TenantID: tenantID, PatientID: patientID, Category: "lab_result", FileName: tag + ".pdf", MimeType: "application/pdf", StoredName: "attachments/" + tag}
		f.mustCreate(&s.file)
		_ = f.attach.Write(tenantID, s.file.StoredName, []byte("%PDF-result-"+tag))
		reviewed := time.Now()
		order := clinical_models.ExamOrder{TenantID: tenantID, Number: uint(patientID), PatientID: patientID, OrderedByID: 1, Status: clinical_models.ExamOrderStatusCompleteResults}
		f.mustCreate(&order)
		item := clinical_models.ExamOrderItem{TenantID: tenantID, ExamOrderID: order.ID, Name: "Hemograma", Category: "laboratory", ReviewedAt: &reviewed}
		f.mustCreate(&item)
		if err := sys.Model(&item).Association("Attachments").Append(&s.file); err != nil {
			t.Fatal(err)
		}
		return s
	}
	mine := seed(f.own.ID, f.ownPat.ID, "mine")
	neighbour := seed(f.own.ID, f.ownPat2.ID, "neighbour")
	foreign := seed(f.other.ID, f.otherPat.ID, "foreign")

	token := f.signIn(f.own, f.ownPat)
	for _, path := range []string{"/portal/appointments", "/portal/invoices", "/portal/documents", "/portal/results"} {
		if n := listLen(t, f.do(http.MethodGet, path, token, nil)); n != 1 {
			t.Fatalf("%s: %d items, want 1", path, n)
		}
	}
	appts := f.do(http.MethodGet, "/portal/appointments", token, nil)
	if strings.Contains(string(appts.Raw), "internal") || strings.Contains(string(appts.Raw), "notes") {
		t.Fatalf("appointments leak internal fields: %s", appts.Raw)
	}

	get := func(path string) result { return f.do(http.MethodGet, path, token, nil) }
	if r := get(fmt.Sprintf("/portal/invoices/%d/ride", mine.invoice.ID)); r.Code != http.StatusOK || string(r.Raw) != "%PDF-ride-mine" {
		t.Fatalf("own RIDE: %d %s", r.Code, r.Raw)
	}
	if r := get(fmt.Sprintf("/portal/invoices/%d/xml", mine.invoice.ID)); r.Code != http.StatusOK || string(r.Raw) != "<factura>mine</factura>" {
		t.Fatalf("own XML: %d %s", r.Code, r.Raw)
	}
	if r := get(fmt.Sprintf("/portal/documents/certificate/%d/file", mine.cert.ID)); r.Code != http.StatusOK || string(r.Raw) != "%PDF-cert-mine" {
		t.Fatalf("own certificate: %d %s", r.Code, r.Raw)
	}
	if r := get(fmt.Sprintf("/portal/results/files/%d", mine.file.ID)); r.Code != http.StatusOK || string(r.Raw) != "%PDF-result-mine" {
		t.Fatalf("own result: %d %s", r.Code, r.Raw)
	}
	for _, s := range []seeded{neighbour, foreign} {
		for _, path := range []string{
			fmt.Sprintf("/portal/invoices/%d/ride", s.invoice.ID),
			fmt.Sprintf("/portal/invoices/%d/xml", s.invoice.ID),
			fmt.Sprintf("/portal/documents/certificate/%d/file", s.cert.ID),
			fmt.Sprintf("/portal/results/files/%d", s.file.ID),
		} {
			if r := get(path); r.Code != http.StatusNotFound {
				t.Fatalf("%s: %d %s", path, r.Code, r.Raw)
			}
		}
	}

	// Each download is audited with the portal patient as actor.
	var logs []audit.AuditLog
	sys.Where("tenant_id = ? AND actor_type = ?", f.own.ID, audit.ActorPortalPatient).Find(&logs)
	if len(logs) != 4 {
		t.Fatalf("audit rows = %d, want 4", len(logs))
	}
	for _, l := range logs {
		if l.UserID != 0 || l.PatientID == nil || *l.PatientID != f.ownPat.ID || l.Action != "READ" {
			t.Fatalf("audit row = %+v", l)
		}
	}
}

func TestResults_OnlyReviewedItemsAndNoVoidedOrders(t *testing.T) {
	f := newFixture(t)
	reviewed := time.Now()
	mk := func(number uint, status string, reviewedAt ...*time.Time) {
		order := clinical_models.ExamOrder{TenantID: f.own.ID, Number: number, PatientID: f.ownPat.ID, OrderedByID: 1, Status: status, DoctorID: &f.doctor.ID}
		f.mustCreate(&order)
		for i, r := range reviewedAt {
			f.mustCreate(&clinical_models.ExamOrderItem{TenantID: f.own.ID, ExamOrderID: order.ID, Name: fmt.Sprintf("Exam %d-%d", number, i), Category: "laboratory", ReviewedAt: r})
		}
	}
	mk(1, clinical_models.ExamOrderStatusPartialResults, &reviewed, nil) // one reviewed, one not
	mk(2, clinical_models.ExamOrderStatusIssued, nil)                    // nothing reviewed: omitted
	mk(3, clinical_models.ExamOrderStatusVoided, &reviewed)              // voided: omitted

	token := f.signIn(f.own, f.ownPat)
	r := f.do(http.MethodGet, "/portal/results", token, nil)
	if listLen(t, r) != 1 {
		t.Fatalf("orders = %s", r.Raw)
	}
	order := r.Body["data"].([]any)[0].(map[string]any)
	items := order["items"].([]any)
	if order["number"] != "ORD-000001" || len(items) != 1 || order["doctor_name"] != "Dra. Ana Ruiz" {
		t.Fatalf("order = %s", r.Raw)
	}
}

func TestDocuments_OnlySigned(t *testing.T) {
	f := newFixture(t)
	signedAt := time.Now()
	record := clinical_models.MedicalRecord{TenantID: f.own.ID, PatientID: f.ownPat.ID, Date: time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)}
	f.mustCreate(&record)
	presc := clinical_models.Prescription{MedicalRecordID: record.ID, Content: "x", DoctorID: &f.doctor.ID,
		DocumentSignature: clinical_models.DocumentSignature{SignedAt: &signedAt, SignedFile: "signed/presc.pdf"}}
	f.mustCreate(&presc)
	_ = f.files.Write(f.own.ID, "signed/presc.pdf", []byte("%PDF-presc"))
	// Unsigned ones: not listed, not downloadable.
	unsignedReport := clinical_models.MedicalReport{TenantID: f.own.ID, PatientID: f.ownPat.ID}
	f.mustCreate(&unsignedReport)
	unsignedCert := clinical_models.MedicalCertificate{TenantID: f.own.ID, PatientID: f.ownPat.ID}
	f.mustCreate(&unsignedCert)
	// A prescription of another patient's record of the same clinic.
	otherRecord := clinical_models.MedicalRecord{TenantID: f.own.ID, PatientID: f.ownPat2.ID, Date: time.Now()}
	f.mustCreate(&otherRecord)
	otherPresc := clinical_models.Prescription{MedicalRecordID: otherRecord.ID, DocumentSignature: clinical_models.DocumentSignature{SignedAt: &signedAt, SignedFile: "signed/other.pdf"}}
	f.mustCreate(&otherPresc)

	token := f.signIn(f.own, f.ownPat)
	r := f.do(http.MethodGet, "/portal/documents", token, nil)
	if listLen(t, r) != 1 {
		t.Fatalf("documents = %s", r.Raw)
	}
	doc := r.Body["data"].([]any)[0].(map[string]any)
	if doc["type"] != "prescription" || doc["date"] != "2026-09-01" || doc["doctor_name"] != "Dra. Ana Ruiz" {
		t.Fatalf("document = %v", doc)
	}
	if r := f.do(http.MethodGet, fmt.Sprintf("/portal/documents/prescription/%d/file", presc.ID), token, nil); r.Code != http.StatusOK || string(r.Raw) != "%PDF-presc" {
		t.Fatalf("download: %d %s", r.Code, r.Raw)
	}
	for _, path := range []string{
		fmt.Sprintf("/portal/documents/report/%d/file", unsignedReport.ID),
		fmt.Sprintf("/portal/documents/certificate/%d/file", unsignedCert.ID),
		fmt.Sprintf("/portal/documents/prescription/%d/file", otherPresc.ID),
		fmt.Sprintf("/portal/documents/bogus/%d/file", presc.ID),
	} {
		if r := f.do(http.MethodGet, path, token, nil); r.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", path, r.Code)
		}
	}
}

func TestInvoices_OnlyAuthorized(t *testing.T) {
	f := newFixture(t)
	pid := f.ownPat.ID
	for i, status := range []string{billing_models.InvoiceStatusAuthorized, "draft", "pending", "rejected", "validated"} {
		key := fmt.Sprintf("%049d", i+1)
		f.mustCreate(&billing_models.Invoice{TenantID: f.own.ID, PatientID: &pid, Status: status, AccessKey: &key, IssueDate: time.Now(),
			EstablishmentCode: "001", EmissionPointCode: "002", Sequential: fmt.Sprintf("%09d", i+1)})
	}
	token := f.signIn(f.own, f.ownPat)
	r := f.do(http.MethodGet, "/portal/invoices", token, nil)
	if listLen(t, r) != 1 || !strings.Contains(string(r.Raw), `"number":"001-002-000000001"`) {
		t.Fatalf("invoices = %s", r.Raw)
	}
	var draft billing_models.Invoice
	tenantdb.System(f.db).Where("status = ?", "draft").First(&draft)
	if r := f.do(http.MethodGet, fmt.Sprintf("/portal/invoices/%d/ride", draft.ID), token, nil); r.Code != http.StatusNotFound {
		t.Fatalf("draft RIDE: %d", r.Code)
	}
}

// --- gating ---

func TestSectionDisabled_Answers404(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(f.own, f.ownPat)
	f.setSettings(f.own, func(s *tenant_models.PortalSettings) { s.Sections.Invoices = false })
	if r := f.do(http.MethodGet, "/portal/invoices", token, nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-001" {
		t.Fatalf("hidden section: %d %s", r.Code, r.Raw)
	}
	if r := f.do(http.MethodGet, "/portal/invoices/1/xml", token, nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-001" {
		t.Fatalf("hidden section download: %d %s", r.Code, r.Raw)
	}
	if r := f.do(http.MethodGet, "/portal/appointments", token, nil); r.Code != http.StatusOK {
		t.Fatalf("visible section: %d", r.Code)
	}
	me := f.do(http.MethodGet, "/portal/me", token, nil)
	if !strings.Contains(string(me.Raw), `"invoices":false`) {
		t.Fatalf("me = %s", me.Raw)
	}
}

func TestPortalNotAvailable_Answers404(t *testing.T) {
	f := newFixture(t)
	basic := company_models.Plan{Name: "Basic", Code: "BASIC", Price: 1}
	f.mustCreate(&basic)
	noPlan := f.clinic("noplan", true, basic.Code)  // plan without the feature
	disabled := f.clinic("off", false, f.plan.Code) // feature, portal off
	f.patient(noPlan.ID, "222", "x@example.com", "")

	for name, slug := range map[string]string{"plan without feature": noPlan.Slug, "disabled": disabled.Slug, "unknown": "no-such-clinic"} {
		for _, r := range []result{
			f.do(http.MethodGet, "/portal/clinics/"+slug, "", nil),
			f.do(http.MethodPost, "/portal/clinics/"+slug+"/login", "", map[string]string{"document": "222"}),
		} {
			if r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-001" {
				t.Fatalf("%s: %d %s", name, r.Code, r.Raw)
			}
		}
	}
	if f.mail.calls != 0 {
		t.Fatal("no code may be sent for a closed portal")
	}
	// A session of a clinic whose portal closes stops working.
	token := f.signIn(f.own, f.ownPat)
	f.setSettings(f.own, func(s *tenant_models.PortalSettings) { s.Enabled = false })
	if r := f.do(http.MethodGet, "/portal/me", token, nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-001" {
		t.Fatalf("closed portal session: %d %s", r.Code, r.Raw)
	}
	// Same when the plan loses the feature.
	f.setSettings(f.own, func(s *tenant_models.PortalSettings) { s.Enabled = true })
	tenantdb.System(f.db).Model(&company_models.Subscription{}).Where("plan_code = ?", f.plan.Code).Update("plan_code", basic.Code)
	if r := f.do(http.MethodGet, "/portal/me", token, nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-001" {
		t.Fatalf("plan without feature session: %d %s", r.Code, r.Raw)
	}
}

func TestGetClinic_PublicCard(t *testing.T) {
	f := newFixture(t)
	r := f.do(http.MethodGet, "/portal/clinics/"+f.own.Slug, "", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("clinic: %d %s", r.Code, r.Raw)
	}
	var envelopeBody struct {
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(r.Raw, &envelopeBody)
	data := string(envelopeBody.Data)
	want := fmt.Sprintf(`{"name":"Clínica own","slug":"%s","has_logo":false,"sections":{"appointments":true,"results":true,"documents":true,"invoices":true},"booking":{"enabled":false,"choose_by":"doctor","new_patients":false},"channels":{"whatsapp":false,"email":true}}`, f.own.Slug)
	if data != want {
		t.Fatalf("clinic = %s", data)
	}
}

// --- WhatsApp delivery ---

func (f *fixture) connectWhatsApp(tenant tenant_models.Tenant, templateStatus string) {
	box, err := whatsapp_models.Box()
	if err != nil {
		f.t.Fatal(err)
	}
	account := whatsapp_models.WhatsAppAccount{TenantID: tenant.ID, WabaID: "WABA", PhoneNumberID: fmt.Sprintf("PH-%d", tenant.ID), Status: whatsapp_models.AccountStatusConnected}
	if err := account.SealAccessToken(box, "token"); err != nil {
		f.t.Fatal(err)
	}
	f.mustCreate(&account)
	f.mustCreate(&whatsapp_models.WhatsAppTemplate{TenantID: tenant.ID, Name: whatsapp_models.TemplatePortalCode, Status: templateStatus})
}

func TestLogin_WhatsAppDeliveryAndFallbacks(t *testing.T) {
	f := newFixture(t)
	withPhone := f.patient(f.own.ID, "3333333333", "tel@example.com", "0991234567")

	t.Run("template not approved: email", func(t *testing.T) {
		f.connectWhatsApp(f.own, whatsapp_models.TemplateStatusPending)
		f.login(f.own, withPhone.Document)
		if len(f.graph.sends()) != 0 || f.mail.last(withPhone.Email) == "" {
			t.Fatal("a pending template must fall back to email")
		}
		if r := f.do(http.MethodGet, "/portal/clinics/"+f.own.Slug, "", nil); strings.Contains(string(r.Raw), `"whatsapp":true`) {
			t.Fatalf("channels must not offer whatsapp: %s", r.Raw)
		}
	})

	tenantdb.System(f.db).Model(&whatsapp_models.WhatsAppTemplate{}).Where("tenant_id = ?", f.own.ID).Update("status", whatsapp_models.TemplateStatusApproved)
	tenantdb.System(f.db).Unscoped().Where("patient_id = ?", withPhone.ID).Delete(&portal_models.PortalLoginCode{})
	emailsBefore := f.mail.calls

	t.Run("approved: whatsapp, counted, no code stored in the body", func(t *testing.T) {
		f.login(f.own, withPhone.Document)
		sends := f.graph.sends()
		if len(sends) != 1 || !strings.Contains(sends[0], `"name":"portal_code"`) || !strings.Contains(sends[0], `"593991234567"`) || f.mail.calls != emailsBefore {
			t.Fatalf("sends = %v, emails %d", sends, f.mail.calls-emailsBefore)
		}
		var msg whatsapp_models.WhatsAppMessage
		tenantdb.System(f.db).Where("tenant_id = ? AND kind = ?", f.own.ID, whatsapp_models.KindPortalCode).First(&msg)
		if msg.SentAt == nil || msg.ConversationID != nil || strings.ContainsAny(msg.Body, "0123456789") {
			t.Fatalf("message = %+v", msg)
		}
		codes := f.codes(withPhone.ID)
		if len(codes) != 1 || codes[0].Channel != portal_models.ChannelWhatsApp {
			t.Fatalf("codes = %+v", codes)
		}
		if r := f.do(http.MethodGet, "/portal/clinics/"+f.own.Slug, "", nil); !strings.Contains(string(r.Raw), `"whatsapp":true`) {
			t.Fatalf("channels = %s", r.Raw)
		}
	})

	t.Run("monthly cap reached: email", func(t *testing.T) {
		tenantdb.System(f.db).Model(&company_models.Plan{}).Where("id = ?", f.plan.ID).
			Update("properties", datatypes.JSONMap{"max_whatsapp_messages": 1})
		sendsBefore := len(f.graph.sends())
		f.login(f.own, withPhone.Document)
		if len(f.graph.sends()) != sendsBefore || f.mail.calls != emailsBefore+1 {
			t.Fatalf("over the cap must go by email (sends %d, emails %d)", len(f.graph.sends())-sendsBefore, f.mail.calls-emailsBefore)
		}
	})

	t.Run("whatsapp send fails: email", func(t *testing.T) {
		tenantdb.System(f.db).Model(&company_models.Plan{}).Where("id = ?", f.plan.ID).Update("properties", datatypes.JSONMap{})
		tenantdb.System(f.db).Unscoped().Where("patient_id = ?", withPhone.ID).Delete(&portal_models.PortalLoginCode{})
		f.graph.mu.Lock()
		f.graph.fail = true
		f.graph.mu.Unlock()
		before := f.mail.calls
		f.login(f.own, withPhone.Document)
		codes := f.codes(withPhone.ID)
		if f.mail.calls != before+1 || len(codes) != 1 || codes[0].Channel != portal_models.ChannelEmail {
			t.Fatalf("fallback: emails %d codes %+v", f.mail.calls-before, codes)
		}
		if r := f.verify(f.own, withPhone.Document, f.mail.last(withPhone.Email)); r.Code != http.StatusOK {
			t.Fatalf("emailed code must work: %d", r.Code)
		}
	})
}
