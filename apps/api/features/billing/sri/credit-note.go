package sri

import "encoding/xml"

// CreditNoteSRI representa el XML "notaCredito" (código SRI 04), ficha técnica v1.1.0.
type CreditNoteSRI struct {
	XMLName        xml.Name        `xml:"notaCredito"`
	ID             string          `xml:"id,attr"`
	Version        string          `xml:"version,attr"`
	XmlnsDS        string          `xml:"xmlns:ds,attr"`
	XmlnsXSI       string          `xml:"xmlns:xsi,attr"`
	TaxInfo        TaxInfo         `json:"infoTributaria" xml:"infoTributaria"`
	CreditNoteInfo CreditNoteInfo  `json:"infoNotaCredito" xml:"infoNotaCredito"`
	Details        Details         `json:"detalles" xml:"detalles"`
	AdditionalInfo *AdditionalInfo `json:"infoAdicional,omitempty" xml:"infoAdicional,omitempty"`
}

type CreditNoteInfo struct {
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
	ModifiedDocCode      string         `json:"codDocModificado" xml:"codDocModificado"` // "01" Factura
	ModifiedDocNumber    string         `json:"numDocModificado" xml:"numDocModificado"` // "001-001-000000123"
	ModifiedDocIssueDate string         `json:"fechaEmisionDocSustento" xml:"fechaEmisionDocSustento"`
	TotalWithoutTaxes    string         `json:"totalSinImpuestos" xml:"totalSinImpuestos"`
	ValueModification    string         `json:"valorModificacion" xml:"valorModificacion"`
	Currency             *string        `json:"moneda,omitempty" xml:"moneda,omitempty"`
	TotalWithTaxes       TotalWithTaxes `json:"totalConImpuestos" xml:"totalConImpuestos"`
	Motivo               string         `json:"motivo" xml:"motivo"`
}
