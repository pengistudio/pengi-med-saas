package whatsapp_services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"pengi-med-saas/core/tenantdb"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlanLimitKey is the plan property (Plan.Properties) with the monthly cap of
// template messages; absent or -1 means unlimited.
const PlanLimitKey = "max_whatsapp_messages"

// Usage notifications (one of each per tenant and month, to the users with
// MANAGE_WHATSAPP).
const (
	NotificationTypeUsageWarning = "whatsapp.usage.warning" // 80% of the cap
	NotificationTypeUsageReached = "whatsapp.usage.reached" // 100% of the cap
	UsageResourceType            = "whatsapp_usage"
	usageWarningPercent          = 80
	managePermission             = "MANAGE_WHATSAPP"
)

// Usage is the tenant's template messages this month against its cap.
// Limit is -1 when the plan sets no cap. PeriodEnd is exclusive (the first
// instant of next month), both in the clinic's time zone.
type Usage struct {
	Used        int64
	Limit       int64
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// Reached reports whether no more template messages may be sent.
func (u Usage) Reached() bool { return u.Limit >= 0 && u.Used >= u.Limit }

// Period is the calendar month containing now in loc: [start, end).
func Period(now time.Time, loc *time.Location) (time.Time, time.Time) {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, 0)
}

// PlanLimit is the tenant's monthly cap (-1 unlimited, also when the tenant
// has no company or active subscription).
func PlanLimit(db *gorm.DB, tenantID uint) int64 {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Select("id").Limit(1).Find(&company).Error; err != nil || company.ID == 0 {
		return -1
	}
	return subscription_middleware.GetPlanLimitForCompany(db, company.ID, PlanLimitKey)
}

// CountMonthly counts the tenant's charged messages (CountedKinds) that
// reached Meta (sent_at set) in [start, end). Replies inside the window,
// system messages, inbound ones and messages that failed or were skipped
// before sending don't count.
func CountMonthly(db *gorm.DB, tenantID uint, start, end time.Time) (int64, error) {
	var n int64
	// Bounds in UTC: SQLite (CI tests) compares timestamps as text, so a
	// -05:00 bound against +00:00 rows would shift the month by five hours.
	err := tenantdb.ForTenant(db, tenantID).Model(&whatsapp_models.WhatsAppMessage{}).
		Where("kind IN ? AND sent_at IS NOT NULL AND sent_at >= ? AND sent_at < ?", whatsapp_models.CountedKinds, start.UTC(), end.UTC()).
		Count(&n).Error
	return n, err
}

// MonthlyUsage is the tenant's usage in the calendar month (clinic time
// zone, ClinicLocation) containing now.
func MonthlyUsage(db *gorm.DB, tenantID uint, now time.Time) (Usage, error) {
	return monthlyUsage(db, tenantID, now, ClinicLocation())
}

func monthlyUsage(db *gorm.DB, tenantID uint, now time.Time, loc *time.Location) (Usage, error) {
	start, end := Period(now, loc)
	u := Usage{Limit: PlanLimit(db, tenantID), PeriodStart: start, PeriodEnd: end}
	used, err := CountMonthly(db, tenantID, start, end)
	if err != nil {
		return u, err
	}
	u.Used = used
	return u, nil
}

// CanSendTemplate reports whether the tenant may send one more template
// message now. Concurrent sends may overshoot the cap slightly; accepted.
func CanSendTemplate(db *gorm.DB, tenantID uint, now time.Time) (bool, Usage, error) {
	return canSendTemplate(db, tenantID, now, ClinicLocation())
}

func canSendTemplate(db *gorm.DB, tenantID uint, now time.Time, loc *time.Location) (bool, Usage, error) {
	start, end := Period(now, loc)
	u := Usage{Limit: PlanLimit(db, tenantID), PeriodStart: start, PeriodEnd: end}
	if u.Limit < 0 {
		return true, u, nil // unlimited: no need to count
	}
	used, err := CountMonthly(db, tenantID, start, end)
	if err != nil {
		return false, u, err
	}
	u.Used = used
	return !u.Reached(), u, nil
}

// NotifyUsage tells the tenant's WhatsApp admins when usage this month
// reaches 80% and 100% of the cap, once per month each (deduplicated by type
// + resource whatsapp_usage/yyyymm, read or not). Failures are logged.
func NotifyUsage(db *gorm.DB, logger *zap.Logger, tenantID uint, now time.Time) {
	notifyUsage(db, logger, tenantID, now, ClinicLocation())
}

func notifyUsage(db *gorm.DB, logger *zap.Logger, tenantID uint, now time.Time, loc *time.Location) {
	u, err := monthlyUsage(db, tenantID, now, loc)
	if err != nil {
		logger.Error("failed to compute whatsapp usage", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if u.Limit <= 0 {
		return
	}
	notifType, key := "", ""
	switch {
	case u.Used >= u.Limit:
		notifType, key = NotificationTypeUsageReached, "notification.whatsapp.usage.reached"
	case u.Used*100 >= u.Limit*usageWarningPercent:
		notifType, key = NotificationTypeUsageWarning, "notification.whatsapp.usage.warning"
	default:
		return
	}
	period := PeriodID(u.PeriodStart)
	tdb := tenantdb.ForTenant(db, tenantID)
	var existing int64
	if err := tdb.Model(&notifications_models.Notification{}).
		Where("type = ? AND resource_type = ? AND resource_id = ?", notifType, UsageResourceType, period).
		Count(&existing).Error; err != nil {
		logger.Error("failed to check whatsapp usage notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if existing > 0 {
		return
	}
	userIDs, err := UsersWithPermission(db, tenantID, managePermission)
	if err != nil {
		logger.Error("failed to resolve whatsapp admins", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
	}
	params, _ := json.Marshal(map[string]string{"used": strconv.FormatInt(u.Used, 10), "limit": strconv.FormatInt(u.Limit, 10)})
	level := notifications_models.NotificationLevelInfo
	rows := make([]notifications_models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		rows = append(rows, notifications_models.Notification{
			TenantID: tenantID, UserID: uid, Type: notifType, ResourceType: UsageResourceType, ResourceID: period,
			MessageKey: key, Params: params, ActionURL: "/settings", Level: level,
		})
	}
	if err := tdb.Create(&rows).Error; err != nil {
		logger.Error("failed to create whatsapp usage notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}

// PeriodID is the month of start as yyyymm (e.g. 202610).
func PeriodID(start time.Time) uint {
	id, _ := strconv.ParseUint(fmt.Sprintf("%04d%02d", start.Year(), int(start.Month())), 10, 32)
	return uint(id)
}

// UsersWithPermission returns the members of the tenant's companies whose role
// has permissionID.
func UsersWithPermission(db *gorm.DB, tenantID uint, permissionID string) ([]uint, error) {
	var userIDs []uint
	err := tenantdb.System(db).Table("environments AS e").
		Distinct("e.user_id").
		Joins("JOIN companies AS c ON c.id = e.company_id AND c.deleted_at IS NULL").
		Joins("JOIN role_permissions AS rp ON rp.role_id = e.role_id").
		Where("c.tenant_id = ? AND e.deleted_at IS NULL AND rp.permission_id = ?", tenantID, permissionID).
		Pluck("e.user_id", &userIDs).Error
	return userIDs, err
}
