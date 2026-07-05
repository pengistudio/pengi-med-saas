package clinical_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MedicalRecordDraftHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewMedicalRecordDraftHandler(db *gorm.DB, logger *zap.Logger) *MedicalRecordDraftHandler {
	return &MedicalRecordDraftHandler{db: db, logger: logger}
}

func (h *MedicalRecordDraftHandler) currentUserID(c *gin.Context) (uint, bool) {
	userID, _, exists := auth_middleware.GetUserFromContext(c)
	if !exists {
		return 0, false
	}
	return uint(userID), true
}

func (h *MedicalRecordDraftHandler) GetDraft(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("patient_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.medical_record.draft.invalid_patient", core_errors.ErrClinicalInvalidRequest)
	}
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrClinicalInvalidRequest)
	}

	var draft clinical_models.MedicalRecordDraft
	err = h.db.Scopes(tenant_middleware.TenantScope(c)).
		Where("user_id = ? AND patient_id = ?", userID, patientID).
		First(&draft).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.SuccessResponse(nil, "clinical.medical_record.draft.not_found")
		}
		h.logger.Error("failed to fetch medical record draft", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.medical_record.draft.error", core_errors.ErrClinicalDraftError)
	}
	return envelope.SuccessResponse(draft, "clinical.medical_record.draft.found")
}

func (h *MedicalRecordDraftHandler) SaveDraft(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("patient_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.medical_record.draft.invalid_patient", core_errors.ErrClinicalInvalidRequest)
	}
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrClinicalInvalidRequest)
	}

	var req clinical_dto.SaveMedicalRecordDraftDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	tenantID := c.GetUint("tenant_id")

	draft := clinical_models.MedicalRecordDraft{
		TenantID:  tenantID,
		UserID:    userID,
		PatientID: uint(patientID),
		Data:      req.Data,
	}

	// Atomic upsert: concurrent saves for the same draft (e.g. two debounced
	// requests racing) must not both take the find-then-create path and
	// collide on the unique (tenant_id, user_id, patient_id) index.
	// Include deleted_at so a re-save resurrects a previously discarded
	// (soft-deleted) draft for this patient instead of silently updating
	// a row that GORM's default scope will keep filtering out.
	err = h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}, {Name: "patient_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at", "deleted_at"}),
	}).Create(&draft).Error
	if err != nil {
		h.logger.Error("failed to upsert medical record draft", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.medical_record.draft.error", core_errors.ErrClinicalDraftError)
	}

	return envelope.SuccessResponse(draft, "clinical.medical_record.draft.saved")
}

func (h *MedicalRecordDraftHandler) DeleteDraft(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("patient_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.medical_record.draft.invalid_patient", core_errors.ErrClinicalInvalidRequest)
	}
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrClinicalInvalidRequest)
	}

	// Hard delete (Unscoped): a soft-deleted row would still occupy the
	// unique (tenant_id, user_id, patient_id) index and break the next
	// upsert for the same patient.
	if err := h.db.Unscoped().Scopes(tenant_middleware.TenantScope(c)).
		Where("user_id = ? AND patient_id = ?", userID, patientID).
		Delete(&clinical_models.MedicalRecordDraft{}).Error; err != nil {
		h.logger.Error("failed to delete medical record draft", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.medical_record.draft.error", core_errors.ErrClinicalDraftError)
	}

	return envelope.SuccessResponse(nil, "clinical.medical_record.draft.discarded")
}
