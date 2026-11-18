package sri

// AdditionalField is a <campoAdicional nombre="...">value</campoAdicional>.
type AdditionalField struct {
	Name  string `json:"@nombre" xml:"nombre,attr"`
	Value string `json:"#" xml:",chardata"`
}

// AdditionalInfo representa el array de AdditionalField en Go. The XSD needs
// at least one campoAdicional: leave the pointer nil when there is none.
type AdditionalInfo struct {
	AdditionalField []AdditionalField `json:"campoAdicional" xml:"campoAdicional"`
}
