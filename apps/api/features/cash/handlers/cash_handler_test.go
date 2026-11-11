package cash_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	tenantA uint = 1
	tenantB uint = 2
	userID       = int64(7)
)

type fakeConverter struct{ html string }

func (f *fakeConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	f.html = html
	return []byte("%PDF-fake"), nil
}

type fixture struct {
	t    *testing.T
	db   *gorm.DB
	h    *CashHandler
	conv *fakeConverter
	// denied are the permissions h.can refuses.
	denied map[string]bool
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &billing_models.Invoice{},
		&cash_models.CashSession{}, &cash_models.Payment{}, &cash_models.CashMovement{},
		&audit.AuditLog{},
	)
	audit.RegisterCallbacks(db)
	conv := &fakeConverter{}
	f := &fixture{t: t, db: db, conv: conv, denied: map[string]bool{}}
	f.h = NewCashHandler(db, zap.NewNop(), pdfrender.New(tenantfiles.Memory(), conv))
	f.h.can = func(_ *gin.Context, id string) bool { return !f.denied[id] }
	return f
}

// ctx builds a request context for tenantID; body (if any) is sent as JSON.
func ctx(tenantID uint, body any, id uint, query string) *gin.Context {
	c, _ := testutils.NewGinContext(tenantID, userID)
	c.Set("user_id", userID)
	c.Set("username", "caja")
	if body != nil {
		b, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(http.MethodPost, "/?"+query, bytes.NewReader(b))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	}
	if id != 0 {
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	}
	return c
}

func (f *fixture) invoice(tenantID uint, total float64, status string) billing_models.Invoice {
	f.t.Helper()
	inv := billing_models.Invoice{TenantID: tenantID, Total: total, Status: status, PaymentMethod: "20", Sequential: "000000001", EstablishmentCode: "001", EmissionPointCode: "001"}
	if err := f.db.Create(&inv).Error; err != nil {
		f.t.Fatal(err)
	}
	return inv
}

func (f *fixture) open(tenantID uint, float float64) SessionDetail {
	f.t.Helper()
	resp := f.h.OpenSession(ctx(tenantID, map[string]any{"opening_float": float}, 0, ""))
	mustOK(f.t, resp)
	return resp.Data.(SessionDetail)
}

func (f *fixture) close(tenantID, sessionID uint, counted float64, note string) envelope.Response {
	return f.h.CloseSession(ctx(tenantID, map[string]any{"counted_cash": counted, "note": note}, sessionID, ""))
}

func (f *fixture) pay(tenantID, invoiceID uint, lines ...map[string]any) envelope.Response {
	return f.h.CreatePayment(ctx(tenantID, map[string]any{"invoice_id": invoiceID, "lines": lines}, 0, ""))
}

func line(method string, amount float64) map[string]any {
	return map[string]any{"method": method, "amount": amount}
}

func mustOK(t *testing.T, resp envelope.Response) {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%s, %+v), want 200", resp.Code, resp.Message, resp.Data)
	}
}

func mustFail(t *testing.T, resp envelope.Response, status int, code core_errors.AppError) {
	t.Helper()
	appErr, _ := resp.Data.(core_errors.AppError)
	if resp.Code != status || appErr.ErrorCode != code.ErrorCode {
		t.Fatalf("got %d %s (%+v), want %d %s", resp.Code, resp.Message, resp.Data, status, code.ErrorCode)
	}
}

func (f *fixture) payments(tenantID, invoiceID uint) InvoicePayments {
	f.t.Helper()
	resp := f.h.GetPayments(ctx(tenantID, nil, 0, fmt.Sprintf("invoice_id=%d", invoiceID)))
	mustOK(f.t, resp)
	return resp.Data.(InvoicePayments)
}

func TestOpenSession_OnlyOneOpenPerTenant(t *testing.T) {
	f := setup(t)
	s := f.open(tenantA, 25)
	if s.Status != cash_models.SessionOpen || s.OpeningFloat != 25 || s.ExpectedCash != nil {
		t.Fatalf("opened session = %+v", s.CashSession)
	}
	mustFail(t, f.h.OpenSession(ctx(tenantA, map[string]any{"opening_float": 10}, 0, "")), http.StatusConflict, core_errors.ErrCashSessionAlreadyOpen)

	// Another tenant has its own desk.
	f.open(tenantB, 0)

	// Once closed, a new one can be opened.
	mustOK(t, f.close(tenantA, s.ID, 25, ""))
	f.open(tenantA, 0)

	// The database enforces it too (partial unique index).
	dup := cash_models.CashSession{TenantID: tenantA, Status: cash_models.SessionOpen, OpenedAt: time.Now().UTC()}
	if err := f.db.Create(&dup).Error; err == nil {
		t.Fatal("a second open session was inserted: the partial unique index is missing")
	}

	var logs int64
	f.db.Model(&audit.AuditLog{}).Where("entity_type = ? AND tenant_id = ?", "cash_sessions", tenantA).Count(&logs)
	if logs < 3 { // open, close, open
		t.Fatalf("audit rows for cash_sessions = %d, want >= 3", logs)
	}
}

func TestCreatePayment_RejectedWhileClosed(t *testing.T) {
	f := setup(t)
	inv := f.invoice(tenantA, 40, billing_models.InvoiceStatusDraft)
	mustFail(t, f.pay(tenantA, inv.ID, line("cash", 40)), http.StatusConflict, core_errors.ErrCashNoOpenSession)

	s := f.open(tenantA, 0)
	mustOK(t, f.close(tenantA, s.ID, 0, ""))
	mustFail(t, f.pay(tenantA, inv.ID, line("cash", 40)), http.StatusConflict, core_errors.ErrCashNoOpenSession)
	mustFail(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "in", "category": "deposit", "amount": 5, "reason": "x"}, 0, "")), http.StatusConflict, core_errors.ErrCashNoOpenSession)
}

func TestCreatePayment_MixedMethods(t *testing.T) {
	f := setup(t)
	f.open(tenantA, 0)
	inv := f.invoice(tenantA, 100, billing_models.InvoiceStatusDraft)

	resp := f.pay(tenantA, inv.ID, line("credit_card", 40), map[string]any{"method": "cash", "amount": 60, "reference": " caja 1 "})
	mustOK(t, resp)
	got := resp.Data.(InvoicePayments)
	if got.PaidAmount != 100 || got.Balance != 0 || got.PaymentStatus != billing_models.PaymentStatusPaid || len(got.Payments) != 2 {
		t.Fatalf("invoice payments = %+v", got)
	}
	if got.Payments[0].InvoiceNumber != "001-001-000000001" {
		t.Fatalf("invoice number = %q", got.Payments[0].InvoiceNumber)
	}

	// Draft invoice: the SRI forma de pago follows the largest method (cash).
	var reloaded billing_models.Invoice
	f.db.First(&reloaded, inv.ID)
	if reloaded.PaymentMethod != "01" || reloaded.Status != billing_models.InvoiceStatusDraft {
		t.Fatalf("invoice payment_method=%q status=%q, want 01 / draft", reloaded.PaymentMethod, reloaded.Status)
	}

	// Already sent to the SRI: the document is never touched.
	sent := f.invoice(tenantA, 30, billing_models.InvoiceStatusAuthorized)
	mustOK(t, f.pay(tenantA, sent.ID, line("cash", 30)))
	reloaded = billing_models.Invoice{}
	f.db.First(&reloaded, sent.ID)
	if reloaded.PaymentMethod != "20" || reloaded.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("authorized invoice changed: payment_method=%q status=%q", reloaded.PaymentMethod, reloaded.Status)
	}

	mustFail(t, f.pay(tenantA, inv.ID, line("bitcoin", 1)), http.StatusBadRequest, core_errors.ErrCashInvalidRequest)
}

func TestCreatePayment_OverpaymentRejected(t *testing.T) {
	f := setup(t)
	f.open(tenantA, 0)
	inv := f.invoice(tenantA, 50, billing_models.InvoiceStatusDraft)

	mustOK(t, f.pay(tenantA, inv.ID, line("cash", 30)))
	if p := f.payments(tenantA, inv.ID); p.PaymentStatus != billing_models.PaymentStatusPartial || p.Balance != 20 {
		t.Fatalf("after partial payment = %+v", p)
	}
	mustFail(t, f.pay(tenantA, inv.ID, line("cash", 20.01)), http.StatusUnprocessableEntity, core_errors.ErrCashOverpayment)
	// Atomic: one valid line plus one that overflows saves nothing.
	mustFail(t, f.pay(tenantA, inv.ID, line("cash", 10), line("transfer", 10.5)), http.StatusUnprocessableEntity, core_errors.ErrCashOverpayment)
	if p := f.payments(tenantA, inv.ID); len(p.Payments) != 1 || p.PaidAmount != 30 {
		t.Fatalf("rejected payments were saved: %+v", p)
	}
	mustOK(t, f.pay(tenantA, inv.ID, line("cash", 20)))
}

func TestVoidVersusRefundAfterClose(t *testing.T) {
	f := setup(t)
	s1 := f.open(tenantA, 0)
	inv := f.invoice(tenantA, 80, billing_models.InvoiceStatusDraft)
	paid := f.pay(tenantA, inv.ID, line("cash", 50), line("transfer", 30)).Data.(InvoicePayments)
	cashPay, transferPay := paid.Payments[0], paid.Payments[1]

	// Same open session: void, not refund.
	mustFail(t, f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 10, "reason": "x"}, transferPay.ID, "")), http.StatusConflict, core_errors.ErrCashRefundUseVoid)
	mustFail(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": " "}, cashPay.ID, "")), http.StatusBadRequest, core_errors.ErrCashInvalidRequest)
	mustOK(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": "cobro duplicado"}, cashPay.ID, "")))
	mustFail(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": "otra vez"}, cashPay.ID, "")), http.StatusConflict, core_errors.ErrCashPaymentVoided)

	// The voided cash no longer counts: expected is 0.
	closed := f.close(tenantA, s1.ID, 0, "")
	mustOK(t, closed)
	if d := closed.Data.(SessionDetail); *d.ExpectedCash != 0 || *d.Difference != 0 {
		t.Fatalf("expected/difference = %v / %v", *d.ExpectedCash, *d.Difference)
	}

	// After the close: no void, refunds only, never more than what is left.
	mustFail(t, f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 10, "reason": "x"}, transferPay.ID, "")), http.StatusConflict, core_errors.ErrCashNoOpenSession)
	s2 := f.open(tenantA, 0)
	mustFail(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": "tarde"}, transferPay.ID, "")), http.StatusConflict, core_errors.ErrCashVoidOutsideSession)
	mustFail(t, f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 30.01, "reason": "x"}, transferPay.ID, "")), http.StatusUnprocessableEntity, core_errors.ErrCashRefundExceedsPaid)
	mustFail(t, f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 5, "reason": "x"}, cashPay.ID, "")), http.StatusConflict, core_errors.ErrCashRefundNotRefundable)

	resp := f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 20, "reason": "paciente no atendido", "method": "cash"}, transferPay.ID, ""))
	mustOK(t, resp)
	refund := resp.Data.(cash_models.Payment)
	if refund.Kind != cash_models.KindRefund || refund.Amount != -20 || refund.CashSessionID != s2.ID || refund.RefundOfPaymentID == nil || *refund.RefundOfPaymentID != transferPay.ID || refund.Method != "cash" {
		t.Fatalf("refund = %+v", refund)
	}
	mustFail(t, f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 10.01, "reason": "x"}, transferPay.ID, "")), http.StatusUnprocessableEntity, core_errors.ErrCashRefundExceedsPaid)
	if p := f.payments(tenantA, inv.ID); p.PaidAmount != 10 || p.PaymentStatus != billing_models.PaymentStatusPartial || len(p.Payments) != 3 {
		t.Fatalf("after refund = %+v", p)
	}

	// A cash refund leaves the drawer: expected = 0 − 20.
	mustFail(t, f.close(tenantA, s2.ID, 0, ""), http.StatusUnprocessableEntity, core_errors.ErrCashCloseNoteRequired)
	mustOK(t, f.close(tenantA, s2.ID, 0, "devolución en efectivo sin fondo"))

	// Nothing was deleted.
	var all int64
	f.db.Unscoped().Model(&cash_models.Payment{}).Count(&all)
	if all != 3 {
		t.Fatalf("payments rows = %d, want 3", all)
	}
}

func TestCloseSession_Blind(t *testing.T) {
	f := setup(t)
	s := f.open(tenantA, 20)
	inv := f.invoice(tenantA, 80, billing_models.InvoiceStatusDraft)
	mustOK(t, f.pay(tenantA, inv.ID, line("cash", 50), line("credit_card", 30)))
	mustOK(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "out", "category": "expense", "amount": 5, "reason": "café"}, 0, "")))
	mustOK(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "in", "category": "adjustment", "amount": 1.5, "reason": "sobrante"}, 0, "")))

	// While open the expected amount is not revealed.
	current := f.h.GetCurrentSession(ctx(tenantA, nil, 0, ""))
	mustOK(t, current)
	if d := current.Data.(SessionDetail); d.ExpectedCash != nil || d.Difference != nil || len(d.Movements) != 2 {
		t.Fatalf("open session detail = %+v", d)
	}

	// expected = 20 + 50 − 5 + 1.5 = 66.5. A difference without a note fails
	// and the error carries no amounts.
	resp := f.close(tenantA, s.ID, 60, "  ")
	mustFail(t, resp, http.StatusUnprocessableEntity, core_errors.ErrCashCloseNoteRequired)
	if b, _ := json.Marshal(resp); bytes.Contains(b, []byte("66.5")) {
		t.Fatalf("the error reveals the expected cash: %s", b)
	}

	resp = f.close(tenantA, s.ID, 60, "faltante")
	mustOK(t, resp)
	d := resp.Data.(SessionDetail)
	if d.Status != cash_models.SessionClosed || *d.ExpectedCash != 66.5 || *d.CountedCash != 60 || *d.Difference != -6.5 || d.CloseNote != "faltante" || d.ClosedAt == nil {
		t.Fatalf("closed session = %+v", d.CashSession)
	}
	var card float64
	for _, m := range d.Totals.ByMethod {
		if m.Method == "credit_card" {
			card = m.Total
		}
	}
	if card != 30 {
		t.Fatalf("credit card total = %v, want 30 (%+v)", card, d.Totals.ByMethod)
	}

	// A closed session is immutable.
	mustFail(t, f.close(tenantA, s.ID, 66.5, ""), http.StatusConflict, core_errors.ErrCashSessionClosed)

	// An exact count needs no note.
	s2 := f.open(tenantA, 10)
	mustOK(t, f.close(tenantA, s2.ID, 10, ""))

	// The history lists both, newest first.
	list := f.h.GetSessions(ctx(tenantA, nil, 0, ""))
	mustOK(t, list)
	if paged := list.Data.(envelope.PagedData); paged.Total != 2 || paged.Items.([]cash_models.CashSession)[0].ID != s2.ID {
		t.Fatalf("history = %+v", paged)
	}
}

func TestCreateMovement_PermissionsAndValidation(t *testing.T) {
	f := setup(t)
	f.open(tenantA, 0)
	f.denied["VOID_PAYMENT"] = true

	mustFail(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "out", "category": "withdrawal", "amount": 5, "reason": "banco"}, 0, "")), http.StatusForbidden, core_errors.ErrPermissionDenied)
	mustOK(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "in", "category": "deposit", "amount": 5, "reason": "cambio"}, 0, "")))
	mustFail(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "out", "category": "deposit", "amount": 5, "reason": "x"}, 0, "")), http.StatusBadRequest, core_errors.ErrCashInvalidRequest)
	mustFail(t, f.h.CreateMovement(ctx(tenantA, map[string]any{"type": "in", "category": "deposit", "amount": 5, "reason": " "}, 0, "")), http.StatusBadRequest, core_errors.ErrCashInvalidRequest)
}

func TestSessionReport(t *testing.T) {
	f := setup(t)
	tenant := tenant_models.Tenant{Name: "Clínica A", Slug: fmt.Sprintf("cash-%d", time.Now().UnixNano()), DisplayToken: fmt.Sprintf("tok-cash-%d", time.Now().UnixNano())}
	if err := f.db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	s := f.open(tenant.ID, 10)

	c := ctx(tenant.ID, nil, s.ID, "")
	f.h.DownloadSessionReport(c)
	if w := c.Writer; w.Status() != http.StatusConflict {
		t.Fatalf("report of an open session = %d, want 409", w.Status())
	}

	inv := f.invoice(tenant.ID, 15, billing_models.InvoiceStatusDraft)
	mustOK(t, f.pay(tenant.ID, inv.ID, line("cash", 15)))
	mustOK(t, f.close(tenant.ID, s.ID, 25, ""))
	rec := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Set("tenant_id", tenant.ID)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(s.ID)}}
	f.h.DownloadSessionReport(c)
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("report = %d %q", rec.Code, rec.Body.String())
	}
	for _, want := range []string{"Clínica A", fmt.Sprintf("%06d", s.ID), "25.00", "001-001-000000001"} {
		if !bytes.Contains([]byte(f.conv.html), []byte(want)) {
			t.Fatalf("report html lacks %q", want)
		}
	}
}

func TestCash_TenantIsolation(t *testing.T) {
	f := setup(t)
	sA := f.open(tenantA, 0)
	invA := f.invoice(tenantA, 20, billing_models.InvoiceStatusDraft)
	payA := f.pay(tenantA, invA.ID, line("cash", 20)).Data.(InvoicePayments).Payments[0]

	// Tenant B sees no open desk, nor A's session, payment or invoice.
	if cur := f.h.GetCurrentSession(ctx(tenantB, nil, 0, "")); cur.Data != nil {
		t.Fatalf("tenant B sees a current session: %+v", cur.Data)
	}
	mustFail(t, f.h.GetSession(ctx(tenantB, nil, sA.ID, "")), http.StatusNotFound, core_errors.ErrCashSessionNotFound)
	mustFail(t, f.close(tenantB, sA.ID, 0, ""), http.StatusNotFound, core_errors.ErrCashSessionNotFound)
	mustFail(t, f.h.GetPayments(ctx(tenantB, nil, 0, fmt.Sprintf("invoice_id=%d", invA.ID))), http.StatusNotFound, core_errors.ErrCashInvoiceNotFound)

	f.open(tenantB, 0)
	mustFail(t, f.pay(tenantB, invA.ID, line("cash", 1)), http.StatusNotFound, core_errors.ErrCashInvoiceNotFound)
	mustFail(t, f.h.VoidPayment(ctx(tenantB, map[string]any{"reason": "x"}, payA.ID, "")), http.StatusNotFound, core_errors.ErrCashPaymentNotFound)
	mustFail(t, f.h.RefundPayment(ctx(tenantB, map[string]any{"amount": 1, "reason": "x"}, payA.ID, "")), http.StatusNotFound, core_errors.ErrCashPaymentNotFound)

	list := f.h.GetSessions(ctx(tenantB, nil, 0, "")).Data.(envelope.PagedData)
	if list.Total != 1 {
		t.Fatalf("tenant B history = %d sessions, want 1", list.Total)
	}

	// A is untouched.
	if p := f.payments(tenantA, invA.ID); p.PaidAmount != 20 || p.Payments[0].Status != cash_models.PaymentValid {
		t.Fatalf("tenant A payments changed: %+v", p)
	}
}

func TestVoidRefund_CannotOverpayInvoice(t *testing.T) {
	f := setup(t)
	s1 := f.open(tenantA, 0)
	inv := f.invoice(tenantA, 40, billing_models.InvoiceStatusDraft)
	original := f.pay(tenantA, inv.ID, line("cash", 40)).Data.(InvoicePayments).Payments[0]
	mustOK(t, f.close(tenantA, s1.ID, 40, ""))

	f.open(tenantA, 0)
	resp := f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 40, "reason": "devolución"}, original.ID, ""))
	mustOK(t, resp)
	refund := resp.Data.(cash_models.Payment)
	mustOK(t, f.pay(tenantA, inv.ID, line("cash", 40)))

	// Voiding the refund would put the invoice at 80 of 40.
	mustFail(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": "error"}, refund.ID, "")), http.StatusUnprocessableEntity, core_errors.ErrCashOverpayment)
	if p := f.payments(tenantA, inv.ID); p.PaidAmount != 40 || p.PaymentStatus != billing_models.PaymentStatusPaid {
		t.Fatalf("after rejected void = %+v", p)
	}
}

func TestVoidRefund_AllowedWhenWithinTotal(t *testing.T) {
	f := setup(t)
	s1 := f.open(tenantA, 0)
	inv := f.invoice(tenantA, 40, billing_models.InvoiceStatusDraft)
	original := f.pay(tenantA, inv.ID, line("cash", 40)).Data.(InvoicePayments).Payments[0]
	mustOK(t, f.close(tenantA, s1.ID, 40, ""))

	f.open(tenantA, 0)
	refund := f.h.RefundPayment(ctx(tenantA, map[string]any{"amount": 40, "reason": "devolución"}, original.ID, "")).Data.(cash_models.Payment)
	mustOK(t, f.h.VoidPayment(ctx(tenantA, map[string]any{"reason": "error"}, refund.ID, "")))
	if p := f.payments(tenantA, inv.ID); p.PaidAmount != 40 || p.PaymentStatus != billing_models.PaymentStatusPaid {
		t.Fatalf("after voiding the refund = %+v", p)
	}
}
