package billing_handlers

import (
	"errors"
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	sri_document "pengi-med-saas/features/billing/sri-document"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// enqueueSriDocument requests SRI processing of the :id document of kind for the
// current tenant. The lifecycle decides what processing means for its status.
func enqueueSriDocument(c *gin.Context, db *gorm.DB, logger *zap.Logger, docs *sri_document.Lifecycle, kind sri_document.Kind) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest)
	}

	err = docs.Enqueue(db.Scopes(tenant_middleware.TenantScope(c)), kind, id)
	switch {
	case err == nil:
		return envelope.SuccessResponse(nil, "billing.invoice.processing.queued")
	case errors.Is(err, sri_document.ErrNotFound):
		return envelope.ErrorResponse(http.StatusNotFound, "billing.invoice.error.not_found", core_errors.ErrBillingInvoiceNotFound)
	case errors.Is(err, sri_document.ErrAlreadyAuthorized):
		return envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.already_authorized", core_errors.ErrBillingInvalidRequest)
	default:
		logger.Error("Failed to enqueue SRI document", zap.String("kind", kind.Name), zap.Uint64("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.enqueue_failed", core_errors.ErrInternal)
	}
}
