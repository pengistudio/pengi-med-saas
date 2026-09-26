package billing_handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"pengi-med-saas/core/tenantdb"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvoiceHandler struct {
	db           *gorm.DB
	logger       *zap.Logger
	sriDocuments *sri_document.Lifecycle
}

func NewInvoiceHandler(db *gorm.DB, logger *zap.Logger, sriDocuments *sri_document.Lifecycle) *InvoiceHandler {
	return &InvoiceHandler{db: db, logger: logger, sriDocuments: sriDocuments}
}

func (h *InvoiceHandler) CreateInvoice(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "billing.invoice.error.tenant_not_found", core_errors.ErrTenantNotFound)
	}
	db := tenantdb.For(c, h.db)

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

	invoice.PatientID = dto.PatientID

	// Compute totals based on CatalogItem (products)
	var items []billing_models.InvoiceItem
	var subtotalAcc, discountAcc, taxAcc, totalAcc float64

	for _, itemDTO := range dto.Items {
		var service billing_models.CatalogItem
		if err := db.First(&service, itemDTO.ProductID).Error; err != nil {
			h.logger.Error("Product/Service not found", zap.Uint("id", itemDTO.ProductID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.product_not_found", core_errors.ErrBillingProductNotFound)
		}

		qty := float64(itemDTO.Quantity)
		grossAmount := service.UnitPrice * qty
		discount := itemDTO.Discount
		if discount < 0 {
			discount = 0
		}
		if discount > grossAmount {
			discount = grossAmount
		}
		subtotal := grossAmount - discount
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
			Discount:         discount,
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
		discountAcc += discount
		taxAcc += taxTotal
		totalAcc += total
	}

	invoice.Items = items

	// Totals are always server-computed from the itemized data above — never trust
	// client-supplied aggregate totals for a document with legal/fiscal effect.
	invoice.Subtotal = subtotalAcc
	invoice.Discount = discountAcc
	invoice.TaxTotal = taxAcc
	invoice.Total = totalAcc

	if invoice.PatientID == nil && math.Round(invoice.Total*100)/100 > billing_models.FinalConsumerMaxTotal {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.final_consumer_limit", core_errors.ErrBillingInvalidRequest)
	}

	// Generate Sequential using GORM transaction to avoid race conditions
	err := db.Transaction(func(tx *gorm.DB) error {
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
	db := tenantdb.For(c, h.db)

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

	baseQuery := db.Model(&billing_models.Invoice{})
	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("sequential ILIKE ? OR status ILIKE ?", like, like)
	}
	if status := c.Query("status"); status != "" {
		baseQuery = baseQuery.Where("status IN ?", strings.Split(status, ","))
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
	db := tenantdb.For(c, h.db)
	invoiceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	var invoice billing_models.Invoice
	if err := db.First(&invoice, invoiceID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	}

	if err := db.Delete(&invoice).Error; err != nil {
		h.logger.Error("Failed to delete invoice", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.delete_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(invoice, "billing.invoice.delete.success")
}

func (h *InvoiceHandler) SRIInvoiceProcessing(c *gin.Context) envelope.Response {
	return enqueueSriDocument(c, h.db, h.logger, h.sriDocuments, sri_document.Invoice)
}

// MultipleSRIInvoiceProcessing queues every listed invoice that can still be
// processed; authorized and unknown ones are skipped.
func (h *InvoiceHandler) MultipleSRIInvoiceProcessing(c *gin.Context) envelope.Response {
	var idList billing_dto.InvoiceIDListDTO
	if err := c.ShouldBindJSON(&idList); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	tenantDB := tenantdb.For(c, h.db)
	for _, id := range idList.IDList {
		err := h.sriDocuments.Enqueue(tenantDB, sri_document.Invoice, uint64(id))
		if err != nil && !errors.Is(err, sri_document.ErrNotFound) && !errors.Is(err, sri_document.ErrAlreadyAuthorized) {
			h.logger.Error("Failed to enqueue invoice for SRI processing", zap.Uint("invoice_id", uint(id)), zap.Error(err))
		}
	}

	return envelope.SuccessResponse(nil, "billing.invoices.processing.queued")
}

// DownloadInvoiceRide serves the RIDE (Representación Impresa del Documento
// Electrónico) PDF for an authorized invoice. If it hasn't been generated yet
// (e.g. the invoice was authorized before this feature existed), it is
// generated on demand.
func (h *InvoiceHandler) DownloadInvoiceRide(c *gin.Context) {
	db := tenantdb.For(c, h.db)
	invoiceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest))
		return
	}

	var invoice billing_models.Invoice
	if err := db.Preload("Patient").Preload("Items").First(&invoice, invoiceID).Error; err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound))
		return
	}

	if invoice.Status != billing_models.InvoiceStatusAuthorized || invoice.AccessKey == nil {
		c.JSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.ride_not_authorized", core_errors.ErrBillingInvoiceRideNotReady))
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	pdfPath := filepath.Join("storage", "tenants", fmt.Sprint(tenantIDVal), "invoices", fmt.Sprintf("%s.pdf", *invoice.AccessKey))

	force := c.Query("force") == "true"
	var errFile error
	var pdfBytes []byte

	if !force {
		pdfBytes, errFile = os.ReadFile(pdfPath)
	}

	if force || errFile != nil {
		var tenant tenant_models.Tenant
		if err := h.db.First(&tenant, tenantIDVal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.ride_generate_failed", core_errors.ErrBillingInvoiceRideGenerate))
			return
		}
		address := tenant.Address
		if address == "" {
			address = "Dirección no provista"
		}
		pdfBytes, errFile = sri_services.GenerateInvoiceRide(invoice, tenant, address, sri_services.ResolveSriEnv())
		if errFile != nil {
			h.logger.Error("Failed to generate RIDE on demand", zap.Uint64("invoice_id", invoiceID), zap.Error(errFile))
			c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.ride_generate_failed", core_errors.ErrBillingInvoiceRideGenerate))
			return
		}

		// Save the newly generated RIDE to cache
		destDir := filepath.Dir(pdfPath)
		_ = os.MkdirAll(destDir, os.ModePerm)
		_ = os.WriteFile(pdfPath, pdfBytes, 0644)
	}

	fileName := fmt.Sprintf("factura_%s-%s-%s.pdf", invoice.EstablishmentCode, invoice.EmissionPointCode, invoice.Sequential)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
