package billing_workers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type claimDoc struct {
	gorm.Model
	Status string
}

func setupClaimDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&claimDoc{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestClaimForProcessing(t *testing.T) {
	stale := time.Now().Add(-StuckProcessingThreshold - time.Minute)
	fresh := time.Now()

	cases := []struct {
		status    string
		updatedAt time.Time
		want      bool
	}{
		{billing_models.InvoiceStatusPending, fresh, true},
		{billing_models.InvoiceStatusFailed, fresh, true},
		{billing_models.InvoiceStatusConnectionError, fresh, true},
		{billing_models.InvoiceStatusProcessing, stale, true},
		{billing_models.InvoiceStatusProcessing, fresh, false},
		{billing_models.InvoiceStatusSigned, stale, false},
		{billing_models.InvoiceStatusValidated, stale, false},
		{billing_models.InvoiceStatusAuthorized, stale, false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_stale=%v", tc.status, tc.updatedAt.Equal(stale)), func(t *testing.T) {
			db := setupClaimDB(t)
			doc := claimDoc{Status: tc.status}
			db.Create(&doc)
			db.Model(&doc).UpdateColumn("updated_at", tc.updatedAt)

			got, err := claimForProcessing(db, &claimDoc{}, uint64(doc.ID))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("claimed = %v, want %v", got, tc.want)
			}

			var reloaded claimDoc
			db.First(&reloaded, doc.ID)
			wantStatus := tc.status
			if tc.want {
				wantStatus = billing_models.InvoiceStatusProcessing
			}
			if reloaded.Status != wantStatus {
				t.Fatalf("status = %q, want %q", reloaded.Status, wantStatus)
			}
		})
	}
}

func TestClaimForProcessingOnlyOnce(t *testing.T) {
	db := setupClaimDB(t)
	doc := claimDoc{Status: billing_models.InvoiceStatusPending}
	db.Create(&doc)

	first, _ := claimForProcessing(db, &claimDoc{}, uint64(doc.ID))
	second, _ := claimForProcessing(db, &claimDoc{}, uint64(doc.ID))
	if !first || second {
		t.Fatalf("first=%v second=%v, want true/false", first, second)
	}
}

func TestClaimForProcessingMissingDocument(t *testing.T) {
	db := setupClaimDB(t)
	got, err := claimForProcessing(db, &claimDoc{}, 999)
	if err != nil || got {
		t.Fatalf("got=%v err=%v, want false/nil", got, err)
	}
}

func TestIsRetryable(t *testing.T) {
	signerDown := fmt.Errorf("%w: %w", sri_services.ErrSignerUnreachable,
		fmt.Errorf("failed to sign XML: %w: %w", sri_services.ErrSriConnection, errors.New("connection refused")))
	sriDownAfterSubmit := fmt.Errorf("failed to authorize XML with SRI: %w: %w", sri_services.ErrSriConnection, errors.New("timeout"))

	cases := map[string]struct {
		err  error
		want bool
	}{
		"transient db error":                {fmt.Errorf("%w: %w", errTransient, errors.New("conn reset")), true},
		"signer unreachable before submit":  {signerDown, true},
		"sri connection error after submit": {sriDownAfterSubmit, false},
		"sri rejection":                     {errors.New("[SRI-ERROR] clave de acceso no registrada"), false},
		"missing tenant sri config":         {errors.New("missing SRI setup for tenant 1"), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isRetryable(tc.err); got != tc.want {
				t.Fatalf("isRetryable = %v, want %v", got, tc.want)
			}
		})
	}
}
