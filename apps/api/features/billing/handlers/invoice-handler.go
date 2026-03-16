package billing_handlers

import (
	"encoding/json"
	"fmt"
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

type InvoiceHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInvoiceHandler(db *gorm.DB, logger *zap.Logger) *InvoiceHandler {
	return &InvoiceHandler{db: db, logger: logger}
}

func (h *InvoiceHandler) CreateInvoice(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "billing.invoice.error.tenant_not_found", core_errors.ErrTenantNotFound)
	}
	tenantScope := tenant_middleware.TenantScope(c)

	var dto billing_dto.CreateInvoiceDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind CreateInvoice DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	invoice := &billing_models.Invoice{
		EmissionType:      "1",
		PaymentMethod:     dto.PaymentMethod,
		Term:              fmt.Sprint(dto.Term),
		TimeUnit:          dto.TimeUnit,
		EstablishmentCode: dto.EstablishmentCode,
		EmissionPointCode: dto.EmissionPointCode,
		Currency:          "USD",
		DocumentCode:      "01",
		IssueDate:         time.Now(),
	}

	if dto.PatientID != nil {
		invoice.PatientID = *dto.PatientID
	}

	// Compute totals based on CatalogItem (products)
	var items []billing_models.InvoiceItem
	var subtotalAcc, discountAcc, taxAcc, totalAcc float64

	for _, itemDTO := range dto.Items {
		var service billing_models.CatalogItem
		if err := h.db.Scopes(tenantScope).First(&service, itemDTO.ProductID).Error; err != nil {
			h.logger.Error("Product/Service not found", zap.Uint("id", itemDTO.ProductID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.product_not_found", core_errors.ErrBillingProductNotFound)
		}

		qty := float64(itemDTO.Quantity)
		subtotal := service.UnitPrice * qty // CatalogItem has no discount property
		taxTotal := subtotal * service.Tax
		total := subtotal + taxTotal

		if service.IceTaxCode != "3000" && service.IceTaxCode != "" {
			taxTotal += subtotal * service.IceTax
		}

		item := billing_models.InvoiceItem{
			ProductID:        itemDTO.ProductID,
			Quantity:         qty,
			Description:      service.Name,
			UnitPrice:        service.UnitPrice,
			Discount:         0.0,
			TaxRate:          service.Tax,
			Subtotal:         subtotal,
			TaxAmount:        taxTotal,
			Total:            total,
			IceTax:           service.IceTax,
			IceTaxCode:       service.IceTaxCode,
			IceTaxPercentage: service.IceTaxPercentageCode,
			TaxCode:          service.TaxCode,
			TaxPercentage:    service.TaxPercentageCode,
		}

		items = append(items, item)
		subtotalAcc += subtotal
		discountAcc += 0.0 // service.Discount
		taxAcc += taxTotal
		totalAcc += total
	}

	invoice.Items = items

	// Assign totals
	if dto.SubTotal != nil {
		invoice.Subtotal = *dto.SubTotal
	} else {
		invoice.Subtotal = subtotalAcc
	}

	if dto.Discount != nil {
		invoice.Discount = *dto.Discount
	} else {
		invoice.Discount = discountAcc
	}

	if dto.TaxTotal != nil {
		invoice.TaxTotal = *dto.TaxTotal
	} else {
		invoice.TaxTotal = taxAcc
	}

	if dto.Total != nil {
		invoice.Total = *dto.Total
	} else {
		invoice.Total = totalAcc
	}

	// Generate Sequential using GORM transaction to avoid race conditions
	err := h.db.Scopes(tenantScope).Transaction(func(tx *gorm.DB) error {
		// Use the existing GenerateSequential method which expects the model to be saved
		// but since we added multi-tenant, it requires the tenant ID populated
		invoice.TenantID = tenantID.(uint)

		_, seqErr := invoice.GenerateSequential(tx)
		if seqErr != nil {
			return seqErr
		}

		return tx.Create(invoice).Error
	})

	if err != nil {
		h.logger.Error("Failed to create Invoice", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.create_failed", core_errors.ErrBillingInvoiceCreateError)
	}

	return envelope.SuccessResponse(invoice, "billing.invoice.create.success")
}

func (h *InvoiceHandler) GetAllInvoices(c *gin.Context) envelope.Response {
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

	baseQuery := h.db.Scopes(tenantScope).Model(&billing_models.Invoice{})
	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("sequential ILIKE ? OR status ILIKE ?", like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count invoices", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoices.error.fetch_failed", core_errors.ErrInternal)
	}

	var invoices []billing_models.Invoice
	if err := baseQuery.Preload("Patient").Order("created_at DESC").Limit(limit).Offset(offset).Find(&invoices).Error; err != nil {
		h.logger.Error("Failed to fetch invoices", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoices.error.fetch_failed", core_errors.ErrInternal)
	}

	return envelope.PagedSuccessResponse(invoices, int(total), page, limit, "billing.invoices.fetch.success")
}

func (h *InvoiceHandler) DeleteInvoiceByID(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)
	invoiceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	var invoice billing_models.Invoice
	if err := h.db.Scopes(tenantScope).First(&invoice, invoiceID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	}

	if err := h.db.Scopes(tenantScope).Delete(&invoice).Error; err != nil {
		h.logger.Error("Failed to delete invoice", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.delete_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(invoice, "billing.invoice.delete.success")
}

func (h *InvoiceHandler) SRIInvoiceProcessing(c *gin.Context) envelope.Response {
	invoiceChannel := rabbitmq.GetChannel(c, "invoice_channel")
	if invoiceChannel == nil {
		h.logger.Error("RabbitMQ channel not found in context")
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.rabbitmq_failed", core_errors.ErrInternal)
	}

	invoiceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	body, err := json.Marshal(&billing_dto.InvoiceDTO{
		InvoiceID: invoiceID,
	})
	if err != nil {
		h.logger.Error("Failed to marshal InvoiceDTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.encode_failed", core_errors.ErrBillingInvalidRequest)
	}

	err = rabbitmq.PublishMessage(invoiceChannel, "invoice_tasks", body)
	if err != nil {
		h.logger.Error("Failed to publish to RabbitMQ", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.enqueue_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "billing.invoice.processing.queued")
}

func (h *InvoiceHandler) MultipleSRIInvoiceProcessing(c *gin.Context) envelope.Response {
	invoiceChannel := rabbitmq.GetChannel(c, "invoice_channel")
	if invoiceChannel == nil {
		h.logger.Error("RabbitMQ channel not found in context")
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.rabbitmq_failed", core_errors.ErrInternal)
	}

	var idList billing_dto.InvoiceIDListDTO
	if err := c.ShouldBindJSON(&idList); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	for _, currentID := range idList.IDList {
		body, err := json.Marshal(&billing_dto.InvoiceDTO{
			InvoiceID: currentID,
		})
		if err != nil {
			h.logger.Error("Failed to marshal InvoiceDTO", zap.Error(err))
			continue // Skip errors on individual payloads
		}

		err = rabbitmq.PublishMessage(invoiceChannel, "invoice_tasks", body)
		if err != nil {
			h.logger.Error("Failed to publish to RabbitMQ", zap.Error(err))
			continue
		}
	}

	return envelope.SuccessResponse(nil, "billing.invoices.processing.queued")
}
