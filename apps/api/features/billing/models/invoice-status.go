package billing_models

const (
	InvoiceStatusDraft           = "draft" // column default for invoices: created, never queued
	InvoiceStatusPending         = "pending"
	InvoiceStatusProcessing      = "processing"
	InvoiceStatusSigned          = "signed"
	InvoiceStatusValidated       = "validated" // received by the SRI (recepción), awaiting authorization
	InvoiceStatusAuthorized      = "authorized"
	InvoiceStatusRejected        = "rejected" // NO AUTORIZADO — resent with the same access key once corrected
	InvoiceStatusFailed          = "failed"
	InvoiceStatusConnectionError = "connection_error" // SRI/sri-xml-signer unreachable — not a rejection of the document
)
