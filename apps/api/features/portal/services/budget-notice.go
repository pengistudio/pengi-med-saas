package portal_services

import (
	"fmt"
	"strconv"
	"sync"
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

// noticeMu keeps parallel logins from creating the day's notice twice.
var noticeMu sync.Mutex

// NotifyCodeBudget tells the clinic's portal admins (MANAGE_PATIENT_PORTAL)
// that its sign-in code budget is spent (budget is BudgetTenantHour or
// BudgetTenantDay): once per clinic per day in the clinic's time zone
// (deduplicated by type + resource portal_code_budget/yyyymmdd, read or not).
// Failures are logged.
func NotifyCodeBudget(db *gorm.DB, logger *zap.Logger, tenantID uint, budget string, now time.Time) {
	noticeMu.Lock()
	defer noticeMu.Unlock()
	day := DayID(now.In(whatsapp_services.ClinicLocation()))
	tdb := tenantdb.ForTenant(db, tenantID)
	var existing int64
	if err := tdb.Model(&notifications_models.Notification{}).
		Where("type = ? AND resource_type = ? AND resource_id = ?", NotificationTypeCodeBudget, CodeBudgetResourceType, day).
		Count(&existing).Error; err != nil {
		logger.Error("portal: failed to check code budget notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if existing > 0 {
		return
	}
	userIDs, err := whatsapp_services.UsersWithPermission(db, tenantID, managePortalPermission)
	if err != nil {
		logger.Error("portal: failed to resolve portal admins", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
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
	if err := tdb.Create(&rows).Error; err != nil {
		logger.Error("portal: failed to create code budget notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}

// DayID is t's date as yyyymmdd (e.g. 20261026).
func DayID(t time.Time) uint {
	id, _ := strconv.ParseUint(fmt.Sprintf("%04d%02d%02d", t.Year(), int(t.Month()), t.Day()), 10, 32)
	return uint(id)
}
