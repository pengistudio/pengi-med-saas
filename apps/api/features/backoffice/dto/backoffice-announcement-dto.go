package backoffice_dto

import "time"

type CreateAnnouncementDTO struct {
	Scope       string     `json:"scope" binding:"required,oneof=global company user"`
	CompanyID   *uint      `json:"company_id"`
	UserID      *uint      `json:"user_id"`
	Title       string     `json:"title" binding:"required,max=120"`
	Body        string     `json:"body" binding:"required,max=1000"`
	Level       string     `json:"level" binding:"required,oneof=info success warning critical"`
	ActionURL   string     `json:"action_url" binding:"max=500"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// AnnouncementDTO is an announcement plus the names of its target, for the
// history list.
type AnnouncementDTO struct {
	ID             uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Scope          string     `json:"scope"`
	CompanyID      *uint      `json:"company_id"`
	CompanyName    string     `json:"company_name"`
	UserID         *uint      `json:"user_id"`
	UserName       string     `json:"user_name"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	Level          string     `json:"level"`
	ActionURL      string     `json:"action_url"`
	ScheduledAt    *time.Time `json:"scheduled_at"`
	SentAt         *time.Time `json:"sent_at"`
	Status         string     `json:"status"`
	RecipientCount int        `json:"recipient_count"`
}
