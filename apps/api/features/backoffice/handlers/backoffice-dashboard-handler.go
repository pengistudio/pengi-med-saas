package backoffice_handlers

import (
	"net/http"
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

type DashboardStats struct {
	TotalCompanies      int64                    `json:"total_companies"`
	TotalUsers          int64                    `json:"total_users"`
	TotalPlans          int64                    `json:"total_plans"`
	TotalFeatures       int64                    `json:"total_features"`
	ActiveSubscriptions int64                    `json:"active_subscriptions"`
	RecentCompanies     []company_models.Company `json:"recent_companies"`
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

	// Recent 5 companies
	h.db.Preload("Tenant").Order("created_at desc").Limit(5).Find(&stats.RecentCompanies)

	return envelope.SuccessResponse(stats, "backoffice.dashboard.stats.success")
}
