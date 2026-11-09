// Package cash_handlers serves the cash desk: cash sessions (open, blind
// close, history), payments against invoices (mixed methods, void, refund),
// manual cash movements and the closing report. Every write is audited
// through the models' IsAuditable (tenantdb.For carries the user).
package cash_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"
	cash_services "pengi-med-saas/features/cash/services"
	subscription_middleware "pengi-med-saas/features/companies/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// staleAfter is how long a session may stay open before the desk warns (no
// auto-close).
const staleAfter = 24 * time.Hour

type CashHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	renderer *pdfrender.Renderer
	// can reports whether the caller holds a permission (plan + role), for
	// checks that depend on the payload. A field so tests can stub it.
	can func(c *gin.Context, permissionID string) bool
}

func NewCashHandler(db *gorm.DB, logger *zap.Logger, renderer *pdfrender.Renderer) *CashHandler {
	return &CashHandler{
		db:       db,
		logger:   logger,
		renderer: renderer,
		can: func(c *gin.Context, permissionID string) bool {
			return subscription_middleware.HasPermission(c, db, permissionID)
		},
	}
}

// forUpdate locks the selected rows until the transaction ends (ignored by
// SQLite, which locks the whole database on write).
var forUpdate = clause.Locking{Strength: "UPDATE"}

// fail is an error response returned from inside a transaction; respond
// turns it back into the response.
func fail(status int, key string, code core_errors.AppError) error {
	return envelope.ErrorResponse(status, key, code)
}

func asResponse(err error) (envelope.Response, bool) {
	var resp envelope.Response
	if errors.As(err, &resp) {
		return resp, true
	}
	return envelope.Response{}, false
}

func (h *CashHandler) internal(msg string, err error) envelope.Response {
	h.logger.Error(msg, zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "cash.error.internal", core_errors.ErrCashInternal)
}

func invalid(key string) envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, key, core_errors.ErrCashInvalidRequest)
}

// currentUser is the authenticated user's ID and name.
func currentUser(c *gin.Context) (uint, string) {
	raw, _ := c.Get("user_id")
	var id uint
	switch v := raw.(type) {
	case int64:
		id = uint(v)
	case uint:
		id = v
	case int:
		id = uint(v)
	}
	return id, c.GetString("username")
}

func paramID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id), err == nil && id > 0
}

// openSession returns the tenant's open session, locked; found is false when
// the desk is closed.
func openSession(tx *gorm.DB) (session cash_models.CashSession, found bool, err error) {
	err = tx.Clauses(forUpdate).Where("status = ?", cash_models.SessionOpen).Limit(1).Find(&session).Error
	return session, err == nil && session.ID != 0, err
}

// lockInvoice loads the invoice for update, so payments against it serialize.
func lockInvoice(tx *gorm.DB, id uint) (billing_models.Invoice, error) {
	var invoice billing_models.Invoice
	if err := tx.Clauses(forUpdate).First(&invoice, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return invoice, fail(http.StatusNotFound, "cash.payment.error.invoice_not_found", core_errors.ErrCashInvoiceNotFound)
		}
		return invoice, err
	}
	return invoice, nil
}

// syncInvoicePaymentMethod sets Invoice.PaymentMethod (the SRI forma de pago)
// to the method that collected the most, but only while the invoice is a
// draft: once queued or sent to the SRI its document is never touched.
func syncInvoicePaymentMethod(tx *gorm.DB, invoice billing_models.Invoice) error {
	if invoice.Status != billing_models.InvoiceStatusDraft {
		return nil
	}
	var rows []struct {
		Method string
		Total  float64
	}
	if err := tx.Model(&cash_models.Payment{}).
		Select("method, SUM(amount) AS total").
		Where("invoice_id = ? AND status = ?", invoice.ID, cash_models.PaymentValid).
		Group("method").
		Scan(&rows).Error; err != nil {
		return err
	}
	totals := map[string]int64{}
	for _, r := range rows {
		totals[r.Method] = cash_models.Cents(r.Total)
	}
	best, bestCents := "", int64(0)
	for _, m := range cash_models.Methods { // fixed order breaks ties
		if totals[m] > bestCents {
			best, bestCents = m, totals[m]
		}
	}
	code := cash_models.MethodSRICodes[best]
	if best == "" || code == invoice.PaymentMethod {
		return nil
	}
	return tx.Model(&billing_models.Invoice{}).Where("id = ?", invoice.ID).Update("payment_method", code).Error
}

// PaymentView is a payment with the number of the invoice it was made on.
type PaymentView struct {
	cash_models.Payment
	InvoiceNumber string `json:"invoice_number"`
}

// SessionDetail is a session with its payments, movements and totals.
// ExpectedCash and Difference stay null while the session is open (blind close).
// The open detail still carries per-method totals and every payment, so the
// expected cash can be derived from it: the blind close is enforced by the UI
// (and by close errors never revealing amounts), not by hiding data here.
// Accepted trade-off.
type SessionDetail struct {
	cash_models.CashSession
	Totals    cash_services.SessionTotals `json:"totals"`
	Payments  []PaymentView               `json:"payments"`
	Movements []cash_models.CashMovement  `json:"movements"`
	// Stale: open for more than 24 hours; the desk should be closed.
	Stale bool `json:"stale"`
}

func (h *CashHandler) detail(db *gorm.DB, session cash_models.CashSession) (SessionDetail, error) {
	var payments []cash_models.Payment
	if err := db.Where("cash_session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&payments).Error; err != nil {
		return SessionDetail{}, err
	}
	movements := []cash_models.CashMovement{}
	if err := db.Where("cash_session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&movements).Error; err != nil {
		return SessionDetail{}, err
	}
	views, err := withInvoiceNumbers(db, payments)
	if err != nil {
		return SessionDetail{}, err
	}
	return SessionDetail{
		CashSession: session,
		Totals:      cash_services.Totals(session, payments, movements),
		Payments:    views,
		Movements:   movements,
		Stale:       session.Status == cash_models.SessionOpen && time.Since(session.OpenedAt) > staleAfter,
	}, nil
}

func withInvoiceNumbers(db *gorm.DB, payments []cash_models.Payment) ([]PaymentView, error) {
	views := make([]PaymentView, len(payments))
	if len(payments) == 0 {
		return views, nil
	}
	ids := make([]uint, 0, len(payments))
	for _, p := range payments {
		ids = append(ids, p.InvoiceID)
	}
	var invoices []billing_models.Invoice
	if err := db.Select("id", "establishment_code", "emission_point_code", "sequential").Where("id IN ?", ids).Find(&invoices).Error; err != nil {
		return nil, err
	}
	numbers := map[uint]string{}
	for _, inv := range invoices {
		numbers[inv.ID] = invoiceNumber(inv)
	}
	for i, p := range payments {
		views[i] = PaymentView{Payment: p, InvoiceNumber: numbers[p.InvoiceID]}
	}
	return views, nil
}

func invoiceNumber(inv billing_models.Invoice) string {
	if inv.EstablishmentCode == "" && inv.EmissionPointCode == "" {
		return inv.Sequential
	}
	return inv.EstablishmentCode + "-" + inv.EmissionPointCode + "-" + inv.Sequential
}
