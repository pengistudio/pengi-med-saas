package integration_models

import (
	"time"

	"gorm.io/gorm"
)

type TenantIntegration struct {
	gorm.Model
	TenantID           uint       `json:"tenant_id" gorm:"uniqueIndex"`
	GoogleAccessToken  string     `json:"-"`
	GoogleRefreshToken string     `json:"-"`
	GoogleTokenExpiry  *time.Time `json:"-"`
	GoogleCalendarID   string     `json:"google_calendar_id" gorm:"default:primary"`
	GoogleConnected    bool       `json:"google_connected"`
}
