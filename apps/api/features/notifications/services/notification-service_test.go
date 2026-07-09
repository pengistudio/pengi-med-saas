package notifications_service_test

import (
	"testing"

	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_service "pengi-med-saas/features/notifications/services"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestCreateIfNotExists_CreatesWhenNoneExists(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	input := notifications_service.CreateNotificationInput{
		TenantID:     1,
		UserID:       1,
		Type:         "clinical.draft.stale",
		ResourceType: "medical_record_draft",
		ResourceID:   10,
		MessageKey:   "notification.clinical.draft.stale",
		Params:       map[string]string{"patient_name": "Juan Perez", "minutes_elapsed": "75"},
		ActionURL:    "/clinical/medical-records/10",
	}

	if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 notification, got %d", count)
	}
}

func TestCreateIfNotExists_SkipsWhenUnreadExists(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	input := notifications_service.CreateNotificationInput{
		TenantID:     1,
		UserID:       1,
		Type:         "clinical.draft.stale",
		ResourceType: "medical_record_draft",
		ResourceID:   10,
		MessageKey:   "notification.clinical.draft.stale",
		Params:       map[string]string{"patient_name": "Juan Perez", "minutes_elapsed": "75"},
		ActionURL:    "/clinical/medical-records/10",
	}

	if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}
	// Second tick, same stale draft, still unread — must be a no-op.
	input.Params["minutes_elapsed"] = "85"
	if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
		t.Fatalf("unexpected error on second create: %v", err)
	}

	var count int64
	db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected still 1 notification (deduped), got %d", count)
	}
}

func TestCreateIfNotExists_CreatesAgainAfterRead(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	input := notifications_service.CreateNotificationInput{
		TenantID:     1,
		UserID:       1,
		Type:         "clinical.draft.stale",
		ResourceType: "medical_record_draft",
		ResourceID:   10,
		MessageKey:   "notification.clinical.draft.stale",
		Params:       map[string]string{"minutes_elapsed": "75"},
	}

	if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var first notifications_models.Notification
	if err := db.First(&first).Error; err != nil {
		t.Fatalf("failed to load created notification: %v", err)
	}

	if err := notifications_service.MarkAsRead(db, 1, 1, first.ID); err != nil {
		t.Fatalf("failed to mark as read: %v", err)
	}

	if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
		t.Fatalf("unexpected error creating after read: %v", err)
	}

	var count int64
	db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 notifications (new reminder after read), got %d", count)
	}
}

func TestMarkAllAsRead_OnlyAffectsOwnUser(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	for _, userID := range []uint{1, 2} {
		input := notifications_service.CreateNotificationInput{
			TenantID:     1,
			UserID:       userID,
			Type:         "clinical.draft.stale",
			ResourceType: "medical_record_draft",
			ResourceID:   uint(10 + userID),
			MessageKey:   "notification.clinical.draft.stale",
			Params:       map[string]string{},
		}
		if err := notifications_service.CreateIfNotExists(db, logger, input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if err := notifications_service.MarkAllAsRead(db, 1, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var user1Unread, user2Unread int64
	db.Model(&notifications_models.Notification{}).Where("user_id = ? AND read_at IS NULL", 1).Count(&user1Unread)
	db.Model(&notifications_models.Notification{}).Where("user_id = ? AND read_at IS NULL", 2).Count(&user2Unread)

	if user1Unread != 0 {
		t.Fatalf("expected user 1 to have 0 unread, got %d", user1Unread)
	}
	if user2Unread != 1 {
		t.Fatalf("expected user 2 to still have 1 unread, got %d", user2Unread)
	}
}
