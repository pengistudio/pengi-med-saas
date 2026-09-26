package clinical_handlers

import (
	"html/template"
	"io"
	"net/http"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
)

type PrescriptionTemplateHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	files  tenantfiles.Store
}

func NewPrescriptionTemplateHandler(db *gorm.DB, logger *zap.Logger, files tenantfiles.Store) *PrescriptionTemplateHandler {
	return &PrescriptionTemplateHandler{db: db, logger: logger, files: files}
}

// prescriptionTemplateName is the tenant file overriding the default prescription
// template; pdfrender picks it up by this same name.
const prescriptionTemplateName = "prescription_template.html"

// GetPrescriptionTemplateStatus returns whether the tenant has a custom template uploaded.
func (h *PrescriptionTemplateHandler) GetPrescriptionTemplateStatus(c *gin.Context) envelope.Response {
	hasCustom := h.files.Exists(tenantdb.TenantID(c), prescriptionTemplateName)
	return envelope.SuccessResponse(gin.H{"has_custom": hasCustom}, "clinical.prescription_template.status")
}

// UploadPrescriptionTemplate accepts an HTML file and stores it as the tenant's custom prescription template.
func (h *PrescriptionTemplateHandler) UploadPrescriptionTemplate(c *gin.Context) envelope.Response {

	file, _, err := c.Request.FormFile("template")
	if err != nil {
		h.logger.Error("Failed to retrieve template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Template file is required", core_errors.ErrClinicalInvalidRequest)
	}
	defer file.Close()

	src, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Template file is required", core_errors.ErrClinicalInvalidRequest)
	}
	// A template that does not parse would break every prescription download.
	if _, err := template.New(prescriptionTemplateName).Parse(string(src)); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.prescription_template.error.invalid", core_errors.ErrClinicalInvalidRequest)
	}
	if err := h.files.Write(tenantdb.TenantID(c), prescriptionTemplateName, src); err != nil {
		h.logger.Error("Failed to save template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save template file", core_errors.ErrInternal)
	}

	h.logger.Info("Prescription template uploaded", zap.Uint("tenant_id", tenantdb.TenantID(c)))
	return envelope.SuccessResponse(gin.H{"has_custom": true}, "clinical.prescription_template.uploaded")
}

// DeletePrescriptionTemplate removes the tenant's custom prescription template (reverts to default).
func (h *PrescriptionTemplateHandler) DeletePrescriptionTemplate(c *gin.Context) envelope.Response {
	if err := h.files.Remove(tenantdb.TenantID(c), prescriptionTemplateName); err != nil {
		h.logger.Error("Failed to delete template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to delete template file", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"has_custom": false}, "clinical.prescription_template.deleted")
}
