package notifications_service

import (
	"encoding/json"
	"pengi-med-saas/core/tenantdb"
	"time"

	notifications_models "pengi-med-saas/features/notifications/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CreateNotificationInput struct {
	TenantID     uint
	UserID       uint
	Type         string
	ResourceType string
	ResourceID   uint
	MessageKey   string
	Params       map[string]string
	ActionURL    string
}

// CreateIfNotExists creates a notification unless an unread one already
// exists for the same tenant/type/resource. This is what keeps a periodic
// background check (e.g. a scheduler ticking every few minutes) from
// spamming the same reminder on every tick while the user hasn't acted on it.
func CreateIfNotExists(db *gorm.DB, logger *zap.Logger, input CreateNotificationInput) error {
	// Count (not First) so the common case — no existing unread notification
	// yet — doesn't trip GORM's default logger, which logs ErrRecordNotFound
	// as an error line on every miss.
	var existingCount int64
	if err := tenantdb.ForTenant(db, input.TenantID).Model(&notifications_models.Notification{}).Where(
		"type = ? AND resource_type = ? AND resource_id = ? AND read_at IS NULL",
		input.Type, input.ResourceType, input.ResourceID,
	).Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 {
		return nil
	}

	params, err := json.Marshal(input.Params)
	if err != nil {
		return err
	}

	notification := notifications_models.Notification{
		TenantID:     input.TenantID,
		UserID:       input.UserID,
		Type:         input.Type,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		MessageKey:   input.MessageKey,
		Params:       params,
		ActionURL:    input.ActionURL,
		Level:        notifications_models.NotificationLevelInfo,
	}

	if err := tenantdb.ForTenant(db, input.TenantID).Create(&notification).Error; err != nil {
		logger.Error("failed to create notification", zap.Error(err), zap.String("type", input.Type))
		return err
	}

	return nil
}

func MarkAsRead(db *gorm.DB, tenantID, userID, notificationID uint) error {
	now := time.Now()
	return tenantdb.ForTenant(db, tenantID).Model(&notifications_models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read_at", now).Error
}

func MarkAllAsRead(db *gorm.DB, tenantID, userID uint) error {
	now := time.Now()
	return tenantdb.ForTenant(db, tenantID).Model(&notifications_models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}

// Delete soft-deletes one of the user's notifications and reports how many
// rows matched, so the caller can tell "not found / not yours" apart.
func Delete(db *gorm.DB, tenantID, userID, notificationID uint) (int64, error) {
	result := tenantdb.ForTenant(db, tenantID).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&notifications_models.Notification{})
	return result.RowsAffected, result.Error
}

func DeleteRead(db *gorm.DB, tenantID, userID uint) error {
	return tenantdb.ForTenant(db, tenantID).
		Where("user_id = ? AND read_at IS NOT NULL", userID).
		Delete(&notifications_models.Notification{}).Error
}
