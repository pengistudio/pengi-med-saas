package services

import (
	"strings"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant "pengi-med-saas/features/tenants/models"
)

func testTenant() tenant.Tenant {
	t := tenant.Tenant{
		CorporateName:     "Consultorio Demo S.A.",
		TradeName:         "Consultorio Demo",
		TaxID:             "1790000000001",
		Address:           "Av. Siempre Viva 123",
		AccountingObliged: false,
	}
	t.ID = 1
	return t
}

func testPatient() clinical_models.Patient {
	return clinical_models.Patient{
		FirstName: "Juan",
		LastName:  "Perez",
		Document:  "1710000000",
	}
}

func TestGenerateInvoice_EnvironmentMatchesSriEnvArgument(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()

	invoice := billing_models.Invoice{
		DocumentCode: "01",
		EmissionType: "1",
		Sequential:   "000000001",
		IssueDate:    time.Now(),
		Patient:      &patient,
		Total:        100,
	}
	products := []CatalogItem{{SKU: "CONS-01", UnitPrice: 100, Tax: 0.12}}
	products[0].ID = 1

	// Regression test: TaxInfo.Environment (and the ambiente digit embedded in the
	// access key itself) used to be hardcoded to "1" (pruebas) regardless of the
	// actual SRI_ENV the document was being sent to.
	sriInvoice, accessKey, err := GenerateInvoice(invoice, products, tenantObj, "001", "001", "Dir", "2")
	if err != nil {
		t.Fatalf("GenerateInvoice failed: %v", err)
	}
	if sriInvoice.TaxInfo.Environment != "2" {
		t.Errorf("expected TaxInfo.Environment %q, got %q", "2", sriInvoice.TaxInfo.Environment)
	}
	// The ambiente digit sits right after the 8-digit date + 2-digit codDoc + 13-digit ruc.
	envDigitIndex := 8 + 2 + 13
	if string(accessKey[envDigitIndex]) != "2" {
		t.Errorf("expected access key ambiente digit at index %d to be '2', got access key %q", envDigitIndex, accessKey)
	}
}

func TestGenerateInvoice_ProducesValidAccessKeyAndXml(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()

	invoice := billing_models.Invoice{
		DocumentCode: "01",
		EmissionType: "1",
		Sequential:   "000000001",
		IssueDate:    time.Now(),
		Patient:      &patient,
		Subtotal:     100,
		Total:        112,
		Items: []billing_models.InvoiceItem{
			{
				ProductID:     1,
				Description:   "Consulta",
				Quantity:      1,
				UnitPrice:     100,
				TaxCode:       "2",
				TaxPercentage: "2",
				TaxRate:       0.12,
				Subtotal:      100,
			},
		},
	}

	products := []CatalogItem{
		{SKU: "CONS-01", UnitPrice: 100, Tax: 0.12},
	}
	products[0].ID = 1

	sriInvoice, accessKey, err := GenerateInvoice(invoice, products, tenantObj, "001", "001", "Dirección Establecimiento", "1")
	if err != nil {
		t.Fatalf("GenerateInvoice failed: %v", err)
	}
	if len(accessKey) != 49 {
		t.Errorf("expected 49-char access key, got %d chars: %s", len(accessKey), accessKey)
	}
	if sriInvoice.TaxInfo.DocumentCode != "01" {
		t.Errorf("expected codDoc 01, got %s", sriInvoice.TaxInfo.DocumentCode)
	}

	xmlStr, err := GenerateInvoiceXml(*sriInvoice)
	if err != nil {
		t.Fatalf("GenerateInvoiceXml failed: %v", err)
	}
	if !strings.Contains(xmlStr, "<factura ") {
		t.Errorf("expected <factura> root element, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<claveAcceso>"+accessKey+"</claveAcceso>") {
		t.Errorf("expected claveAcceso in XML to match generated access key")
	}
}

func TestGenerateInvoice_NilPatient_EmitsFinalConsumer(t *testing.T) {
	tenantObj := testTenant()

	invoice := billing_models.Invoice{
		DocumentCode: "01",
		EmissionType: "1",
		Sequential:   "000000001",
		IssueDate:    time.Now(),
		Subtotal:     100,
		Total:        112,
		Items: []billing_models.InvoiceItem{
			{
				ProductID:     1,
				Description:   "Consulta",
				Quantity:      1,
				UnitPrice:     100,
				TaxCode:       "2",
				TaxPercentage: "2",
				TaxRate:       0.12,
				Subtotal:      100,
			},
		},
	}

	products := []CatalogItem{
		{SKU: "CONS-01", UnitPrice: 100, Tax: 0.12},
	}
	products[0].ID = 1

	sriInvoice, accessKey, err := GenerateInvoice(invoice, products, tenantObj, "001", "001", "Dirección Establecimiento", "1")
	if err != nil {
		t.Fatalf("GenerateInvoice failed: %v", err)
	}

	xmlStr, err := GenerateInvoiceXml(*sriInvoice)
	if err != nil {
		t.Fatalf("GenerateInvoiceXml failed: %v", err)
	}
	if !strings.Contains(xmlStr, "<claveAcceso>"+accessKey+"</claveAcceso>") {
		t.Errorf("expected claveAcceso in XML to match generated access key")
	}
	if !strings.Contains(xmlStr, FinalConsumerIdentification) {
		t.Errorf("expected XML to contain Consumidor Final identification %q, got: %s", FinalConsumerIdentification, xmlStr)
	}
	if !strings.Contains(xmlStr, FinalConsumerIdentificationType) {
		t.Errorf("expected XML to contain Consumidor Final identification type %q, got: %s", FinalConsumerIdentificationType, xmlStr)
	}
	if !strings.Contains(xmlStr, FinalConsumerSocialReason) {
		t.Errorf("expected XML to contain Consumidor Final social reason %q, got: %s", FinalConsumerSocialReason, xmlStr)
	}
}

func TestGenerateCreditNote_ProducesValidXmlReferencingInvoice(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()

	originalInvoice := billing_models.Invoice{
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Sequential:        "000000042",
		IssueDate:         time.Now(),
		Patient:           &patient,
	}

	creditNote := billing_models.CreditNote{
		DocumentCode: billing_models.CreditNoteDocumentCode,
		EmissionType: "1",
		Sequential:   "000000001",
		IssueDate:    time.Now(),
		Reason:       "Anulación por error en el servicio facturado",
		Invoice:      originalInvoice,
		Subtotal:     100,
		Total:        112,
		Items: []billing_models.CreditNoteItem{
			{
				ProductID:     1,
				Description:   "Consulta",
				Quantity:      1,
				UnitPrice:     100,
				TaxCode:       "2",
				TaxPercentage: "2",
				TaxRate:       0.12,
				Subtotal:      100,
			},
		},
	}

	products := []CatalogItem{
		{SKU: "CONS-01", UnitPrice: 100, Tax: 0.12},
	}
	products[0].ID = 1

	sriCreditNote, accessKey, err := GenerateCreditNote(creditNote, products, tenantObj, "001", "001", "Dirección Establecimiento", "1")
	if err != nil {
		t.Fatalf("GenerateCreditNote failed: %v", err)
	}
	if sriCreditNote.TaxInfo.DocumentCode != billing_models.CreditNoteDocumentCode {
		t.Errorf("expected codDoc %s, got %s", billing_models.CreditNoteDocumentCode, sriCreditNote.TaxInfo.DocumentCode)
	}
	if sriCreditNote.CreditNoteInfo.ModifiedDocNumber != "001-001-000000042" {
		t.Errorf("expected numDocModificado 001-001-000000042, got %s", sriCreditNote.CreditNoteInfo.ModifiedDocNumber)
	}
	if sriCreditNote.CreditNoteInfo.Motivo != creditNote.Reason {
		t.Errorf("expected motivo %q, got %q", creditNote.Reason, sriCreditNote.CreditNoteInfo.Motivo)
	}

	xmlStr, err := GenerateCreditNoteXml(*sriCreditNote)
	if err != nil {
		t.Fatalf("GenerateCreditNoteXml failed: %v", err)
	}
	if !strings.Contains(xmlStr, "<notaCredito ") {
		t.Errorf("expected <notaCredito> root element, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<claveAcceso>"+accessKey+"</claveAcceso>") {
		t.Errorf("expected claveAcceso in XML to match generated access key")
	}
}

func TestGenerateDebitNote_ProducesValidXmlReferencingInvoice(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()

	originalInvoice := billing_models.Invoice{
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Sequential:        "000000042",
		IssueDate:         time.Now(),
		Patient:           &patient,
	}

	debitNote := billing_models.DebitNote{
		DocumentCode: billing_models.DebitNoteDocumentCode,
		EmissionType: "1",
		Sequential:   "000000001",
		IssueDate:    time.Now(),
		Invoice:      originalInvoice,
		Subtotal:     10,
		Total:        11.2,
		Motives: []billing_models.DebitNoteMotive{
			{
				Reason:            "Interés por mora",
				Value:             10,
				TaxCode:           "2",
				TaxPercentageCode: "2",
				TaxRate:           0.12,
			},
		},
	}

	sriDebitNote, accessKey, err := GenerateDebitNote(debitNote, tenantObj, "001", "001", "Dirección Establecimiento", "1")
	if err != nil {
		t.Fatalf("GenerateDebitNote failed: %v", err)
	}
	if sriDebitNote.TaxInfo.DocumentCode != billing_models.DebitNoteDocumentCode {
		t.Errorf("expected codDoc %s, got %s", billing_models.DebitNoteDocumentCode, sriDebitNote.TaxInfo.DocumentCode)
	}
	if sriDebitNote.DebitNoteInfo.ModifiedDocNumber != "001-001-000000042" {
		t.Errorf("expected numDocModificado 001-001-000000042, got %s", sriDebitNote.DebitNoteInfo.ModifiedDocNumber)
	}
	if len(sriDebitNote.Motives.Motive) != 1 || sriDebitNote.Motives.Motive[0].Reason != "Interés por mora" {
		t.Errorf("expected 1 motive with reason 'Interés por mora', got %+v", sriDebitNote.Motives.Motive)
	}

	xmlStr, err := GenerateDebitNoteXml(*sriDebitNote)
	if err != nil {
		t.Fatalf("GenerateDebitNoteXml failed: %v", err)
	}
	if !strings.Contains(xmlStr, "<notaDebito ") {
		t.Errorf("expected <notaDebito> root element, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<claveAcceso>"+accessKey+"</claveAcceso>") {
		t.Errorf("expected claveAcceso in XML to match generated access key")
	}
}
