package company_services

import (
	"testing"
	"time"

	company_models "pengi-med-saas/features/companies/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

// Renewing a subscription that ends on 31 January by one month must end on the
// last day of February, not overflow into March.
func TestApplyPaidSubscription_RenewalClampsToTheEndOfTheMonth(t *testing.T) {
	db := testutils.SetupTestDB(t, &company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{})
	loc := company_models.SubscriptionLocation
	company := company_models.Company{TradeName: "Clínica Norte", PlanCode: "pro"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	sub := company_models.Subscription{
		CompanyID: company.ID,
		PlanCode:  "pro",
		Status:    "active",
		ExpiresAt: time.Date(2027, time.January, 31, 23, 59, 59, 0, loc),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}

	ApplyPaidSubscription(db, zap.NewNop(), &sub, &company, &company_models.SubscriptionPayment{Months: 1})

	want := time.Date(2027, time.February, 28, 23, 59, 59, 0, loc)
	if !sub.ExpiresAt.Equal(want) {
		t.Fatalf("renewed expiry = %s, want %s", sub.ExpiresAt.In(loc), want)
	}
}

// Renewing a subscription that already expired counts the paid months from
// today, not from the day it expired: the months in between were not used.
func TestApplyPaidSubscription_ExpiredRenewalStartsToday(t *testing.T) {
	db := testutils.SetupTestDB(t, &company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{})
	loc := company_models.SubscriptionLocation
	now = func() time.Time { return time.Date(2027, time.October, 10, 15, 0, 0, 0, loc) }
	t.Cleanup(func() { now = time.Now })

	company := company_models.Company{TradeName: "Clínica Norte", PlanCode: "pro"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	sub := company_models.Subscription{
		CompanyID: company.ID,
		PlanCode:  "pro",
		Status:    "expired",
		ExpiresAt: time.Date(2027, time.June, 16, 23, 59, 59, 0, loc),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}

	ApplyPaidSubscription(db, zap.NewNop(), &sub, &company, &company_models.SubscriptionPayment{Months: 1})

	want := time.Date(2027, time.November, 10, 23, 59, 59, 0, loc)
	if !sub.ExpiresAt.Equal(want) {
		t.Fatalf("renewed expiry = %s, want %s", sub.ExpiresAt.In(loc), want)
	}
}
