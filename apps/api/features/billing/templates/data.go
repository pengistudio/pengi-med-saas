package billing_templates

import "html/template"

// InvoiceRideData is the model passed to invoice_ride_template.html.
type InvoiceRideData struct {
	// Emisor
	TradeName            string
	CorporateName        string
	TaxID                string
	MainAddress          string
	EstablishmentAddress string
	Establishment        string
	EmissionPoint        string
	SpecialContributor   string
	LogoDataURI          template.URL // e.g. "data:image/png;base64,..." — empty if the tenant has no logo. template.URL (built from the tenant's own file, not user text) so html/template keeps the data: URI instead of replacing it with #ZgotmplZ

	// Documento
	Environment       string // "PRUEBAS" | "PRODUCCIÓN"
	AccessKey         string
	AuthorizationDate string
	IssueDate         string
	Sequential        string
	BarcodeBase64     string

	// Receptor
	BuyerName           string
	BuyerIdentification string

	// Detalle
	Items []InvoiceRideItem

	// Totales
	Subtotal      string
	Discount      string
	TaxSummary    []InvoiceRideTaxSummary
	Total         string
	PaymentMethod string
}

type InvoiceRideItem struct {
	Description string
	Quantity    string
	UnitPrice   string
	Discount    string
	Total       string
}

type InvoiceRideTaxSummary struct {
	Label string // e.g. "IVA 12%"
	Base  string
	Value string
}
