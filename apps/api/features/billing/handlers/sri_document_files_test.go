package billing_handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type pdfConverter struct{ calls int }

func (p *pdfConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	p.calls++
	return []byte("%PDF-" + fmt.Sprint(len(html))), nil
}

// filesFixture is two tenants, each with an authorized invoice, credit note and
// debit note, plus the lifecycle's in-memory files.
type filesFixture struct {
	db      *gorm.DB
	files   *tenantfiles.MemoryStore
	conv    *pdfConverter
	sri     *sri_document.Lifecycle
	tenants [2]uint
	invoice [2]uint
	credit  [2]uint
	debit   [2]uint
	key     map[string]string // kind name + tenant index -> access key
}

func newFilesFixture(t *testing.T) *filesFixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &clinical_models.Patient{},
		&billing_models.Invoice{}, &billing_models.InvoiceItem{}, &billing_models.CatalogItem{},
		&billing_models.CreditNote{}, &billing_models.CreditNoteItem{},
		&billing_models.DebitNote{}, &billing_models.DebitNoteMotive{},
	)
	f := &filesFixture{db: db, files: tenantfiles.Memory(), conv: &pdfConverter{}, key: map[string]string{}}
	f.sri = sri_document.New(db, zap.NewNop(), nil, nil,
		sri_document.Documents{Files: f.files, Renderer: pdfrender.New(f.files, f.conv)}, "1")

	stamp := time.Now().UnixNano() % 1000000
	now := time.Now()
	for i := range 2 {
		tenant := &tenant_models.Tenant{Name: "Files", Slug: fmt.Sprintf("files-%d-%d", stamp, i),
			DisplayToken: fmt.Sprintf("tok-files-%d-%d", stamp, i), TaxID: "1790000000001", CorporateName: "Demo"}
		mustCreate(t, db, tenant)
		f.tenants[i] = tenant.ID

		patient := &clinical_models.Patient{TenantID: tenant.ID, FirstName: "Ana", LastName: "Lopez",
			Document: fmt.Sprintf("17%08d", stamp+int64(i)), Email: "ana@example.com"}
		mustCreate(t, db, patient)
		product := &billing_models.CatalogItem{TenantID: tenant.ID, Name: "Consulta", SKU: "CONS-01", UnitPrice: 40}
		mustCreate(t, db, product)

		invKey := accessKey("01", stamp, i)
		invoice := &billing_models.Invoice{TenantID: tenant.ID, PatientID: testutils.Ptr(patient.ID), Sequential: "000000001",
			EstablishmentCode: "001", EmissionPointCode: "001", Status: billing_models.InvoiceStatusAuthorized,
			AccessKey: &invKey, AuthorizedAt: &now, IssueDate: now, PaymentMethod: "01", Total: 40}
		mustCreate(t, db, invoice)
		f.invoice[i] = invoice.ID

		cnKey := accessKey("04", stamp, i)
		credit := &billing_models.CreditNote{TenantID: tenant.ID, InvoiceID: invoice.ID, Sequential: "000000002",
			EstablishmentCode: "001", EmissionPointCode: "001", Status: billing_models.InvoiceStatusAuthorized,
			AccessKey: &cnKey, AuthorizedAt: &now, IssueDate: now, Reason: "Devolución", Subtotal: 40, Total: 40,
			Items: []billing_models.CreditNoteItem{{ProductID: product.ID, Description: "Consulta", Quantity: 1, UnitPrice: 40, Subtotal: 40}}}
		mustCreate(t, db, credit)
		f.credit[i] = credit.ID

		dnKey := accessKey("05", stamp, i)
		debit := &billing_models.DebitNote{TenantID: tenant.ID, InvoiceID: invoice.ID, Sequential: "000000003",
			EstablishmentCode: "001", EmissionPointCode: "001", Status: billing_models.InvoiceStatusAuthorized,
			AccessKey: &dnKey, AuthorizedAt: &now, IssueDate: now, Subtotal: 5, Total: 5,
			Motives: []billing_models.DebitNoteMotive{{Reason: "Interés por mora", Value: 5}}}
		mustCreate(t, db, debit)
		f.debit[i] = debit.ID

		for _, doc := range []struct {
			kind sri_document.Kind
			key  string
		}{{sri_document.Invoice, invKey}, {sri_document.CreditNote, cnKey}, {sri_document.DebitNote, dnKey}} {
			f.key[fmt.Sprint(doc.kind.Name, i)] = doc.key
			_ = f.files.Write(tenant.ID, sri_document.SignedXMLName(doc.kind, doc.key), []byte("<comprobante>"+doc.key+"</comprobante>"))
		}
	}
	return f
}

func mustCreate(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatalf("create %T: %v", v, err)
	}
}

// accessKey is a 49-digit key unique per test run, tenant and document code.
func accessKey(code string, stamp int64, i int) string {
	return fmt.Sprintf("16092026%s1790000000001100100%017d%d11", code, stamp*10+int64(i), i)
}

func (f *filesFixture) call(tenant int, id uint, handler func(*gin.Context)) *httptest.ResponseRecorder {
	c, w := testutils.NewGinContext(f.tenants[tenant], 1)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	handler(c)
	return w
}

func TestDownloadSignedXML_AuthorizedDocuments(t *testing.T) {
	f := newFilesFixture(t)
	cases := []struct {
		kind    sri_document.Kind
		id      uint
		handler func(*gin.Context)
	}{
		{sri_document.Invoice, f.invoice[0], NewInvoiceHandler(f.db, zap.NewNop(), f.sri).DownloadInvoiceXML},
		{sri_document.CreditNote, f.credit[0], NewCreditNoteHandler(f.db, zap.NewNop(), f.sri).DownloadCreditNoteXML},
		{sri_document.DebitNote, f.debit[0], NewDebitNoteHandler(f.db, zap.NewNop(), f.sri).DownloadDebitNoteXML},
	}
	for _, tc := range cases {
		t.Run(tc.kind.Name, func(t *testing.T) {
			key := f.key[tc.kind.Name+"0"]
			w := f.call(0, tc.id, tc.handler)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/xml") {
				t.Errorf("Content-Type = %q", ct)
			}
			if cd := w.Header().Get("Content-Disposition"); cd != "attachment; filename="+key+".xml" {
				t.Errorf("Content-Disposition = %q", cd)
			}
			if w.Body.String() != "<comprobante>"+key+"</comprobante>" {
				t.Errorf("body = %q, want the stored signed XML", w.Body.String())
			}

			// Another tenant's id is not found, never served.
			if w := f.call(1, tc.id, tc.handler); w.Code != http.StatusNotFound {
				t.Errorf("other tenant: status = %d, want 404", w.Code)
			}
		})
	}
}

func TestDownloadSriFiles_RequireAuthorization(t *testing.T) {
	f := newFilesFixture(t)
	for _, model := range []any{&billing_models.Invoice{}, &billing_models.CreditNote{}, &billing_models.DebitNote{}} {
		if err := f.db.Model(model).Where("tenant_id = ?", f.tenants[0]).Update("status", billing_models.InvoiceStatusValidated).Error; err != nil {
			t.Fatalf("set validated: %v", err)
		}
	}
	credit := NewCreditNoteHandler(f.db, zap.NewNop(), f.sri)
	debit := NewDebitNoteHandler(f.db, zap.NewNop(), f.sri)
	for name, call := range map[string]func() *httptest.ResponseRecorder{
		"invoice xml": func() *httptest.ResponseRecorder {
			return f.call(0, f.invoice[0], NewInvoiceHandler(f.db, zap.NewNop(), f.sri).DownloadInvoiceXML)
		},
		"credit note xml":  func() *httptest.ResponseRecorder { return f.call(0, f.credit[0], credit.DownloadCreditNoteXML) },
		"credit note ride": func() *httptest.ResponseRecorder { return f.call(0, f.credit[0], credit.DownloadCreditNoteRide) },
		"debit note xml":   func() *httptest.ResponseRecorder { return f.call(0, f.debit[0], debit.DownloadDebitNoteXML) },
		"debit note ride":  func() *httptest.ResponseRecorder { return f.call(0, f.debit[0], debit.DownloadDebitNoteRide) },
	} {
		w := call()
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "E-BILL-014") {
			t.Errorf("%s: status = %d body = %s, want 400 E-BILL-014", name, w.Code, w.Body.String())
		}
	}
}

func TestDownloadSignedXML_MissingFile(t *testing.T) {
	f := newFilesFixture(t)
	_ = f.files.Remove(f.tenants[0], sri_document.SignedXMLName(sri_document.CreditNote, f.key["credit_note0"]))
	w := f.call(0, f.credit[0], NewCreditNoteHandler(f.db, zap.NewNop(), f.sri).DownloadCreditNoteXML)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "E-BILL-015") {
		t.Errorf("status = %d body = %s, want 404 E-BILL-015", w.Code, w.Body.String())
	}
}

// A note authorized before RIDEs were stored has none: the download renders
// it, caches it, and serves the cached one from then on.
func TestDownloadNoteRide_RendersAndCaches(t *testing.T) {
	f := newFilesFixture(t)
	cases := []struct {
		kind    sri_document.Kind
		id      [2]uint
		prefix  string
		handler func(*gin.Context)
	}{
		{sri_document.CreditNote, f.credit, "nota_credito", NewCreditNoteHandler(f.db, zap.NewNop(), f.sri).DownloadCreditNoteRide},
		{sri_document.DebitNote, f.debit, "nota_debito", NewDebitNoteHandler(f.db, zap.NewNop(), f.sri).DownloadDebitNoteRide},
	}
	for _, tc := range cases {
		t.Run(tc.kind.Name, func(t *testing.T) {
			before := f.conv.calls
			w := f.call(0, tc.id[0], tc.handler)
			if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), "%PDF") {
				t.Fatalf("status = %d body = %.200s, want 200 PDF", w.Code, w.Body.String())
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
				t.Errorf("Content-Type = %q", ct)
			}
			if cd := w.Header().Get("Content-Disposition"); cd != "attachment; filename="+tc.prefix+"_001-001-"+map[string]string{"credit_note": "000000002", "debit_note": "000000003"}[tc.kind.Name]+".pdf" {
				t.Errorf("Content-Disposition = %q", cd)
			}
			name := sri_document.RideFileName(tc.kind, f.key[tc.kind.Name+"0"])
			if !f.files.Exists(f.tenants[0], name) {
				t.Fatalf("RIDE not cached at %s", name)
			}
			if w := f.call(0, tc.id[0], tc.handler); w.Code != http.StatusOK || f.conv.calls != before+1 {
				t.Errorf("second download: status %d, renders %d, want the cached RIDE", w.Code, f.conv.calls-before)
			}
			if w := f.call(1, tc.id[0], tc.handler); w.Code != http.StatusNotFound {
				t.Errorf("other tenant: status = %d, want 404", w.Code)
			}
		})
	}
}
