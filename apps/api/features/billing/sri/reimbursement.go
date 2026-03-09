package sri

type TaxDetail struct {
	Code              string `json:"codigo" xml:"codigo"`
	PercentageCode    string `json:"codigoPorcentaje" xml:"codigoPorcentaje"`
	Rate              string `json:"tarifa" xml:"tarifa"`
	ReimbursementBase string `json:"baseImponibleReembolso" xml:"baseImponibleReembolso"`
	ReimbursementTax  string `json:"impuestoReembolso" xml:"impuestoReembolso"`
}

type TaxDetails struct {
	TaxDetail []TaxDetail `json:"detalleImpuesto" xml:"detalleImpuesto"`
}

type ReimbursementCompensation struct {
	Code  string `json:"codigo" xml:"codigo"`
	Rate  string `json:"tarifa" xml:"tarifa"`
	Value string `json:"valor" xml:"valor"`
}

type ReimbursementCompensations struct {
	ReimbursementCompensation []ReimbursementCompensation `json:"compensacionesReembolso" xml:"compensacionesReembolso"`
}

type ReimbursementDetail struct {
	ProviderIDType             string                     `json:"tipoIdentificacionProveedorReembolso" xml:"tipoIdentificacionProveedorReembolso"`
	ProviderID                 string                     `json:"identificacionProveedorReembolso" xml:"identificacionProveedorReembolso"`
	ProviderCountryCode        string                     `json:"codPaisPagoProveedorReembolso" xml:"codPaisPagoProveedorReembolso"`
	ProviderType               string                     `json:"tipoProveedorReembolso" xml:"tipoProveedorReembolso"`
	DocCode                    string                     `json:"codDocReembolso" xml:"codDocReembolso"`
	DocEstablishment           string                     `json:"estabDocReembolso" xml:"estabDocReembolso"`
	DocEmissionPoint           string                     `json:"ptoEmiDocReembolso" xml:"ptoEmiDocReembolso"`
	DocSequential              string                     `json:"secuencialDocReembolso" xml:"secuencialDocReembolso"`
	DocIssueDate               string                     `json:"fechaEmisionDocReembolso" xml:"fechaEmisionDocReembolso"`
	DocAuthorizationNumber     string                     `json:"numeroautorizacionDocReemb" xml:"numeroautorizacionDocReemb"`
	TaxDetails                 TaxDetails                 `json:"detalleImpuestos" xml:"detalleImpuestos"`
	ReimbursementCompensations ReimbursementCompensations `json:"compensacionesReembolso" xml:"compensacionesReembolso"`
}

type Reimbursements struct {
	ReimbursementDetail []ReimbursementDetail `json:"reembolsoDetalle" xml:"reembolsoDetalle"`
}
