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

	// Información adicional: buyer contact data and, when the buyer is a third
	// party, the patient. Empty when there is none.
	AdditionalInfo []RideAdditionalField

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

// NoteRideHeader is what the RIDE of a nota de crédito and a nota de débito
// share (ficha técnica offline, Anexo 2): issuer, the comprobante's own box,
// the buyer and the invoice it modifies. Embedded, so templates read its
// fields directly ({{.AccessKey}}).
type NoteRideHeader struct {
	// Emisor
	TradeName            string
	CorporateName        string
	TaxID                string
	MainAddress          string
	EstablishmentAddress string
	SpecialContributor   string // Nº resolución contribuyente especial; empty if not one
	AccountingObliged    string // "SI" | "NO"
	WithholdingAgent     string // Nº resolución agente de retención; empty if not one
	RimpeTaxpayer        string // RIMPE legend; empty if not RIMPE
	LogoDataURI          template.URL

	// Comprobante
	Establishment     string
	EmissionPoint     string
	Sequential        string
	AccessKey         string // also the número de autorización
	AuthorizationDate string
	Environment       string // "PRUEBAS" | "PRODUCCIÓN"
	EmissionType      string // "NORMAL"
	BarcodeBase64     string

	// Receptor y comprobante que se modifica
	BuyerName            string
	BuyerIdentification  string
	IssueDate            string
	ModifiedDocType      string // "FACTURA"
	ModifiedDocNumber    string // estab-ptoEmi-secuencial
	ModifiedDocIssueDate string

	AdditionalInfo []RideAdditionalField
}

// RideAdditionalField is a row of Información Adicional.
type RideAdditionalField struct {
	Name  string
	Value string
}

// CreditNoteRideData is the model passed to credit_note_ride_template.html.
type CreditNoteRideData struct {
	NoteRideHeader
	Reason     string // razón de modificación
	Items      []CreditNoteRideItem
	Subtotal   string
	Discount   string
	TaxSummary []InvoiceRideTaxSummary
	Total      string
}

type CreditNoteRideItem struct {
	Code        string
	Quantity    string
	Description string
	UnitPrice   string
	Discount    string
	Total       string
}

// DebitNoteRideData is the model passed to debit_note_ride_template.html.
type DebitNoteRideData struct {
	NoteRideHeader
	Motives    []DebitNoteRideMotive
	Payments   []RidePayment
	Subtotal   string
	TaxSummary []InvoiceRideTaxSummary
	Total      string
}

type DebitNoteRideMotive struct {
	Reason string
	Value  string
}

// RidePayment is a row of the "Forma de pago | Valor" table.
type RidePayment struct {
	Method string
	Value  string
}
