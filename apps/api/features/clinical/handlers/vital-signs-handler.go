package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type VitalSignsHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewVitalSignsHandler(db *gorm.DB, logger *zap.Logger) *VitalSignsHandler {
	return &VitalSignsHandler{db: db, logger: logger}
}

// vitalSignsOwner is what a vital signs row hangs from: a medical record or,
// at triage, an appointment. Vital signs have no tenant_id of their own; they
// inherit their owner's, so every request first checks the owner's tenant.
type vitalSignsOwner struct {
	column   string // vital_signs column holding the owner's id
	model    any    // owner model, to check it belongs to the tenant
	notFound core_errors.AppError
}

var (
	recordOwner      = vitalSignsOwner{"medical_record_id", &clinical_models.MedicalRecord{}, core_errors.ErrClinicalRecordNotFound}
	appointmentOwner = vitalSignsOwner{"appointment_id", &clinical_models.Appointment{}, core_errors.ErrClinicalAppointmentNotFound}
)

// UpsertVitalSigns saves a medical record's vital signs.
func (h *VitalSignsHandler) UpsertVitalSigns(c *gin.Context) envelope.Response {
	return h.upsert(c, recordOwner)
}

// GetVitalSigns returns a medical record's vital signs.
func (h *VitalSignsHandler) GetVitalSigns(c *gin.Context) envelope.Response {
	return h.get(c, recordOwner)
}

// UpsertAppointmentVitalSigns saves the vital signs taken at triage, before the
// medical record exists.
func (h *VitalSignsHandler) UpsertAppointmentVitalSigns(c *gin.Context) envelope.Response {
	return h.upsert(c, appointmentOwner)
}

// GetAppointmentVitalSigns returns the vital signs taken at triage.
func (h *VitalSignsHandler) GetAppointmentVitalSigns(c *gin.Context) envelope.Response {
	return h.get(c, appointmentOwner)
}

func (h *VitalSignsHandler) upsert(c *gin.Context, owner vitalSignsOwner) envelope.Response {
	ownerID, errResp := h.ownerID(c, owner)
	if errResp != nil {
		return *errResp
	}

	var body clinical_dto.VitalSignsInput
	if err := c.ShouldBindJSON(&body); err != nil {
		h.logger.Error("invalid vital signs payload", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.vital_signs.error.invalid_payload", core_errors.ErrClinicalInvalidRequest)
	}
	input := body.Measurements()
	if owner.column == recordOwner.column {
		input.MedicalRecordID = &ownerID
	} else {
		input.AppointmentID = &ownerID
	}

	var existing clinical_models.VitalSigns
	result := h.db.Where(owner.column+" = ?", ownerID).First(&existing)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		h.logger.Error("failed to query vital signs", zap.Error(result.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.fetch_failed", core_errors.ErrInternal)
	}

	if result.Error == gorm.ErrRecordNotFound {
		if err := tenantdb.For(c, h.db).Create(&input).Error; err != nil {
			h.logger.Error("failed to create vital signs", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.save_failed", core_errors.ErrInternal)
		}
		return envelope.SuccessResponse(input, "clinical.vital_signs.save.success")
	}

	if err := tenantdb.For(c, h.db).Model(&existing).Updates(&input).Error; err != nil {
		h.logger.Error("failed to update vital signs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.save_failed", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(existing, "clinical.vital_signs.save.success")
}

func (h *VitalSignsHandler) get(c *gin.Context, owner vitalSignsOwner) envelope.Response {
	ownerID, errResp := h.ownerID(c, owner)
	if errResp != nil {
		return *errResp
	}

	var vitalSigns clinical_models.VitalSigns
	if err := h.db.Where(owner.column+" = ?", ownerID).First(&vitalSigns).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.SuccessResponse(nil, "clinical.vital_signs.not_found")
		}
		h.logger.Error("failed to fetch vital signs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.vital_signs.error.fetch_failed", core_errors.ErrInternal)
	}

	audit.RecordAccess(h.db, c, "vital_signs", vitalSigns.ID, nil)

	return envelope.SuccessResponse(vitalSigns, "clinical.vital_signs.fetch.success")
}

// ownerID parses :id and checks that the owner belongs to the caller's tenant
// (another tenant's owner is reported as not found).
func (h *VitalSignsHandler) ownerID(c *gin.Context, owner vitalSignsOwner) (uint, *envelope.Response) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.logger.Error("invalid vital signs owner ID", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.vital_signs.error.invalid_id", core_errors.ErrClinicalInvalidRequest)
		return 0, &resp
	}
	var count int64
	tenantdb.For(c, h.db).Model(owner.model).Where("id = ?", id).Count(&count)
	if count != 1 {
		resp := envelope.ErrorResponse(http.StatusNotFound, "error.not_found", owner.notFound)
		return 0, &resp
	}
	return uint(id), nil
}
