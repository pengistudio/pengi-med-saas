package services

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
)

func TestBuildInvoiceRideData_AndRenderTemplate(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()
	authorizedAt := time.Now()
	accessKey := "2607202601179000000000110010010000000011855268517"

	invoice := billing_models.Invoice{
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Sequential:        "000000001",
		IssueDate:         time.Now(),
		AccessKey:         &accessKey,
		AuthorizedAt:      &authorizedAt,
		Patient:           patient,
		Subtotal:          100,
		Total:             112,
		PaymentMethod:     "01",
		Items: []billing_models.InvoiceItem{
			{
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
	invoice.Patient.ID = 1

	data, err := buildInvoiceRideData(invoice, tenantObj, "Dirección Establecimiento", "1")
	if err != nil {
		t.Fatalf("buildInvoiceRideData failed: %v", err)
	}
	if data.Environment != "PRUEBAS" {
		t.Errorf("expected Environment PRUEBAS, got %s", data.Environment)
	}
	if data.AccessKey != accessKey {
		t.Errorf("expected AccessKey %s, got %s", accessKey, data.AccessKey)
	}
	if data.QRCodeBase64 == "" {
		t.Error("expected non-empty QRCodeBase64")
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.Items))
	}
	if len(data.TaxSummary) != 1 {
		t.Fatalf("expected 1 tax summary entry, got %d", len(data.TaxSummary))
	}

	tmpl, err := template.ParseFiles("../../templates/invoice_ride_template.html")
	if err != nil {
		t.Fatalf("failed to parse RIDE template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to render RIDE template: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, accessKey) {
		t.Error("expected rendered HTML to contain the access key")
	}
	if !strings.Contains(html, "data:image/png;base64,") {
		t.Error("expected rendered HTML to embed the QR code image")
	}
	if !strings.Contains(html, "Consulta") {
		t.Error("expected rendered HTML to contain the invoice item description")
	}
}

func TestBuildInvoiceRideData_WithLogo(t *testing.T) {
	tenantObj := testTenant()
	patient := testPatient()
	accessKey := "2607202601179000000000110010010000000011855268517"

	logoPath := filepath.Join(t.TempDir(), "logo.png")
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(logoPath, pngSignature, 0644); err != nil {
		t.Fatalf("failed to write fixture logo: %v", err)
	}
	tenantObj.LogoPath = &logoPath

	invoice := billing_models.Invoice{
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Sequential:        "000000001",
		IssueDate:         time.Now(),
		AccessKey:         &accessKey,
		Patient:           patient,
		Total:             112,
	}

	data, err := buildInvoiceRideData(invoice, tenantObj, "Dir", "1")
	if err != nil {
		t.Fatalf("buildInvoiceRideData failed: %v", err)
	}
	if data.LogoDataURI == "" {
		t.Fatal("expected non-empty LogoDataURI when tenant has a logo")
	}
	if !strings.HasPrefix(data.LogoDataURI, "data:image/png;base64,") {
		t.Errorf("expected LogoDataURI to be a PNG data URI, got %q", data.LogoDataURI)
	}

	tmpl, err := template.ParseFiles("../../templates/invoice_ride_template.html")
	if err != nil {
		t.Fatalf("failed to parse RIDE template: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to render RIDE template: %v", err)
	}
	if !strings.Contains(buf.String(), `class="logo"`) {
		t.Error("expected rendered HTML to include the logo <img>")
	}
}

func TestBuildInvoiceRideData_RequiresAccessKey(t *testing.T) {
	tenantObj := testTenant()
	invoice := billing_models.Invoice{IssueDate: time.Now()}

	if _, err := buildInvoiceRideData(invoice, tenantObj, "Dir", "1"); err == nil {
		t.Fatal("expected error when invoice has no access key")
	}
}
