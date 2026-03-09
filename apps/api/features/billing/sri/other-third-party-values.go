package sri

type ThirdPartyValue struct {
	Concept string `json:"concepto" xml:"concepto"`
	Total   string `json:"total" xml:"total"`
}

type OtherThirdPartyValues struct {
	LineOfBusiness []ThirdPartyValue `json:"rubro" xml:"rubro"`
}
