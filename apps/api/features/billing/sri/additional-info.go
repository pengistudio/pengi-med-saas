package sri

type AdditionalField struct {
	Name  string `json:"@nombre" xml:"@nombre"`
	Value string `json:"#" xml:"#"`
}

// AdditionalInfo representa el array de AdditionalField en Go
type AdditionalInfo struct {
	AdditionalField []AdditionalField `json:"campoAdicional" xml:"campoAdicional"`
}
