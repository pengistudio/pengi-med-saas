package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type VitalSignsHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewVitalSignsHandler(db *gorm.DB, logger *zap.Logger) *VitalSignsHandler {
	return &VitalSignsHandler{db: db, logger: logger}
}

func (h *VitalSignsHandler) UpsertVitalSigns(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	recordID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.vital_signs.error.invalid_id", core_errors.ErrClinicalInvalidRequest)
	}

	var input clinical_models.VitalSigns
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Error("invalid vital signs payload", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.vital_signs.error.invalid_payload", core_errors.ErrClinicalInvalidRequest)
	}
	input.MedicalRecordID = uint(recordID)

	var existing clinical_models.VitalSigns
	result := h.db.Where("medical_record_id = ?", recordID).First(&existing)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		h.logger.Error("failed to query vital signs", zap.Error(result.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.fetch_failed", core_errors.ErrInternal)
	}

	if result.Error == gorm.ErrRecordNotFound {
		if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(&input).Error; err != nil {
			h.logger.Error("failed to create vital signs", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.save_failed", core_errors.ErrInternal)
		}
		return envelope.SuccessResponse(input, "clinical.vital_signs.save.success")
	}

	// Update existing record
	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Model(&existing).Updates(&input).Error; err != nil {
		h.logger.Error("failed to update vital signs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.save_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(existing, "clinical.vital_signs.save.success")
}

func (h *VitalSignsHandler) GetVitalSigns(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	recordID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.vital_signs.error.invalid_id", core_errors.ErrClinicalInvalidRequest)
	}

	var vitalSigns clinical_models.VitalSigns
	if err := h.db.Where("medical_record_id = ?", recordID).First(&vitalSigns).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.SuccessResponse(nil, "clinical.vital_signs.not_found")
		}
		h.logger.Error("failed to fetch vital signs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.fetch_failed", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(vitalSigns, "clinical.vital_signs.fetch.success")
}
