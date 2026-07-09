package notifications_models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model
	TenantID     uint           `json:"tenant_id" gorm:"index:idx_notif_tenant_user_read;index:idx_notif_tenant_type_resource;not null"`
	UserID       uint           `json:"user_id" gorm:"index:idx_notif_tenant_user_read;not null"`
	Type         string         `json:"type" gorm:"index:idx_notif_tenant_type_resource;not null"`
	ResourceType string         `json:"resource_type" gorm:"index:idx_notif_tenant_type_resource"`
	ResourceID   uint           `json:"resource_id" gorm:"index:idx_notif_tenant_type_resource"`
	MessageKey   string         `json:"message_key" gorm:"not null"`
	Params       datatypes.JSON `json:"params" gorm:"type:jsonb;default:'{}'"`
	ActionURL    string         `json:"action_url"`
	ReadAt       *time.Time     `json:"read_at" gorm:"index:idx_notif_tenant_user_read"`
}

func (Notification) IsAuditable() bool { return false }
