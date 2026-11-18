package services

import (
	"strings"

	"pengi-med-saas/core/ecid"
	"pengi-med-saas/core/logger"
	billing_models "pengi-med-saas/features/billing/models"
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
	b := patientBuyer(patient)
	return b.Identification, b.IdentificationType, b.Name
}

// Buyer is who a comprobante is issued to (comprador): a third-party customer,
// the patient, or Consumidor Final.
type Buyer struct {
	Identification     string
	IdentificationType string // 04 RUC, 05 cédula, 06 pasaporte, 07 Consumidor Final
	Name               string // razón social / nombres y apellidos
	Address            string
	Email              string
	Phone              string
}

// IsFinalConsumer reports whether b is the Consumidor Final buyer.
func (b Buyer) IsFinalConsumer() bool {
	return b.IdentificationType == FinalConsumerIdentificationType
}

// Snapshot is b as stored on an invoice.
func (b Buyer) Snapshot() billing_models.BuyerSnapshot {
	return billing_models.BuyerSnapshot{
		BuyerIdentification:     b.Identification,
		BuyerIdentificationType: b.IdentificationType,
		BuyerName:               b.Name,
		BuyerAddress:            b.Address,
		BuyerEmail:              b.Email,
		BuyerPhone:              b.Phone,
	}
}

// FinalConsumerBuyer is the SRI-mandated Consumidor Final buyer.
func FinalConsumerBuyer() Buyer {
	return Buyer{
		Identification:     FinalConsumerIdentification,
		IdentificationType: FinalConsumerIdentificationType,
		Name:               FinalConsumerSocialReason,
	}
}

// CustomerBuyer is a third-party customer as buyer.
func CustomerBuyer(c billing_models.Customer) Buyer {
	id := strings.TrimSpace(c.Identification)
	return Buyer{
		Identification:     id,
		IdentificationType: ResolveBuyerIdentificationType(id),
		Name:               strings.TrimSpace(c.Name),
		Address:            strings.TrimSpace(c.Address),
		Email:              strings.TrimSpace(c.Email),
		Phone:              strings.TrimSpace(c.Phone),
	}
}

// patientBuyer is the patient as buyer; Consumidor Final when there is no
// patient or it has no document. Patients have no address on file.
func patientBuyer(patient *clinical_models.Patient) Buyer {
	if patient == nil || patient.Document == "" {
		return FinalConsumerBuyer()
	}
	return Buyer{
		Identification:     patient.Document,
		IdentificationType: ResolveBuyerIdentificationType(patient.Document),
		Name:               patient.FirstName + " " + patient.LastName,
		Email:              strings.TrimSpace(patient.Email),
		Phone:              strings.TrimSpace(patient.Phone),
	}
}

// ResolveLiveBuyer is the buyer an invoice would declare now, ignoring its
// snapshot: customer if given, else the patient, else Consumidor Final. Used
// to write the snapshot (on create and before each signing attempt).
func ResolveLiveBuyer(customer *billing_models.Customer, patient *clinical_models.Patient) Buyer {
	if customer != nil {
		return CustomerBuyer(*customer)
	}
	return patientBuyer(patient)
}

// InvoiceBuyer is the buyer an invoice declares, and so its notes and emails:
// its snapshot when recorded; else (invoices from before snapshots) live from
// its loaded Customer or Patient, else Consumidor Final. Never fails.
func InvoiceBuyer(invoice billing_models.Invoice) Buyer {
	if s := invoice.BuyerSnapshot; !s.IsEmpty() {
		return Buyer{
			Identification:     s.BuyerIdentification,
			IdentificationType: s.BuyerIdentificationType,
			Name:               s.BuyerName,
			Address:            s.BuyerAddress,
			Email:              strings.TrimSpace(s.BuyerEmail),
			Phone:              s.BuyerPhone,
		}
	}
	return ResolveLiveBuyer(invoice.Customer, invoice.Patient)
}

// invoiceBuyer is InvoiceBuyer for the document docType/docID, logging a
// dangling patient reference when the buyer is resolved live from the patient
// (invoices from before snapshots).
func invoiceBuyer(docType string, docID uint, invoice billing_models.Invoice) Buyer {
	if invoice.BuyerSnapshot.IsEmpty() && invoice.Customer == nil {
		WarnOnDanglingPatientRef(docType, docID, invoice.PatientID, invoice.Patient)
	}
	return InvoiceBuyer(invoice)
}

// PatientName is the treated patient to show next to the buyer when the buyer
// is a third party (campo adicional "Paciente"); "" when the buyer is the
// patient or there is none.
func PatientName(invoice billing_models.Invoice, buyer Buyer) string {
	if invoice.CustomerID == nil || invoice.Patient == nil {
		return ""
	}
	if invoice.Patient.Document != "" && invoice.Patient.Document == buyer.Identification {
		return ""
	}
	return strings.TrimSpace(invoice.Patient.FirstName + " " + invoice.Patient.LastName)
}

// ValidateCustomerIdentification checks a customer's (trimmed) identification:
// never the Consumidor Final id; 10 digits must be a valid cédula (módulo 10);
// 13 digits must end in 001 (RUC, its checksum is not verified); anything else
// is a pasaporte/other id of up to 20 characters without spaces.
func ValidateCustomerIdentification(id string) bool {
	if id == "" || id == FinalConsumerIdentification || len(id) > billing_models.CustomerIdentificationMaxLength {
		return false
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return false
	}
	digitsOnly := true
	for _, r := range id {
		if r < '0' || r > '9' {
			digitsOnly = false
			break
		}
	}
	switch {
	case digitsOnly && len(id) == 10:
		return ecid.ValidCedula(id)
	case digitsOnly && len(id) == 13:
		return strings.HasSuffix(id, "001")
	default:
		return true
	}
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
