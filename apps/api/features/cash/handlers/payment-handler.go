package cash_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	cash_dto "pengi-med-saas/features/cash/dto"
	cash_models "pengi-med-saas/features/cash/models"
	cash_services "pengi-med-saas/features/cash/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InvoicePayments is what an invoice was charged: its payments (voided and
// refunds included) and the derived collection status.
type InvoicePayments struct {
	InvoiceID     uint          `json:"invoice_id"`
	Total         float64       `json:"total"`
	PaidAmount    float64       `json:"paid_amount"`
	Balance       float64       `json:"balance"`
	PaymentStatus string        `json:"payment_status"`
	Payments      []PaymentView `json:"payments"`
}

func (h *CashHandler) invoicePayments(db *gorm.DB, invoice billing_models.Invoice) (InvoicePayments, error) {
	var payments []cash_models.Payment
	if err := db.Where("invoice_id = ?", invoice.ID).Order("created_at ASC, id ASC").Find(&payments).Error; err != nil {
		return InvoicePayments{}, err
	}
	views, err := withInvoiceNumbers(db, payments)
	if err != nil {
		return InvoicePayments{}, err
	}
	paid := 0.0
	for _, p := range payments {
		if p.Status == cash_models.PaymentValid {
			paid += p.Amount
		}
	}
	paid = cash_models.Round2(paid)
	return InvoicePayments{
		InvoiceID:     invoice.ID,
		Total:         cash_models.Round2(invoice.Total),
		PaidAmount:    paid,
		Balance:       cash_models.Round2(invoice.Total - paid),
		PaymentStatus: cash_services.PaymentStatus(invoice.Total, paid),
		Payments:      views,
	}, nil
}

// CreatePayment charges an invoice in the open session, with one line per
// method (mixed payment). All lines are saved or none, and the invoice's
// valid payments may not exceed its total. A draft invoice gets its SRI forma
// de pago from the payments; the SRI document itself is never touched.
func (h *CashHandler) CreatePayment(c *gin.Context) envelope.Response {
	var req cash_dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalid("cash.payment.error.invalid_payload")
	}
	var newCents int64
	for i := range req.Lines {
		line := &req.Lines[i]
		line.Method = strings.TrimSpace(line.Method)
		line.Reference = strings.TrimSpace(line.Reference)
		if !cash_models.IsMethod(line.Method) {
			return invalid("cash.payment.error.invalid_method")
		}
		if cash_models.Cents(line.Amount) <= 0 {
			return invalid("cash.payment.error.invalid_amount")
		}
		newCents += cash_models.Cents(line.Amount)
	}

	db := tenantdb.For(c, h.db)
	userID, _ := currentUser(c)
	var invoice billing_models.Invoice
	err := db.Transaction(func(tx *gorm.DB) error {
		session, found, err := openSession(tx)
		if err != nil {
			return err
		}
		if !found {
			return fail(http.StatusConflict, "cash.session.error.no_open", core_errors.ErrCashNoOpenSession)
		}
		if invoice, err = lockInvoice(tx, req.InvoiceID); err != nil {
			return err
		}
		paid, err := cash_services.PaidAmounts(tx, []uint{invoice.ID})
		if err != nil {
			return err
		}
		if cash_models.Cents(paid[invoice.ID])+newCents > cash_models.Cents(invoice.Total) {
			return fail(http.StatusUnprocessableEntity, "cash.payment.error.overpayment", core_errors.ErrCashOverpayment)
		}
		for _, line := range req.Lines {
			payment := cash_models.Payment{
				TenantID:      invoice.TenantID,
				InvoiceID:     invoice.ID,
				CashSessionID: session.ID,
				Method:        line.Method,
				Amount:        cash_models.Round2(line.Amount),
				Reference:     line.Reference,
				Kind:          cash_models.KindPayment,
				Status:        cash_models.PaymentValid,
				CreatedBy:     userID,
			}
			if err := tx.Create(&payment).Error; err != nil {
				return err
			}
		}
		return syncInvoicePaymentMethod(tx, invoice)
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		return h.internal("failed to create payment", err)
	}
	result, err := h.invoicePayments(db, invoice)
	if err != nil {
		return h.internal("failed to fetch invoice payments", err)
	}
	return envelope.SuccessResponse(result, "cash.payment.create.success")
}

// GetPayments lists what an invoice was charged (?invoice_id=).
func (h *CashHandler) GetPayments(c *gin.Context) envelope.Response {
	invoiceID, err := strconv.ParseUint(c.Query("invoice_id"), 10, 64)
	if err != nil || invoiceID == 0 {
		return invalid("cash.payment.error.invoice_required")
	}
	db := tenantdb.For(c, h.db)
	var invoice billing_models.Invoice
	if err := db.First(&invoice, invoiceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return envelope.ErrorResponse(http.StatusNotFound, "cash.payment.error.invoice_not_found", core_errors.ErrCashInvoiceNotFound)
		}
		return h.internal("failed to fetch invoice", err)
	}
	result, err := h.invoicePayments(db, invoice)
	if err != nil {
		return h.internal("failed to fetch invoice payments", err)
	}
	return envelope.SuccessResponse(result, "cash.payments.fetch.success")
}

// lockPayment loads a payment for update.
func lockPayment(tx *gorm.DB, id uint) (cash_models.Payment, error) {
	var payment cash_models.Payment
	if err := tx.Clauses(forUpdate).First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return payment, fail(http.StatusNotFound, "cash.payment.error.not_found", core_errors.ErrCashPaymentNotFound)
		}
		return payment, err
	}
	return payment, nil
}

// VoidPayment voids a payment (or refund) of the open session where it was
// made. After that session closed, the money can only be refunded.
func (h *CashHandler) VoidPayment(c *gin.Context) envelope.Response {
	id, ok := paramID(c)
	if !ok {
		return invalid("cash.payment.error.invalid_id")
	}
	var req cash_dto.VoidPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		return invalid("cash.payment.error.reason_required")
	}

	db := tenantdb.For(c, h.db)
	userID, _ := currentUser(c)
	var payment cash_models.Payment
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		if payment, err = lockPayment(tx, id); err != nil {
			return err
		}
		if payment.Status == cash_models.PaymentVoided {
			return fail(http.StatusConflict, "cash.payment.error.already_voided", core_errors.ErrCashPaymentVoided)
		}
		session, found, err := openSession(tx)
		if err != nil {
			return err
		}
		if !found || session.ID != payment.CashSessionID {
			return fail(http.StatusConflict, "cash.payment.error.void_outside_session", core_errors.ErrCashVoidOutsideSession)
		}
		invoice, err := lockInvoice(tx, payment.InvoiceID)
		if err != nil {
			return err
		}
		// Voiding a refund adds its money back to the invoice: it must not
		// leave the invoice charged above its total.
		if payment.Kind == cash_models.KindRefund {
			paid, err := cash_services.PaidAmounts(tx, []uint{invoice.ID})
			if err != nil {
				return err
			}
			if cash_models.Cents(paid[invoice.ID])-cash_models.Cents(payment.Amount) > cash_models.Cents(invoice.Total) {
				return fail(http.StatusUnprocessableEntity, "cash.payment.error.overpayment", core_errors.ErrCashOverpayment)
			}
		}
		now := time.Now().UTC()
		payment.Status = cash_models.PaymentVoided
		payment.VoidReason = strings.TrimSpace(req.Reason)
		payment.VoidedBy = &userID
		payment.VoidedAt = &now
		if err := tx.Save(&payment).Error; err != nil {
			return err
		}
		return syncInvoicePaymentMethod(tx, invoice)
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		return h.internal("failed to void payment", err)
	}
	return envelope.SuccessResponse(payment, "cash.payment.void.success")
}

// RefundPayment gives back money of a payment whose session is already
// closed: a refund (negative amount) in the open session, linked to the
// original, never more than what is left of it.
func (h *CashHandler) RefundPayment(c *gin.Context) envelope.Response {
	id, ok := paramID(c)
	if !ok {
		return invalid("cash.payment.error.invalid_id")
	}
	var req cash_dto.RefundPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalid("cash.payment.error.invalid_payload")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return invalid("cash.payment.error.reason_required")
	}
	if cash_models.Cents(req.Amount) <= 0 {
		return invalid("cash.payment.error.invalid_amount")
	}
	method := strings.TrimSpace(req.Method)
	if method != "" && !cash_models.IsMethod(method) {
		return invalid("cash.payment.error.invalid_method")
	}

	db := tenantdb.For(c, h.db)
	userID, _ := currentUser(c)
	var refund cash_models.Payment
	err := db.Transaction(func(tx *gorm.DB) error {
		original, err := lockPayment(tx, id)
		if err != nil {
			return err
		}
		if original.Kind != cash_models.KindPayment || original.Status != cash_models.PaymentValid {
			return fail(http.StatusConflict, "cash.payment.error.not_refundable", core_errors.ErrCashRefundNotRefundable)
		}
		session, found, err := openSession(tx)
		if err != nil {
			return err
		}
		if !found {
			return fail(http.StatusConflict, "cash.session.error.no_open", core_errors.ErrCashNoOpenSession)
		}
		if session.ID == original.CashSessionID {
			return fail(http.StatusConflict, "cash.payment.error.refund_use_void", core_errors.ErrCashRefundUseVoid)
		}
		var refunded struct{ Total float64 }
		if err := tx.Model(&cash_models.Payment{}).
			Select("COALESCE(SUM(amount), 0) AS total").
			Where("refund_of_payment_id = ? AND status = ?", original.ID, cash_models.PaymentValid).
			Scan(&refunded).Error; err != nil {
			return err
		}
		// Refund amounts are negative: what is left is the original plus them.
		if cash_models.Cents(req.Amount) > cash_models.Cents(original.Amount)+cash_models.Cents(refunded.Total) {
			return fail(http.StatusUnprocessableEntity, "cash.payment.error.refund_exceeds", core_errors.ErrCashRefundExceedsPaid)
		}
		invoice, err := lockInvoice(tx, original.InvoiceID)
		if err != nil {
			return err
		}
		if method == "" {
			method = original.Method
		}
		originalID := original.ID
		refund = cash_models.Payment{
			TenantID:          original.TenantID,
			InvoiceID:         original.InvoiceID,
			CashSessionID:     session.ID,
			Method:            method,
			Amount:            -cash_models.Round2(req.Amount),
			Kind:              cash_models.KindRefund,
			Status:            cash_models.PaymentValid,
			Reason:            reason,
			RefundOfPaymentID: &originalID,
			CreatedBy:         userID,
		}
		if err := tx.Create(&refund).Error; err != nil {
			return err
		}
		return syncInvoicePaymentMethod(tx, invoice)
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		return h.internal("failed to refund payment", err)
	}
	return envelope.SuccessResponse(refund, "cash.payment.refund.success")
}
