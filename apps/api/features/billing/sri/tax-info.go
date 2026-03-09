package sri

type TaxInfo struct {
	/*
	  pruebas 1
	  produccion 2
	*/
	Environment   string `json:"ambiente" xml:"ambiente"`
	EmissionType  string `json:"tipoEmision" xml:"tipoEmision"`
	CorporateName string `json:"razonSocial" xml:"razonSocial"`
	TradeName     string `json:"nombreComercial" xml:"nombreComercial"`
	TaxID         string `json:"ruc" xml:"ruc"`
	AccessKey     string `json:"claveAcceso" xml:"claveAcceso"`
	/*
	  FACTURA 01
	  LIQUIDACIÓN DE COMPRA DE
	  BIENES Y PRESTACIÓN DE
	  SERVICIOS 03
	  NOTA DE CRÉDITO 04
	  NOTA DE DÉBITO 05
	  GUÍA DE REMISIÓN 06
	  COMPROBANTE DE RETENCIÓN 07
	*/
	DocumentCode          string  `json:"codDoc" xml:"codDoc"`
	Establishment         string  `json:"estab" xml:"estab"`
	EmissionPoint         string  `json:"ptoEmi" xml:"ptoEmi"`
	Sequential            string  `json:"secuencial" xml:"secuencial"`
	HeadquartersAddress   string  `json:"dirMatriz" xml:"dirMatriz"`
	MicroenterpriseRegime *string `json:"regimenMicroempresas,omitempty" xml:"regimenMicroempresas,omitempty"`
	WithholdingAgent      *string `json:"agenteRetencion,omitempty" xml:"agenteRetencion,omitempty"`
	RimpeTaxpayer         *string `json:"contribuyenteRimpe,omitempty" xml:"contribuyenteRimpe,omitempty"`
}
