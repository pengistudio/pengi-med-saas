package billing_handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

type DebitNoteHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDebitNoteHandler(db *gorm.DB, logger *zap.Logger) *DebitNoteHandler {
	return &DebitNoteHandler{db: db, logger: logger}
}

func (h *DebitNoteHandler) CreateDebitNote(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "billing.invoice.error.tenant_not_found", core_errors.ErrTenantNotFound)
	}
	tenantScope := tenant_middleware.TenantScope(c)

	var dto billing_dto.CreateDebitNoteDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind CreateDebitNote DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	var invoice billing_models.Invoice
	if err := h.db.Scopes(tenantScope).First(&invoice, dto.InvoiceID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	}
	if invoice.Status != billing_models.InvoiceStatusAuthorized {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.debit_note.error.invoice_not_authorized", core_errors.ErrBillingInvoiceNotAuthorized)
	}

	debitNote := &billing_models.DebitNote{
		InvoiceID:         invoice.ID,
		EmissionType:      "1",
		EstablishmentCode: invoice.EstablishmentCode,
		EmissionPointCode: invoice.EmissionPointCode,
		Currency:          "DOLAR",
		DocumentCode:      billing_models.DebitNoteDocumentCode,
		IssueDate:         time.Now(),
	}

	var motives []billing_models.DebitNoteMotive
	var subtotalAcc, taxAcc, totalAcc float64

	for _, motiveDTO := range dto.Motives {
		taxAmount := motiveDTO.Value * motiveDTO.TaxRate
		motives = append(motives, billing_models.DebitNoteMotive{
			Reason:            motiveDTO.Reason,
			Value:             motiveDTO.Value,
			TaxCode:           motiveDTO.TaxCode,
			TaxPercentageCode: motiveDTO.TaxPercentageCode,
			TaxRate:           motiveDTO.TaxRate,
		})
		subtotalAcc += motiveDTO.Value
		taxAcc += taxAmount
		totalAcc += motiveDTO.Value + taxAmount
	}

	debitNote.Motives = motives
	debitNote.Subtotal = subtotalAcc
	debitNote.TaxTotal = taxAcc
	debitNote.Total = totalAcc

	err := h.db.Scopes(tenantScope).Transaction(func(tx *gorm.DB) error {
		debitNote.TenantID = tenantID.(uint)

		if _, seqErr := debitNote.GenerateSequential(tx); seqErr != nil {
			return seqErr
		}

		return tx.Create(debitNote).Error
	})

	if err != nil {
		h.logger.Error("Failed to create DebitNote", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.debit_note.error.create_failed", core_errors.ErrBillingDebitNoteCreateError)
	}

	return envelope.SuccessResponse(debitNote, "billing.debit_note.create.success")
}

func (h *DebitNoteHandler) GetAllDebitNotes(c *gin.Context) envelope.Response {
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

	baseQuery := h.db.Scopes(tenantScope).Model(&billing_models.DebitNote{})
	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("sequential ILIKE ? OR status ILIKE ?", like, like)
	}
	if status := c.Query("status"); status != "" {
		baseQuery = baseQuery.Where("status IN ?", strings.Split(status, ","))
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count debit notes", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.debit_notes.error.fetch_failed", core_errors.ErrInternal)
	}

	var debitNotes []billing_models.DebitNote
	if err := baseQuery.Preload("Invoice").Preload("Invoice.Patient").Preload("Motives").Order("created_at DESC").Limit(limit).Offset(offset).Find(&debitNotes).Error; err != nil {
		h.logger.Error("Failed to fetch debit notes", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.debit_notes.error.fetch_failed", core_errors.ErrInternal)
	}

	return envelope.PagedSuccessResponse(debitNotes, int(total), page, limit, "billing.debit_notes.fetch.success")
}

func (h *DebitNoteHandler) SRIDebitNoteProcessing(c *gin.Context) envelope.Response {
	channel := rabbitmq.GetChannel(c, "invoice_channel")
	if channel == nil {
		h.logger.Error("RabbitMQ channel not found in context")
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.rabbitmq_failed", core_errors.ErrInternal)
	}

	debitNoteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	tenantScope := tenant_middleware.TenantScope(c)
	var debitNote billing_models.DebitNote
	if err := h.db.Scopes(tenantScope).First(&debitNote, debitNoteID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	}
	if debitNote.Status == billing_models.InvoiceStatusAuthorized {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.already_authorized", core_errors.ErrBillingInvalidRequest)
	}

	if err := h.db.Scopes(tenantScope).Model(&debitNote).
		Update("status", billing_models.InvoiceStatusPending).Error; err != nil {
		h.logger.Error("Failed to mark debit note as pending", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.enqueue_failed", core_errors.ErrInternal)
	}

	body, err := json.Marshal(&billing_dto.DebitNoteDTO{DebitNoteID: debitNoteID})
	if err != nil {
		h.logger.Error("Failed to marshal DebitNoteDTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.encode_failed", core_errors.ErrBillingInvalidRequest)
	}

	if err := rabbitmq.PublishMessage(channel, "debit_note_tasks", body); err != nil {
		h.logger.Error("Failed to publish to RabbitMQ", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.enqueue_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "billing.invoice.processing.queued")
}
