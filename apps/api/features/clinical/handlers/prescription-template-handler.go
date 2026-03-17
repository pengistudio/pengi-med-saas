package clinical_handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
)

type PrescriptionTemplateHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPrescriptionTemplateHandler(db *gorm.DB, logger *zap.Logger) *PrescriptionTemplateHandler {
	return &PrescriptionTemplateHandler{db: db, logger: logger}
}

func prescriptionTemplatePath(tenantID any) string {
	return filepath.Join("storage", "tenants", fmt.Sprint(tenantID), "prescription_template.html")
}

// GetPrescriptionTemplateStatus returns whether the tenant has a custom template uploaded.
func (h *PrescriptionTemplateHandler) GetPrescriptionTemplateStatus(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")
	path := prescriptionTemplatePath(tenantID)
	_, err := os.Stat(path)
	hasCustom := err == nil
	return envelope.SuccessResponse(gin.H{"has_custom": hasCustom}, "clinical.prescription_template.status")
}

// UploadPrescriptionTemplate accepts an HTML file and stores it as the tenant's custom prescription template.
func (h *PrescriptionTemplateHandler) UploadPrescriptionTemplate(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")

	file, _, err := c.Request.FormFile("template")
	if err != nil {
		h.logger.Error("Failed to retrieve template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Template file is required", core_errors.ErrClinicalInvalidRequest)
	}
	defer file.Close()

	uploadDir := filepath.Join("storage", "tenants", fmt.Sprint(tenantID))
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		h.logger.Error("Failed to create storage directory", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create storage directory", core_errors.ErrInternal)
	}

	tmplPath := prescriptionTemplatePath(tenantID)
	out, err := os.Create(tmplPath)
	if err != nil {
		h.logger.Error("Failed to create template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create template file", core_errors.ErrInternal)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		h.logger.Error("Failed to save template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to save template file", core_errors.ErrInternal)
	}

	h.logger.Info("Prescription template uploaded", zap.Any("tenant_id", tenantID))
	return envelope.SuccessResponse(gin.H{"has_custom": true}, "clinical.prescription_template.uploaded")
}

// DeletePrescriptionTemplate removes the tenant's custom prescription template (reverts to default).
func (h *PrescriptionTemplateHandler) DeletePrescriptionTemplate(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")
	tmplPath := prescriptionTemplatePath(tenantID)

	if err := os.Remove(tmplPath); err != nil && !os.IsNotExist(err) {
		h.logger.Error("Failed to delete template file", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to delete template file", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(gin.H{"has_custom": false}, "clinical.prescription_template.deleted")
}
