package cash_dto

// OpenSessionRequest opens the cash desk with the cash already in the drawer.
type OpenSessionRequest struct {
	OpeningFloat *float64 `json:"opening_float"`
}

// CloseSessionRequest is a blind close: the cashier sends what was counted
// without seeing the expected amount. Note is required when they differ.
type CloseSessionRequest struct {
	CountedCash *float64 `json:"counted_cash" binding:"required"`
	Note        string   `json:"note"`
}

// PaymentLine is one method of a (possibly mixed) payment.
type PaymentLine struct {
	Method    string  `json:"method" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
	Reference string  `json:"reference"`
}

// CreatePaymentRequest charges an invoice; all lines are saved or none.
type CreatePaymentRequest struct {
	InvoiceID uint          `json:"invoice_id" binding:"required"`
	Lines     []PaymentLine `json:"lines" binding:"required,min=1,dive"`
}

// VoidPaymentRequest voids a payment of the open session.
type VoidPaymentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RefundPaymentRequest gives back part or all of a payment from a closed
// session. Method defaults to the original payment's method.
type RefundPaymentRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	Reason string  `json:"reason" binding:"required"`
	Method string  `json:"method"`
}

// CreateMovementRequest records cash put into or taken out of the drawer.
type CreateMovementRequest struct {
	Type     string  `json:"type" binding:"required"`
	Category string  `json:"category" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
	Reason   string  `json:"reason" binding:"required"`
}
