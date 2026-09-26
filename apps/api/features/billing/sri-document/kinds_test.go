package sri_document

import (
	"fmt"
	"strings"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func TestInvoiceKind_BuildsXmlWithTheGivenAccessKeyAndFrozenPrices(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{},
		&billing_models.Invoice{}, &billing_models.InvoiceItem{}, &billing_models.CatalogItem{})

	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "T", Slug: fmt.Sprintf("k-%d", now), DisplayToken: fmt.Sprintf("tok-k-%d", now), TaxID: "1790011223001"}
	db.Create(&tenant)
	product := billing_models.CatalogItem{TenantID: tenant.ID, Name: "Consulta", SKU: "CONS-01", UnitPrice: 99, Tax: 0.15}
	db.Create(&product)
	invoice := billing_models.Invoice{TenantID: tenant.ID, DocumentCode: "01", EmissionType: "1", Sequential: "000000007",
		IssueDate: time.Now(), EstablishmentCode: "001", EmissionPointCode: "001", Total: 57.5}
	db.Create(&invoice)
	db.Create(&billing_models.InvoiceItem{InvoiceID: invoice.ID, ProductID: product.ID, Quantity: 1, UnitPrice: 50, TaxRate: 0.15, Subtotal: 50})

	key := "0109202601179001122300110010010000000071234567811"
	xml, err := Invoice.BuildXML(db, invoice.ID, tenant, key, "1")
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
