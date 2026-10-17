package clinical_handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// A medical record is never erased: deleting it sets deleted_at with who and
// why (one audited UPDATE), and restoring clears them. Its SOAP, prescription,
// vital signs, documents and attachments are left as they are, so a restore
// brings everything back.

// recordDeleteReason validates the required deletion reason.
func recordDeleteReason(c *gin.Context) (string, *envelope.Response) {
	var req deleteAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.medical_record.delete.reason.required", core_errors.ErrClinicalInvalidRequest)
		return "", &resp
	}
	reason := strings.TrimSpace(req.Reason)
	if utf8.RuneCountInString(reason) > attachmentDeleteReasonMax {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.medical_record.delete.reason.too_long", core_errors.ErrClinicalInvalidRequest)
		return "", &resp
	}
	return reason, nil
}

// DeleteMedicalRecord soft-deletes a medical record. The reason is required.
func (h *MedicalRecordHandler) DeleteMedicalRecord(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	reason, errResp := recordDeleteReason(c)
	if errResp != nil {
		return *errResp
	}

	db := tenantdb.For(c, h.db)
	var record clinical_models.MedicalRecord
	if err := db.First(&record, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	updates := map[string]any{"deleted_at": time.Now(), "delete_reason": reason}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		updates["deleted_by_id"] = uint(uid)
	}
	if err := db.Model(&record).Updates(updates).Error; err != nil {
		h.logger.Error("failed to delete medical record", zap.Uint("id", record.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.medical_record.delete.error", core_errors.ErrClinicalRecordUpdateError)
	}
	return envelope.SuccessResponse(nil, "clinical.medical_record.delete.success")
}

// deletedMedicalRecord is a deleted record with who deleted it.
type deletedMedicalRecord struct {
	clinical_models.MedicalRecord
	DeletedByName string `json:"deleted_by_name,omitempty"`
}

// ListDeletedMedicalRecords returns the patient's deleted records, most
// recently deleted first, with who, when (deleted_at) and why.
func (h *MedicalRecordHandler) ListDeletedMedicalRecords(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var patient clinical_models.Patient
	if err := db.Select("id").First(&patient, patientID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	records := []clinical_models.MedicalRecord{}
	if err := db.Unscoped().
		Where("patient_id = ? AND deleted_at IS NOT NULL", patient.ID).
		Order("deleted_at DESC").Order("id DESC").Find(&records).Error; err != nil {
		h.logger.Error("failed to list deleted medical records", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	names := map[uint]string{}
	var ids []uint
	for _, r := range records {
		if r.DeletedByID != nil {
			ids = append(ids, *r.DeletedByID)
		}
	}
	if len(ids) > 0 {
		var users []user_models.User
		if err := h.db.Select("id", "user_name").Where("id IN ?", ids).Find(&users).Error; err == nil {
			for _, u := range users {
				names[u.ID] = u.UserName
			}
		}
	}
	out := make([]deletedMedicalRecord, 0, len(records))
	for _, r := range records {
		d := deletedMedicalRecord{MedicalRecord: r}
		if r.DeletedByID != nil {
			d.DeletedByName = names[*r.DeletedByID]
		}
		out = append(out, d)
	}
	return envelope.SuccessResponse(out, "clinical.medical_record.deleted.list.success")
}

// RestoreMedicalRecord brings a deleted record back. One that is not deleted
// (or not in the tenant) is 404.
func (h *MedicalRecordHandler) RestoreMedicalRecord(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var record clinical_models.MedicalRecord
	if err := db.Unscoped().Where("deleted_at IS NOT NULL").First(&record, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if err := db.Unscoped().Model(&record).Updates(map[string]any{"deleted_at": nil, "deleted_by_id": nil, "delete_reason": ""}).Error; err != nil {
		h.logger.Error("failed to restore medical record", zap.Uint("id", record.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.medical_record.restore.error", core_errors.ErrClinicalRecordUpdateError)
	}
	return envelope.SuccessResponse(nil, "clinical.medical_record.restore.success")
}
