package services

import (
	billing_models "pengi-med-saas/features/billing/models"
	invoiceSRI "pengi-med-saas/features/billing/sri"
	billing_templates "pengi-med-saas/features/billing/templates"
)

// SRI limits of infoAdicional (ficha técnica offline v2.26): up to 15
// campoAdicional, each value up to 300 characters.
const (
	maxAdditionalFields     = 15
	maxAdditionalFieldValue = 300
)

// PatientFieldName is the campoAdicional naming the treated patient when the
// buyer is a third party.
const PatientFieldName = "Paciente"

// additionalField is a row of infoAdicional / Información adicional.
type additionalField struct{ name, value string }

// buyerFields is the buyer's contact data plus, for a third-party buyer, the
// patient. Email and phone first, as the RIDEs always showed them.
func buyerFields(invoice billing_models.Invoice, buyer Buyer) []additionalField {
	var fields []additionalField
	add := func(name, value string) {
		if value != "" {
			fields = append(fields, additionalField{name, value})
		}
	}
	add("Email", buyer.Email)
	add("Teléfono", buyer.Phone)
	add("Dirección", buyer.Address)
	add(PatientFieldName, PatientName(invoice, buyer))
	return fields
}

// xmlAdditionalInfo is the comprobante's infoAdicional: only the patient of a
// third-party buyer (the buyer's own data is in infoFactura / infoNota). nil
// when there is nothing to declare, since an empty infoAdicional fails the XSD.
func xmlAdditionalInfo(invoice billing_models.Invoice, buyer Buyer) *invoiceSRI.AdditionalInfo {
	name := PatientName(invoice, buyer)
	if name == "" {
		return nil
	}
	return &invoiceSRI.AdditionalInfo{AdditionalField: []invoiceSRI.AdditionalField{
		{Name: PatientFieldName, Value: truncateRunes(name, maxAdditionalFieldValue)},
	}}
}

// rideAdditionalInfo is the RIDE's Información adicional; nil when empty.
func rideAdditionalInfo(invoice billing_models.Invoice, buyer Buyer) []billing_templates.RideAdditionalField {
	var rows []billing_templates.RideAdditionalField
	for _, f := range buyerFields(invoice, buyer) {
		if len(rows) == maxAdditionalFields {
			break
		}
		rows = append(rows, billing_templates.RideAdditionalField{Name: f.name, Value: truncateRunes(f.value, maxAdditionalFieldValue)})
	}
	return rows
}

// buyerAddress is direccionComprador: "S/N" when the buyer has none on file.
func buyerAddress(buyer Buyer) string {
	if buyer.Address == "" {
		return "S/N"
	}
	return truncateRunes(buyer.Address, maxAdditionalFieldValue)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
