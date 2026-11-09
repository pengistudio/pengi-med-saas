// Package cash_services holds the cash logic shared by the cash handlers and
// billing: session totals, the expected cash of a blind close and the
// collection status of invoices.
package cash_services

import (
	cash_models "pengi-med-saas/features/cash/models"

	"gorm.io/gorm"
)

// MethodTotal is the net amount (payments minus refunds) of one method.
type MethodTotal struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
	Count  int     `json:"count"`
}

// SessionTotals are the figures of a session computed from its valid payments
// and its movements.
type SessionTotals struct {
	ByMethod     []MethodTotal `json:"by_method"`
	CashIn       float64       `json:"cash_in"`       // cash payments received
	CashRefunded float64       `json:"cash_refunded"` // cash given back (positive)
	MovementsIn  float64       `json:"movements_in"`
	MovementsOut float64       `json:"movements_out"`
	// expected is what the drawer should hold; only revealed once closed.
	expected float64
}

// Expected is opening float + cash received − cash refunded + in − out.
func (t SessionTotals) Expected() float64 { return t.expected }

// ComputeTotals adds up the session's valid payments and movements. db must be
// bound to the session's tenant.
func ComputeTotals(db *gorm.DB, session cash_models.CashSession) (SessionTotals, error) {
	var payments []cash_models.Payment
	if err := db.Where("cash_session_id = ? AND status = ?", session.ID, cash_models.PaymentValid).Find(&payments).Error; err != nil {
		return SessionTotals{}, err
	}
	var movements []cash_models.CashMovement
	if err := db.Where("cash_session_id = ?", session.ID).Find(&movements).Error; err != nil {
		return SessionTotals{}, err
	}
	return Totals(session, payments, movements), nil
}

// Totals computes SessionTotals from already loaded rows (voided payments are
// skipped).
func Totals(session cash_models.CashSession, payments []cash_models.Payment, movements []cash_models.CashMovement) SessionTotals {
	byMethod := map[string]*MethodTotal{}
	var t SessionTotals
	for _, p := range payments {
		if p.Status != cash_models.PaymentValid {
			continue
		}
		mt := byMethod[p.Method]
		if mt == nil {
			mt = &MethodTotal{Method: p.Method}
			byMethod[p.Method] = mt
		}
		mt.Total += p.Amount
		mt.Count++
		if p.Method == cash_models.MethodCash {
			if p.Amount >= 0 {
				t.CashIn += p.Amount
			} else {
				t.CashRefunded -= p.Amount
			}
		}
	}
	for _, m := range cash_models.Methods {
		if mt := byMethod[m]; mt != nil {
			mt.Total = cash_models.Round2(mt.Total)
			t.ByMethod = append(t.ByMethod, *mt)
		}
	}
	if t.ByMethod == nil {
		t.ByMethod = []MethodTotal{}
	}
	for _, m := range movements {
		if m.Type == cash_models.MovementIn {
			t.MovementsIn += m.Amount
		} else {
			t.MovementsOut += m.Amount
		}
	}
	t.CashIn = cash_models.Round2(t.CashIn)
	t.CashRefunded = cash_models.Round2(t.CashRefunded)
	t.MovementsIn = cash_models.Round2(t.MovementsIn)
	t.MovementsOut = cash_models.Round2(t.MovementsOut)
	t.expected = cash_models.Round2(session.OpeningFloat + t.CashIn - t.CashRefunded + t.MovementsIn - t.MovementsOut)
	return t
}
