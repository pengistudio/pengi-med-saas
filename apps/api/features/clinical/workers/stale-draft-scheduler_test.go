package clinical_workers

import (
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestSchedulerNotifiesStaleDraft(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&clinical_models.Patient{},
		&clinical_models.MedicalRecordDraft{},
		&notifications_models.Notification{},
	)
	logger := zap.NewNop()

	patient := clinical_models.Patient{TenantID: 1, FirstName: "Juan", LastName: "Perez", Document: "DOC-1", Phone: "555"}
	if err := db.Create(&patient).Error; err != nil {
		t.Fatalf("failed to create patient: %v", err)
	}

	draft := clinical_models.MedicalRecordDraft{TenantID: 1, UserID: 1, PatientID: patient.ID}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	// Backdate updated_at past the threshold — bypassing GORM's auto-touch.
	staleTime := time.Now().Add(-90 * time.Minute)
	if err := db.Model(&draft).UpdateColumn("updated_at", staleTime).Error; err != nil {
		t.Fatalf("failed to backdate draft: %v", err)
	}

	scheduler := NewStaleDraftScheduler(db, logger)
	scheduler.checkStaleDrafts()

	var notifications []notifications_models.Notification
	if err := db.Find(&notifications).Error; err != nil {
		t.Fatalf("failed to fetch notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	if notifications[0].Type != "clinical.draft.stale" {
		t.Fatalf("expected type clinical.draft.stale, got %s", notifications[0].Type)
	}
	if notifications[0].UserID != draft.UserID || notifications[0].TenantID != draft.TenantID {
		t.Fatalf("notification not scoped to the draft's tenant/user")
	}
}

func TestSchedulerSkipsFreshDraft(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&clinical_models.Patient{},
		&clinical_models.MedicalRecordDraft{},
		&notifications_models.Notification{},
	)
	logger := zap.NewNop()

	patient := clinical_models.Patient{TenantID: 1, FirstName: "Juan", LastName: "Perez", Document: "DOC-2", Phone: "555"}
	if err := db.Create(&patient).Error; err != nil {
		t.Fatalf("failed to create patient: %v", err)
	}

	draft := clinical_models.MedicalRecordDraft{TenantID: 1, UserID: 1, PatientID: patient.ID}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	// updated_at defaults to "now" on create — well within the threshold.

	scheduler := NewStaleDraftScheduler(db, logger)
	scheduler.checkStaleDrafts()

	var count int64
	db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 notifications for a fresh draft, got %d", count)
	}
}

func TestSchedulerSkipsAlreadyNotifiedDraft(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&clinical_models.Patient{},
		&clinical_models.MedicalRecordDraft{},
		&notifications_models.Notification{},
	)
	logger := zap.NewNop()

	patient := clinical_models.Patient{TenantID: 1, FirstName: "Juan", LastName: "Perez", Document: "DOC-3", Phone: "555"}
	if err := db.Create(&patient).Error; err != nil {
		t.Fatalf("failed to create patient: %v", err)
	}

	draft := clinical_models.MedicalRecordDraft{TenantID: 1, UserID: 1, PatientID: patient.ID}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	staleTime := time.Now().Add(-90 * time.Minute)
	if err := db.Model(&draft).UpdateColumn("updated_at", staleTime).Error; err != nil {
		t.Fatalf("failed to backdate draft: %v", err)
	}

	scheduler := NewStaleDraftScheduler(db, logger)
	scheduler.checkStaleDrafts()
	scheduler.checkStaleDrafts() // second tick, draft still unread and still stale

	var count int64
	db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 notification (deduped across ticks), got %d", count)
	}
}
