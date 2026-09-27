package notifications_service

import (
	"encoding/json"
	"errors"
	"time"

	notifications_models "pengi-med-saas/features/notifications/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	AnnouncementNotificationType = "announcement"
	AnnouncementMessageKey       = "notification.announcement"
)

// ErrNoRecipients: the announcement's target resolved to nobody (e.g. the
// user is no longer linked to that company).
var ErrNoRecipients = errors.New("announcement has no recipients")

type Recipient struct {
	TenantID uint
	UserID   uint
}

// ResolveRecipients lists the (tenant, user) pairs an announcement reaches
// right now: every active user linked to a live company, narrowed to one
// company or one user of that company by the announcement's scope. db must be
// a tenantdb.System handle — the query spans tenants.
func ResolveRecipients(db *gorm.DB, a *notifications_models.Announcement) ([]Recipient, error) {
	query := db.Table("environments AS e").
		Select("DISTINCT c.tenant_id AS tenant_id, e.user_id AS user_id").
		Joins("JOIN companies AS c ON c.id = e.company_id AND c.deleted_at IS NULL").
		Joins("JOIN users AS u ON u.id = e.user_id AND u.deleted_at IS NULL").
		Where("e.deleted_at IS NULL")

	switch a.Scope {
	case notifications_models.AnnouncementScopeGlobal:
	case notifications_models.AnnouncementScopeCompany:
		if a.CompanyID == nil {
			return nil, ErrNoRecipients
		}
		query = query.Where("e.company_id = ?", *a.CompanyID)
	case notifications_models.AnnouncementScopeUser:
		if a.CompanyID == nil || a.UserID == nil {
			return nil, ErrNoRecipients
		}
		query = query.Where("e.company_id = ? AND e.user_id = ?", *a.CompanyID, *a.UserID)
	default:
		return nil, ErrNoRecipients
	}

	var recipients []Recipient
	if err := query.Scan(&recipients).Error; err != nil {
		return nil, err
	}
	return recipients, nil
}

// Dispatch fans an announcement out into one Notification per recipient and
// marks it sent, in a single transaction: either every recipient gets it or
// none does. On failure the announcement is marked failed. db must be a
// tenantdb.System handle.
func Dispatch(db *gorm.DB, logger *zap.Logger, a *notifications_models.Announcement) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		recipients, err := ResolveRecipients(tx, a)
		if err != nil {
			return err
		}
		if len(recipients) == 0 {
			return ErrNoRecipients
		}

		params, err := json.Marshal(map[string]string{"title": a.Title, "body": a.Body})
		if err != nil {
			return err
		}

		notifications := make([]notifications_models.Notification, 0, len(recipients))
		for _, r := range recipients {
			notifications = append(notifications, notifications_models.Notification{
				TenantID:     r.TenantID,
				UserID:       r.UserID,
				Type:         AnnouncementNotificationType,
				ResourceType: AnnouncementNotificationType,
				ResourceID:   a.ID,
				MessageKey:   AnnouncementMessageKey,
				Params:       params,
				ActionURL:    a.ActionURL,
				Level:        a.Level,
			})
		}
		if err := tx.CreateInBatches(&notifications, 500).Error; err != nil {
			return err
		}

		now := time.Now()
		a.Status = notifications_models.AnnouncementStatusSent
		a.SentAt = &now
		a.RecipientCount = len(recipients)
		return tx.Model(a).Select("status", "sent_at", "recipient_count").Updates(a).Error
	})
	if err != nil {
		logger.Error("failed to dispatch announcement", zap.Uint("announcement_id", a.ID), zap.Error(err))
		a.Status = notifications_models.AnnouncementStatusFailed
		if updateErr := db.Model(a).Update("status", a.Status).Error; updateErr != nil {
			logger.Error("failed to mark announcement as failed", zap.Uint("announcement_id", a.ID), zap.Error(updateErr))
		}
		return err
	}
	return nil
}
