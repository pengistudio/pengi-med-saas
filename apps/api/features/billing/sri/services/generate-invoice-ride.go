package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"pengi-med-saas/core/utils"
	billing_models "pengi-med-saas/features/billing/models"
	tenant "pengi-med-saas/features/tenants/models"
)

type InvoiceRideItem struct {
	Description string
	Quantity    string
	UnitPrice   string
	Discount    string
	Total       string
}

type InvoiceRideTaxSummary struct {
	Label string // e.g. "IVA 12%"
	Base  string
	Value string
}

// InvoiceRideTemplateData is the model passed to invoice_ride_template.html.
type InvoiceRideTemplateData struct {
	// Emisor
	TradeName            string
	CorporateName        string
	TaxID                string
	MainAddress          string
	EstablishmentAddress string
	Establishment        string
	EmissionPoint        string
	SpecialContributor   string
	LogoDataURI          string // e.g. "data:image/png;base64,..." — empty if the tenant has no logo

	// Documento
	Environment       string // "PRUEBAS" | "PRODUCCIÓN"
	AccessKey         string
	AuthorizationDate string
	IssueDate         string
	Sequential        string
	QRCodeBase64      string

	// Receptor
	BuyerName           string
	BuyerIdentification string

	// Detalle
	Items []InvoiceRideItem

	// Totales
	Subtotal      string
	Discount      string
	TaxSummary    []InvoiceRideTaxSummary
	Total         string
	PaymentMethod string
}

func buildInvoiceRideData(invoice billing_models.Invoice, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) (InvoiceRideTemplateData, error) {
	if invoice.AccessKey == nil {
		return InvoiceRideTemplateData{}, fmt.Errorf("invoice %d has no access key yet", invoice.ID)
	}

	qrBase64, err := GenerateQRCodeBase64(*invoice.AccessKey)
	if err != nil {
		return InvoiceRideTemplateData{}, err
	}

	environment := "PRUEBAS"
	if sriEnv == "2" {
		environment = "PRODUCCIÓN"
	}

	authDate := ""
	if invoice.AuthorizedAt != nil {
		authDate = invoice.AuthorizedAt.Format("02/01/2006 15:04:05")
	}

	var items []InvoiceRideItem
	for _, item := range invoice.Items {
		items = append(items, InvoiceRideItem{
			Description: item.Description,
			Quantity:    fmt.Sprintf("%.2f", item.Quantity),
			UnitPrice:   fmt.Sprintf("%.2f", item.UnitPrice),
			Discount:    fmt.Sprintf("%.2f", item.Discount),
			Total:       fmt.Sprintf("%.2f", item.Subtotal),
		})
	}

	type taxKey struct {
		code           string
		percentageCode string
	}
	grouped := make(map[taxKey]float64)
	for _, item := range invoice.Items {
		if item.TaxCode == "" || item.TaxPercentage == "" {
			continue
		}
		key := taxKey{code: item.TaxCode, percentageCode: item.TaxPercentage}
		grouped[key] += item.Subtotal * item.TaxRate
	}
	var taxSummary []InvoiceRideTaxSummary
	for key, value := range grouped {
		label := "IVA"
		if key.code == "3" {
			label = "ICE"
		}
		taxSummary = append(taxSummary, InvoiceRideTaxSummary{
			Label: label,
			Base:  fmt.Sprintf("%.2f", invoice.Subtotal),
			Value: fmt.Sprintf("%.2f", value),
		})
	}

	buyerIdentification, _, buyerName := ResolveBuyerInfo(invoice.Patient)

	specialContributor := ""
	if tenantObj.SpecialContributorNumber != nil {
		specialContributor = *tenantObj.SpecialContributorNumber
	}

	logoDataURI := ""
	if tenantObj.LogoPath != nil {
		if logoBytes, err := os.ReadFile(*tenantObj.LogoPath); err == nil {
			logoDataURI = fmt.Sprintf("data:%s;base64,%s", http.DetectContentType(logoBytes), base64.StdEncoding.EncodeToString(logoBytes))
		}
	}

	return InvoiceRideTemplateData{
		TradeName:            tenantObj.TradeName,
		CorporateName:        tenantObj.CorporateName,
		TaxID:                tenantObj.TaxID,
		MainAddress:          tenantObj.Address,
		EstablishmentAddress: establishmentAddress,
		Establishment:        invoice.EstablishmentCode,
		EmissionPoint:        invoice.EmissionPointCode,
		SpecialContributor:   specialContributor,
		LogoDataURI:          logoDataURI,
		Environment:          environment,
		AccessKey:            *invoice.AccessKey,
		AuthorizationDate:    authDate,
		IssueDate:            invoice.IssueDate.Format("02/01/2006"),
		Sequential:           invoice.Sequential,
		QRCodeBase64:         qrBase64,
		BuyerName:            buyerName,
		BuyerIdentification:  buyerIdentification,
		Items:                items,
		Subtotal:             fmt.Sprintf("%.2f", invoice.Subtotal),
		Discount:             fmt.Sprintf("%.2f", invoice.Discount),
		TaxSummary:           taxSummary,
		Total:                fmt.Sprintf("%.2f", invoice.Total),
		PaymentMethod:        invoice.PaymentMethod,
	}, nil
}

// GenerateInvoiceRide renders the RIDE (Representación Impresa del Documento
// Electrónico) for an already-authorized invoice and returns the PDF bytes.
func GenerateInvoiceRide(invoice billing_models.Invoice, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) ([]byte, error) {
	data, err := buildInvoiceRideData(invoice, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return nil, err
	}

	customPath := filepath.Join("storage", "tenants", fmt.Sprint(tenantObj.ID), "invoice_ride_template.html")
	tmplPath := "features/billing/templates/invoice_ride_template.html"
	if _, statErr := os.Stat(customPath); statErr == nil {
		tmplPath = customPath
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error loading RIDE template: %w", err)
	}

	var htmlBuffer bytes.Buffer
	if err := tmpl.Execute(&htmlBuffer, data); err != nil {
		return nil, fmt.Errorf("error rendering RIDE template: %w", err)
	}

	gotenbergURL := os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		gotenbergURL = "http://gotenberg:3000"
	}

	client := utils.NewGotenbergClient(gotenbergURL)
	return client.GeneratePDFFromHTMLWithOptions(htmlBuffer.String(), utils.A4Portrait)
}
