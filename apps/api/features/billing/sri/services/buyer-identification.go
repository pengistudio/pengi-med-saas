package services

import (
	"strings"

	clinical_models "pengi-med-saas/features/clinical/models"
)

// SRI-mandated identification for a document (factura/nota) issued without a
// buyer on file — i.e. a "Consumidor Final" sale.
const (
	FinalConsumerIdentification     = "9999999999999"
	FinalConsumerIdentificationType = "07"
	FinalConsumerSocialReason       = "CONSUMIDOR FINAL"
)

// ResolveBuyerIdentificationType returns the SRI buyer identification type code:
//
//	04 RUC (13 digits, ends in 001)
//	05 Cédula (10 digits)
//	06 Pasaporte / otro (anything else non-empty)
//	07 Consumidor Final (no document)
func ResolveBuyerIdentificationType(document string) string {
	if document == "" {
		return FinalConsumerIdentificationType
	}

	digitsOnly := true
	for _, r := range document {
		if r < '0' || r > '9' {
			digitsOnly = false
			break
		}
	}

	switch {
	case digitsOnly && len(document) == 13 && strings.HasSuffix(document, "001"):
		return "04"
	case digitsOnly && len(document) == 10:
		return "05"
	default:
		return "06"
	}
}

// ResolveBuyerInfo returns the SRI buyer identification, identification type and
// social reason for a document. When patient is nil (no patient attached — a
// "Consumidor Final" sale), it returns the SRI-mandated Consumidor Final triple
// instead of an empty/zero buyer.
func ResolveBuyerInfo(patient *clinical_models.Patient) (identification, identificationType, socialReason string) {
	if patient == nil {
		return FinalConsumerIdentification, FinalConsumerIdentificationType, FinalConsumerSocialReason
	}
	return patient.Document, ResolveBuyerIdentificationType(patient.Document), patient.FirstName + " " + patient.LastName
}
