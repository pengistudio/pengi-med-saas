package sri

import "encoding/xml"

// DebitNoteSRI representa el XML "notaDebito" (código SRI 05), ficha técnica v1.0.0.
type DebitNoteSRI struct {
	XMLName        xml.Name        `xml:"notaDebito"`
	ID             string          `xml:"id,attr"`
	Version        string          `xml:"version,attr"`
	XmlnsDS        string          `xml:"xmlns:ds,attr"`
	XmlnsXSI       string          `xml:"xmlns:xsi,attr"`
	TaxInfo        TaxInfo         `json:"infoTributaria" xml:"infoTributaria"`
	DebitNoteInfo  DebitNoteInfo   `json:"infoNotaDebito" xml:"infoNotaDebito"`
	Motives        DebitMotives    `json:"motivos" xml:"motivos"`
	AdditionalInfo *AdditionalInfo `json:"infoAdicional,omitempty" xml:"infoAdicional,omitempty"`
}

type DebitNoteInfo struct {
	IssueDate            string  `json:"fechaEmision" xml:"fechaEmision"`
	EstablishmentAddress string  `json:"dirEstablecimiento" xml:"dirEstablecimiento"`
	SpecialContributor   *string `json:"contribuyenteEspecial,omitempty" xml:"contribuyenteEspecial,omitempty"`
	AccountingObliged    string  `json:"obligadoContabilidad" xml:"obligadoContabilidad"`
	/*
	  RUC 04
	  CÉDULA 05
	  PASAPORTE 06
	  VENTA A CONSUMIDOR FINAL* 07
	  IDENTIFICACIÓN DELEXTERIOR* 08
	*/
	BuyerIdentificationType string `json:"tipoIdentificacionComprador" xml:"tipoIdentificacionComprador"`
	BuyerSocialReason       string `json:"razonSocialComprador" xml:"razonSocialComprador"`
	BuyerIdentification     string `json:"identificacionComprador" xml:"identificacionComprador"`
	// Documento que se está modificando (la factura original)
	ModifiedDocCode      string     `json:"codDocModificado" xml:"codDocModificado"` // "01" Factura
	ModifiedDocNumber    string     `json:"numDocModificado" xml:"numDocModificado"` // "001-001-000000123"
	ModifiedDocIssueDate string     `json:"fechaEmisionDocSustento" xml:"fechaEmisionDocSustento"`
	TotalWithoutTaxes    string     `json:"totalSinImpuestos" xml:"totalSinImpuestos"`
	Currency             *string    `json:"moneda,omitempty" xml:"moneda,omitempty"`
	Taxes                DebitTaxes `json:"impuestos" xml:"impuestos"`
	ValueTotal           string     `json:"valorTotal" xml:"valorTotal"`
}

type DebitTaxes struct {
	Taxes []Tax `json:"impuesto" xml:"impuesto"`
}

type DebitMotive struct {
	Reason string `json:"razon" xml:"razon"`
	Value  string `json:"valor" xml:"valor"`
}

type DebitMotives struct {
	Motive []DebitMotive `json:"motivo" xml:"motivo"`
}
