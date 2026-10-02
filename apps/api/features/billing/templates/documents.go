package billing_templates

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/utils"
)

// sampleAccessKey is a well-formed 49-digit clave de acceso.
const sampleAccessKey = "1509202601179234567800110010010000012341234567813"

// InvoiceRide is the RIDE of an invoice (Representación Impresa del Documento
// Electrónico). Its file name is also the name tenant templates are stored
// under: never rename it.
var InvoiceRide = pdfrender.Document{
	ID:       "invoice_ride",
	File:     "invoice_ride_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "billing",
	Sample:   sampleInvoiceRide(),
	Variants: []any{InvoiceRideData{AccessKey: sampleAccessKey}},
	// The SRI requires the access key on every RIDE.
	Required: []pdfrender.RequiredField{{Field: ".AccessKey", Value: sampleAccessKey}},
}

func sampleInvoiceRide() InvoiceRideData {
	return InvoiceRideData{
		TradeName:            "Consultorio Médico Andrade",
		CorporateName:        "ANDRADE LÓPEZ MARÍA FERNANDA",
		TaxID:                "1792345678001",
		MainAddress:          "Av. Amazonas N34-120 y Av. República, Quito",
		EstablishmentAddress: "Av. Amazonas N34-120 y Av. República, Quito",
		Establishment:        "001",
		EmissionPoint:        "001",
		Environment:          "PRUEBAS",
		AccessKey:            sampleAccessKey,
		AuthorizationDate:    "15/09/2026 10:32:05",
		IssueDate:            "15/09/2026",
		Sequential:           "000001234",
		BarcodeBase64:        sampleBarcode(sampleAccessKey),
		BuyerName:            "Juan Carlos Pérez Mora",
		BuyerIdentification:  "1712345678",
		Items: []InvoiceRideItem{
			{Description: "Consulta médica general", Quantity: "1.00", UnitPrice: "40.00", Discount: "0.00", Total: "40.00"},
			{Description: "Electrocardiograma", Quantity: "1.00", UnitPrice: "25.00", Discount: "5.00", Total: "20.00"},
			{Description: "Curación de herida", Quantity: "2.00", UnitPrice: "10.00", Discount: "0.00", Total: "20.00"},
		},
		Subtotal:      "80.00",
		Discount:      "5.00",
		TaxSummary:    []InvoiceRideTaxSummary{{Label: "IVA", Base: "80.00", Value: "0.00"}},
		Total:         "80.00",
		PaymentMethod: "01",
	}
}

// sampleBarcode is the Code128 PNG of the access key, as the real RIDE carries.
func sampleBarcode(accessKey string) string {
	bc, err := code128.Encode(accessKey)
	if err != nil {
		return ""
	}
	scaled, err := barcode.Scale(bc, bc.Bounds().Max.X*4, 300)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
