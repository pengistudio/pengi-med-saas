package notifications_handlers

import (
	"net/http/httptest"
	"testing"

	notifications_dto "pengi-med-saas/features/notifications/dto"
	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_service "pengi-med-saas/features/notifications/services"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// newAuthedContext builds a gin context scoped to a tenant/user, matching
// what auth_middleware.GetUserFromContext expects ("user_id" + "username"),
// which testutils.NewGinContext does not set on its own.
func newAuthedContext(tenantID uint, userID int64) *gin.Context {
	c, _ := testutils.NewGinContext(tenantID, userID)
	c.Set("user_id", userID)
	c.Set("username", "doctor")
	c.Request = httptest.NewRequest("GET", "/notifications", nil)
	return c
}

func TestListNotifications_ScopedToTenantAndUser(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	// tenant 1 / user 1's own notification
	if err := notifications_service.CreateIfNotExists(db, logger, notifications_service.CreateNotificationInput{
		TenantID: 1, UserID: 1, Type: "clinical.draft.stale", ResourceType: "medical_record_draft", ResourceID: 1,
		MessageKey: "notification.clinical.draft.stale", Params: map[string]string{},
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	// tenant 1 / user 2's notification — must not show up for user 1
	if err := notifications_service.CreateIfNotExists(db, logger, notifications_service.CreateNotificationInput{
		TenantID: 1, UserID: 2, Type: "clinical.draft.stale", ResourceType: "medical_record_draft", ResourceID: 2,
		MessageKey: "notification.clinical.draft.stale", Params: map[string]string{},
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	// tenant 2 / user 1's notification — must not leak across tenants
	if err := notifications_service.CreateIfNotExists(db, logger, notifications_service.CreateNotificationInput{
		TenantID: 2, UserID: 1, Type: "clinical.draft.stale", ResourceType: "medical_record_draft", ResourceID: 3,
		MessageKey: "notification.clinical.draft.stale", Params: map[string]string{},
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	handler := NewNotificationHandler(db, logger)
	c := newAuthedContext(1, 1)

	response := handler.ListNotifications(c)
	if response.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Message)
	}

	data, ok := response.Data.(notifications_dto.ListNotificationsResponse)
	if !ok {
		t.Fatalf("unexpected response data type: %T", response.Data)
	}
	if data.Total != 1 || len(data.Items) != 1 {
		t.Fatalf("expected exactly 1 notification for tenant 1/user 1, got total=%d items=%d", data.Total, len(data.Items))
	}
	if data.UnreadCount != 1 {
		t.Fatalf("expected unread_count 1, got %d", data.UnreadCount)
	}
}

func TestMarkAsRead_SetsReadAt(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	if err := notifications_service.CreateIfNotExists(db, logger, notifications_service.CreateNotificationInput{
		TenantID: 1, UserID: 1, Type: "clinical.draft.stale", ResourceType: "medical_record_draft", ResourceID: 1,
		MessageKey: "notification.clinical.draft.stale", Params: map[string]string{},
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	var created notifications_models.Notification
	if err := db.First(&created).Error; err != nil {
		t.Fatalf("failed to load seeded notification: %v", err)
	}

	handler := NewNotificationHandler(db, logger)
	c := newAuthedContext(1, 1)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	response := handler.MarkAsRead(c)
	if response.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Message)
	}

	var updated notifications_models.Notification
	if err := db.First(&updated, created.ID).Error; err != nil {
		t.Fatalf("failed to reload notification: %v", err)
	}
	if updated.ReadAt == nil {
		t.Fatalf("expected ReadAt to be set")
	}
}

func TestMarkAllAsRead_DoesNotAffectOtherUser(t *testing.T) {
	db := testutils.SetupTestDB(t, &notifications_models.Notification{})
	logger := zap.NewNop()

	for _, userID := range []uint{1, 2} {
		if err := notifications_service.CreateIfNotExists(db, logger, notifications_service.CreateNotificationInput{
			TenantID: 1, UserID: userID, Type: "clinical.draft.stale", ResourceType: "medical_record_draft", ResourceID: userID,
			MessageKey: "notification.clinical.draft.stale", Params: map[string]string{},
		}); err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	handler := NewNotificationHandler(db, logger)
	c := newAuthedContext(1, 1)

	response := handler.MarkAllAsRead(c)
	if response.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Message)
	}

	var user2Unread int64
	db.Model(&notifications_models.Notification{}).Where("user_id = ? AND read_at IS NULL", 2).Count(&user2Unread)
	if user2Unread != 1 {
		t.Fatalf("expected user 2's notification to remain unread, got %d unread", user2Unread)
	}
}
