package y2026

import (
	"testing"
	"time"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	"pengi-med-saas/testutils"
)

func TestSubscriptionExpiryEndOfDay_MovesBackofficeDatesToTheEndOfThatDayInEcuador(t *testing.T) {
	db := testutils.SetupTestDB(t, &company_models.Subscription{})
	backoffice := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC) // "16/11" as the backoffice stored it
	trial := time.Date(2026, 10, 10, 14, 23, 7, 0, time.UTC)    // signup: now + 14 days
	for _, at := range []time.Time{backoffice, trial} {
		if err := db.Create(&company_models.Subscription{PlanCode: "pro", Status: "active", ExpiresAt: at, CompanyID: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}

	m, ok := database.GlobalDBMap["DB20260926_2"]
	if !ok {
		t.Fatal("migration DB20260926_2 not registered")
	}
	if err := m.Execute(db); err != nil {
		t.Fatal(err)
	}

	var subs []company_models.Subscription
	db.Order("id").Find(&subs)
	if want := time.Date(2026, 11, 17, 4, 59, 59, 0, time.UTC); !subs[0].ExpiresAt.Equal(want) {
		t.Fatalf("backoffice subscription = %s, want %s", subs[0].ExpiresAt.UTC(), want)
	}
	if !subs[1].ExpiresAt.Equal(trial) {
		t.Fatalf("trial subscription changed to %s", subs[1].ExpiresAt.UTC())
	}
}
