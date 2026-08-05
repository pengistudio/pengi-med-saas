package services

import (
	"strings"

	"pengi-med-saas/core/logger"
	clinical_models "pengi-med-saas/features/clinical/models"

	"go.uber.org/zap"
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
	if patient == nil || patient.Document == "" {
		return FinalConsumerIdentification, FinalConsumerIdentificationType, FinalConsumerSocialReason
	}
	return patient.Document, ResolveBuyerIdentificationType(patient.Document), patient.FirstName + " " + patient.LastName
}

// WarnOnDanglingPatientRef logs when a fiscal document's PatientID still points at a
// patient row but the Patient association didn't resolve (e.g. the patient was
// soft-deleted and the query wasn't Unscoped). Left unnoticed, this is indistinguishable
// from a genuine Consumidor Final sale and would silently mis-declare the buyer identity
// on a legally-binding document — call this right before ResolveBuyerInfo so the anomaly
// is at least observable in logs. It does not block generation; ResolveBuyerInfo still
// falls back to the Consumidor Final triple either way.
func WarnOnDanglingPatientRef(docType string, docID uint, patientID *uint, patient *clinical_models.Patient) {
	if patientID != nil && patient == nil {
		logger.Warn("dangling patient reference on fiscal document, falling back to Consumidor Final",
			zap.String("document_type", docType),
			zap.Uint("document_id", docID),
			zap.Uint("patient_id", *patientID),
		)
	}
}
