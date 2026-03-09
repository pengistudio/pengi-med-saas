package sri

import "encoding/xml"

type InvoiceSRI struct {
	XMLName                      xml.Name                      `xml:"factura"`
	ID                           string                        `xml:"id,attr"`
	Version                      string                        `xml:"version,attr"`
	XmlnsDS                      string                        `xml:"xmlns:ds,attr"`
	XmlnsXSI                     string                        `xml:"xmlns:xsi,attr"`
	TaxInfo                      TaxInfo                       `json:"infoTributaria" xml:"infoTributaria"`
	InvoiceInfo                  InvoiceInfo                   `json:"infoFactura" xml:"infoFactura"`
	Details                      Details                       `json:"detalles" xml:"detalles"`
	Reimbursements               *Reimbursements               `json:"reembolsos,omitempty" xml:"reembolsos,omitempty"`
	Retentions                   *Retentions                   `json:"retenciones,omitempty" xml:"retenciones,omitempty"`
	RemissionGuideSubstituteInfo *RemissionGuideSubstituteInfo `json:"infoSustitutivaGuiaRemision,omitempty" xml:"infoSustitutivaGuiaRemision,omitempty"`
	OtherThirdPartyValues        *OtherThirdPartyValues        `json:"otrosRubrosTerceros,omitempty" xml:"otrosRubrosTerceros,omitempty"`
	NegotiableType               *NegotiableType               `json:"tipoNegociable,omitempty" xml:"tipoNegociable,omitempty"`
	FiscalMachine                *FiscalMachine                `json:"maquinaFiscal,omitempty" xml:"maquinaFiscal,omitempty"`
	AdditionalInfo               *AdditionalInfo               `json:"infoAdicional,omitempty" xml:"infoAdicional,omitempty"`
}

type InvoiceInput struct {
	TaxInfo                      TaxInfo                       `json:"infoTributaria" xml:"infoTributaria"`
	InvoiceInfo                  InvoiceInfo                   `json:"infoFactura" xml:"infoFactura"`
	Details                      Details                       `json:"detalles" xml:"detalles"`
	Reimbursements               *Reimbursements               `json:"reembolsos,omitempty" xml:"reembolsos,omitempty"`
	Retentions                   *Retentions                   `json:"retenciones,omitempty" xml:"retenciones,omitempty"`
	RemissionGuideSubstituteInfo *RemissionGuideSubstituteInfo `json:"infoSustitutivaGuiaRemision,omitempty" xml:"infoSustitutivaGuiaRemision,omitempty"`
	OtherThirdPartyValues        *OtherThirdPartyValues        `json:"otrosRubrosTerceros,omitempty" xml:"otrosRubrosTerceros,omitempty"`
	NegotiableType               *NegotiableType               `json:"tipoNegociable,omitempty" xml:"tipoNegociable,omitempty"`
	FiscalMachine                *FiscalMachine                `json:"maquinaFiscal,omitempty" xml:"maquinaFiscal,omitempty"`
	AdditionalInfo               *AdditionalInfo               `json:"infoAdicional,omitempty" xml:"additionalInfo,omitempty"`
}

type NegotiableType struct {
	Email string `json:"correo" xml:"correo"`
}

type FiscalMachine struct {
	Brand  string `json:"marca" xml:"brand"`
	Model  string `json:"modelo" xml:"modelo"`
	Series string `json:"serie" xml:"serie"`
}

// Define other types like TaxInfo, InvoiceInfo, Details, Reimbursements, Retentions, RemisionGuideSustitutiveInfo, OtherThirdPartyValues, and AdditionalInfo similarly.
