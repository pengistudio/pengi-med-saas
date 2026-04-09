package backoffice_handlers

import (
	"encoding/json"
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeSubscriptionHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeSubscriptionHandler(db *gorm.DB, logger *zap.Logger) *BackofficeSubscriptionHandler {
	return &BackofficeSubscriptionHandler{db: db, logger: logger}
}

// applyPlanFeaturedToTenant applies enabled features from the plan to the tenant
func (h *BackofficeSubscriptionHandler) applyPlanFeaturesToTenant(subscription *company_models.Subscription) error {
	// Fetch the plan with full data
	var plan company_models.Plan
	if err := h.db.Where("code = ?", subscription.PlanCode).First(&plan).Error; err != nil {
		return err
	}

	// Fetch the company and tenant
	var company company_models.Company
	if err := h.db.First(&company, subscription.CompanyID).Error; err != nil {
		return err
	}

	var tenant tenant_models.Tenant
	if err := h.db.First(&tenant, company.TenantID).Error; err != nil {
		return err
	}

	// Extract enabled_features from plan properties
	enabledFeatures := tenant_models.DefaultEnabledFeatures()
	if properties := plan.Properties; properties != nil {
		if featuresData, ok := properties["enabled_features"]; ok {
			if featuresJSON, err := json.Marshal(featuresData); err == nil {
				json.Unmarshal(featuresJSON, &enabledFeatures)
			}
		}
	}

	// Update tenant with new enabled features
	featuresJSON, err := json.Marshal(enabledFeatures)
	if err != nil {
		return err
	}

	if err := h.db.Model(&tenant).Update("enabled_features", string(featuresJSON)).Error; err != nil {
		return err
	}

	h.logger.Info("Applied plan features to tenant",
		zap.Uint("company_id", subscription.CompanyID),
		zap.Uint("tenant_id", tenant.ID),
		zap.String("plan_code", subscription.PlanCode))

	return nil
}

// ── DTOs ────────────────────────────────────────────────────────────────────

type CreateSubscriptionRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	PlanCode  string `json:"plan_code" binding:"required"`
	Status    string `json:"status" binding:"required"`
	ExpiresAt string `json:"expires_at" binding:"required"`
}

type UpdateSubscriptionRequest struct {
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	PlanCode  string `json:"plan_code"`
}

// ── Handlers ────────────────────────────────────────────────────────────────

func (h *BackofficeSubscriptionHandler) GetSubscriptions(c *gin.Context) envelope.Response {
	var subscriptions []company_models.Subscription
	if err := h.db.Preload("Plan").Preload("Company").Find(&subscriptions).Error; err != nil {
		h.logger.Error("Failed to fetch subscriptions", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining subscriptions", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(subscriptions, "backoffice.subscription.list.success")
}

func (h *BackofficeSubscriptionHandler) GetSubscriptionsByCompany(c *gin.Context) envelope.Response {
	companyID := c.Param("id")
	var subscriptions []company_models.Subscription
	if err := h.db.Preload("Plan").Where("company_id = ?", companyID).Find(&subscriptions).Error; err != nil {
		h.logger.Error("Failed to fetch subscriptions for company", zap.String("company_id", companyID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining subscriptions", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(subscriptions, "backoffice.subscription.list.success")
}

func (h *BackofficeSubscriptionHandler) CreateSubscription(c *gin.Context) envelope.Response {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid date format, use RFC3339", core_errors.ErrBackofficeInvalidRequest)
	}

	subscription := company_models.Subscription{
		CompanyID: req.CompanyID,
		PlanCode:  req.PlanCode,
		Status:    req.Status,
		ExpiresAt: expiresAt,
	}

	if err := h.db.Create(&subscription).Error; err != nil {
		h.logger.Error("Failed to create subscription", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating subscription", core_errors.ErrInternal)
	}

	// Apply plan features to tenant
	if err := h.applyPlanFeaturesToTenant(&subscription); err != nil {
		h.logger.Warn("Failed to apply plan features to tenant", zap.Error(err))
	}

	h.db.Preload("Plan").First(&subscription, subscription.ID)
	h.logger.Info("Subscription created", zap.Uint("company_id", req.CompanyID), zap.String("plan_code", req.PlanCode))
	return envelope.New(http.StatusCreated, "backoffice.subscription.create.success", subscription)
}

func (h *BackofficeSubscriptionHandler) UpdateSubscription(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var subscription company_models.Subscription
	if err := h.db.First(&subscription, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Subscription not found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.PlanCode != "" {
		updates["plan_code"] = req.PlanCode
	}
	if req.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "Invalid date format", core_errors.ErrBackofficeInvalidRequest)
		}
		updates["expires_at"] = expiresAt
	}

	if len(updates) > 0 {
		h.db.Model(&subscription).Updates(updates)
	}

	// If plan changed, apply new plan features to tenant
	if req.PlanCode != "" {
		if err := h.applyPlanFeaturesToTenant(&subscription); err != nil {
			h.logger.Warn("Failed to apply plan features to tenant", zap.Error(err))
		}
	}

	h.db.Preload("Plan").First(&subscription, subscription.ID)
	return envelope.SuccessResponse(subscription, "backoffice.subscription.update.success")
}

func (h *BackofficeSubscriptionHandler) DeleteSubscription(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var subscription company_models.Subscription
	if err := h.db.First(&subscription, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Subscription not found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	if err := h.db.Delete(&subscription).Error; err != nil {
		h.logger.Error("Failed to delete subscription", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting subscription", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "backoffice.subscription.delete.success")
}
