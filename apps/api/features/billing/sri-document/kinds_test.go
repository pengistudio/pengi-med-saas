package sri_document

import (
	"fmt"
	"pengi-med-saas/core/tenantdb"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func TestInvoiceKind_BuildsXmlWithTheGivenAccessKeyAndFrozenPrices(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{},
		&billing_models.Invoice{}, &billing_models.InvoiceItem{}, &billing_models.CatalogItem{})

	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "T", Slug: fmt.Sprintf("k-%d", now), DisplayToken: fmt.Sprintf("tok-k-%d", now), TaxID: "1790011223001", Address: "Av. Siempre Viva 123"}
	db.Create(&tenant)
	product := billing_models.CatalogItem{TenantID: tenant.ID, Name: "Consulta", SKU: "CONS-01", UnitPrice: 99, Tax: 0.15}
	db.Create(&product)
	invoice := billing_models.Invoice{TenantID: tenant.ID, DocumentCode: "01", EmissionType: "1", Sequential: "000000007",
		IssueDate: time.Now(), EstablishmentCode: "001", EmissionPointCode: "001", Total: 57.5}
	db.Create(&invoice)
	db.Create(&billing_models.InvoiceItem{InvoiceID: invoice.ID, ProductID: product.ID, Quantity: 1, UnitPrice: 50, TaxRate: 0.15, Subtotal: 50})

	key := "0109202601179001122300110010010000000071234567811"
	xml, err := Invoice.BuildXML(tenantdb.System(db), invoice.ID, tenant, key, "1")
	if err != nil {
		t.Fatalf("BuildXML: %v", err)
	}
	if !strings.Contains(xml, "<claveAcceso>"+key+"</claveAcceso>") {
		t.Fatalf("XML does not declare the given access key:\n%s", xml)
	}
	if !strings.Contains(xml, "<precioUnitario>50") {
		t.Fatalf("XML does not use the price frozen on the item:\n%s", xml)
	}
}

type htmlConverter struct{ html string }

func (h *htmlConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	h.html = html
	return []byte("%PDF"), nil
}

// Once authorized, a nota stores its RIDE next to its signed XML, rendered from
// the system handle the lifecycle uses.
func TestNoteKinds_OnAuthorizedStoreTheRide(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{},
		&billing_models.Invoice{}, &billing_models.CatalogItem{},
		&billing_models.CreditNote{}, &billing_models.CreditNoteItem{},
		&billing_models.DebitNote{}, &billing_models.DebitNoteMotive{})

	now := time.Now()
	stamp := now.UnixNano()
	tenant := tenant_models.Tenant{Name: "T", Slug: fmt.Sprintf("ride-%d", stamp), DisplayToken: fmt.Sprintf("tok-ride-%d", stamp), TaxID: "1790011223001", Address: "Av. Siempre Viva 123"}
	db.Create(&tenant)
	product := billing_models.CatalogItem{TenantID: tenant.ID, Name: "Consulta", SKU: "CONS-77", UnitPrice: 50}
	db.Create(&product)
	invoice := billing_models.Invoice{TenantID: tenant.ID, Sequential: "000000009", EstablishmentCode: "001", EmissionPointCode: "001", IssueDate: now}
	db.Create(&invoice)

	cnKey := "1609202604179001122300110010010000000101234567811"
	creditNote := billing_models.CreditNote{TenantID: tenant.ID, InvoiceID: invoice.ID, AccessKey: &cnKey, AuthorizedAt: &now, IssueDate: now,
		Sequential: "000000010", EstablishmentCode: "001", EmissionPointCode: "001", Reason: "Devolución",
		Items: []billing_models.CreditNoteItem{{ProductID: product.ID, Description: "Consulta", Quantity: 1, UnitPrice: 50, Subtotal: 50}}}
	db.Create(&creditNote)
	dnKey := "1609202605179001122300110010010000000111234567817"
	debitNote := billing_models.DebitNote{TenantID: tenant.ID, InvoiceID: invoice.ID, AccessKey: &dnKey, AuthorizedAt: &now, IssueDate: now,
		Sequential: "000000011", EstablishmentCode: "001", EmissionPointCode: "001",
		Motives: []billing_models.DebitNoteMotive{{Reason: "Interés por mora", Value: 3}}}
	db.Create(&debitNote)

	files := tenantfiles.Memory()
	conv := &htmlConverter{}
	docs := Documents{Files: files, Renderer: pdfrender.New(files, conv)}

	if err := CreditNote.OnAuthorized(tenantdb.System(db), docs, creditNote.ID, tenant, "1"); err != nil {
		t.Fatalf("credit note OnAuthorized: %v", err)
	}
	if !files.Exists(tenant.ID, "credit-notes/"+cnKey+".pdf") {
		t.Fatal("credit note RIDE not stored next to its XML")
	}
	if !strings.Contains(conv.html, "CONS-77") || !strings.Contains(conv.html, "NOTA DE CR") {
		t.Error("credit note RIDE lacks the line's catalog code or its title")
	}

	if err := DebitNote.OnAuthorized(tenantdb.System(db), docs, debitNote.ID, tenant, "1"); err != nil {
		t.Fatalf("debit note OnAuthorized: %v", err)
	}
	if !files.Exists(tenant.ID, "debit-notes/"+dnKey+".pdf") {
		t.Fatal("debit note RIDE not stored next to its XML")
	}
	if !strings.Contains(conv.html, "Interés por mora") {
		t.Error("debit note RIDE lacks its motive")
	}
}
