// Package cash_models holds the cash desk (caja) records: the cash session a
// clinic opens and closes each shift, the payments received against invoices
// during it and the manual cash movements (expenses, withdrawals, deposits,
// adjustments). Nothing here is ever deleted: a payment is voided or refunded,
// a closed session is immutable. Money is float64 rounded to cents, like
// billing_models.Invoice. Times are stored in UTC.
package cash_models

import (
	"math"
	"time"

	"gorm.io/gorm"
)

// Session status.
const (
	SessionOpen   = "open"
	SessionClosed = "closed"
)

// Payment kind and status.
const (
	KindPayment = "payment"
	KindRefund  = "refund"

	PaymentValid  = "valid"
	PaymentVoided = "voided"
)

// Movement type and category.
const (
	MovementIn  = "in"
	MovementOut = "out"

	CategoryExpense    = "expense"
	CategoryWithdrawal = "withdrawal"
	CategoryDeposit    = "deposit"
	CategoryAdjustment = "adjustment"
)

// Payment methods.
const (
	MethodCash       = "cash"
	MethodTransfer   = "transfer"
	MethodCreditCard = "credit_card"
	MethodDebitCard  = "debit_card"
	MethodCheck      = "check"
	MethodOther      = "other"
)

// Methods lists the payment methods in display order.
var Methods = []string{MethodCash, MethodTransfer, MethodCreditCard, MethodDebitCard, MethodCheck, MethodOther}

// MethodSRICodes maps each method to its SRI "forma de pago" code (ficha
// técnica, tabla 24): the value Invoice.PaymentMethod carries to the XML.
// Transfers, checks and anything else that goes through a bank are "20"
// (otros con utilización del sistema financiero).
var MethodSRICodes = map[string]string{
	MethodCash:       "01", // sin utilización del sistema financiero
	MethodTransfer:   "20",
	MethodCreditCard: "19",
	MethodDebitCard:  "16",
	MethodCheck:      "20",
	MethodOther:      "20",
}

// IsMethod reports whether m is a known payment method.
func IsMethod(m string) bool {
	_, ok := MethodSRICodes[m]
	return ok
}

// CashSession is one opening-to-closing period of the clinic's cash desk. A
// tenant has at most one open session (partial unique index). Counted,
// Expected and Difference are set on close and stay null while open.
type CashSession struct {
	gorm.Model
	TenantID     uint       `gorm:"not null;index;uniqueIndex:idx_cash_sessions_one_open,where:status = 'open'" json:"tenant_id"`
	Status       string     `gorm:"type:varchar(10);not null;default:'open'" json:"status"`
	OpenedBy     uint       `json:"opened_by"`
	OpenedByName string     `json:"opened_by_name"`
	OpenedAt     time.Time  `gorm:"not null" json:"opened_at"`
	OpeningFloat float64    `json:"opening_float"`
	ClosedBy     *uint      `json:"closed_by"`
	ClosedByName string     `json:"closed_by_name"`
	ClosedAt     *time.Time `json:"closed_at"`
	CountedCash  *float64   `json:"counted_cash"`
	ExpectedCash *float64   `json:"expected_cash"`
	Difference   *float64   `json:"difference"`
	CloseNote    string     `gorm:"type:text" json:"close_note"`
}

func (CashSession) TableName() string { return "cash_sessions" }
func (CashSession) IsAuditable() bool { return true }

// Payment is money received (kind payment, positive Amount) or given back
// (kind refund, negative Amount, RefundOfPaymentID set) against an invoice,
// inside a cash session. The sum of an invoice's valid payments is what has
// been collected for it.
type Payment struct {
	gorm.Model
	TenantID          uint       `gorm:"not null;index" json:"tenant_id"`
	InvoiceID         uint       `gorm:"not null;index" json:"invoice_id"`
	CashSessionID     uint       `gorm:"not null;index" json:"cash_session_id"`
	Method            string     `gorm:"type:varchar(20);not null" json:"method"`
	Amount            float64    `json:"amount"`
	Reference         string     `json:"reference"`
	Kind              string     `gorm:"type:varchar(10);not null;default:'payment'" json:"kind"`
	Status            string     `gorm:"type:varchar(10);not null;default:'valid'" json:"status"`
	Reason            string     `gorm:"type:text" json:"reason"` // why a refund was given
	VoidReason        string     `gorm:"type:text" json:"void_reason"`
	VoidedBy          *uint      `json:"voided_by"`
	VoidedAt          *time.Time `json:"voided_at"`
	RefundOfPaymentID *uint      `gorm:"index" json:"refund_of_payment_id"`
	CreatedBy         uint       `json:"created_by"`
}

func (Payment) TableName() string { return "cash_payments" }
func (Payment) IsAuditable() bool { return true }

// CashMovement is cash put into (in) or taken out of (out) the drawer that is
// not a payment. Amount is always positive; Type gives the sign.
type CashMovement struct {
	gorm.Model
	TenantID      uint    `gorm:"not null;index" json:"tenant_id"`
	CashSessionID uint    `gorm:"not null;index" json:"cash_session_id"`
	Type          string  `gorm:"type:varchar(5);not null" json:"type"`
	Category      string  `gorm:"type:varchar(20);not null" json:"category"`
	Amount        float64 `json:"amount"`
	Reason        string  `gorm:"type:text;not null" json:"reason"`
	CreatedBy     uint    `json:"created_by"`
}

func (CashMovement) TableName() string { return "cash_movements" }
func (CashMovement) IsAuditable() bool { return true }

// Round2 rounds money to cents.
func Round2(v float64) float64 { return math.Round(v*100) / 100 }

// Cents is v in whole cents, for exact comparisons.
func Cents(v float64) int64 { return int64(math.Round(v * 100)) }
