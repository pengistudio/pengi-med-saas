package sri

type InvoiceInfo struct {
	IssueDate            string  `json:"fechaEmision" xml:"fechaEmision"`
	EstablishmentAddress string  `json:"dirEstablecimiento" xml:"dirEstablecimiento"`
	SpecialContributor   *string `json:"contribuyenteEspecial,omitempty" xml:"contribuyenteEspecial,omitempty"`
	AccountingObliged    string  `json:"obligadoContabilidad" xml:"obligadoContabilidad"`
	ForeignTrade         *string `json:"comercioExterior,omitempty" xml:"comercioExterior,omitempty"`
	IncoTermInvoice      *string `json:"incoTermFactura,omitempty" xml:"incoTermFactura,omitempty"`
	IncoTermPlace        *string `json:"lugarIncoTerm,omitempty" xml:"lugarIncoTerm,omitempty"`
	CountryOfOrigin      *string `json:"paisOrigen,omitempty" xml:"paisOrigen,omitempty"`
	PortOfShipment       *string `json:"puertoEmbarque,omitempty" xml:"puertoEmbarque,omitempty"`
	PortOfDestination    *string `json:"puertoDestino,omitempty" xml:"puertoDestino,omitempty"`
	DestinationCountry   *string `json:"paisDestino,omitempty" xml:"paisDestino,omitempty"`
	AcquisitionCountry   *string `json:"paisAdquisicion,omitempty" xml:"paisAdquisicion,omitempty"`
	/*
	  RUC 04
	  CÉDULA 05
	  PASAPORTE 06
	  VENTA A CONSUMIDOR FINAL* 07
	  IDENTIFICACIÓN DELEXTERIOR* 08
	*/
	BuyerIdentificationType    string         `json:"tipoIdentificacionComprador" xml:"tipoIdentificacionComprador"`
	ShippingGuide              *string        `json:"guiaRemision,omitempty" xml:"guiaRemision,omitempty"`
	BuyerSocialReason          string         `json:"razonSocialComprador" xml:"razonSocialComprador"`
	BuyerIdentification        string         `json:"identificacionComprador" xml:"identificacionComprador"`
	BuyerAddress               string         `json:"direccionComprador" xml:"direccionComprador"`
	TotalWithoutTaxes          string         `json:"totalSinImpuestos" xml:"totalSinImpuestos"`
	TotalSubsidy               *string        `json:"totalSubsidio,omitempty" xml:"totalSubsidio,omitempty"`
	IncoTermTotalWithoutTaxes  *string        `json:"incoTermTotalSinImpuestos,omitempty" xml:"incoTermTotalSinImpuestos,omitempty"`
	TotalDiscount              string         `json:"totalDescuento" xml:"totalDescuento"`
	ReimbursementDocCode       *string        `json:"codDocReembolso,omitempty" xml:"codDocReembolso,omitempty"`
	TotalReimbursementReceipts *string        `json:"totalComprobantesReembolso,omitempty" xml:"totalComprobantesReembolso,omitempty"`
	TotalReimbursementTaxBase  *string        `json:"totalBaseImponibleReembolso,omitempty" xml:"totalBaseImponibleReembolso,omitempty"`
	TotalReimbursementTax      *string        `json:"totalImpuestoReembolso,omitempty" xml:"totalImpuestoReembolso,omitempty"`
	TotalWithTaxes             TotalWithTaxes `json:"totalConImpuestos" xml:"totalConImpuestos"`
	Compensations              *Compensations `json:"compensaciones,omitempty" xml:"compensaciones,omitempty"`
	Tip                        *string        `json:"propina,omitempty" xml:"propina,omitempty"`
	InternationalFreight       *string        `json:"fleteInternacional,omitempty" xml:"fleteInternacional,omitempty"`
	InternationalInsurance     *string        `json:"seguroInternacional,omitempty" xml:"seguroInternacional,omitempty"`
	CustomsExpenses            *string        `json:"gastosAduaneros,omitempty" xml:"gastosAduaneros,omitempty"`
	OtherTransportExpenses     *string        `json:"gastosTransporteOtros,omitempty" xml:"gastosTransporteOtros,omitempty"`
	TotalAmount                string         `json:"importeTotal" xml:"importeTotal"`
	Currency                   string         `json:"moneda" xml:"moneda"`
	LicensePlate               *string        `json:"placa,omitempty" xml:"placa,omitempty"`
	Payments                   Payments       `json:"pagos" xml:"pagos"`
	VatRetentionValue          *string        `json:"valorRetIva,omitempty" xml:"valorRetIva,omitempty"`
	IncomeTaxRetentionValue    *string        `json:"valorRetRenta,omitempty" xml:"valorRetRenta,omitempty"`
}

type TotalWithTax struct {
	/*
	  IVA 2
	  ICE 3
	  IRBPNR 5
	*/
	Code string `json:"codigo" xml:"codigo"`
	/*
	  IVA
	  0% 0
	  12% 2
	  14% 3
	  No Objeto de Impuesto 6
	  Exento de IVA 7
	  IVA diferenciado4 8

	  ICE - Ver tabla 18 de la ficha tecnica de comprobantes electronicos
	*/
	PercentageCode     string  `json:"codigoPorcentaje" xml:"codigoPorcentaje"`
	AdditionalDiscount string  `json:"descuentoAdicional" xml:"descuentoAdicional"`
	TaxableBase        string  `json:"baseImponible" xml:"baseImponible"`
	Rate               *string `json:"tarifa,omitempty" xml:"tarifa,omitempty"`
	Value              string  `json:"valor" xml:"valor"`
	VatRefundValue     *string `json:"valorDevolucionIva,omitempty" xml:"valorDevolucionIva,omitempty"`
}

type TotalWithTaxes struct {
	TotalTax []TotalWithTax `json:"totalImpuesto" xml:"totalImpuesto"`
}

type Compensation struct {
	Code  string `json:"codigo" xml:"codigo"`
	Rate  string `json:"tarifa" xml:"tarifa"`
	Value string `json:"valor" xml:"valor"`
}

type Compensations struct {
	Compensation []Compensation `json:"compensacion" xml:"compensacion"`
}

type Payment struct {
	PaymentMethod string `json:"formaPago" xml:"formaPago"`
	Total         string `json:"total" xml:"total"`
	Term          string `json:"plazo" xml:"plazo"`
	TimeUnit      string `json:"unidadTiempo" xml:"unidadTiempo"`
}

type Payments struct {
	Payment []Payment `json:"pago" xml:"pago"`
}
