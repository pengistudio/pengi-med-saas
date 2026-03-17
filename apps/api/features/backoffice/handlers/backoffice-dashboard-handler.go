package backoffice_handlers

import (
	"net/http"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	company_models "pengi-med-saas/features/companies/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeDashboardHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeDashboardHandler(db *gorm.DB, logger *zap.Logger) *BackofficeDashboardHandler {
	return &BackofficeDashboardHandler{db: db, logger: logger}
}

type ExpiringSubscription struct {
	ID          uint      `json:"id"`
	CompanyName string    `json:"company_name"`
	PlanCode    string    `json:"plan_code"`
	ExpiresAt   time.Time `json:"expires_at"`
	DaysLeft    int       `json:"days_left"`
}

type DashboardStats struct {
	TotalCompanies         int64                    `json:"total_companies"`
	TotalUsers             int64                    `json:"total_users"`
	TotalPlans             int64                    `json:"total_plans"`
	TotalFeatures          int64                    `json:"total_features"`
	ActiveSubscriptions    int64                    `json:"active_subscriptions"`
	RecentCompanies        []company_models.Company `json:"recent_companies"`
	ExpiringSubscriptions  []ExpiringSubscription   `json:"expiring_subscriptions"`
}

func (h *BackofficeDashboardHandler) GetDashboardStats(c *gin.Context) envelope.Response {
	var stats DashboardStats

	if err := h.db.Model(&company_models.Company{}).Count(&stats.TotalCompanies).Error; err != nil {
		h.logger.Error("Failed to count companies", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining stats", core_errors.ErrInternal)
	}

	if err := h.db.Model(&backoffice_models.BackofficeUser{}).Count(&stats.TotalUsers).Error; err != nil {
		h.logger.Error("Failed to count users", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining stats", core_errors.ErrInternal)
	}

	if err := h.db.Model(&company_models.Plan{}).Count(&stats.TotalPlans).Error; err != nil {
		h.logger.Error("Failed to count plans", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining stats", core_errors.ErrInternal)
	}

	if err := h.db.Model(&company_models.Feature{}).Count(&stats.TotalFeatures).Error; err != nil {
		h.logger.Error("Failed to count features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining stats", core_errors.ErrInternal)
	}

	if err := h.db.Model(&company_models.Subscription{}).Where("status = ?", "active").Count(&stats.ActiveSubscriptions).Error; err != nil {
		h.logger.Error("Failed to count active subscriptions", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining stats", core_errors.ErrInternal)
	}

	// Recent 5 companies with active subscription
	h.db.Preload("Tenant").Preload("Subscriptions", "status = ? AND expires_at > NOW()", "active").Order("created_at desc").Limit(5).Find(&stats.RecentCompanies)

	// Subscriptions expiring in the next 30 days
	now := time.Now()
	in30Days := now.AddDate(0, 0, 30)
	var expiringSubs []company_models.Subscription
	h.db.Preload("Company").
		Where("status = ? AND expires_at >= ? AND expires_at <= ?", "active", now, in30Days).
		Order("expires_at ASC").
		Find(&expiringSubs)

	stats.ExpiringSubscriptions = make([]ExpiringSubscription, 0, len(expiringSubs))
	for _, s := range expiringSubs {
		daysLeft := int(time.Until(s.ExpiresAt).Hours() / 24)
		stats.ExpiringSubscriptions = append(stats.ExpiringSubscriptions, ExpiringSubscription{
			ID:          s.ID,
			CompanyName: s.Company.TradeName,
			PlanCode:    s.PlanCode,
			ExpiresAt:   s.ExpiresAt,
			DaysLeft:    daysLeft,
		})
	}

	return envelope.SuccessResponse(stats, "backoffice.dashboard.stats.success")
}
