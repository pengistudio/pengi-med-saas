package billing_dto

type CreateCreditNoteDTO struct {
	InvoiceID uint                `json:"invoice_id" binding:"required"`
	Reason    string              `json:"reason" binding:"required"`
	Items     []CreateInvoiceItem `json:"items" binding:"required,min=1"`
}

type CreditNoteDTO struct {
	CreditNoteID uint64 `json:"credit_note_id"`
}
