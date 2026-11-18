package services

import (
	"strings"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	billing_templates "pengi-med-saas/features/billing/templates"
)

func TestValidateCustomerIdentification(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"1710034065", true},           // valid cédula
		{"1712345678", false},          // bad módulo 10 check digit
		{"1790012345001", true},        // RUC: 13 digits ending in 001
		{"1790012345002", false},       // 13 digits not ending in 001
		{"9999999999999", false},       // Consumidor Final id
		{"AB123456", true},             // pasaporte
		{"A1234567890123456789", true}, // 20 characters
		{"A12345678901234567890", false},
		{"AB 123", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ValidateCustomerIdentification(c.id); got != c.want {
			t.Errorf("ValidateCustomerIdentification(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

func TestCustomerBuyer_IdentificationTypes(t *testing.T) {
	for id, want := range map[string]string{"1790012345001": "04", "1710034065": "05", "AB123456": "06"} {
		if got := CustomerBuyer(billing_models.Customer{Identification: id, Name: "X"}).IdentificationType; got != want {
			t.Errorf("type of %q = %q, want %q", id, got, want)
		}
	}
}

func testCustomer() billing_models.Customer {
	c := billing_models.Customer{
		Identification: "1790012345001",
		Name:           "ASEGURADORA DEL SUR S.A.",
		Address:        "Av. Amazonas N24-03, Quito",
		Email:          "facturas@aseguradora.ec",
		Phone:          "022345678",
	}
	c.ID = 7
	return c
}

// customerInvoice is an invoice bought by testCustomer for testPatient, with
// the snapshot the build step records.
func customerInvoice() billing_models.Invoice {
	inv := testModifiedInvoice()
	customer := testCustomer()
	inv.CustomerID = &customer.ID
	inv.BuyerSnapshot = CustomerBuyer(customer).Snapshot()
	return inv
}

func TestInvoiceBuyer_SnapshotLegacyAndFinalConsumer(t *testing.T) {
	// Snapshot wins over the live patient.
	inv := customerInvoice()
	if b := InvoiceBuyer(inv); b.Identification != "1790012345001" || b.IdentificationType != "04" || b.Name != "ASEGURADORA DEL SUR S.A." || b.Email != "facturas@aseguradora.ec" {
		t.Fatalf("snapshot buyer = %+v", b)
	}

	// Legacy: no snapshot → the patient, live.
	legacy := testModifiedInvoice()
	if b := InvoiceBuyer(legacy); b.Identification != "1710000000" || b.IdentificationType != "05" || b.Name != "Juan Perez" || b.Email != "juan@example.com" {
		t.Fatalf("legacy buyer = %+v", b)
	}

	// Neither → Consumidor Final.
	if b := InvoiceBuyer(billing_models.Invoice{}); b != FinalConsumerBuyer() {
		t.Fatalf("no buyer = %+v, want Consumidor Final", b)
	}
}

func TestGenerateInvoice_CustomerBuyerDeclaresThePatientAsAdditionalInfo(t *testing.T) {
	inv := customerInvoice()
	inv.DocumentCode, inv.EmissionType, inv.Total, inv.Subtotal = "01", "1", 100, 100

	sriInvoice, err := GenerateInvoice(inv, nil, testTenant(), "001", "001", "Matriz", "1", testAccessKey(t, "1"))
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	xmlStr, err := GenerateInvoiceXml(*sriInvoice)
	if err != nil {
		t.Fatalf("GenerateInvoiceXml: %v", err)
	}
	assertContains(t, xmlStr,
		"<tipoIdentificacionComprador>04</tipoIdentificacionComprador>",
		"<razonSocialComprador>ASEGURADORA DEL SUR S.A.</razonSocialComprador>",
		"<identificacionComprador>1790012345001</identificacionComprador>",
		"<direccionComprador>Av. Amazonas N24-03, Quito</direccionComprador>",
		`<infoAdicional>`, `<campoAdicional nombre="Paciente">Juan Perez</campoAdicional>`,
	)
	if strings.Contains(xmlStr, "1710000000") {
		t.Fatalf("the patient's document must not be declared as buyer:\n%s", xmlStr)
	}
	if strings.Index(xmlStr, "<infoAdicional>") < strings.Index(xmlStr, "</detalles>") {
		t.Fatalf("infoAdicional must follow detalles (XSD order)")
	}
}

func TestGenerateInvoice_PatientBuyerHasNoAdditionalInfo(t *testing.T) {
	inv := testModifiedInvoice()
	inv.DocumentCode, inv.EmissionType = "01", "1"
	inv.BuyerSnapshot = ResolveLiveBuyer(nil, inv.Patient).Snapshot()

	sriInvoice, err := GenerateInvoice(inv, nil, testTenant(), "001", "001", "Matriz", "1", testAccessKey(t, "1"))
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	xmlStr, _ := GenerateInvoiceXml(*sriInvoice)
	assertContains(t, xmlStr,
		"<identificacionComprador>1710000000</identificacionComprador>",
		"<tipoIdentificacionComprador>05</tipoIdentificacionComprador>",
		"<direccionComprador>S/N</direccionComprador>",
	)
	if strings.Contains(xmlStr, "infoAdicional") {
		t.Fatalf("no infoAdicional expected (an empty one fails the XSD):\n%s", xmlStr)
	}
}

func TestNotes_DeclareTheBuyerOfTheirInvoice(t *testing.T) {
	inv := customerInvoice()
	issued := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)

	creditNote := billing_models.CreditNote{DocumentCode: billing_models.CreditNoteDocumentCode, EmissionType: "1",
		Sequential: "000000001", IssueDate: issued, Reason: "Devolución", Invoice: inv}
	sriCredit, err := GenerateCreditNote(creditNote, nil, testTenant(), "001", "001", "Matriz", "1", testAccessKey(t, "1"))
	if err != nil {
		t.Fatalf("GenerateCreditNote: %v", err)
	}
	creditXML, _ := GenerateCreditNoteXml(*sriCredit)

	debitNote := billing_models.DebitNote{DocumentCode: "05", EmissionType: "1", Sequential: "000000001", IssueDate: issued, Invoice: inv}
	sriDebit, err := GenerateDebitNote(debitNote, testTenant(), "001", "001", "Matriz", "1", testAccessKey(t, "1"))
	if err != nil {
		t.Fatalf("GenerateDebitNote: %v", err)
	}
	debitXML, _ := GenerateDebitNoteXml(*sriDebit)

	for name, xmlStr := range map[string]string{"credit": creditXML, "debit": debitXML} {
		assertContains(t, xmlStr,
			"<tipoIdentificacionComprador>04</tipoIdentificacionComprador>",
			"<razonSocialComprador>ASEGURADORA DEL SUR S.A.</razonSocialComprador>",
			"<identificacionComprador>1790012345001</identificacionComprador>",
			`<campoAdicional nombre="Paciente">Juan Perez</campoAdicional>`,
		)
		if strings.Contains(xmlStr, "1710000000") {
			t.Fatalf("%s note declares the patient as buyer:\n%s", name, xmlStr)
		}
	}
}

func TestRides_ShowTheInvoiceBuyerAndThePatient(t *testing.T) {
	inv := customerInvoice()
	key := testAccessKey(t, "1")
	authorizedAt := time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC)
	inv.AccessKey, inv.AuthorizedAt = &key, &authorizedAt

	invoiceData, err := buildInvoiceRideData(inv, testTenant(), "Matriz", "1")
	if err != nil {
		t.Fatalf("buildInvoiceRideData: %v", err)
	}
	if invoiceData.BuyerName != "ASEGURADORA DEL SUR S.A." || invoiceData.BuyerIdentification != "1790012345001" {
		t.Fatalf("invoice RIDE buyer = %q %q", invoiceData.BuyerName, invoiceData.BuyerIdentification)
	}
	want := []billing_templates.RideAdditionalField{
		{Name: "Email", Value: "facturas@aseguradora.ec"},
		{Name: "Teléfono", Value: "022345678"},
		{Name: "Dirección", Value: "Av. Amazonas N24-03, Quito"},
		{Name: "Paciente", Value: "Juan Perez"},
	}
	if len(invoiceData.AdditionalInfo) != len(want) {
		t.Fatalf("invoice RIDE additional info = %+v, want %+v", invoiceData.AdditionalInfo, want)
	}
	for i := range want {
		if invoiceData.AdditionalInfo[i] != want[i] {
			t.Fatalf("invoice RIDE additional info = %+v, want %+v", invoiceData.AdditionalInfo, want)
		}
	}
	out := renderedHTML(t, billing_templates.InvoiceRide, invoiceData)
	assertContains(t, out, "Información Adicional", "Paciente", "Juan Perez", "ASEGURADORA DEL SUR S.A.", "facturas@aseguradora.ec")

	noteKey := testCreditNoteKey
	creditNote := billing_models.CreditNote{AccessKey: &noteKey, AuthorizedAt: &authorizedAt, IssueDate: authorizedAt,
		EstablishmentCode: "001", EmissionPointCode: "002", Sequential: "000000005", Invoice: inv}
	noteData, err := buildCreditNoteRideData(creditNote, nil, testTenant(), "Matriz", "1")
	if err != nil {
		t.Fatalf("buildCreditNoteRideData: %v", err)
	}
	if noteData.BuyerName != "ASEGURADORA DEL SUR S.A." || noteData.BuyerIdentification != "1790012345001" {
		t.Fatalf("note RIDE buyer = %q %q", noteData.BuyerName, noteData.BuyerIdentification)
	}
	if last := noteData.AdditionalInfo[len(noteData.AdditionalInfo)-1]; last.Name != "Paciente" || last.Value != "Juan Perez" {
		t.Fatalf("note RIDE additional info = %+v", noteData.AdditionalInfo)
	}
}

func TestAdditionalInfo_TruncatesTo300Characters(t *testing.T) {
	inv := customerInvoice()
	inv.Patient.FirstName = strings.Repeat("Á", 400)
	info := xmlAdditionalInfo(inv, InvoiceBuyer(inv))
	if info == nil || len([]rune(info.AdditionalField[0].Value)) != 300 {
		t.Fatalf("campoAdicional value not truncated to 300 characters")
	}
}

func TestValidateCustomerIdentification_CedulaRejectsBadProvinceAndThirdDigit(t *testing.T) {
	// 99xxxxxxxx has a correct módulo-10 digit but province 99; 1760034065-style
	// numbers with third digit >= 6 are not natural-person cédulas.
	for _, id := range []string{"9910034065", "1760034060"} {
		if ValidateCustomerIdentification(id) {
			t.Errorf("%s should be rejected", id)
		}
	}
	if !ValidateCustomerIdentification("1710034065") {
		t.Error("valid cédula rejected")
	}
}
