package cash_services

import (
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	"pengi-med-saas/testutils"
)

func TestApplyPaymentStatus(t *testing.T) {
	db := testutils.SetupTestDB(t, &clinical_models.Patient{}, &billing_models.Invoice{}, &cash_models.CashSession{}, &cash_models.Payment{})
	const tenant, other uint = 1, 2
	firstOpen := time.Date(2026, 11, 1, 13, 0, 0, 0, time.UTC)
	mk := func(tenantID uint, total float64, created time.Time) billing_models.Invoice {
		inv := billing_models.Invoice{TenantID: tenantID, Total: total}
		inv.CreatedAt = created
		if err := db.Create(&inv).Error; err != nil {
			t.Fatal(err)
		}
		return inv
	}
	pay := func(inv billing_models.Invoice, amount float64, status string) {
		p := cash_models.Payment{TenantID: inv.TenantID, InvoiceID: inv.ID, CashSessionID: 1, Method: "cash", Amount: amount, Kind: "payment", Status: status}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}

	oldUnpaid := mk(tenant, 40, firstOpen.Add(-time.Hour))
	oldCharged := mk(tenant, 40, firstOpen.Add(-time.Hour))
	unpaid := mk(tenant, 40, firstOpen.Add(time.Hour))
	partial := mk(tenant, 40, firstOpen.Add(time.Hour))
	paid := mk(tenant, 40, firstOpen.Add(time.Hour))
	refunded := mk(tenant, 40, firstOpen.Add(time.Hour))
	otherTenant := mk(other, 40, firstOpen.Add(time.Hour))

	// Before the tenant's first session there is no status at all.
	invoices := []billing_models.Invoice{unpaid}
	if err := ApplyPaymentStatus(tenantdb.ForTenant(db, tenant), invoices); err != nil || invoices[0].PaymentStatus != nil {
		t.Fatalf("without sessions: status=%v err=%v, want nil", invoices[0].PaymentStatus, err)
	}

	session := cash_models.CashSession{TenantID: tenant, Status: cash_models.SessionClosed, OpenedAt: firstOpen}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	pay(oldCharged, 40, cash_models.PaymentValid)
	pay(partial, 15, cash_models.PaymentValid)
	pay(partial, 25, cash_models.PaymentVoided)
	pay(paid, 30, cash_models.PaymentValid)
	pay(paid, 10, cash_models.PaymentValid)
	pay(refunded, 40, cash_models.PaymentValid)
	pay(refunded, -40, cash_models.PaymentValid)
	pay(otherTenant, 40, cash_models.PaymentValid)

	invoices = []billing_models.Invoice{oldUnpaid, oldCharged, unpaid, partial, paid, refunded}
	if err := ApplyPaymentStatus(tenantdb.ForTenant(db, tenant), invoices); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		status string
		paid   float64
	}{{"", 0}, {"paid", 40}, {"unpaid", 0}, {"partial", 15}, {"paid", 40}, {"unpaid", 0}}
	for i, w := range want {
		got := invoices[i]
		if w.status == "" {
			if got.PaymentStatus != nil || got.PaidAmount != nil {
				t.Errorf("invoice %d: status=%v paid=%v, want null (issued before the first session)", i, got.PaymentStatus, got.PaidAmount)
			}
			continue
		}
		if got.PaymentStatus == nil || *got.PaymentStatus != w.status || *got.PaidAmount != w.paid {
			t.Errorf("invoice %d: status=%v paid=%v, want %s %v", i, got.PaymentStatus, got.PaidAmount, w.status, w.paid)
		}
	}

	// The other tenant never opened a session: its payments don't leak in, and
	// its invoice has no status.
	others := []billing_models.Invoice{otherTenant}
	if err := ApplyPaymentStatus(tenantdb.ForTenant(db, other), others); err != nil || others[0].PaymentStatus != nil {
		t.Fatalf("other tenant: status=%v err=%v", others[0].PaymentStatus, err)
	}
}
