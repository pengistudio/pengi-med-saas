package billing_handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CreditNoteHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCreditNoteHandler(db *gorm.DB, logger *zap.Logger) *CreditNoteHandler {
	return &CreditNoteHandler{db: db, logger: logger}
}

func (h *CreditNoteHandler) CreateCreditNote(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "billing.invoice.error.tenant_not_found", core_errors.ErrTenantNotFound)
	}
	tenantScope := tenant_middleware.TenantScope(c)

	var dto billing_dto.CreateCreditNoteDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind CreateCreditNote DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	var invoice billing_models.Invoice
	if err := h.db.Scopes(tenantScope).First(&invoice, dto.InvoiceID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	}
	if invoice.Status != billing_models.InvoiceStatusAuthorized {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.credit_note.error.invoice_not_authorized", core_errors.ErrBillingInvoiceNotAuthorized)
	}

	creditNote := &billing_models.CreditNote{
		InvoiceID:         invoice.ID,
		EmissionType:      "1",
		EstablishmentCode: invoice.EstablishmentCode,
		EmissionPointCode: invoice.EmissionPointCode,
		Currency:          "DOLAR",
		DocumentCode:      billing_models.CreditNoteDocumentCode,
		Reason:            dto.Reason,
		IssueDate:         time.Now(),
	}

	var items []billing_models.CreditNoteItem
	var subtotalAcc, taxAcc, totalAcc float64

	for _, itemDTO := range dto.Items {
		var product billing_models.CatalogItem
		if err := h.db.Scopes(tenantScope).First(&product, itemDTO.ProductID).Error; err != nil {
			h.logger.Error("Product/Service not found", zap.Uint("id", itemDTO.ProductID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.product_not_found", core_errors.ErrBillingProductNotFound)
		}

		qty := float64(itemDTO.Quantity)
		subtotal := product.UnitPrice * qty
		taxTotal := subtotal * product.Tax
		total := subtotal + taxTotal

		if product.IceTaxCode != "3000" && product.IceTaxCode != "" {
			taxTotal += subtotal * product.IceTax
		}

		items = append(items, billing_models.CreditNoteItem{
			ProductID:        itemDTO.ProductID,
			Quantity:         qty,
			Description:      product.Name,
			UnitPrice:        product.UnitPrice,
			TaxRate:          product.Tax,
			Subtotal:         subtotal,
			TaxAmount:        taxTotal,
			Total:            total,
			IceTax:           product.IceTax,
			IceTaxCode:       product.IceTaxCode,
			IceTaxPercentage: product.IceTaxPercentageCode,
			TaxCode:          product.TaxCode,
			TaxPercentage:    product.TaxPercentageCode,
		})

		subtotalAcc += subtotal
		taxAcc += taxTotal
		totalAcc += total
	}

	creditNote.Items = items
	creditNote.Subtotal = subtotalAcc
	creditNote.TaxTotal = taxAcc
	creditNote.Total = totalAcc

	err := h.db.Scopes(tenantScope).Transaction(func(tx *gorm.DB) error {
		creditNote.TenantID = tenantID.(uint)

		if _, seqErr := creditNote.GenerateSequential(tx); seqErr != nil {
			return seqErr
		}

		return tx.Create(creditNote).Error
	})

	if err != nil {
		h.logger.Error("Failed to create CreditNote", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.credit_note.error.create_failed", core_errors.ErrBillingCreditNoteCreateError)
	}

	return envelope.SuccessResponse(creditNote, "billing.credit_note.create.success")
}

func (h *CreditNoteHandler) GetAllCreditNotes(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := h.db.Scopes(tenantScope).Model(&billing_models.CreditNote{})
	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("sequential ILIKE ? OR status ILIKE ?", like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count credit notes", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.credit_notes.error.fetch_failed", core_errors.ErrInternal)
	}

	var creditNotes []billing_models.CreditNote
	if err := baseQuery.Preload("Invoice").Preload("Invoice.Patient").Order("created_at DESC").Limit(limit).Offset(offset).Find(&creditNotes).Error; err != nil {
		h.logger.Error("Failed to fetch credit notes", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.credit_notes.error.fetch_failed", core_errors.ErrInternal)
	}

	return envelope.PagedSuccessResponse(creditNotes, int(total), page, limit, "billing.credit_notes.fetch.success")
}

func (h *CreditNoteHandler) SRICreditNoteProcessing(c *gin.Context) envelope.Response {
	channel := rabbitmq.GetChannel(c, "invoice_channel")
	if channel == nil {
		h.logger.Error("RabbitMQ channel not found in context")
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.rabbitmq_failed", core_errors.ErrInternal)
	}

	creditNoteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	body, err := json.Marshal(&billing_dto.CreditNoteDTO{CreditNoteID: creditNoteID})
	if err != nil {
		h.logger.Error("Failed to marshal CreditNoteDTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.encode_failed", core_errors.ErrBillingInvalidRequest)
	}

	if err := rabbitmq.PublishMessage(channel, "credit_note_tasks", body); err != nil {
		h.logger.Error("Failed to publish to RabbitMQ", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.enqueue_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "billing.invoice.processing.queued")
}
