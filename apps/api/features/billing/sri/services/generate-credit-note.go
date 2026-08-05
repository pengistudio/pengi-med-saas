package services

import (
	"encoding/xml"
	"fmt"

	billing_models "pengi-med-saas/features/billing/models"
	invoiceSRI "pengi-med-saas/features/billing/sri"
	tenant "pengi-med-saas/features/tenants/models"
)

// reorderCreditNoteTaxInfo mirrors reorderTaxInfo but for a Nota de Crédito (codDoc 04).
func reorderCreditNoteTaxInfo(creditNote billing_models.CreditNote, accessKey string, tenantObj tenant.Tenant, establishmentCode string, emissionCode string, sriEnv string) invoiceSRI.TaxInfo {
	return invoiceSRI.TaxInfo{
		Environment:           sriEnv,
		EmissionType:          creditNote.EmissionType,
		CorporateName:         tenantObj.CorporateName,
		TradeName:             tenantObj.TradeName,
		TaxID:                 tenantObj.TaxID,
		AccessKey:             accessKey,
		DocumentCode:          creditNote.DocumentCode,
		Establishment:         establishmentCode,
		EmissionPoint:         emissionCode,
		Sequential:            creditNote.Sequential,
		HeadquartersAddress:   tenantObj.Address,
		MicroenterpriseRegime: tenantObj.MicroenterpriseRegime,
		WithholdingAgent:      tenantObj.WithholdingAgent,
		RimpeTaxpayer:         tenantObj.RimpeTaxpayer,
	}
}

// reorderCreditNoteDetails mirrors reorderDetails but for CreditNoteItem, reusing the
// same generic Detail/Tax structs as factura.
func reorderCreditNoteDetails(items []billing_models.CreditNoteItem, services []CatalogItem) invoiceSRI.Details {
	var details []invoiceSRI.Detail
	for _, item := range items {
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
		if item.IceTaxCode != "" && item.IceTaxCode != "3000" {
			tax = append(tax, iceTax)
		}

		mainCode := findProduct(item.ProductID, services)
		if mainCode == nil {
			return invoiceSRI.Details{}
		}

		detail := invoiceSRI.Detail{
			MainCode:             mainCode.SKU,
			SecondaryCode:        generateCodigoAuxiliar(),
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
	return invoiceSRI.Details{Details: details}
}

func reorderCreditNoteInfo(creditNote billing_models.CreditNote, tenantObj tenant.Tenant, establishmentAddress string) invoiceSRI.CreditNoteInfo {
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
	for _, item := range creditNote.Items {
		if item.TaxCode != "" && item.TaxPercentage != "" {
			key := taxKey{Code: item.TaxCode, PercentageCode: item.TaxPercentage}
			if _, exists := groupedTaxes[key]; !exists {
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

			var currentBase, currentValue float64
			fmt.Sscanf(groupedTaxes[key].TaxableBase, "%f", &currentBase)
			fmt.Sscanf(groupedTaxes[key].Value, "%f", &currentValue)

			groupedTaxes[key].TaxableBase = fmt.Sprintf("%.2f", currentBase+item.Subtotal)
			groupedTaxes[key].Value = fmt.Sprintf("%.2f", currentValue+(item.Subtotal*item.TaxRate))
		}

		if item.IceTaxCode != "" && item.IceTaxCode != "3000" && item.IceTaxPercentage != "" {
			key := taxKey{Code: item.IceTaxCode, PercentageCode: item.IceTaxPercentage}
			if _, exists := groupedTaxes[key]; !exists {
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

			groupedTaxes[key].TaxableBase = fmt.Sprintf("%.2f", currentBase+item.Subtotal)
			groupedTaxes[key].Value = fmt.Sprintf("%.2f", currentValue+(item.Subtotal*item.IceTax))
		}
	}

	var totalWithTaxes []invoiceSRI.TotalWithTax
	for _, taxInfo := range groupedTaxes {
		totalWithTaxes = append(totalWithTaxes, *taxInfo)
	}

	buyerIdentification, buyerIdentificationType, buyerSocialReason := ResolveBuyerInfo(creditNote.Invoice.Patient)

	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", creditNote.Invoice.EstablishmentCode, creditNote.Invoice.EmissionPointCode, creditNote.Invoice.Sequential)

	currency := "DOLAR"
	info := invoiceSRI.CreditNoteInfo{
		IssueDate:               creditNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: buyerIdentificationType,
		BuyerSocialReason:       buyerSocialReason,
		BuyerIdentification:     buyerIdentification,
		ModifiedDocCode:         "01", // Factura
		ModifiedDocNumber:       modifiedDocNumber,
		ModifiedDocIssueDate:    creditNote.Invoice.IssueDate.Format("02/01/2006"),
		TotalWithoutTaxes:       fmt.Sprintf("%.2f", creditNote.Subtotal),
		ValueModification:       fmt.Sprintf("%.2f", creditNote.Total),
		Currency:                &currency,
		TotalWithTaxes: invoiceSRI.TotalWithTaxes{
			TotalTax: totalWithTaxes,
		},
		Motivo: creditNote.Reason,
	}
	return info
}

// GenerateCreditNoteXml marshals a CreditNoteSRI into the XML document sent to the signer.
func GenerateCreditNoteXml(creditNoteSRI invoiceSRI.CreditNoteSRI) (string, error) {
	output, err := xml.MarshalIndent(creditNoteSRI, "", "  ")
	if err != nil {
		return "", err
	}
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	return xmlHeader + string(output), nil
}

// GenerateCreditNote builds the CreditNoteSRI struct and access key for a given CreditNote.
func GenerateCreditNote(creditNote billing_models.CreditNote, services []CatalogItem, tenantObj tenant.Tenant, establishmentCode string, emissionCode string, establishmentAddress string, sriEnv string) (*invoiceSRI.CreditNoteSRI, string, error) {
	accessKey, err := GenerateAccessKey(creditNote.IssueDate, creditNote.DocumentCode, tenantObj.TaxID, establishmentCode, emissionCode, creditNote.Sequential, creditNote.EmissionType, sriEnv)
	if err != nil {
		return nil, "", err
	}

	taxInfo := reorderCreditNoteTaxInfo(creditNote, accessKey, tenantObj, establishmentCode, emissionCode, sriEnv)
	creditNoteInfo := reorderCreditNoteInfo(creditNote, tenantObj, establishmentAddress)
	details := reorderCreditNoteDetails(creditNote.Items, services)

	creditNoteSRIObj := &invoiceSRI.CreditNoteSRI{
		ID:             "comprobante",
		Version:        "1.1.0",
		XmlnsDS:        "http://www.w3.org/2000/09/xmldsig#",
		XmlnsXSI:       "http://www.w3.org/2001/XMLSchema-instance",
		TaxInfo:        taxInfo,
		CreditNoteInfo: creditNoteInfo,
		Details:        details,
	}

	return creditNoteSRIObj, accessKey, nil
}
