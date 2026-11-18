package billing_handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// sendSriDocumentEmailRequest: Email overrides the buyer's address; blank
// sends to the buyer.
type sendSriDocumentEmailRequest struct {
	Email string `json:"email"`
}

// EmailInvoice emails an authorized invoice (RIDE + authorized XML).
func (h *InvoiceHandler) EmailInvoice(c *gin.Context) envelope.Response {
	return emailSriDocument(c, h.db, h.logger, h.sriDocuments, sri_document.Invoice,
		envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound))
}

// EmailCreditNote emails an authorized credit note (RIDE + authorized XML).
func (h *CreditNoteHandler) EmailCreditNote(c *gin.Context) envelope.Response {
	return emailSriDocument(c, h.db, h.logger, h.sriDocuments, sri_document.CreditNote, creditNoteNotFound())
}

// EmailDebitNote emails an authorized debit note (RIDE + authorized XML).
func (h *DebitNoteHandler) EmailDebitNote(c *gin.Context) envelope.Response {
	return emailSriDocument(c, h.db, h.logger, h.sriDocuments, sri_document.DebitNote, debitNoteNotFound())
}

// emailSriDocument sends an authorized comprobante of the request's tenant to
// the body's email, or to its buyer when blank, and answers its delivery
// state (emailed_at, emailed_to, email_error).
func emailSriDocument(c *gin.Context, db *gorm.DB, logger *zap.Logger, sriDocuments *sri_document.Lifecycle, kind sri_document.Kind, notFound envelope.Response) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}
	var req sendSriDocumentEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_payload", core_errors.ErrBillingInvalidRequest)
	}

	// Through the request's tenant: another tenant's id is not found.
	var doc sriDocumentFile
	if err := tenantdb.For(c, db).Model(kind.Model).Where("id = ?", id).Take(&doc).Error; err != nil {
		return notFound
	}
	if doc.Status != billing_models.InvoiceStatusAuthorized || doc.AccessKey == nil || *doc.AccessKey == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.document.error.not_authorized", core_errors.ErrBillingDocumentNotAuthorized)
	}

	state, err := sriDocuments.SendEmail(kind, tenantdb.TenantID(c), doc.ID, req.Email)
	switch {
	case err == nil:
		return envelope.SuccessResponse(state, "billing.document.email.success")
	case errors.Is(err, sri_document.ErrNotFound):
		return notFound
	case errors.Is(err, sri_document.ErrNotAuthorized):
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.document.error.not_authorized", core_errors.ErrBillingDocumentNotAuthorized)
	case errors.Is(err, sri_document.ErrNoRecipient):
		return envelope.ErrorResponse(http.StatusUnprocessableEntity, "billing.document.email.error.no_recipient", core_errors.ErrBillingEmailNoRecipient)
	case errors.Is(err, sri_document.ErrInvalidRecipient):
		return envelope.ErrorResponse(http.StatusBadRequest, sri_document.EmailErrorInvalidRecipient, core_errors.ErrBillingEmailInvalid)
	case errors.Is(err, sri_document.ErrEmailTooSoon):
		return envelope.ErrorResponse(http.StatusTooManyRequests, "billing.document.email.error.too_soon", core_errors.ErrBillingEmailTooSoon)
	case errors.Is(err, sri_document.ErrEmailAttachments):
		return envelope.ErrorResponse(http.StatusInternalServerError, sri_document.EmailErrorAttachments, core_errors.ErrBillingEmailAttachments)
	case errors.Is(err, sri_document.ErrEmailSendFailed):
		return envelope.ErrorResponse(http.StatusBadGateway, sri_document.EmailErrorSendFailed, core_errors.ErrBillingEmailSendFailed)
	default:
		logger.Error("Failed to email SRI document", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, sri_document.EmailErrorSendFailed, core_errors.ErrBillingEmailSendFailed)
	}
}
