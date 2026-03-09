package sri

type AdditionalDetail struct {
	Nombre string `json:"@nombre" xml:"@nombre"`
	Value  string `json:"#" xml:"#"`
}

type AdditionalDetails struct {
	AdditionalDetail []AdditionalDetail `json:"detAdicional" xml:"detAdicional"`
}

type Tax struct {
	Code           string `json:"codigo" xml:"codigo"`
	CodePercentage string `json:"codigoPorcentaje" xml:"codigoPorcentaje"`
	Tariff         string `json:"tarifa" xml:"tarifa"`
	TaxableBase    string `json:"baseImponible" xml:"baseImponible"`
	Value          string `json:"valor" xml:"valor"`
}

type Taxes struct {
	Taxes []Tax `json:"impuesto" xml:"impuesto"`
}

type Detail struct {
	MainCode             string             `json:"codigoPrincipal" xml:"codigoPrincipal"`
	SecondaryCode        string             `json:"codigoAuxiliar" xml:"codigoAuxiliar"`
	Description          string             `json:"descripcion" xml:"descripcion"`
	UnitMeasure          *string            `json:"unidadMedida,omitempty" xml:"unidadMedida,omitempty"`
	Quantity             string             `json:"cantidad" xml:"cantidad"`
	UnitPrice            string             `json:"precioUnitario" xml:"precioUnitario"`
	PriceWithoutSubsidy  *string            `json:"precioSinSubsidio,omitempty" xml:"precioSinSubsidio,omitempty"`
	Discount             string             `json:"descuento" xml:"descuento"`
	TotalPriceWithoutTax string             `json:"precioTotalSinImpuesto" xml:"precioTotalSinImpuesto"`
	AdditionalDetails    *AdditionalDetails `json:"detallesAdicionales,omitempty" xml:"detallesAdicionales,omitempty"`
	Taxes                Taxes              `json:"impuestos" xml:"impuestos"`
}

type Details struct {
	Details []Detail `json:"detalle" xml:"detalle"`
}
