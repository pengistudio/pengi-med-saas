package billing_handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/mailer"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_models "pengi-med-saas/features/clinical/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type emailMailer struct {
	err  error
	to   []string
	sent []mailer.SriDocumentEmail
}

func (m *emailMailer) SendSriDocument(to string, e mailer.SriDocumentEmail) error {
	m.to = append(m.to, to)
	m.sent = append(m.sent, e)
	return m.err
}

type emailCase struct {
	kind    sri_document.Kind
	id      uint
	model   any
	handler envelope.Action
}

func emailCases(f *filesFixture) []emailCase {
	return []emailCase{
		{sri_document.Invoice, f.invoice[0], &billing_models.Invoice{}, NewInvoiceHandler(f.db, zap.NewNop(), f.sri).EmailInvoice},
		{sri_document.CreditNote, f.credit[0], &billing_models.CreditNote{}, NewCreditNoteHandler(f.db, zap.NewNop(), f.sri).EmailCreditNote},
		{sri_document.DebitNote, f.debit[0], &billing_models.DebitNote{}, NewDebitNoteHandler(f.db, zap.NewNop(), f.sri).EmailDebitNote},
	}
}

func (f *filesFixture) post(tenant int, id uint, body string, handler envelope.Action) *httptest.ResponseRecorder {
	c, w := testutils.NewGinContext(f.tenants[tenant], 1)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	envelope.Handle(handler)(c)
	return w
}

func emailState(t *testing.T, w *httptest.ResponseRecorder) billing_models.EmailDelivery {
	t.Helper()
	var resp struct {
		Data billing_models.EmailDelivery `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return resp.Data
}

func TestEmailSriDocument_SendsToTheBuyer(t *testing.T) {
	f := newFilesFixture(t)
	m := &emailMailer{}
	f.sri.WithMailer(m, nil)
	for i, tc := range emailCases(f) {
		t.Run(tc.kind.Name, func(t *testing.T) {
			key := f.key[tc.kind.Name+"0"]
			w := f.post(0, tc.id, "", tc.handler)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", w.Code, w.Body.String())
			}
			state := emailState(t, w)
			if state.EmailedAt == nil || state.EmailedTo == nil || *state.EmailedTo != "ana@example.com" || state.EmailError != nil {
				t.Errorf("state = %+v", state)
			}
			// Notes go to the patient of their invoice.
			if len(m.to) != i+1 || m.to[i] != "ana@example.com" {
				t.Fatalf("recipients = %v", m.to)
			}
			atts := m.sent[i].Attachments
			if len(atts) != 2 || atts[0].Filename != key+".pdf" || !strings.HasPrefix(string(atts[0].Content), "%PDF") ||
				atts[1].Filename != key+".xml" || string(atts[1].Content) != "<autorizacion>"+key+"</autorizacion>" {
				t.Errorf("attachments = %+v, want the RIDE and the authorized XML", atts)
			}
			var saved billing_models.EmailDelivery
			f.db.Model(tc.model).Where("id = ?", tc.id).Take(&saved)
			if saved.EmailedTo == nil || *saved.EmailedTo != "ana@example.com" || saved.EmailedAt == nil {
				t.Errorf("persisted = %+v", saved)
			}

			// Right after a send: refused, nothing sent.
			if w := f.post(0, tc.id, `{"email":"otro@example.com"}`, tc.handler); w.Code != http.StatusTooManyRequests ||
				!strings.Contains(w.Body.String(), "E-BILL-020") {
				t.Errorf("resend: status = %d body = %s, want 429 E-BILL-020", w.Code, w.Body.String())
			}
			if len(m.to) != i+1 {
				t.Errorf("sent during the cooldown")
			}
		})
	}
	subjects := []string{m.sent[0].Subject, m.sent[1].Subject, m.sent[2].Subject}
	for i, want := range []string{"Factura 001-001-000000001 — Demo", "Nota de crédito 001-001-000000002 — Demo", "Nota de débito 001-001-000000003 — Demo"} {
		if subjects[i] != want {
			t.Errorf("subject = %q, want %q", subjects[i], want)
		}
	}
}

func TestEmailSriDocument_Errors(t *testing.T) {
	f := newFilesFixture(t)
	m := &emailMailer{}
	f.sri.WithMailer(m, nil)
	invoice := NewInvoiceHandler(f.db, zap.NewNop(), f.sri).EmailInvoice

	if w := f.post(0, f.invoice[0], `{"email":"Ana <ana@example.com>"}`, invoice); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "E-BILL-018") {
		t.Errorf("invalid email: status = %d body = %s, want 400 E-BILL-018", w.Code, w.Body.String())
	}
	if w := f.post(1, f.invoice[0], "", invoice); w.Code != http.StatusNotFound {
		t.Errorf("other tenant: status = %d, want 404", w.Code)
	}
	if w := f.post(0, f.invoice[0], `{"email":`, invoice); w.Code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", w.Code)
	}

	m.err = errors.New("resend API error: status 500")
	if w := f.post(0, f.invoice[0], "", invoice); w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "E-BILL-019") {
		t.Errorf("send failure: status = %d body = %s, want 502 E-BILL-019", w.Code, w.Body.String())
	}
	var saved billing_models.Invoice
	f.db.First(&saved, f.invoice[0])
	if saved.EmailedAt != nil || saved.EmailError == nil || *saved.EmailError != sri_document.EmailErrorSendFailed {
		t.Errorf("after a failure: emailed_at = %v email_error = %v", saved.EmailedAt, saved.EmailError)
	}
	m.err = nil

	f.db.Model(&clinical_models.Patient{}).Where("tenant_id = ?", f.tenants[0]).Update("email", "")
	if w := f.post(0, f.invoice[0], "", invoice); w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "E-BILL-017") {
		t.Errorf("no recipient: status = %d body = %s, want 422 E-BILL-017", w.Code, w.Body.String())
	}
	// An explicit address still works for a buyer without email.
	if w := f.post(0, f.invoice[0], `{"email":"caja@example.com"}`, invoice); w.Code != http.StatusOK || m.to[len(m.to)-1] != "caja@example.com" {
		t.Errorf("override: status = %d body = %s", w.Code, w.Body.String())
	}

	for _, tc := range emailCases(f) {
		f.db.Model(tc.model).Where("id = ?", tc.id).Update("status", billing_models.InvoiceStatusValidated)
		if w := f.post(0, tc.id, "", tc.handler); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "E-BILL-014") {
			t.Errorf("%s not authorized: status = %d body = %s, want 400 E-BILL-014", tc.kind.Name, w.Code, w.Body.String())
		}
	}
}

func TestListEndpoints_ReturnEmailDelivery(t *testing.T) {
	f := newFilesFixture(t)
	for name, handler := range map[string]envelope.Action{
		"invoices":     NewInvoiceHandler(f.db, zap.NewNop(), f.sri).GetAllInvoices,
		"credit notes": NewCreditNoteHandler(f.db, zap.NewNop(), f.sri).GetAllCreditNotes,
		"debit notes":  NewDebitNoteHandler(f.db, zap.NewNop(), f.sri).GetAllDebitNotes,
	} {
		c, w := testutils.NewGinContext(f.tenants[0], 1)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		envelope.Handle(handler)(c)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d: %s", name, w.Code, w.Body.String())
		}
		for _, field := range []string{`"emailed_at":null`, `"emailed_to":null`, `"email_error":null`} {
			if !strings.Contains(w.Body.String(), field) {
				t.Errorf("%s: response has no %s", name, field)
			}
		}
	}
}
