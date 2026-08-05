package services

import (
	"encoding/xml"
	"fmt"

	billing_models "pengi-med-saas/features/billing/models"
	invoiceSRI "pengi-med-saas/features/billing/sri"
	tenant "pengi-med-saas/features/tenants/models"
)

func reorderDebitNoteTaxInfo(debitNote billing_models.DebitNote, accessKey string, tenantObj tenant.Tenant, establishmentCode string, emissionCode string, sriEnv string) invoiceSRI.TaxInfo {
	return invoiceSRI.TaxInfo{
		Environment:           sriEnv,
		EmissionType:          debitNote.EmissionType,
		CorporateName:         tenantObj.CorporateName,
		TradeName:             tenantObj.TradeName,
		TaxID:                 tenantObj.TaxID,
		AccessKey:             accessKey,
		DocumentCode:          debitNote.DocumentCode,
		Establishment:         establishmentCode,
		EmissionPoint:         emissionCode,
		Sequential:            debitNote.Sequential,
		HeadquartersAddress:   tenantObj.Address,
		MicroenterpriseRegime: tenantObj.MicroenterpriseRegime,
		WithholdingAgent:      tenantObj.WithholdingAgent,
		RimpeTaxpayer:         tenantObj.RimpeTaxpayer,
	}
}

func reorderDebitMotives(debitNote billing_models.DebitNote) invoiceSRI.DebitMotives {
	var motives []invoiceSRI.DebitMotive
	for _, motive := range debitNote.Motives {
		motives = append(motives, invoiceSRI.DebitMotive{
			Reason: motive.Reason,
			Value:  fmt.Sprintf("%.2f", motive.Value),
		})
	}
	return invoiceSRI.DebitMotives{Motive: motives}
}

func reorderDebitNoteInfo(debitNote billing_models.DebitNote, tenantObj tenant.Tenant, establishmentAddress string) invoiceSRI.DebitNoteInfo {
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

	groupedTaxes := make(map[taxKey]*invoiceSRI.Tax)

	for _, motive := range debitNote.Motives {
		if motive.TaxCode == "" || motive.TaxPercentageCode == "" {
			continue
		}
		key := taxKey{Code: motive.TaxCode, PercentageCode: motive.TaxPercentageCode}
		if _, exists := groupedTaxes[key]; !exists {
			groupedTaxes[key] = &invoiceSRI.Tax{
				Code:           motive.TaxCode,
				CodePercentage: motive.TaxPercentageCode,
				Tariff:         fmt.Sprintf("%.2f", motive.TaxRate),
				TaxableBase:    "0.00",
				Value:          "0.00",
			}
		}

		var currentBase, currentValue float64
		fmt.Sscanf(groupedTaxes[key].TaxableBase, "%f", &currentBase)
		fmt.Sscanf(groupedTaxes[key].Value, "%f", &currentValue)

		groupedTaxes[key].TaxableBase = fmt.Sprintf("%.2f", currentBase+motive.Value)
		groupedTaxes[key].Value = fmt.Sprintf("%.2f", currentValue+(motive.Value*motive.TaxRate))
	}

	var taxes []invoiceSRI.Tax
	for _, tax := range groupedTaxes {
		taxes = append(taxes, *tax)
	}

	buyerIdentification, buyerIdentificationType, buyerSocialReason := ResolveBuyerInfo(debitNote.Invoice.Patient)

	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", debitNote.Invoice.EstablishmentCode, debitNote.Invoice.EmissionPointCode, debitNote.Invoice.Sequential)
	currency := "DOLAR"

	return invoiceSRI.DebitNoteInfo{
		IssueDate:               debitNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: buyerIdentificationType,
		BuyerSocialReason:       buyerSocialReason,
		BuyerIdentification:     buyerIdentification,
		ModifiedDocCode:         "01", // Factura
		ModifiedDocNumber:       modifiedDocNumber,
		ModifiedDocIssueDate:    debitNote.Invoice.IssueDate.Format("02/01/2006"),
		TotalWithoutTaxes:       fmt.Sprintf("%.2f", debitNote.Subtotal),
		Currency:                &currency,
		Taxes:                   invoiceSRI.DebitTaxes{Taxes: taxes},
		ValueTotal:              fmt.Sprintf("%.2f", debitNote.Total),
	}
}

// GenerateDebitNoteXml marshals a DebitNoteSRI into the XML document sent to the signer.
func GenerateDebitNoteXml(debitNoteSRI invoiceSRI.DebitNoteSRI) (string, error) {
	output, err := xml.MarshalIndent(debitNoteSRI, "", "  ")
	if err != nil {
		return "", err
	}
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	return xmlHeader + string(output), nil
}

// GenerateDebitNote builds the DebitNoteSRI struct and access key for a given DebitNote.
func GenerateDebitNote(debitNote billing_models.DebitNote, tenantObj tenant.Tenant, establishmentCode string, emissionCode string, establishmentAddress string, sriEnv string) (*invoiceSRI.DebitNoteSRI, string, error) {
	accessKey, err := GenerateAccessKey(debitNote.IssueDate, debitNote.DocumentCode, tenantObj.TaxID, establishmentCode, emissionCode, debitNote.Sequential, debitNote.EmissionType, sriEnv)
	if err != nil {
		return nil, "", err
	}

	taxInfo := reorderDebitNoteTaxInfo(debitNote, accessKey, tenantObj, establishmentCode, emissionCode, sriEnv)
	debitNoteInfo := reorderDebitNoteInfo(debitNote, tenantObj, establishmentAddress)
	motives := reorderDebitMotives(debitNote)

	debitNoteSRIObj := &invoiceSRI.DebitNoteSRI{
		ID:            "comprobante",
		Version:       "1.0.0",
		XmlnsDS:       "http://www.w3.org/2000/09/xmldsig#",
		XmlnsXSI:      "http://www.w3.org/2001/XMLSchema-instance",
		TaxInfo:       taxInfo,
		DebitNoteInfo: debitNoteInfo,
		Motives:       motives,
	}

	return debitNoteSRIObj, accessKey, nil
}
