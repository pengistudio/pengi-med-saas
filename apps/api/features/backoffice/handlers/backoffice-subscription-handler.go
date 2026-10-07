package backoffice_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeSubscriptionHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeSubscriptionHandler(db *gorm.DB, logger *zap.Logger) *BackofficeSubscriptionHandler {
	return &BackofficeSubscriptionHandler{db: tenantdb.System(db), logger: logger}
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(subscriptions, "backoffice.subscription.list.success")
}

func (h *BackofficeSubscriptionHandler) GetSubscriptionByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var subscription company_models.Subscription
	if err := h.db.Preload("Plan").Preload("Company").First(&subscription, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}
	return envelope.SuccessResponse(subscription, "backoffice.subscription.found")
}

func (h *BackofficeSubscriptionHandler) CreateSubscription(c *gin.Context) envelope.Response {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}

	expiresAt, err := company_models.ParseExpiry(req.ExpiresAt)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.subscription.invalid_date", core_errors.ErrBackofficeInvalidRequest)
	}

	subscription := company_models.Subscription{
		CompanyID: req.CompanyID,
		PlanCode:  req.PlanCode,
		Status:    req.Status,
		ExpiresAt: expiresAt,
	}

	if err := h.db.Create(&subscription).Error; err != nil {
		h.logger.Error("Failed to create subscription", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	h.db.Preload("Plan").First(&subscription, subscription.ID)
	h.logger.Info("Subscription created", zap.Uint("company_id", req.CompanyID), zap.String("plan_code", req.PlanCode))
	return envelope.New(http.StatusCreated, "backoffice.subscription.create.success", subscription)
}

func (h *BackofficeSubscriptionHandler) UpdateSubscription(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var subscription company_models.Subscription
	if err := h.db.First(&subscription, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.PlanCode != "" {
		updates["plan_code"] = req.PlanCode
	}
	if req.ExpiresAt != "" {
		expiresAt, err := company_models.ParseExpiry(req.ExpiresAt)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.subscription.invalid_date", core_errors.ErrBackofficeInvalidRequest)
		}
		updates["expires_at"] = expiresAt
	}

	if len(updates) > 0 {
		h.db.Model(&subscription).Updates(updates)
	}

	h.db.Preload("Plan").First(&subscription, subscription.ID)
	return envelope.SuccessResponse(subscription, "backoffice.subscription.update.success")
}

func (h *BackofficeSubscriptionHandler) DeleteSubscription(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var subscription company_models.Subscription
	if err := h.db.First(&subscription, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	if err := h.db.Delete(&subscription).Error; err != nil {
		h.logger.Error("Failed to delete subscription", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "backoffice.subscription.delete.success")
}
