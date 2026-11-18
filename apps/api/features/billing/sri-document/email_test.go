package sri_document

import (
	"errors"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/mailer"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
)

type sentEmail struct {
	to    string
	email mailer.SriDocumentEmail
}

type fakeMailer struct {
	err  error
	sent []sentEmail
}

func (m *fakeMailer) SendSriDocument(to string, e mailer.SriDocumentEmail) error {
	m.sent = append(m.sent, sentEmail{to, e})
	return m.err
}

const emailKey = "0109202601179001122300110010010000001231234567810"

// newEmailFixture is the lifecycle fixture with the real invoice email (buyer
// patient, RIDE, authorized XML) and a fake mailer.
func newEmailFixture(t *testing.T) (*fixture, *fakeMailer) {
	t.Helper()
	f := newFixture(t)
	if err := f.db.AutoMigrate(&clinical_models.Patient{}, &billing_models.InvoiceItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	f.tenant.CorporateName = "Clínica Demo S.A."
	if err := f.db.Model(&f.tenant).Update("corporate_name", f.tenant.CorporateName).Error; err != nil {
		t.Fatalf("update tenant: %v", err)
	}
	f.kind.EmailTitle = Invoice.EmailTitle
	f.kind.Email = Invoice.Email
	m := &fakeMailer{}
	f.docs.WithMailer(m, nil)
	return f, m
}

// emailInvoice is a failed invoice with a known access key and its stored RIDE,
// bought by a patient with email (no patient when email is nil).
func (f *fixture) emailInvoice(email *string) billing_models.Invoice {
	f.t.Helper()
	inv := f.invoice(billing_models.InvoiceStatusFailed)
	fields := map[string]any{"access_key": emailKey, "total": 46}
	if email != nil {
		patient := clinical_models.Patient{TenantID: f.tenant.ID, FirstName: "Ana", LastName: "Lopez",
			Document: "1710034065", Email: *email}
		if err := f.db.Create(&patient).Error; err != nil {
			f.t.Fatalf("create patient: %v", err)
		}
		fields["patient_id"] = patient.ID
	}
	if err := f.db.Model(&inv).Updates(fields).Error; err != nil {
		f.t.Fatalf("update invoice: %v", err)
	}
	_ = f.files.Write(f.tenant.ID, RideName(emailKey), []byte("%PDF-ride"))
	return inv
}

func ptr(s string) *string { return &s }

func TestEmail_AuthorizedInvoiceIsEmailedWithRideAndAuthorizedXML(t *testing.T) {
	f, m := newEmailFixture(t)
	inv := f.emailInvoice(ptr("  ana@example.com "))

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("emails sent = %d, want 1", len(m.sent))
	}
	sent := m.sent[0]
	if sent.to != "ana@example.com" {
		t.Errorf("to = %q, want the buyer's trimmed email", sent.to)
	}
	if want := "Factura 001-001-000000123 — Clínica Demo S.A."; sent.email.Subject != want {
		t.Errorf("subject = %q, want %q", sent.email.Subject, want)
	}
	body := strings.Join(sent.email.Lines, "\n")
	for _, want := range []string{"Clínica Demo S.A.", "001-001-000000123", "$46.00", "01/09/2026"} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q does not contain %q", body, want)
		}
	}
	if !strings.Contains(sent.email.Footer, emailKey) {
		t.Errorf("footer %q does not carry the access key", sent.email.Footer)
	}
	if len(sent.email.Attachments) != 2 {
		t.Fatalf("attachments = %d, want RIDE + XML", len(sent.email.Attachments))
	}
	pdf, xml := sent.email.Attachments[0], sent.email.Attachments[1]
	if pdf.Filename != emailKey+".pdf" || string(pdf.Content) != "%PDF-ride" {
		t.Errorf("RIDE attachment = %s %q", pdf.Filename, pdf.Content)
	}
	stored, _ := f.files.Read(f.tenant.ID, AuthorizedXMLName(f.kind, emailKey))
	if xml.Filename != emailKey+".xml" || !strings.Contains(string(xml.Content), "<autorizacion><estado>AUTORIZADO</estado>") ||
		string(xml.Content) != string(stored) {
		t.Errorf("XML attachment = %s %q, want the stored authorized envelope", xml.Filename, xml.Content)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q", got.Status)
	}
	if got.EmailedAt == nil || got.EmailedTo == nil || *got.EmailedTo != "ana@example.com" || got.EmailError != nil {
		t.Errorf("delivery = %v %v %v, want emailed to the buyer", got.EmailedAt, got.EmailedTo, got.EmailError)
	}

	// A second authorization run (duplicate task, sweep) never sends again.
	doc, err := f.docs.load(f.db, f.kind, uint64(inv.ID))
	if err != nil {
		t.Fatal(err)
	}
	f.docs.authorize(f.kind, doc, f.tenant, emailKey)
	if len(m.sent) != 1 {
		t.Errorf("emails sent after a re-run = %d, want 1", len(m.sent))
	}
}

func TestEmail_MailerFailureKeepsTheAuthorization(t *testing.T) {
	f, m := newEmailFixture(t)
	m.err = errors.New("resend API error: status 500")
	inv := f.emailInvoice(ptr("ana@example.com"))

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v, want the authorization to succeed", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized || got.AuthorizedAt == nil {
		t.Fatalf("status = %q authorized_at = %v, want authorized", got.Status, got.AuthorizedAt)
	}
	if got.EmailedAt != nil || got.EmailedTo != nil {
		t.Errorf("emailed_at = %v emailed_to = %v, want nil after a failed send", got.EmailedAt, got.EmailedTo)
	}
	if got.EmailError == nil || *got.EmailError != EmailErrorSendFailed {
		t.Errorf("email_error = %v, want %s", got.EmailError, EmailErrorSendFailed)
	}
}

func TestEmail_NoBuyerEmailSendsNothing(t *testing.T) {
	for name, email := range map[string]*string{"consumidor final": nil, "patient without email": ptr("  ")} {
		t.Run(name, func(t *testing.T) {
			f, m := newEmailFixture(t)
			inv := f.emailInvoice(email)
			if err := f.process(inv.ID); err != nil {
				t.Fatalf("process: %v", err)
			}
			got := f.reload(inv.ID)
			if len(m.sent) != 0 || got.EmailedAt != nil || got.EmailError != nil {
				t.Errorf("sent %d, emailed_at %v, email_error %v: want nothing", len(m.sent), got.EmailedAt, got.EmailError)
			}
			if _, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, ""); !errors.Is(err, ErrNoRecipient) {
				t.Errorf("SendEmail err = %v, want ErrNoRecipient", err)
			}
		})
	}
}

func TestEmail_InvalidBuyerEmailIsRecorded(t *testing.T) {
	f, m := newEmailFixture(t)
	inv := f.emailInvoice(ptr("ana at example"))
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(inv.ID)
	if len(m.sent) != 0 || got.EmailError == nil || *got.EmailError != EmailErrorInvalidRecipient {
		t.Errorf("sent %d, email_error %v: want nothing sent and %s", len(m.sent), got.EmailError, EmailErrorInvalidRecipient)
	}
}

func TestSendEmail_CooldownOverrideAndRestoreOnFailure(t *testing.T) {
	f, m := newEmailFixture(t)
	inv := f.emailInvoice(ptr("ana@example.com"))
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	if _, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, "otro@example.com"); !errors.Is(err, ErrEmailTooSoon) {
		t.Fatalf("err = %v, want ErrEmailTooSoon right after a send", err)
	}
	earlier := time.Now().Add(-2 * EmailCooldown)
	f.db.Model(&billing_models.Invoice{}).Where("id = ?", inv.ID).Update("emailed_at", earlier)

	if _, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, "Name <otro@example.com>"); !errors.Is(err, ErrInvalidRecipient) {
		t.Fatalf("err = %v, want ErrInvalidRecipient", err)
	}
	if got := f.reload(inv.ID); got.EmailError != nil {
		t.Errorf("email_error = %v: a mistyped address is not recorded", *got.EmailError)
	}

	state, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, " otro@example.com ")
	if err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	if len(m.sent) != 2 || m.sent[1].to != "otro@example.com" || state.EmailedTo == nil || *state.EmailedTo != "otro@example.com" {
		t.Fatalf("sent = %+v state = %+v, want the override", m.sent, state)
	}

	// A failed resend leaves the last successful send recorded.
	f.db.Model(&billing_models.Invoice{}).Where("id = ?", inv.ID).Update("emailed_at", earlier)
	m.err = errors.New("boom")
	if _, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, "tercero@example.com"); !errors.Is(err, ErrEmailSendFailed) {
		t.Fatalf("err = %v, want ErrEmailSendFailed", err)
	}
	got := f.reload(inv.ID)
	if got.EmailedTo == nil || *got.EmailedTo != "otro@example.com" || got.EmailedAt == nil || got.EmailedAt.Sub(earlier).Abs() > time.Second {
		t.Errorf("delivery = %v %v, want the previous send kept", got.EmailedAt, got.EmailedTo)
	}
	if got.EmailError == nil || *got.EmailError != EmailErrorSendFailed {
		t.Errorf("email_error = %v", got.EmailError)
	}
}

func TestSendEmail_OnlyAuthorizedDocuments(t *testing.T) {
	f, m := newEmailFixture(t)
	inv := f.emailInvoice(ptr("ana@example.com"))
	if _, err := f.docs.SendEmail(f.kind, f.tenant.ID, inv.ID, ""); !errors.Is(err, ErrNotAuthorized) {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
	if _, err := f.docs.SendEmail(f.kind, f.tenant.ID+1000, inv.ID, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("other tenant err = %v, want ErrNotFound", err)
	}
	if len(m.sent) != 0 {
		t.Errorf("sent %d emails", len(m.sent))
	}
}

func TestValidEmail(t *testing.T) {
	for s, want := range map[string]bool{
		"ana@example.com":        true,
		"ana.lopez+x@clinica.ec": true,
		"ana@localhost":          false,
		"Ana <ana@example.com>":  false,
		"a@example.com, b@x.com": false,
		"ana at example.com":     false,
		"":                       false,
	} {
		if got := validEmail(s); got != want {
			t.Errorf("validEmail(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestEmail_TextsAreInTheCatalog(t *testing.T) {
	keys := []string{"billing.document.email.subject", "billing.document.email.heading", "billing.document.email.intro",
		"billing.document.email.summary", "billing.document.email.attachments", "billing.document.email.footer",
		EmailErrorSendFailed, EmailErrorInvalidRecipient, EmailErrorAttachments}
	for _, kind := range Kinds {
		if kind.Email == nil || kind.EmailTitle == "" {
			t.Errorf("%s is not emailed", kind.Name)
		}
		keys = append(keys, kind.EmailTitle)
	}
	for _, key := range keys {
		if catalogText(key) == key {
			t.Errorf("%s is not in the message catalog", key)
		}
	}
}
