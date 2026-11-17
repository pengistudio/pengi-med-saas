package services

import (
	"html"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	billing_models "pengi-med-saas/features/billing/models"
	billing_templates "pengi-med-saas/features/billing/templates"
	tenant "pengi-med-saas/features/tenants/models"
)

const (
	testCreditNoteKey = "1609202604179000000000110010010000000051234567811"
	testDebitNoteKey  = "1609202605179000000000110010010000000071234567817"
)

func testModifiedInvoice() billing_models.Invoice {
	patient := testPatient()
	patient.ID = 1
	patient.Email = "juan@example.com"
	invoiceIssued := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	return billing_models.Invoice{
		EstablishmentCode: "001",
		EmissionPointCode: "002",
		Sequential:        "000000123",
		IssueDate:         invoiceIssued,
		PatientID:         &patient.ID,
		Patient:           &patient,
		PaymentMethod:     "19",
	}
}

// testRideTenant sets every optional issuer field the RIDE header prints.
func testRideTenant() tenantRideFixture {
	tenantObj := testTenant()
	tenantObj.AccountingObliged = true
	rimpe := "CONTRIBUYENTE RÉGIMEN RIMPE"
	agent := "NAC-DNCRASC20-00000001"
	special := "12345"
	tenantObj.RimpeTaxpayer = &rimpe
	tenantObj.WithholdingAgent = &agent
	tenantObj.SpecialContributorNumber = &special
	return tenantRideFixture{tenantObj}
}

type tenantRideFixture struct{ tenant tenant.Tenant }

// renderedHTML renders doc's default template with data, as the PDF will show it.
func renderedHTML(t *testing.T, doc pdfrender.Document, data any) string {
	t.Helper()
	conv := &capturingConverter{}
	if _, err := pdfrender.New(tenantfiles.Memory(), conv).Render(1, doc, data); err != nil {
		t.Fatalf("render %s: %v", doc.ID, err)
	}
	return html.UnescapeString(conv.html)
}

func assertContains(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("rendered RIDE is missing %q", w)
		}
	}
}

func TestBuildCreditNoteRideData_AndRenderTemplate(t *testing.T) {
	fx := testRideTenant()
	key := testCreditNoteKey
	authorizedAt := time.Date(2026, 9, 16, 9, 15, 42, 0, time.UTC)
	creditNote := billing_models.CreditNote{
		AccessKey:         &key,
		AuthorizedAt:      &authorizedAt,
		IssueDate:         time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC),
		EstablishmentCode: "001",
		EmissionPointCode: "002",
		Sequential:        "000000005",
		Reason:            "Devolución por servicio no prestado",
		Invoice:           testModifiedInvoice(),
		Subtotal:          45,
		Total:             51.75,
		Items: []billing_models.CreditNoteItem{
			{ProductID: 7, Description: "Ecografía", Quantity: 1, UnitPrice: 50, Discount: 5, Subtotal: 45,
				TaxCode: "2", TaxPercentage: "4", TaxRate: 0.15},
		},
	}

	data, err := buildCreditNoteRideData(creditNote, map[uint]string{7: "ECO-01"}, fx.tenant, "Sucursal Norte", "2")
	if err != nil {
		t.Fatalf("buildCreditNoteRideData: %v", err)
	}
	if data.Environment != "PRODUCCIÓN" || data.AccountingObliged != "SI" || data.BarcodeBase64 == "" {
		t.Errorf("unexpected header: env=%q obligado=%q barcode=%v", data.Environment, data.AccountingObliged, data.BarcodeBase64 != "")
	}
	if len(data.TaxSummary) != 1 || data.TaxSummary[0].Label != "IVA 15%" || data.TaxSummary[0].Value != "6.75" {
		t.Fatalf("unexpected tax summary: %+v", data.TaxSummary)
	}

	out := renderedHTML(t, billing_templates.CreditNoteRide, data)
	assertContains(t, out,
		"NOTA DE CRÉDITO", "R.U.C.: 1790000000001", "No. 001-002-000000005",
		"NÚMERO DE AUTORIZACIÓN", key, "16/09/2026 09:15:42", "AMBIENTE: PRODUCCIÓN", "EMISIÓN: NORMAL",
		"OBLIGADO A LLEVAR CONTABILIDAD: SI", "Contribuyente Especial Nro: 12345",
		"Agente de Retención Resolución No. NAC-DNCRASC20-00000001", "CONTRIBUYENTE RÉGIMEN RIMPE",
		"Dir. Sucursal: Sucursal Norte",
		"Juan Perez", "1710000000", "Fecha Emisión: 16/09/2026",
		"FACTURA 001-002-000000123", "Fecha Emisión (Comprobante a modificar): 15/09/2026",
		"Razón de Modificación: Devolución por servicio no prestado",
		"ECO-01", "Ecografía", "50.00", "-5.00", "IVA 15%", "6.75", "VALOR TOTAL", "51.75",
		"Información Adicional", "juan@example.com",
		"data:image/png;base64,",
	)
}

func TestBuildDebitNoteRideData_AndRenderTemplate(t *testing.T) {
	fx := testRideTenant()
	key := testDebitNoteKey
	authorizedAt := time.Date(2026, 9, 17, 11, 0, 0, 0, time.UTC)
	debitNote := billing_models.DebitNote{
		AccessKey:         &key,
		AuthorizedAt:      &authorizedAt,
		IssueDate:         time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC),
		EstablishmentCode: "001",
		EmissionPointCode: "002",
		Sequential:        "000000007",
		Invoice:           testModifiedInvoice(),
		Subtotal:          10,
		Total:             11.5,
		Motives: []billing_models.DebitNoteMotive{
			{Reason: "Interés por mora", Value: 4, TaxCode: "2", TaxPercentageCode: "4", TaxRate: 0.15},
			{Reason: "Gastos de cobranza", Value: 6, TaxCode: "2", TaxPercentageCode: "4", TaxRate: 0.15},
		},
	}

	data, err := buildDebitNoteRideData(debitNote, fx.tenant, "Sucursal Norte", "1")
	if err != nil {
		t.Fatalf("buildDebitNoteRideData: %v", err)
	}
	if len(data.TaxSummary) != 1 || data.TaxSummary[0].Base != "10.00" || data.TaxSummary[0].Value != "1.50" {
		t.Fatalf("unexpected tax summary: %+v", data.TaxSummary)
	}
	if len(data.Payments) != 1 || data.Payments[0].Method != "TARJETA DE CRÉDITO" || data.Payments[0].Value != "11.50" {
		t.Fatalf("unexpected payments: %+v", data.Payments)
	}

	out := renderedHTML(t, billing_templates.DebitNoteRide, data)
	assertContains(t, out,
		"NOTA DE DÉBITO", "No. 001-002-000000007", key, "AMBIENTE: PRUEBAS",
		"FACTURA 001-002-000000123", "Fecha Emisión (Comprobante a modificar): 15/09/2026",
		"RAZÓN DE LA MODIFICACIÓN", "VALOR DE LA MODIFICACIÓN", "Interés por mora", "4.00", "Gastos de cobranza", "6.00",
		"Forma de pago", "TARJETA DE CRÉDITO", "11.50", "IVA 15%", "1.50", "VALOR TOTAL",
		"Información Adicional", "juan@example.com",
	)
}

func TestNoteRideData_ConsumidorFinalAndPlainTenant(t *testing.T) {
	key := testDebitNoteKey
	invoice := testModifiedInvoice()
	invoice.PatientID, invoice.Patient, invoice.PaymentMethod = nil, nil, ""
	debitNote := billing_models.DebitNote{AccessKey: &key, IssueDate: time.Now(), Invoice: invoice}

	data, err := buildDebitNoteRideData(debitNote, testTenant(), "Dir", "1")
	if err != nil {
		t.Fatalf("buildDebitNoteRideData: %v", err)
	}
	if data.BuyerName != FinalConsumerSocialReason || data.BuyerIdentification != FinalConsumerIdentification {
		t.Errorf("expected Consumidor Final buyer, got %q %q", data.BuyerName, data.BuyerIdentification)
	}
	if data.AccountingObliged != "NO" || data.RimpeTaxpayer != "" || data.AdditionalInfo != nil || data.Payments != nil {
		t.Errorf("unexpected optional fields: %+v", data.NoteRideHeader)
	}
	out := renderedHTML(t, billing_templates.DebitNoteRide, data)
	if strings.Contains(out, "Agente de Retención") || strings.Contains(out, "Contribuyente Especial") || strings.Contains(out, "Información Adicional") {
		t.Error("optional rows must not print when not set")
	}
}

func TestNoteRideData_RequiresAccessKey(t *testing.T) {
	if _, err := buildCreditNoteRideData(billing_models.CreditNote{}, nil, testTenant(), "Dir", "1"); err == nil {
		t.Error("credit note without access key: expected error")
	}
	if _, err := buildDebitNoteRideData(billing_models.DebitNote{}, testTenant(), "Dir", "1"); err == nil {
		t.Error("debit note without access key: expected error")
	}
}
