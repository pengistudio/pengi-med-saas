package cash_services

import (
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"

	"gorm.io/gorm"
)

// PaidAmounts returns the net collected amount (valid payments minus refunds)
// of each invoice that has any valid payment. db must be bound to the tenant.
func PaidAmounts(db *gorm.DB, invoiceIDs []uint) (map[uint]float64, error) {
	out := map[uint]float64{}
	if len(invoiceIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		InvoiceID uint
		Paid      float64
	}
	if err := db.Model(&cash_models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Where("invoice_id IN ? AND status = ?", invoiceIDs, cash_models.PaymentValid).
		Group("invoice_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.InvoiceID] = cash_models.Round2(r.Paid)
	}
	return out, nil
}

// PaymentStatus derives unpaid | partial | paid from what was collected.
func PaymentStatus(total, paid float64) string {
	switch {
	case cash_models.Cents(paid) <= 0:
		return billing_models.PaymentStatusUnpaid
	case cash_models.Cents(paid) < cash_models.Cents(total):
		return billing_models.PaymentStatusPartial
	default:
		return billing_models.PaymentStatusPaid
	}
}

// FirstSessionAt is when the tenant opened its first cash session, or nil if
// it never used the cash desk.
func FirstSessionAt(db *gorm.DB) (*time.Time, error) {
	var first cash_models.CashSession
	err := db.Order("opened_at ASC").Limit(1).Find(&first).Error
	if err != nil || first.ID == 0 {
		return nil, err
	}
	return &first.OpenedAt, nil
}

// ApplyPaymentStatus fills PaidAmount and PaymentStatus on each invoice. They
// stay null for invoices created before the tenant's first cash session that
// have no payment (no backfill: those were charged outside the desk), and for
// every invoice of a tenant that never opened one.
func ApplyPaymentStatus(db *gorm.DB, invoices []billing_models.Invoice) error {
	if len(invoices) == 0 {
		return nil
	}
	firstAt, err := FirstSessionAt(db)
	if err != nil || firstAt == nil {
		return err
	}
	ids := make([]uint, len(invoices))
	for i, inv := range invoices {
		ids[i] = inv.ID
	}
	paid, err := PaidAmounts(db, ids)
	if err != nil {
		return err
	}
	for i := range invoices {
		inv := &invoices[i]
		amount, charged := paid[inv.ID]
		if !charged && inv.CreatedAt.Before(*firstAt) {
			continue
		}
		status := PaymentStatus(inv.Total, amount)
		inv.PaidAmount = &amount
		inv.PaymentStatus = &status
	}
	return nil
}
