package billing_handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// sriDocumentFile is the slice of a comprobante its file downloads read.
type sriDocumentFile struct {
	ID                uint
	Status            string
	AccessKey         *string
	EstablishmentCode string
	EmissionPointCode string
	Sequential        string
}

// authorizedSriDocument loads a comprobante of kind through the request's
// tenant (another tenant's id is not found) and checks the SRI authorized it.
// On failure it has already written the translated error.
func authorizedSriDocument(c *gin.Context, db *gorm.DB, kind sri_document.Kind, notFound envelope.Response) (sriDocumentFile, bool) {
	var doc sriDocumentFile
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "billing.invoice.error.invalid_id", core_errors.ErrBillingInvalidRequest))
		return doc, false
	}
	if err := tenantdb.For(c, db).Model(kind.Model).Where("id = ?", id).Take(&doc).Error; err != nil {
		envelope.Write(c, notFound)
		return doc, false
	}
	if doc.Status != billing_models.InvoiceStatusAuthorized || doc.AccessKey == nil || *doc.AccessKey == "" {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "billing.document.error.not_authorized", core_errors.ErrBillingDocumentNotAuthorized))
		return doc, false
	}
	return doc, true
}

// serveSignedXML answers the signed XML the SRI authorized, as stored by the
// lifecycle, named after its access key.
func serveSignedXML(c *gin.Context, db *gorm.DB, logger *zap.Logger, sriDocuments *sri_document.Lifecycle, kind sri_document.Kind, notFound envelope.Response) {
	doc, ok := authorizedSriDocument(c, db, kind, notFound)
	if !ok {
		return
	}
	xml, err := sriDocuments.Documents().Files.Read(tenantdb.TenantID(c), sri_document.SignedXMLName(kind, *doc.AccessKey))
	if err != nil {
		logger.Error("Failed to read signed XML", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "billing.document.error.xml_unavailable", core_errors.ErrBillingDocumentXMLMissing))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xml", *doc.AccessKey))
	c.Data(http.StatusOK, "application/xml", xml)
}

// noteRide renders (or reads the cached) RIDE of an authorized note.
type noteRide func(db *gorm.DB, docs sri_document.Documents, id uint, tenant tenant_models.Tenant) ([]byte, error)

// serveNoteRide answers an authorized note's RIDE PDF, rendering and caching
// it when it is missing (notes authorized before RIDEs were stored).
func serveNoteRide(c *gin.Context, db *gorm.DB, logger *zap.Logger, sriDocuments *sri_document.Lifecycle, kind sri_document.Kind, notFound envelope.Response, filePrefix string, ride noteRide) {
	doc, ok := authorizedSriDocument(c, db, kind, notFound)
	if !ok {
		return
	}
	var tenant tenant_models.Tenant
	if err := db.First(&tenant, tenantdb.TenantID(c)).Error; err != nil {
		logger.Error("Failed to load tenant for RIDE", zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.ride_generate_failed", core_errors.ErrBillingInvoiceRideGenerate))
		return
	}
	pdf, err := ride(tenantdb.For(c, db), sriDocuments.Documents(), doc.ID, tenant)
	if err != nil {
		logger.Error("Failed to get RIDE", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "billing.invoice.error.ride_generate_failed", core_errors.ErrBillingInvoiceRideGenerate))
		return
	}
	fileName := fmt.Sprintf("%s_%s-%s-%s.pdf", filePrefix, doc.EstablishmentCode, doc.EmissionPointCode, doc.Sequential)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdf)
}
