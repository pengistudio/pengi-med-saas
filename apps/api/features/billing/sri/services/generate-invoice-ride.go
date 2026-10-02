package services

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"pengi-med-saas/core/pdfrender"
	billing_models "pengi-med-saas/features/billing/models"
	billing_templates "pengi-med-saas/features/billing/templates"
	tenant "pengi-med-saas/features/tenants/models"
)

func buildInvoiceRideData(invoice billing_models.Invoice, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) (billing_templates.InvoiceRideData, error) {
	if invoice.AccessKey == nil {
		return billing_templates.InvoiceRideData{}, fmt.Errorf("invoice %d has no access key yet", invoice.ID)
	}

	bcBase64, err := GenerateBarcodeBase64(*invoice.AccessKey)
	if err != nil {
		return billing_templates.InvoiceRideData{}, err
	}

	environment := "PRUEBAS"
	if sriEnv == "2" {
		environment = "PRODUCCIÓN"
	}

	authDate := ""
	if invoice.AuthorizedAt != nil {
		authDate = invoice.AuthorizedAt.Format("02/01/2006 15:04:05")
	}

	var items []billing_templates.InvoiceRideItem
	for _, item := range invoice.Items {
		items = append(items, billing_templates.InvoiceRideItem{
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
	var taxSummary []billing_templates.InvoiceRideTaxSummary
	for key, value := range grouped {
		label := "IVA"
		if key.code == "3" {
			label = "ICE"
		}
		taxSummary = append(taxSummary, billing_templates.InvoiceRideTaxSummary{
			Label: label,
			Base:  fmt.Sprintf("%.2f", invoice.Subtotal),
			Value: fmt.Sprintf("%.2f", value),
		})
	}

	WarnOnDanglingPatientRef("invoice_ride", invoice.ID, invoice.PatientID, invoice.Patient)
	buyerIdentification, _, buyerName := ResolveBuyerInfo(invoice.Patient)

	specialContributor := ""
	if tenantObj.SpecialContributorNumber != nil {
		specialContributor = *tenantObj.SpecialContributorNumber
	}

	var logoDataURI template.URL
	if tenantObj.LogoPath != nil {
		if logoBytes, err := os.ReadFile(*tenantObj.LogoPath); err == nil {
			logoDataURI = template.URL(fmt.Sprintf("data:%s;base64,%s", http.DetectContentType(logoBytes), base64.StdEncoding.EncodeToString(logoBytes)))
		}
	}

	return billing_templates.InvoiceRideData{
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
		BarcodeBase64:        bcBase64,
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
func GenerateInvoiceRide(renderer *pdfrender.Renderer, invoice billing_models.Invoice, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) ([]byte, error) {
	data, err := buildInvoiceRideData(invoice, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return nil, err
	}
	return renderer.Render(tenantObj.ID, billing_templates.InvoiceRide, data)
}
