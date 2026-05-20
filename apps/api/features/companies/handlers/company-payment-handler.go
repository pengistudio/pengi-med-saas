package company_handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/utils"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CompanyPaymentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	dlocal *utils.DlocalClient
}

func NewCompanyPaymentHandler(db *gorm.DB, logger *zap.Logger) *CompanyPaymentHandler {
	return &CompanyPaymentHandler{
		db:     db,
		logger: logger,
		dlocal: utils.NewDlocalClient(),
	}
}

// ── DTOs ────────────────────────────────────────────────────────────────────

type PaySubscriptionRequest struct {
	PlanCode string `json:"plan_code"`
	Months   int    `json:"months"`
}

type PaySubscriptionResponse struct {
	CheckoutURL string  `json:"checkout_url"`
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Free        bool    `json:"free"`
	SwitchType  string  `json:"switch_type"` // "paid" | "immediate" | "deferred"
}

type PricingOptionResponse struct {
	Months int     `json:"months"`
	Price  float64 `json:"price"`
}

type PlanOptionResponse struct {
	Code            string                 `json:"code"`
	Name            string                 `json:"name"`
	Tier            int                    `json:"tier"`
	Price           float64                `json:"price"`
	EnabledFeatures map[string]interface{} `json:"enabled_features"`
	Pricings        []PricingOptionResponse `json:"pricings"`
}

type SubscriptionDetailResponse struct {
	PlanName          string     `json:"plan_name"`
	PlanCode          string     `json:"plan_code"`
	PlanTier          int        `json:"plan_tier"`
	Status            string     `json:"status"`
	ExpiresAt         time.Time  `json:"expires_at"`
	DaysLeft          int        `json:"days_left"`
	Amount            float64    `json:"amount"`
	LastPaymentAmount float64    `json:"last_payment_amount"`
	LastPaymentMonths int        `json:"last_payment_months"`
	NextPlanCode      string     `json:"next_plan_code"`
	PlanChangeAt      *time.Time `json:"plan_change_at"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// applyPlanFeatures syncs the tenant's enabled_features from the given plan code.
// Package-level so DashboardHandler can reuse it.
func applyPlanFeatures(db *gorm.DB, logger *zap.Logger, companyID uint, newPlanCode string) error {
	var plan company_models.Plan
	if err := db.Where("code = ?", newPlanCode).First(&plan).Error; err != nil {
		return err
	}

	var company company_models.Company
	if err := db.First(&company, companyID).Error; err != nil {
		return err
	}

	var tenant tenant_models.Tenant
	if err := db.First(&tenant, company.TenantID).Error; err != nil {
		return err
	}

	enabledFeatures := tenant_models.DefaultEnabledFeatures()
	if plan.Properties != nil {
		if featuresData, ok := plan.Properties["enabled_features"]; ok {
			if featuresJSON, err := json.Marshal(featuresData); err == nil {
				json.Unmarshal(featuresJSON, &enabledFeatures)
			}
		}
	}

	featuresJSON, err := json.Marshal(enabledFeatures)
	if err != nil {
		return err
	}

	if err := db.Model(&tenant).Update("enabled_features", string(featuresJSON)).Error; err != nil {
		return err
	}

	logger.Info("Applied plan features to tenant",
		zap.Uint("company_id", companyID),
		zap.Uint("tenant_id", tenant.ID),
		zap.String("plan_code", newPlanCode))

	return nil
}

func (h *CompanyPaymentHandler) applyPlanFeaturesToTenant(subscription *company_models.Subscription, newPlanCode string) error {
	return applyPlanFeatures(h.db, h.logger, subscription.CompanyID, newPlanCode)
}

// checkAndApplyPendingPlanChange applies a deferred plan change if its scheduled date has passed.
// Returns true if a change was applied (caller should reload sub.Plan).
func checkAndApplyPendingPlanChange(db *gorm.DB, logger *zap.Logger, sub *company_models.Subscription, company *company_models.Company) bool {
	if sub.NextPlanCode == "" || sub.PlanChangeAt == nil {
		return false
	}
	if time.Now().Before(*sub.PlanChangeAt) {
		return false
	}
	newCode := sub.NextPlanCode
	sub.PlanCode = newCode
	sub.NextPlanCode = ""
	sub.PlanChangeAt = nil
	db.Model(company).Update("plan_code", newCode)
	if err := applyPlanFeatures(db, logger, sub.CompanyID, newCode); err != nil {
		logger.Error("Failed to apply deferred plan features", zap.Error(err))
	}
	db.Save(sub)
	logger.Info("Applied deferred plan change", zap.Uint("subscription_id", sub.ID), zap.String("plan_code", newCode))
	return true
}

// resolvePriceAndMonths returns the amount and billing months for a plan+months combo.
// Falls back to plan.Price * months if no specific pricing entry exists.
func resolvePriceAndMonths(plan company_models.Plan, months int) (float64, int) {
	if months <= 0 {
		months = 1
	}
	for _, p := range plan.Pricings {
		if p.Months == months {
			return p.Price, months
		}
	}
	// No specific pricing entry — multiply base monthly price
	return plan.Price * float64(months), months
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *CompanyPaymentHandler) GetAvailablePlans(c *gin.Context) envelope.Response {
	var plans []company_models.Plan
	if err := h.db.Preload("Pricings").Where("can_renew = ?", true).Find(&plans).Error; err != nil {
		h.logger.Error("failed to fetch plans", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "subscription.plans.fetch.error", core_errors.ErrInternal)
	}

	result := make([]PlanOptionResponse, 0, len(plans))
	for _, p := range plans {
		var ef map[string]interface{}
		if p.Properties != nil {
			ef, _ = p.Properties["enabled_features"].(map[string]interface{})
		}
		pricings := make([]PricingOptionResponse, 0, len(p.Pricings))
		for _, pr := range p.Pricings {
			pricings = append(pricings, PricingOptionResponse{Months: pr.Months, Price: pr.Price})
		}
		result = append(result, PlanOptionResponse{
			Code:            p.Code,
			Name:            p.Name,
			Tier:            p.Tier,
			Price:           p.Price,
			EnabledFeatures: ef,
			Pricings:        pricings,
		})
	}

	return envelope.SuccessResponse(result, "subscription.plans.fetch.success")
}

func (h *CompanyPaymentHandler) PaySubscription(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
	}

	var sub company_models.Subscription
	if err := h.db.Preload("Plan.Pricings").Where("company_id = ?", company.ID).Order("expires_at DESC").First(&sub).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.subscription.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	var req PaySubscriptionRequest
	_ = c.ShouldBindJSON(&req)

	isPlanChange := req.PlanCode != "" && req.PlanCode != sub.PlanCode

	targetPlan := sub.Plan
	if isPlanChange {
		var newPlan company_models.Plan
		if err := h.db.Preload("Pricings").Where("code = ?", req.PlanCode).First(&newPlan).Error; err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "subscription.plan.not_found", core_errors.ErrBackofficePlanNotFound)
		}
		targetPlan = newPlan
	}

	// ── Plan switching logic (price-based) ───────────────────────────────────
	if isPlanChange {
		isActive := sub.ExpiresAt.After(time.Now())

		if targetPlan.Price == 0 {
			// Free plan — immediate switch at no cost
			sub.PlanCode = targetPlan.Code
			sub.NextPlanCode = ""
			sub.PlanChangeAt = nil
			h.db.Model(&company).Update("plan_code", targetPlan.Code)
			if err := h.applyPlanFeaturesToTenant(&sub, targetPlan.Code); err != nil {
				h.logger.Error("Failed to apply plan features on free plan switch", zap.Error(err))
			}
			h.db.Save(&sub)
			h.logger.Info("Free plan switch applied immediately",
				zap.Uint("subscription_id", sub.ID),
				zap.String("to", targetPlan.Code))
			return envelope.SuccessResponse(PaySubscriptionResponse{
				Free:       true,
				SwitchType: "immediate",
			}, "company.payment.switch.free")
		}

		if targetPlan.Price < sub.Plan.Price && isActive && req.Months <= 0 {
			// Cheaper plan while active and no explicit months — deferred to period end
			planChangeAt := sub.ExpiresAt
			sub.NextPlanCode = targetPlan.Code
			sub.PlanChangeAt = &planChangeAt
			h.db.Save(&sub)
			h.logger.Info("Plan downgrade deferred",
				zap.Uint("subscription_id", sub.ID),
				zap.String("from", sub.PlanCode),
				zap.String("to", targetPlan.Code))
			return envelope.SuccessResponse(PaySubscriptionResponse{
				Free:       true,
				SwitchType: "deferred",
			}, "company.payment.switch.deferred")
		}

		if targetPlan.Price > sub.Plan.Price && sub.Plan.Price > 0 {
			// More expensive plan — charge pro-rated difference (only when current plan is also paid)
			daysLeft := time.Until(sub.ExpiresAt).Hours() / 24
			if daysLeft < 0 {
				daysLeft = 0
			}
			prorationAmount := math.Round((targetPlan.Price-sub.Plan.Price)*daysLeft/30.0*100) / 100
			if prorationAmount < 1.00 {
				// Proration below $1 — treat as free immediate switch
				sub.PlanCode = targetPlan.Code
				sub.NextPlanCode = ""
				sub.PlanChangeAt = nil
				h.db.Model(&company).Update("plan_code", targetPlan.Code)
				if err := h.applyPlanFeaturesToTenant(&sub, targetPlan.Code); err != nil {
					h.logger.Error("Failed to apply plan features on upgrade", zap.Error(err))
				}
				h.db.Save(&sub)
				return envelope.SuccessResponse(PaySubscriptionResponse{
					Free:       true,
					SwitchType: "immediate",
				}, "company.payment.switch.free")
			}

			// Create dlocal payment for proration only (months=0 signals upgrade)
			orderID := uuid.New().String()
			dlocalResp, err := h.dlocal.CreatePayment(utils.DlocalCreatePaymentRequest{
				Amount:          prorationAmount,
				Currency:        "USD",
				OrderID:         orderID,
				Description:     fmt.Sprintf("Upgrade a %s - prorrateo", targetPlan.Name),
				NotificationURL: os.Getenv("DLOCAL_NOTIFICATION_URL"),
				SuccessURL:      os.Getenv("FRONTEND_URL") + "/subscription?status=success",
			})
			if err != nil {
				h.logger.Error("Failed to create dlocal upgrade payment", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "company.payment.create.error", core_errors.ErrBackofficePaymentCreateFailed)
			}

			h.db.Model(&company_models.SubscriptionPayment{}).
				Where("company_id = ? AND status = ?", company.ID, "pending").
				Update("status", "cancelled")

			payment := company_models.SubscriptionPayment{
				CompanyID:       company.ID,
				SubscriptionID:  sub.ID,
				DlocalPaymentID: dlocalResp.ID,
				OrderID:         orderID,
				Amount:          prorationAmount,
				Currency:        "USD",
				Status:          "pending",
				CheckoutURL:     dlocalResp.URL(),
				TargetPlanCode:  targetPlan.Code,
				Months:          0, // 0 = upgrade proration, no period extension
			}
			if err := h.db.Create(&payment).Error; err != nil {
				h.logger.Error("Failed to save upgrade payment", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "company.payment.create.error", core_errors.ErrBackofficePaymentCreateFailed)
			}

			return envelope.SuccessResponse(PaySubscriptionResponse{
				CheckoutURL: dlocalResp.URL(),
				OrderID:     orderID,
				Amount:      prorationAmount,
				Free:        false,
				SwitchType:  "paid",
			}, "company.payment.initiate.success")
		}
		// Same-price plan or downgrade-now (months > 0): falls through to paid renewal below
	}

	// ── Same-plan renewal ──────────────────────────────────────────────────────
	amount, months := resolvePriceAndMonths(targetPlan, req.Months)

	orderID := uuid.New().String()
	dlocalResp, err := h.dlocal.CreatePayment(utils.DlocalCreatePaymentRequest{
		Amount:          amount,
		Currency:        "USD",
		OrderID:         orderID,
		Description:     fmt.Sprintf("Suscripción Pengi - %s (%d mes(es))", targetPlan.Name, months),
		NotificationURL: os.Getenv("DLOCAL_NOTIFICATION_URL"),
		SuccessURL:      os.Getenv("FRONTEND_URL") + "/subscription?status=success",
	})
	if err != nil {
		h.logger.Error("Failed to create dlocal payment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "company.payment.create.error", core_errors.ErrBackofficePaymentCreateFailed)
	}

	h.db.Model(&company_models.SubscriptionPayment{}).
		Where("company_id = ? AND status = ?", company.ID, "pending").
		Update("status", "cancelled")

	payment := company_models.SubscriptionPayment{
		CompanyID:       company.ID,
		SubscriptionID:  sub.ID,
		DlocalPaymentID: dlocalResp.ID,
		OrderID:         orderID,
		Amount:          amount,
		Currency:        "USD",
		Status:          "pending",
		CheckoutURL:     dlocalResp.URL(),
		TargetPlanCode:  targetPlan.Code,
		Months:          months,
	}
	if err := h.db.Create(&payment).Error; err != nil {
		h.logger.Error("Failed to save subscription payment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "company.payment.create.error", core_errors.ErrBackofficePaymentCreateFailed)
	}

	return envelope.SuccessResponse(PaySubscriptionResponse{
		CheckoutURL: dlocalResp.URL(),
		OrderID:     orderID,
		Amount:      amount,
		Free:        false,
		SwitchType:  "paid",
	}, "company.payment.initiate.success")
}

func (h *CompanyPaymentHandler) ConfirmPayment(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
	}

	var payment company_models.SubscriptionPayment
	if err := h.db.Where("company_id = ? AND status = ?", company.ID, "pending").
		Order("created_at DESC").First(&payment).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.payment.pending.not_found", core_errors.ErrBackofficePaymentNotFound)
	}

	payment.Status = "paid"
	if err := h.db.Save(&payment).Error; err != nil {
		h.logger.Error("Failed to confirm payment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "company.payment.confirm.error", core_errors.ErrBackofficePaymentCreateFailed)
	}

	var sub company_models.Subscription
	if err := h.db.First(&sub, payment.SubscriptionID).Error; err == nil {
		isUpgrade := payment.Months == 0 && payment.TargetPlanCode != "" && payment.TargetPlanCode != sub.PlanCode

		if isUpgrade {
			// Upgrade proration paid — apply plan immediately, no period extension
			sub.PlanCode = payment.TargetPlanCode
			sub.Status = "active"
			sub.NextPlanCode = ""
			sub.PlanChangeAt = nil
			h.db.Model(&company).Update("plan_code", payment.TargetPlanCode)
			if err := h.applyPlanFeaturesToTenant(&sub, payment.TargetPlanCode); err != nil {
				h.logger.Error("Failed to apply plan features on upgrade confirm", zap.Error(err))
			}
			h.db.Save(&sub)
			h.logger.Info("Upgrade confirmed, plan applied immediately",
				zap.Uint("subscription_id", sub.ID),
				zap.String("plan_code", payment.TargetPlanCode))
		} else {
			months := payment.Months
			if months <= 0 {
				months = 1
			}

			isActivePlanChange := payment.TargetPlanCode != "" &&
				payment.TargetPlanCode != sub.PlanCode &&
				sub.ExpiresAt.After(time.Now())

			if isActivePlanChange {
				// Active subscription with plan change — defer features to period end
				planChangeAt := sub.ExpiresAt
				sub.ExpiresAt = planChangeAt.AddDate(0, months, 0)
				sub.NextPlanCode = payment.TargetPlanCode
				sub.PlanChangeAt = &planChangeAt
			} else {
				sub.ExpiresAt = sub.ExpiresAt.AddDate(0, months, 0)
				if payment.TargetPlanCode != "" && payment.TargetPlanCode != sub.PlanCode {
					sub.PlanCode = payment.TargetPlanCode
					sub.NextPlanCode = ""
					sub.PlanChangeAt = nil
					h.db.Model(&company).Update("plan_code", payment.TargetPlanCode)
					if err := h.applyPlanFeaturesToTenant(&sub, payment.TargetPlanCode); err != nil {
						h.logger.Error("Failed to apply plan features after plan change", zap.Error(err))
					}
				}
			}

			sub.Status = "active"
			h.db.Save(&sub)
			h.logger.Info("Subscription renewed",
				zap.Uint("subscription_id", sub.ID),
				zap.Int("months", months),
				zap.Time("new_expires_at", sub.ExpiresAt),
				zap.Bool("plan_change_deferred", isActivePlanChange))
		}
	}

	return envelope.SuccessResponse(nil, "company.payment.confirm.success")
}

func (h *CompanyPaymentHandler) GetMySubscription(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
	}

	var sub company_models.Subscription
	if err := h.db.Preload("Plan").Where("company_id = ?", company.ID).Order("expires_at DESC").First(&sub).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.subscription.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	if checkAndApplyPendingPlanChange(h.db, h.logger, &sub, &company) {
		h.db.Preload("Plan").First(&sub, sub.ID)
	}

	daysLeft := int(time.Until(sub.ExpiresAt).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}

	lastPaymentAmount := sub.Plan.Price
	lastPaymentMonths := 1
	var lastPayment company_models.SubscriptionPayment
	if err := h.db.Where("company_id = ? AND status = ?", company.ID, "paid").
		Order("created_at DESC").First(&lastPayment).Error; err == nil {
		lastPaymentAmount = lastPayment.Amount
		if lastPayment.Months > 0 {
			lastPaymentMonths = lastPayment.Months
		}
	}

	return envelope.SuccessResponse(SubscriptionDetailResponse{
		PlanName:          sub.Plan.Name,
		PlanCode:          sub.Plan.Code,
		PlanTier:          sub.Plan.Tier,
		Status:            sub.Status,
		ExpiresAt:         sub.ExpiresAt,
		DaysLeft:          daysLeft,
		Amount:            sub.Plan.Price,
		LastPaymentAmount: lastPaymentAmount,
		LastPaymentMonths: lastPaymentMonths,
		NextPlanCode:      sub.NextPlanCode,
		PlanChangeAt:      sub.PlanChangeAt,
	}, "company.subscription.fetch.success")
}

func (h *CompanyPaymentHandler) CancelPlanChange(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
	}

	var sub company_models.Subscription
	if err := h.db.Where("company_id = ?", company.ID).Order("expires_at DESC").First(&sub).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.subscription.not_found", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	if sub.NextPlanCode == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "subscription.plan.no_pending_change", core_errors.ErrBackofficeSubscriptionNotFound)
	}

	sub.NextPlanCode = ""
	sub.PlanChangeAt = nil
	if err := h.db.Save(&sub).Error; err != nil {
		h.logger.Error("Failed to cancel plan change", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "subscription.plan.cancel_change_error", core_errors.ErrInternal)
	}

	h.logger.Info("Plan change cancelled", zap.Uint("subscription_id", sub.ID))
	return envelope.SuccessResponse(nil, "subscription.plan.cancel_change_success")
}

func (h *CompanyPaymentHandler) GetSubscriptionPayments(c *gin.Context) envelope.Response {
	tenantID := c.GetUint("tenant_id")

	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "company.not_found", core_errors.ErrCompanyNotFound)
	}

	h.db.Model(&company_models.SubscriptionPayment{}).
		Where("company_id = ? AND status = ? AND created_at < ?", company.ID, "pending", time.Now().Add(-24*time.Hour)).
		Update("status", "cancelled")

	var payments []company_models.SubscriptionPayment
	if err := h.db.Where("company_id = ?", company.ID).Order("created_at DESC").Find(&payments).Error; err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "company.subscription.payments.fetch.error", core_errors.ErrBackofficePaymentNotFound)
	}

	return envelope.SuccessResponse(payments, "company.subscription.payments.fetch.success")
}
