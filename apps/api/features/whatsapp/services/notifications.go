package whatsapp_services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// New-message notifications: one per user and conversation while unread.
const (
	NotificationTypeMessage       = "whatsapp.message"
	ConversationResourceType      = "whatsapp_conversation"
	MessageNotificationKey        = "notification.whatsapp.message"
	inboxPermission               = "USE_WHATSAPP_INBOX"
	notificationPreviewMaxRunes   = 80
	notificationParamName         = "name"
	notificationParamCount        = "count"
	notificationParamPreview      = "preview"
	notificationParamContentType  = "content_type"
	notificationParamConversation = "conversation_id"
)

// MessageNotificationURL is where a new-message notification leads.
func MessageNotificationURL(convID uint) string {
	return fmt.Sprintf("/whatsapp?c=%d", convID)
}

// NotifyInbound tells every user with USE_WHATSAPP_INBOX that msg arrived in
// conv. A user who still has an unread notification of that conversation
// gets it updated (count + 1, latest preview, updated_at) instead of a new
// one. Params: name (patient or phone), count (number), preview, content_type.
// Failures are logged: the message is already stored.
func NotifyInbound(db *gorm.DB, logger *zap.Logger, tenantID uint, conv whatsapp_models.WhatsAppConversation, msg whatsapp_models.WhatsAppMessage) {
	userIDs, err := UsersWithPermission(db, tenantID, inboxPermission)
	if err != nil {
		logger.Error("failed to resolve whatsapp inbox users", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
	}
	tdb := tenantdb.ForTenant(db, tenantID)
	name := conversationName(tdb, conv)
	preview := truncateRunes(Preview(msg.Body), notificationPreviewMaxRunes)
	now := time.Now()
	for _, uid := range userIDs {
		var existing notifications_models.Notification
		if err := tdb.Where("user_id = ? AND type = ? AND resource_type = ? AND resource_id = ? AND read_at IS NULL",
			uid, NotificationTypeMessage, ConversationResourceType, conv.ID).
			Order("id DESC").Limit(1).Find(&existing).Error; err != nil {
			logger.Error("failed to load whatsapp message notification", zap.Uint("user_id", uid), zap.Error(err))
			continue
		}
		count := 1
		if existing.ID != 0 {
			var prev map[string]any
			if json.Unmarshal(existing.Params, &prev) == nil {
				if c, ok := prev[notificationParamCount].(float64); ok && c >= 1 {
					count = int(c) + 1
				}
			}
		}
		params, _ := json.Marshal(map[string]any{
			notificationParamName: name, notificationParamCount: count, notificationParamPreview: preview,
			notificationParamContentType: msg.ContentType, notificationParamConversation: conv.ID,
		})
		if existing.ID != 0 {
			err = tdb.Model(&existing).Updates(map[string]any{"params": params, "updated_at": now}).Error
		} else {
			err = tdb.Create(&notifications_models.Notification{
				TenantID: tenantID, UserID: uid, Type: NotificationTypeMessage, ResourceType: ConversationResourceType,
				ResourceID: conv.ID, MessageKey: MessageNotificationKey, Params: params,
				ActionURL: MessageNotificationURL(conv.ID), Level: notifications_models.NotificationLevelInfo,
			}).Error
		}
		if err != nil {
			logger.Error("failed to upsert whatsapp message notification", zap.Uint("user_id", uid), zap.Error(err))
		}
	}
}

// MarkConversationNotificationsRead marks userID's new-message notifications
// of conversation convID as read (db bound to the tenant).
func MarkConversationNotificationsRead(db *gorm.DB, userID, convID uint) error {
	return db.Model(&notifications_models.Notification{}).
		Where("user_id = ? AND type = ? AND resource_type = ? AND resource_id = ? AND read_at IS NULL",
			userID, NotificationTypeMessage, ConversationResourceType, convID).
		Update("read_at", time.Now()).Error
}

// conversationName is the linked patient's name, else the number (+E.164).
func conversationName(db *gorm.DB, conv whatsapp_models.WhatsAppConversation) string {
	if conv.PatientID != nil {
		var p clinical_models.Patient
		if err := db.Select("id", "first_name", "last_name").Where("id = ?", *conv.PatientID).Limit(1).Find(&p).Error; err == nil && p.ID != 0 {
			if name := strings.TrimSpace(p.FirstName + " " + p.LastName); name != "" {
				return name
			}
		}
	}
	return "+" + conv.Phone
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
