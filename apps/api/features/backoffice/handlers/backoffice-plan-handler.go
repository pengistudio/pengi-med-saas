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

type PricingInput struct {
	Months int     `json:"months" binding:"required,oneof=1 3 6 9 12"`
	Price  float64 `json:"price" binding:"required,gt=0"`
}

type CreatePlanRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Code         string                 `json:"code" binding:"required"`
	Tier         int                    `json:"tier" binding:"required,min=0,max=10"`
	CanRenew     *bool                  `json:"can_renew"`
	Properties   map[string]interface{} `json:"properties"`
	FeatureCodes []string               `json:"feature_codes"`
	Pricings     []PricingInput         `json:"pricings"`
}

type UpdatePlanRequest struct {
	Name         string                 `json:"name"`
	Tier         int                    `json:"tier"`
	CanRenew     *bool                  `json:"can_renew"`
	Properties   map[string]interface{} `json:"properties"`
	FeatureCodes []string               `json:"feature_codes"`
	Pricings     []PricingInput         `json:"pricings"`
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

	var features []company_models.Feature
	if err := h.db.Preload("Permissions").Where("code IN ?", featureCodes).Find(&features).Error; err != nil {
		return nil, err
	}

	categoriesFound := make(map[string]bool)
	for _, feature := range features {
		for _, perm := range feature.Permissions {
			categoriesFound[perm.Category] = true
		}
	}

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

// basePriceFromPricings returns the monthly price from pricings, or 0 if not found.
func basePriceFromPricings(pricings []PricingInput) float64 {
	for _, p := range pricings {
		if p.Months == 1 {
			return p.Price
		}
	}
	// Fall back to the cheapest per-month rate
	var best float64
	for _, p := range pricings {
		perMonth := p.Price / float64(p.Months)
		if best == 0 || perMonth < best {
			best = perMonth
		}
	}
	return best
}

func (h *BackofficePlanHandler) syncPricings(planID uint, pricings []PricingInput) {
	if len(pricings) == 0 {
		return
	}
	// Hard-delete existing pricings to avoid unique index conflicts on re-create
	if err := h.db.Unscoped().Where("plan_id = ?", planID).Delete(&company_models.PlanPricing{}).Error; err != nil {
		h.logger.Error("Failed to delete plan pricings", zap.Uint("plan_id", planID), zap.Error(err))
		return
	}
	for _, p := range pricings {
		if err := h.db.Create(&company_models.PlanPricing{
			PlanID: planID,
			Months: p.Months,
			Price:  p.Price,
		}).Error; err != nil {
			h.logger.Error("Failed to create plan pricing", zap.Uint("plan_id", planID), zap.Int("months", p.Months), zap.Error(err))
		}
	}
}

// ── Handlers ────────────────────────────────────────────────────────────────

func (h *BackofficePlanHandler) GetPlans(c *gin.Context) envelope.Response {
	var plans []company_models.Plan
	if err := h.db.Preload("Features").Preload("Pricings").Find(&plans).Error; err != nil {
		h.logger.Error("Failed to fetch plans", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining plans", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(plans, "backoffice.plan.list.success")
}

func (h *BackofficePlanHandler) GetPlanByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var plan company_models.Plan
	if err := h.db.Preload("Features.Permissions").Preload("Pricings").First(&plan, id).Error; err != nil {
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

	enabledFeatures, err := h.calculateEnabledFeatures(req.FeatureCodes)
	if err != nil {
		h.logger.Error("Failed to calculate enabled features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating plan", core_errors.ErrInternal)
	}

	if req.Properties == nil {
		req.Properties = make(map[string]interface{})
	}
	req.Properties["enabled_features"] = enabledFeatures

	canRenew := true
	if req.CanRenew != nil {
		canRenew = *req.CanRenew
	}
	plan := company_models.Plan{
		Name:       req.Name,
		Code:       req.Code,
		Tier:       req.Tier,
		CanRenew:   canRenew,
		Price:      basePriceFromPricings(req.Pricings),
		Properties: req.Properties,
	}

	if err := h.db.Create(&plan).Error; err != nil {
		h.logger.Error("Failed to create plan", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating plan", core_errors.ErrInternal)
	}

	if len(req.FeatureCodes) > 0 {
		var features []company_models.Feature
		h.db.Where("code IN ?", req.FeatureCodes).Find(&features)
		if err := h.db.Model(&plan).Association("Features").Replace(features); err != nil {
			h.logger.Error("Failed to attach features", zap.Error(err))
		}
	}

	h.syncPricings(plan.ID, req.Pricings)

	h.db.Preload("Features").Preload("Pricings").First(&plan, plan.ID)
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
	if req.Tier > 0 {
		updates["tier"] = req.Tier
	}
	if req.CanRenew != nil {
		updates["can_renew"] = *req.CanRenew
	}
	if req.Pricings != nil {
		updates["price"] = basePriceFromPricings(req.Pricings)
	}

	if req.FeatureCodes != nil {
		enabledFeatures, err := h.calculateEnabledFeatures(req.FeatureCodes)
		if err != nil {
			h.logger.Error("Failed to calculate enabled features", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating plan", core_errors.ErrInternal)
		}
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

	if req.Pricings != nil {
		h.syncPricings(plan.ID, req.Pricings)
	}

	h.db.Preload("Features").Preload("Pricings").First(&plan, plan.ID)
	return envelope.SuccessResponse(plan, "backoffice.plan.update.success")
}

func (h *BackofficePlanHandler) DeletePlan(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var plan company_models.Plan
	if err := h.db.First(&plan, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Plan not found", core_errors.ErrBackofficePlanNotFound)
	}

	h.db.Model(&plan).Association("Features").Clear()
	h.db.Unscoped().Where("plan_id = ?", plan.ID).Delete(&company_models.PlanPricing{})

	if err := h.db.Delete(&plan).Error; err != nil {
		h.logger.Error("Failed to delete plan", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting plan", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "backoffice.plan.delete.success")
}
