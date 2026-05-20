package backoffice_handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/utils"
	company_models "pengi-med-saas/features/companies/models"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// resolvePriceFromPricings returns the price for the given months from plan pricings.
// Falls back to plan.Price * months if no specific pricing entry exists.
func resolvePriceFromPricings(plan company_models.Plan, months int) float64 {
	if months <= 0 {
		months = 1
	}
	for _, p := range plan.Pricings {
		if p.Months == months {
			return p.Price
		}
	}
	return plan.Price * float64(months)
}

type BackofficePaymentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	dlocal *utils.DlocalClient
}

func NewBackofficePaymentHandler(db *gorm.DB, logger *zap.Logger) *BackofficePaymentHandler {
	return &BackofficePaymentHandler{
		db:     db,
		logger: logger,
		dlocal: utils.NewDlocalClient(),
	}
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type GeneratePaymentsRequest struct {
	SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
}

type GeneratePaymentResult struct {
	SubscriptionID uint    `json:"subscription_id"`
	CompanyName    string  `json:"company_name"`
	Amount         float64 `json:"amount"`
	CheckoutURL    string  `json:"checkout_url"`
	Status         string  `json:"status"`
}

type DlocalWebhookPayload struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *BackofficePaymentHandler) GeneratePayments(c *gin.Context) envelope.Response {
	var req GeneratePaymentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.payment.request.invalid", core_errors.ErrBackofficeInvalidRequest)
	}

	results := make([]GeneratePaymentResult, 0, len(req.SubscriptionIDs))

	for _, subID := range req.SubscriptionIDs {
		var sub company_models.Subscription
		if err := h.db.Preload("Plan.Pricings").Preload("Company").First(&sub, subID).Error; err != nil {
			h.logger.Warn("Subscription not found, skipping", zap.Uint("subscription_id", subID))
			continue
		}

		// Default to 1 month for backoffice-generated payments
		months := 1
		amount := resolvePriceFromPricings(sub.Plan, months)

		orderID := uuid.New().String()

		dlocalResp, err := h.dlocal.CreatePayment(utils.DlocalCreatePaymentRequest{
			Amount:          amount,
			Currency:        "USD",
			OrderID:         orderID,
			Description:     fmt.Sprintf("Suscripción Pengi - %s (%d mes(es))", sub.Plan.Name, months),
			NotificationURL: os.Getenv("DLOCAL_NOTIFICATION_URL"),
			SuccessURL:      os.Getenv("FRONTEND_URL") + "/subscription?status=success",
		})
		if err != nil {
			h.logger.Error("Failed to create dlocal payment", zap.Uint("subscription_id", subID), zap.Error(err))
			continue
		}

		checkoutURL := dlocalResp.URL()

		payment := company_models.SubscriptionPayment{
			CompanyID:       sub.CompanyID,
			SubscriptionID:  sub.ID,
			DlocalPaymentID: dlocalResp.ID,
			OrderID:         orderID,
			Amount:          amount,
			Currency:        "USD",
			Status:          "pending",
			CheckoutURL:     checkoutURL,
			TargetPlanCode:  sub.PlanCode,
			Months:          months,
		}
		if err := h.db.Create(&payment).Error; err != nil {
			h.logger.Error("Failed to save payment", zap.Uint("subscription_id", subID), zap.Error(err))
			continue
		}

		results = append(results, GeneratePaymentResult{
			SubscriptionID: sub.ID,
			CompanyName:    sub.Company.LegalName,
			Amount:         amount,
			CheckoutURL:    checkoutURL,
			Status:         "pending",
		})
	}

	return envelope.SuccessResponse(results, "backoffice.payment.generate.success")
}

func (h *BackofficePaymentHandler) GetPayments(c *gin.Context) envelope.Response {
	query := h.db.Preload("Subscription.Plan")

	if companyID := c.Query("company_id"); companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var payments []company_models.SubscriptionPayment
	if err := query.Find(&payments).Error; err != nil {
		h.logger.Error("Failed to fetch payments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "backoffice.payment.fetch.error", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(payments, "backoffice.payment.fetch.success")
}

func (h *BackofficePaymentHandler) HandleDlocalWebhook(c *gin.Context) envelope.Response {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.webhook.body.read.error", core_errors.ErrInvalidRequest)
	}

	// Verify HMAC-SHA256 signature
	secret := os.Getenv("DLOCAL_SECRET")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	received := c.GetHeader("X-Signature")

	if !hmac.Equal([]byte(expected), []byte(received)) {
		h.logger.Warn("Invalid dlocal webhook signature")
		return envelope.ErrorResponse(http.StatusUnauthorized, "backoffice.webhook.signature.invalid", core_errors.ErrBackofficeWebhookInvalidSig)
	}

	var payload DlocalWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.webhook.payload.invalid", core_errors.ErrInvalidRequest)
	}

	var payment company_models.SubscriptionPayment
	if err := h.db.Where("order_id = ?", payload.OrderID).First(&payment).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "backoffice.payment.not_found", core_errors.ErrBackofficePaymentNotFound)
	}

	payment.Status = strings.ToLower(payload.Status)
	if payload.Status != "" {
		payment.DlocalPaymentID = payload.ID
	}
	h.db.Save(&payment)

	if payload.Status == "PAID" {
		var sub company_models.Subscription
		if err := h.db.First(&sub, payment.SubscriptionID).Error; err == nil {
			months := payment.Months
			if months <= 0 {
				months = 1
			}
			sub.ExpiresAt = sub.ExpiresAt.AddDate(0, months, 0)
			sub.Status = "active"
			h.db.Save(&sub)
			h.logger.Info("Subscription renewed via dlocal payment",
				zap.Uint("subscription_id", sub.ID),
				zap.Int("months", months),
				zap.Time("new_expires_at", sub.ExpiresAt))
		}
	}

	return envelope.SuccessResponse(nil, "backoffice.payment.webhook.processed")
}
