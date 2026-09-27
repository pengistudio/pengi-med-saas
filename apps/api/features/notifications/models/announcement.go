package notifications_models

import (
	"time"

	"gorm.io/gorm"
)

const (
	AnnouncementScopeGlobal  = "global"
	AnnouncementScopeCompany = "company"
	AnnouncementScopeUser    = "user"

	AnnouncementStatusScheduled = "scheduled"
	AnnouncementStatusSent      = "sent"
	AnnouncementStatusCancelled = "cancelled"
	AnnouncementStatusFailed    = "failed"

	NotificationLevelInfo = "info"
)

// Announcement is a message written by a backoffice admin. It has no TenantID:
// it belongs to the platform, and on dispatch it is fanned out into one
// Notification per recipient (tenant, user) that exists at that moment.
type Announcement struct {
	gorm.Model
	Scope          string     `json:"scope" gorm:"not null"`
	CompanyID      *uint      `json:"company_id"`
	UserID         *uint      `json:"user_id"`
	Title          string     `json:"title" gorm:"not null"`
	Body           string     `json:"body" gorm:"not null"`
	Level          string     `json:"level" gorm:"not null;default:info"`
	ActionURL      string     `json:"action_url"`
	ScheduledAt    *time.Time `json:"scheduled_at" gorm:"index:idx_announcement_status_scheduled"`
	SentAt         *time.Time `json:"sent_at"`
	Status         string     `json:"status" gorm:"not null;index:idx_announcement_status_scheduled"`
	RecipientCount int        `json:"recipient_count" gorm:"not null;default:0"`
	CreatedByID    uint       `json:"created_by_id"`
}
