package billing_dto

type CreateDebitNoteMotive struct {
	Reason            string  `json:"reason" binding:"required"`
	Value             float64 `json:"value" binding:"required,gt=0"`
	TaxCode           string  `json:"tax_code" binding:"required"`
	TaxPercentageCode string  `json:"tax_percentage_code" binding:"required"`
	TaxRate           float64 `json:"tax_rate"`
}

type CreateDebitNoteDTO struct {
	InvoiceID uint                    `json:"invoice_id" binding:"required"`
	Motives   []CreateDebitNoteMotive `json:"motives" binding:"required,min=1"`
}

type DebitNoteDTO struct {
	DebitNoteID uint64 `json:"debit_note_id"`
}
