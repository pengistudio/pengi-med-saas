package portal_services

import (
	"fmt"
	"strconv"
	"time"

	"pengi-med-saas/core/tenantdb"
	notifications_models "pengi-med-saas/features/notifications/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Staff notification when a clinic spends its sign-in code budget.
const (
	NotificationTypeCodeBudget = "portal.code_budget.reached"
	CodeBudgetResourceType     = "portal_code_budget"
	managePortalPermission     = "MANAGE_PATIENT_PORTAL"
)

// NotifyCodeBudget tells the clinic's portal admins (MANAGE_PATIENT_PORTAL)
// that its sign-in code budget is spent (budget is BudgetTenantHour or
// BudgetTenantDay): once per clinic per day in the clinic's time zone
// (deduplicated by type + resource portal_code_budget/yyyymmdd, read or not).
// Failures are logged.
func NotifyCodeBudget(db *gorm.DB, logger *zap.Logger, tenantID uint, budget string, now time.Time) {
	day := DayID(now.In(whatsapp_services.ClinicLocation()))
	// The lock (per clinic and day, across API instances) keeps parallel
	// logins from creating the day's notice twice.
	err := tenantdb.ForTenant(db, tenantID).Transaction(func(tx *gorm.DB) error {
		if err := advisoryLock(tx, fmt.Sprintf("portal-code-budget-notice:%d:%d", tenantID, day)); err != nil {
			return err
		}
		var existing int64
		if err := tx.Model(&notifications_models.Notification{}).
			Where("type = ? AND resource_type = ? AND resource_id = ?", NotificationTypeCodeBudget, CodeBudgetResourceType, day).
			Count(&existing).Error; err != nil {
			return fmt.Errorf("check code budget notifications: %w", err)
		}
		if existing > 0 {
			return nil
		}
		userIDs, err := whatsapp_services.UsersWithPermission(tx, tenantID, managePortalPermission)
		if err != nil {
			return fmt.Errorf("resolve portal admins: %w", err)
		}
		if len(userIDs) == 0 {
			return nil
		}
		key := "notification.portal.code_budget.day"
		if budget == BudgetTenantHour {
			key = "notification.portal.code_budget.hour"
		}
		rows := make([]notifications_models.Notification, 0, len(userIDs))
		for _, uid := range userIDs {
			rows = append(rows, notifications_models.Notification{
				TenantID: tenantID, UserID: uid, Type: NotificationTypeCodeBudget, ResourceType: CodeBudgetResourceType, ResourceID: day,
				MessageKey: key, Params: []byte("{}"), ActionURL: "/settings/portal", Level: notifications_models.NotificationLevelInfo,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create code budget notifications: %w", err)
		}
		return nil
	})
	if err != nil {
		logger.Error("portal: failed to notify the code budget", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}

// DayID is t's date as yyyymmdd (e.g. 20261026).
func DayID(t time.Time) uint {
	id, _ := strconv.ParseUint(fmt.Sprintf("%04d%02d%02d", t.Year(), int(t.Month()), t.Day()), 10, 32)
	return uint(id)
}
