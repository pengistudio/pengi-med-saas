package sri

type Retention struct {
	Code           string `json:"codigo" xml:"codigo"`
	PercentageCode string `json:"codigoPorcentaje" xml:"codigoPorcentaje"`
	Rate           string `json:"tarifa" xml:"tarifa"`
	Value          string `json:"valor" xml:"valor"`
}

type Retentions struct {
	RetentionList []Retention `json:"retencion" xml:"retencion"`
}
