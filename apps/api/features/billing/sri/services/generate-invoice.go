package services

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"

	"math/big"
	invoice "pengi-med-saas/features/billing/models"
	invoiceSRI "pengi-med-saas/features/billing/sri"
	tenant "pengi-med-saas/features/tenants/models"
	"strconv"
	"time"
)

type Invoice = invoice.Invoice
type CatalogItem = invoice.CatalogItem

// Modulo11 calcula el dígito verificador
func Modulo11(input string) string {
	factor := 2
	sum := 0

	for i := len(input) - 1; i >= 0; i-- {
		num := int(input[i] - '0')
		sum += num * factor
		factor++
		if factor > 7 {
			factor = 2
		}
	}

	mod := 11 - (sum % 11)
	if mod == 11 {
		return "0"
	}
	if mod == 10 {
		return "1"
	}
	return strconv.Itoa(mod)
}

func generateCodigoAuxiliar() string {
	// Ejemplo: AUX-280525-AB12
	datePart := time.Now().Format("060102") // YYMMDD
	randomPart := generateRandomString(4)
	return fmt.Sprintf("AUX-%s-%s", datePart, randomPart)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// GenerateAccessKey genera la clave de acceso para el SRI
func GenerateAccessKey(invoice Invoice, tenantObj tenant.Tenant, establishmentCode string, emissionCode string) string {
	max := big.NewInt(90000000) // Rango: 0 - 89999999
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}

	result := int(n.Int64() + 10000000)          // Asegura que tenga 8 dígitos
	date := invoice.IssueDate.Format("02012006") // ddmmaaaa
	docType := invoice.DocumentCode              // "01"
	ruc := tenantObj.TaxID                       // RUC de la empresa (13 dígitos)
	environment := "1"                           // "1" = pruebas, "2" = producción
	establishment := establishmentCode           // Ej: "001"
	point := emissionCode                        // Ej: "001"
	sequence := invoice.Sequential               // "000000123"
	randomCode := strconv.Itoa(result)           // Puedes generar uno aleatorio de 8 dígitos
	emissionType := invoice.EmissionType         // "1" = normal

	base := date + docType + ruc + environment + establishment + point + sequence + randomCode + emissionType
	verifier := Modulo11(base)

	return base + verifier // total: 49 caracteres
}

// Helper function to parse date in DD/MM/YYYY format
func ParseDateFromDDMMYYYY(dateString string) (time.Time, error) {
	layout := "02/01/2006"
	date, err := time.Parse(layout, dateString)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format or value. Expected format: DD/MM/YYYY")
	}
	return date, nil
}

// ReorderTaxInfo function to reorder fields as per Go struct
func reorderTaxInfo(invoice Invoice, accessKey string, tenantObj tenant.Tenant, establishmentCode string, emissionCode string) invoiceSRI.TaxInfo {
	return invoiceSRI.TaxInfo{
		Environment:         "1", //Pruebas
		EmissionType:        invoice.EmissionType,
		CorporateName:       tenantObj.CorporateName,
		TradeName:           tenantObj.TradeName,
		TaxID:               tenantObj.TaxID,
		AccessKey:           accessKey,
		DocumentCode:        invoice.DocumentCode,
		Establishment:       establishmentCode,
		EmissionPoint:       emissionCode,
		Sequential:          invoice.Sequential,
		HeadquartersAddress: tenantObj.Address,
	}
}

func findProduct(id uint, services []CatalogItem) *CatalogItem {
	for i := range services {
		if services[i].ID == id {
			return &services[i]
		}
	}
	return nil
}

func reorderDetails(invoice Invoice, services []CatalogItem) invoiceSRI.Details {
	var details []invoiceSRI.Detail
	for _, item := range invoice.Items {

		var tax []invoiceSRI.Tax
		tax = append(tax, invoiceSRI.Tax{

			Code:           item.TaxCode,
			CodePercentage: item.TaxPercentage,
			Tariff:         fmt.Sprintf("%.2f", item.TaxRate),
			TaxableBase:    fmt.Sprintf("%.2f", item.Subtotal),
			Value:          fmt.Sprintf("%.2f", item.Subtotal*item.TaxRate),
		})

		iceTax := invoiceSRI.Tax{
			Code:           item.IceTaxCode,
			CodePercentage: item.IceTaxPercentage,
			Tariff:         fmt.Sprintf("%.2f", item.IceTax),
			TaxableBase:    fmt.Sprintf("%.2f", item.Subtotal),
			Value:          fmt.Sprintf("%.2f", item.Subtotal*item.IceTax),
		}

		if item.IceTaxCode != "3000" {
			tax = append(tax, iceTax)
		}

		mainCode := findProduct(uint(item.ProductID), services)
		if mainCode == nil {
			return invoiceSRI.Details{}
		}

		SecondaryCode := generateCodigoAuxiliar()

		detail := invoiceSRI.Detail{
			MainCode:             mainCode.SKU,
			SecondaryCode:        SecondaryCode,
			Description:          item.Description,
			Quantity:             fmt.Sprintf("%.2f", item.Quantity),
			UnitPrice:            fmt.Sprintf("%.2f", item.UnitPrice),
			Discount:             fmt.Sprintf("%.2f", item.Discount),
			TotalPriceWithoutTax: fmt.Sprintf("%.2f", item.Subtotal),
			Taxes: invoiceSRI.Taxes{
				Taxes: tax,
			},
		}
		details = append(details, detail)
	}
	return invoiceSRI.Details{
		Details: details,
	}
}

func reorderInvoiceInfo(invoice Invoice, tenantObj tenant.Tenant, establishmentAddress string) invoiceSRI.InvoiceInfo {
	payment := []invoiceSRI.Payment{
		{
			PaymentMethod: invoice.PaymentMethod,
			Total:         fmt.Sprintf("%.2f", invoice.Total),
			Term:          invoice.Term,
			TimeUnit:      invoice.TimeUnit,
		},
	}

	var accountingObliged string
	if tenantObj.AccountingObliged {
		accountingObliged = "SI"
	} else {
		accountingObliged = "NO"
	}

	type taxKey struct {
		Code           string
		PercentageCode string
	}

	groupedTaxes := make(map[taxKey]*invoiceSRI.TotalWithTax)
	var totalDiscount float64

	for _, item := range invoice.Items {
		// Group Main Tax (e.g. IVA)
		if item.TaxCode != "" && item.TaxPercentage != "" {
			key := taxKey{Code: item.TaxCode, PercentageCode: item.TaxPercentage}
			if _, exists := groupedTaxes[key]; !exists {
				// Init if not exists
				rateStr := fmt.Sprintf("%.2f", item.TaxRate)
				groupedTaxes[key] = &invoiceSRI.TotalWithTax{
					Code:               item.TaxCode,
					PercentageCode:     item.TaxPercentage,
					TaxableBase:        "0.00",
					Value:              "0.00",
					Rate:               &rateStr,
					AdditionalDiscount: "0.00",
				}
			}

			// Accumulate using floats to prevent precision issues
			var currentBase, currentValue float64
			fmt.Sscanf(groupedTaxes[key].TaxableBase, "%f", &currentBase)
			fmt.Sscanf(groupedTaxes[key].Value, "%f", &currentValue)

			newBase := currentBase + item.Subtotal
			newValue := currentValue + (item.Subtotal * item.TaxRate)

			groupedTaxes[key].TaxableBase = fmt.Sprintf("%.2f", newBase)
			groupedTaxes[key].Value = fmt.Sprintf("%.2f", newValue)
		}

		// Group ICE Tax
		if item.IceTaxCode != "" && item.IceTaxCode != "3000" && item.IceTaxPercentage != "" {
			key := taxKey{Code: item.IceTaxCode, PercentageCode: item.IceTaxPercentage}
			if _, exists := groupedTaxes[key]; !exists {
				// Init if not exists
				rateStr := fmt.Sprintf("%.2f", item.IceTax)
				groupedTaxes[key] = &invoiceSRI.TotalWithTax{
					Code:               item.IceTaxCode,
					PercentageCode:     item.IceTaxPercentage,
					TaxableBase:        "0.00",
					Value:              "0.00",
					Rate:               &rateStr,
					AdditionalDiscount: "0.00",
				}
			}

			var currentBase, currentValue float64
			fmt.Sscanf(groupedTaxes[key].TaxableBase, "%f", &currentBase)
			fmt.Sscanf(groupedTaxes[key].Value, "%f", &currentValue)

			newBase := currentBase + item.Subtotal
			newValue := currentValue + (item.Subtotal * item.IceTax)

			groupedTaxes[key].TaxableBase = fmt.Sprintf("%.2f", newBase)
			groupedTaxes[key].Value = fmt.Sprintf("%.2f", newValue)
		}

		totalDiscount += item.Discount
	}

	var totalWithTaxes []invoiceSRI.TotalWithTax
	for _, taxInfo := range groupedTaxes {
		totalWithTaxes = append(totalWithTaxes, *taxInfo)
	}

	return invoiceSRI.InvoiceInfo{
		IssueDate:               invoice.IssueDate.Format("02/01/2006"),
		BuyerIdentification:     invoice.Patient.Document,
		BuyerAddress:            "S/N", // As Pengi's Patient model does not have an address yet
		BuyerIdentificationType: "05",  // SRI code for Cedula
		BuyerSocialReason:       invoice.Patient.FirstName + " " + invoice.Patient.LastName,
		EstablishmentAddress:    establishmentAddress,
		AccountingObliged:       accountingObliged,
		Currency:                "USD",
		TotalAmount:             fmt.Sprintf("%.2f", invoice.Total),
		TotalWithoutTaxes:       fmt.Sprintf("%.2f", invoice.Subtotal),
		TotalWithTaxes: invoiceSRI.TotalWithTaxes{
			TotalTax: totalWithTaxes,
		},
		TotalDiscount: fmt.Sprintf("%.2f", totalDiscount),
		Payments:      invoiceSRI.Payments{Payment: payment},
	}
}

// Function to generate the access key

// Function to generate the invoiceSRI XML
func GenerateInvoiceXml(invoiceSRI invoiceSRI.InvoiceSRI) (string, error) {
	output, err := xml.MarshalIndent(invoiceSRI, "", "  ")
	if err != nil {
		return "", err
	}
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	return xmlHeader + string(output), nil
}

// Function to generate the invoiceSRI with input data
func GenerateInvoice(invoice Invoice, services []CatalogItem, tenantObj tenant.Tenant, establishmentCode string, emissionCode string, establishmentAddress string) (*invoiceSRI.InvoiceSRI, string, error) {

	accessKey := GenerateAccessKey(invoice, tenantObj, establishmentCode, emissionCode)
	infoTributariaData := reorderTaxInfo(invoice, accessKey, tenantObj, establishmentCode, emissionCode)
	invoiceInfo := reorderInvoiceInfo(invoice, tenantObj, establishmentAddress)
	invoiceDetails := reorderDetails(invoice, services)

	invoiceSRIObj := &invoiceSRI.InvoiceSRI{
		ID:          "comprobante",
		Version:     "1.1.0",
		XmlnsDS:     "http://www.w3.org/2000/09/xmldsig#",
		XmlnsXSI:    "http://www.w3.org/2001/XMLSchema-instance",
		TaxInfo:     infoTributariaData,
		InvoiceInfo: invoiceInfo,
		Details:     invoiceDetails,
	}

	return invoiceSRIObj, accessKey, nil
}
