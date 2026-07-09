package notifications_dto

import (
	"encoding/json"
	"time"

	notifications_models "pengi-med-saas/features/notifications/models"
)

// ID/CreatedAt/UpdatedAt are deliberately left without json tags (Go's
// default is the capitalized field name) to match gorm.Model's own JSON
// shape, which every BaseModel-derived frontend type expects.
type NotificationDTO struct {
	ID           uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Type         string            `json:"type"`
	ResourceType string            `json:"resource_type"`
	ResourceID   uint              `json:"resource_id"`
	MessageKey   string            `json:"message_key"`
	Params       map[string]string `json:"params"`
	ActionURL    string            `json:"action_url"`
	ReadAt       *time.Time        `json:"read_at"`
}

type ListNotificationsResponse struct {
	Items       []NotificationDTO `json:"items"`
	UnreadCount int               `json:"unread_count"`
	Total       int               `json:"total"`
	Page        int               `json:"page"`
	Limit       int               `json:"limit"`
}

func ToNotificationDTO(n notifications_models.Notification) NotificationDTO {
	params := map[string]string{}
	if len(n.Params) > 0 {
		_ = json.Unmarshal(n.Params, &params)
	}
	return NotificationDTO{
		ID:           n.ID,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
		Type:         n.Type,
		ResourceType: n.ResourceType,
		ResourceID:   n.ResourceID,
		MessageKey:   n.MessageKey,
		Params:       params,
		ActionURL:    n.ActionURL,
		ReadAt:       n.ReadAt,
	}
}

func ToNotificationDTOs(notifications []notifications_models.Notification) []NotificationDTO {
	dtos := make([]NotificationDTO, 0, len(notifications))
	for _, n := range notifications {
		dtos = append(dtos, ToNotificationDTO(n))
	}
	return dtos
}
