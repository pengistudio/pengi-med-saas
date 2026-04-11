package backoffice_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficePlanHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficePlanHandler(db *gorm.DB, logger *zap.Logger) *BackofficePlanHandler {
	return &BackofficePlanHandler{db: db, logger: logger}
}

// ── DTOs ────────────────────────────────────────────────────────────────────

type CreatePlanRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Code         string                 `json:"code" binding:"required"`
	Price        float64                `json:"price" binding:"required"`
	Properties   map[string]interface{} `json:"properties"`
	FeatureCodes []string               `json:"feature_codes"`
}

type UpdatePlanRequest struct {
	Name         string                 `json:"name"`
	Price        *float64               `json:"price"`
	Properties   map[string]interface{} `json:"properties"`
	FeatureCodes []string               `json:"feature_codes"`
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func (h *BackofficePlanHandler) calculateEnabledFeatures(featureCodes []string) (map[string]interface{}, error) {
	enabledFeatures := tenant_models.EnabledFeatures{
		Clinical: false,
		Billing:  false,
		Team:     false,
		Kanban:   false,
	}

	if len(featureCodes) == 0 {
		return map[string]interface{}{
			"clinical": false,
			"billing":  false,
			"team":     false,
			"kanban":   false,
		}, nil
	}

	// Get features and their permissions
	var features []company_models.Feature
	if err := h.db.Preload("Permissions").Where("code IN ?", featureCodes).Find(&features).Error; err != nil {
		return nil, err
	}

	// Track which categories have at least one permission
	categoriesFound := make(map[string]bool)

	for _, feature := range features {
		for _, perm := range feature.Permissions {
			categoriesFound[perm.Category] = true
		}
	}

	// Set enabled features based on found categories
	if categoriesFound["CLINICAL"] {
		enabledFeatures.Clinical = true
	}
	if categoriesFound["BILLING"] {
		enabledFeatures.Billing = true
	}
	if categoriesFound["TEAM"] {
		enabledFeatures.Team = true
	}
	if categoriesFound["KANBAN"] {
		enabledFeatures.Kanban = true
	}

	return map[string]interface{}{
		"clinical": enabledFeatures.Clinical,
		"billing":  enabledFeatures.Billing,
		"team":     enabledFeatures.Team,
		"kanban":   enabledFeatures.Kanban,
	}, nil
}

// ── Handlers ────────────────────────────────────────────────────────────────

func (h *BackofficePlanHandler) GetPlans(c *gin.Context) envelope.Response {
	var plans []company_models.Plan
	if err := h.db.Preload("Features").Find(&plans).Error; err != nil {
		h.logger.Error("Failed to fetch plans", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining plans", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(plans, "backoffice.plan.list.success")
}

func (h *BackofficePlanHandler) GetPlanByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var plan company_models.Plan
	if err := h.db.Preload("Features.Permissions").First(&plan, id).Error; err != nil {
		h.logger.Error("Plan not found", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Plan not found", core_errors.ErrBackofficePlanNotFound)
	}
	return envelope.SuccessResponse(plan, "backoffice.plan.found")
}

func (h *BackofficePlanHandler) CreatePlan(c *gin.Context) envelope.Response {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	// Calculate enabled features based on selected features
	enabledFeatures, err := h.calculateEnabledFeatures(req.FeatureCodes)
	if err != nil {
		h.logger.Error("Failed to calculate enabled features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating plan", core_errors.ErrInternal)
	}

	// Merge enabled_features into properties
	if req.Properties == nil {
		req.Properties = make(map[string]interface{})
	}
	req.Properties["enabled_features"] = enabledFeatures

	plan := company_models.Plan{
		Name:       req.Name,
		Code:       req.Code,
		Price:      req.Price,
		Properties: req.Properties,
	}

	if err := h.db.Create(&plan).Error; err != nil {
		h.logger.Error("Failed to create plan", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating plan", core_errors.ErrInternal)
	}

	// Attach features by code
	if len(req.FeatureCodes) > 0 {
		var features []company_models.Feature
		h.db.Where("code IN ?", req.FeatureCodes).Find(&features)
		if err := h.db.Model(&plan).Association("Features").Replace(features); err != nil {
			h.logger.Error("Failed to attach features", zap.Error(err))
		}
	}

	h.db.Preload("Features").First(&plan, plan.ID)
	h.logger.Info("Plan created", zap.String("code", req.Code))
	return envelope.New(http.StatusCreated, "backoffice.plan.create.success", plan)
}

func (h *BackofficePlanHandler) UpdatePlan(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var plan company_models.Plan
	if err := h.db.First(&plan, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Plan not found", core_errors.ErrBackofficePlanNotFound)
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}

	// Handle features update with automatic enabled_features calculation
	if req.FeatureCodes != nil {
		enabledFeatures, err := h.calculateEnabledFeatures(req.FeatureCodes)
		if err != nil {
			h.logger.Error("Failed to calculate enabled features", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating plan", core_errors.ErrInternal)
		}

		// Merge properties with enabled_features
		if req.Properties == nil {
			req.Properties = make(map[string]interface{})
		}
		req.Properties["enabled_features"] = enabledFeatures
	}

	if req.Properties != nil {
		updates["properties"] = req.Properties
	}

	if len(updates) > 0 {
		h.db.Model(&plan).Updates(updates)
	}

	if req.FeatureCodes != nil {
		var features []company_models.Feature
		h.db.Where("code IN ?", req.FeatureCodes).Find(&features)
		if err := h.db.Model(&plan).Association("Features").Replace(features); err != nil {
			h.logger.Error("Failed to update features", zap.Error(err))
		}
	}

	h.db.Preload("Features").First(&plan, plan.ID)
	return envelope.SuccessResponse(plan, "backoffice.plan.update.success")
}

func (h *BackofficePlanHandler) DeletePlan(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var plan company_models.Plan
	if err := h.db.First(&plan, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Plan not found", core_errors.ErrBackofficePlanNotFound)
	}

	h.db.Model(&plan).Association("Features").Clear()

	if err := h.db.Delete(&plan).Error; err != nil {
		h.logger.Error("Failed to delete plan", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting plan", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "backoffice.plan.delete.success")
}
