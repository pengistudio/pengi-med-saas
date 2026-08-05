package billing_models

const (
	InvoiceStatusPending         = "pending"
	InvoiceStatusProcessing      = "processing"
	InvoiceStatusSigned          = "signed"
	InvoiceStatusValidated       = "validated"
	InvoiceStatusAuthorized      = "authorized"
	InvoiceStatusFailed          = "failed"
	InvoiceStatusConnectionError = "connection_error" // SRI/sri-xml-signer unreachable — not a rejection of the document
)
