package sri_document

import (
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// newBuyerFixture is the lifecycle fixture with the real invoice XML build, so
// the buyer snapshot is written as in production.
func newBuyerFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	if err := f.db.AutoMigrate(&clinical_models.Patient{}, &billing_models.Customer{}, &billing_models.InvoiceItem{},
		&billing_models.CatalogItem{}, &billing_models.CreditNote{}, &billing_models.CreditNoteItem{},
		&billing_models.DebitNote{}, &billing_models.DebitNoteMotive{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	f.kind.BuildXML = Invoice.BuildXML
	f.kind.Email = Invoice.Email
	return f
}

func (f *fixture) create(v any) {
	f.t.Helper()
	if err := f.db.Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

// customerInvoice is a pending invoice of the fixture's tenant bought by a
// customer (RUC) for a patient.
func (f *fixture) customerInvoice() (billing_models.Invoice, billing_models.Customer) {
	f.t.Helper()
	patient := clinical_models.Patient{TenantID: f.tenant.ID, FirstName: "Ana", LastName: "Lopez",
		Document: "1710034065", Email: "ana@example.com"}
	f.create(&patient)
	customer := billing_models.Customer{TenantID: f.tenant.ID, Identification: "1790012345001",
		Name: "ASEGURADORA V1", Email: "v1@aseguradora.ec"}
	f.create(&customer)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.db.Model(&inv).Updates(map[string]any{"patient_id": patient.ID, "customer_id": customer.ID}).Error; err != nil {
		f.t.Fatalf("update invoice: %v", err)
	}
	return f.reload(inv.ID), customer
}

func (f *fixture) renameCustomer(c billing_models.Customer, name, email string) {
	f.t.Helper()
	if err := f.db.Model(&c).Updates(map[string]any{"name": name, "email": email}).Error; err != nil {
		f.t.Fatalf("update customer: %v", err)
	}
}

func (f *fixture) emailRecipient(kind Kind, id uint) string {
	f.t.Helper()
	content, err := kind.Email(f.db, f.docs.Documents(), id, f.tenant)
	if err != nil {
		f.t.Fatalf("email content: %v", err)
	}
	return content.BuyerEmail
}

func TestBuyerSnapshot_WrittenAtBuildRefreshedOnRetryFrozenOnceReceived(t *testing.T) {
	f := newBuyerFixture(t)
	inv, customer := f.customerInvoice()
	if !inv.BuyerSnapshot.IsEmpty() {
		t.Fatalf("fixture invoice starts with a snapshot")
	}

	// First attempt: NO AUTORIZADO. The XML declared the customer as of then.
	f.gateway.auth = Authorization{Status: NotAuthorized, Message: "ERROR"}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusRejected {
		t.Fatalf("status = %q, want rejected", got.Status)
	}
	if got.BuyerName != "ASEGURADORA V1" || got.BuyerIdentification != "1790012345001" || got.BuyerIdentificationType != "04" || got.BuyerEmail != "v1@aseguradora.ec" {
		t.Fatalf("snapshot after first build = %+v", got.BuyerSnapshot)
	}
	first := f.gateway.signedInputs[0]
	for _, want := range []string{"<razonSocialComprador>ASEGURADORA V1</razonSocialComprador>",
		"<identificacionComprador>1790012345001</identificacionComprador>",
		`<campoAdicional nombre="Paciente">Ana Lopez</campoAdicional>`} {
		if !strings.Contains(first, want) {
			t.Fatalf("first XML lacks %s:\n%s", want, first)
		}
	}

	// The user corrects the customer and retries: same key, new buyer data.
	f.renameCustomer(customer, "ASEGURADORA V2", "v2@aseguradora.ec")
	if err := f.docs.Enqueue(tenantdb.ForTenant(f.db, f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	f.gateway.auth = Authorization{Status: Authorized, AuthorizedAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process retry: %v", err)
	}
	got = f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized || got.BuyerName != "ASEGURADORA V2" || got.BuyerEmail != "v2@aseguradora.ec" {
		t.Fatalf("after retry: status=%q snapshot=%+v", got.Status, got.BuyerSnapshot)
	}
	if !strings.Contains(f.gateway.signedInputs[1], "<razonSocialComprador>ASEGURADORA V2</razonSocialComprador>") {
		t.Fatalf("retry XML does not declare the corrected customer:\n%s", f.gateway.signedInputs[1])
	}
	signedXML := f.signedXML(got)

	// Authorized: editing or deleting the customer changes nothing stored.
	f.renameCustomer(customer, "ASEGURADORA V3", "v3@aseguradora.ec")
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process authorized: %v", err)
	}
	if err := f.db.Delete(&customer).Error; err != nil {
		t.Fatalf("delete customer: %v", err)
	}
	got = f.reload(inv.ID)
	if got.BuyerName != "ASEGURADORA V2" || got.BuyerEmail != "v2@aseguradora.ec" {
		t.Fatalf("authorized snapshot rewritten: %+v", got.BuyerSnapshot)
	}
	if f.gateway.signCalls != 2 || f.signedXML(got) != signedXML {
		t.Fatalf("authorized invoice was signed again (sign calls %d)", f.gateway.signCalls)
	}
	if to := f.emailRecipient(Invoice, inv.ID); to != "v2@aseguradora.ec" {
		t.Fatalf("email recipient = %q, want the buyer as authorized", to)
	}
}

func TestBuyerSnapshot_NotRewrittenWhileReceived(t *testing.T) {
	f := newBuyerFixture(t)
	inv, customer := f.customerInvoice()
	f.gateway.auth = Authorization{Status: AuthorizationPending}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusValidated {
		t.Fatalf("status = %q, want validated", got.Status)
	}

	f.renameCustomer(customer, "ASEGURADORA V2", "v2@aseguradora.ec")
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process received: %v", err)
	}
	got := f.reload(inv.ID)
	if got.BuyerName != "ASEGURADORA V1" || f.gateway.signCalls != 1 {
		t.Fatalf("received invoice rebuilt: snapshot=%+v sign calls=%d", got.BuyerSnapshot, f.gateway.signCalls)
	}
}

func TestBuyerSnapshot_LegacyAndFinalConsumer(t *testing.T) {
	f := newBuyerFixture(t)

	// An authorized invoice from before snapshots: the buyer is the patient, live.
	patient := clinical_models.Patient{TenantID: f.tenant.ID, FirstName: "Luis", LastName: "Mora",
		Document: "1710034065", Email: "luis@example.com"}
	f.create(&patient)
	legacy := f.invoice(billing_models.InvoiceStatusAuthorized)
	if err := f.db.Model(&legacy).Updates(map[string]any{"patient_id": patient.ID, "access_key": emailKey}).Error; err != nil {
		t.Fatalf("update invoice: %v", err)
	}
	if to := f.emailRecipient(Invoice, legacy.ID); to != "luis@example.com" {
		t.Fatalf("legacy email recipient = %q, want the patient's", to)
	}

	// No patient, no customer: Consumidor Final is declared and recorded.
	fc := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.process(fc.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(fc.ID)
	if got.BuyerIdentification != "9999999999999" || got.BuyerIdentificationType != "07" || got.BuyerName != "CONSUMIDOR FINAL" {
		t.Fatalf("Consumidor Final snapshot = %+v", got.BuyerSnapshot)
	}
	if strings.Contains(f.gateway.signedInputs[0], "infoAdicional") {
		t.Fatalf("Consumidor Final XML has infoAdicional:\n%s", f.gateway.signedInputs[0])
	}
}

func TestBuyerSnapshot_OtherTenantsCustomerFailsTheBuild(t *testing.T) {
	f := newBuyerFixture(t)
	foreign := billing_models.Customer{TenantID: f.tenant.ID + 1000, Identification: "1790012345001", Name: "OTRO"}
	f.create(&foreign)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.db.Model(&inv).Update("customer_id", foreign.ID).Error; err != nil {
		t.Fatalf("update invoice: %v", err)
	}
	_ = f.process(inv.ID)
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || f.gateway.signCalls != 0 || !got.BuyerSnapshot.IsEmpty() {
		t.Fatalf("status=%q sign calls=%d snapshot=%+v, want failed before signing", got.Status, f.gateway.signCalls, got.BuyerSnapshot)
	}
}

func TestNotes_EmailTheBuyerOfTheirInvoice(t *testing.T) {
	f := newBuyerFixture(t)
	inv, _ := f.customerInvoice()
	snapshot := billing_models.BuyerSnapshot{BuyerIdentification: "1790012345001", BuyerIdentificationType: "04",
		BuyerName: "ASEGURADORA V1", BuyerEmail: " pagos@aseguradora.ec "}
	if err := f.db.Model(&inv).Updates(snapshot.Columns()).Error; err != nil {
		t.Fatalf("update invoice: %v", err)
	}
	key := emailKey
	credit := billing_models.CreditNote{TenantID: f.tenant.ID, InvoiceID: inv.ID, AccessKey: &key}
	f.create(&credit)
	debit := billing_models.DebitNote{TenantID: f.tenant.ID, InvoiceID: inv.ID, AccessKey: &key}
	f.create(&debit)

	if to := f.emailRecipient(CreditNote, credit.ID); to != "pagos@aseguradora.ec" {
		t.Fatalf("credit note recipient = %q", to)
	}
	if to := f.emailRecipient(DebitNote, debit.ID); to != "pagos@aseguradora.ec" {
		t.Fatalf("debit note recipient = %q", to)
	}
}
