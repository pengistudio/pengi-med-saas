package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type MedicalRecordHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewMedicalRecordHandler(db *gorm.DB, logger *zap.Logger) *MedicalRecordHandler {
	return &MedicalRecordHandler{db: db, logger: logger}
}

func (h *MedicalRecordHandler) GetMedicalRecords(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	var records []clinical_models.MedicalRecord
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("SOAPRecord").Preload("Prescription").Order("created_at desc").Where("patient_id = ?", id).Find(&records).Error; err != nil {
		h.logger.Error("Failed to fetch medical records", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	return envelope.SuccessResponse(records, "clinical.medical_record.list.success")
}

func (h *MedicalRecordHandler) GetMedicalRecord(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid medical record ID format", core_errors.ErrAuthInvalidRequest)
	}

	var record clinical_models.MedicalRecord
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("SOAPRecord").Preload("Prescription").First(&record, id).Error; err != nil {
		h.logger.Error("Failed to fetch medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	return envelope.SuccessResponse(record, "clinical.medical_record.found")
}

func (h *MedicalRecordHandler) CreateMedicalRecord(c *gin.Context) envelope.Response {
	var newRecord clinical_dto.CreateMedicalRecordDTO
	if err := c.ShouldBind(&newRecord); err != nil {
		h.logger.Error("Invalid create medical record request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	record := &clinical_models.MedicalRecord{
		Date:                     newRecord.Date,
		NextAppointmentDate:      newRecord.NextAppointmentDate,
		NextAppointmentStartTime: newRecord.NextAppointmentStartTime,
		NextAppointmentEndTime:   newRecord.NextAppointmentEndTime,
		Motive:                   newRecord.Motive,
		Observation:              newRecord.Observation,
		PatientID:                newRecord.PatientID,
		SOAPRecord:               newRecord.SOAPRecord,
	}

	// Create prescription if provided
	if newRecord.Prescription != nil && (newRecord.Prescription.Content != "" || newRecord.Prescription.Indications != "") {
		record.Prescription = newRecord.Prescription
	}

	tenantID, exists := c.Get("tenant_id")
	if exists {
		record.TenantID = tenantID.(uint)
	}

	if err := h.db.Create(record).Error; err != nil {
		h.logger.Error("Failed to create medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordCreateError)
	}

	h.logger.Info("Medical record created successfully", zap.Uint("id", record.ID))
	return envelope.SuccessResponse(record, "clinical.medical_record.create.success")
}

func (h *MedicalRecordHandler) UpdateMedicalRecord(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid medical record ID format", core_errors.ErrAuthInvalidRequest)
	}

	var medicalRecord clinical_models.MedicalRecord
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	var updatedRecord clinical_dto.UpdateMedicalRecordDTO
	if err := c.ShouldBind(&updatedRecord); err != nil {
		h.logger.Error("Invalid update medical record request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// Build update map only with provided fields for MedicalRecord
	record := make(map[string]interface{})
	if updatedRecord.Date != nil {
		record["date"] = *updatedRecord.Date
	}
	if updatedRecord.NextAppointmentDate != nil {
		record["next_appointment_date"] = *updatedRecord.NextAppointmentDate
	}
	if updatedRecord.Motive != nil {
		record["motive"] = *updatedRecord.Motive
	}
	if updatedRecord.Observation != nil {
		record["observation"] = *updatedRecord.Observation
	}

	// Update MedicalRecord fields
	if len(record) > 0 {
		if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&medicalRecord).Updates(record).Error; err != nil {
			h.logger.Error("Failed to update medical record fields", zap.Error(err))
			return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
		}
	}

	// Update SOAPRecord if provided
	if updatedRecord.SOAPRecord != nil {
		if medicalRecord.SOAPRecordID == 0 {
			// Create new SOAP record if it doesn't exist
			newSOAP := *updatedRecord.SOAPRecord
			if err := h.db.Create(&newSOAP).Error; err != nil {
				h.logger.Error("Failed to create SOAP record during update", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
			}
			medicalRecord.SOAPRecordID = newSOAP.ID
			if err := h.db.Save(&medicalRecord).Error; err != nil {
				h.logger.Error("Failed to link new SOAP record to medical record", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
			}
		} else {
			// Update existing SOAP record
			soapUpdates := make(map[string]interface{})
			if updatedRecord.SOAPRecord.Subjective != "" {
				soapUpdates["subjective"] = updatedRecord.SOAPRecord.Subjective
			}
			if updatedRecord.SOAPRecord.Objective != "" {
				soapUpdates["objective"] = updatedRecord.SOAPRecord.Objective
			}
			if updatedRecord.SOAPRecord.Assessment != "" {
				soapUpdates["assessment"] = updatedRecord.SOAPRecord.Assessment
			}
			if updatedRecord.SOAPRecord.Plan != "" {
				soapUpdates["plan"] = updatedRecord.SOAPRecord.Plan
			}

			if len(soapUpdates) > 0 {
				if err := h.db.Model(&clinical_models.SOAPRecord{}).Where("id = ?", medicalRecord.SOAPRecordID).Updates(soapUpdates).Error; err != nil {
					h.logger.Error("Failed to update SOAP record", zap.Error(err))
					return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
				}
			}
		}
	}

	// Update Prescription if provided
	if updatedRecord.Prescription != nil {
		if medicalRecord.PrescriptionID == nil {
			// Create new prescription if it doesn't exist
			newPrescription := *updatedRecord.Prescription
			if err := h.db.Create(&newPrescription).Error; err != nil {
				h.logger.Error("Failed to create prescription during update", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
			}
			medicalRecord.PrescriptionID = &newPrescription.ID
			if err := h.db.Save(&medicalRecord).Error; err != nil {
				h.logger.Error("Failed to link new prescription to medical record", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
			}
		} else {
			// Update existing prescription
			prescriptionUpdates := make(map[string]interface{})
			if updatedRecord.Prescription.Content != "" {
				prescriptionUpdates["content"] = updatedRecord.Prescription.Content
			}
			if updatedRecord.Prescription.Indications != "" {
				prescriptionUpdates["indications"] = updatedRecord.Prescription.Indications
			}

			if len(prescriptionUpdates) > 0 {
				if err := h.db.Model(&clinical_models.Prescription{}).Where("id = ?", *medicalRecord.PrescriptionID).Updates(prescriptionUpdates).Error; err != nil {
					h.logger.Error("Failed to update prescription", zap.Error(err))
					return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalRecordUpdateError)
				}
			}
		}
	}

	// Reload the medical record with updated SOAP and Prescription data
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	h.logger.Info("Medical record updated successfully", zap.Uint("id", medicalRecord.ID))
	return envelope.SuccessResponse(medicalRecord, "clinical.medical_record.update.success")
}

func (h *MedicalRecordHandler) UpdatePrescription(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid medical record ID format", core_errors.ErrAuthInvalidRequest)
	}

	var prescriptionData clinical_dto.UpdatePrescriptionDTO
	if err := c.ShouldBindJSON(&prescriptionData); err != nil {
		h.logger.Error("Invalid update prescription request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid prescription data", core_errors.ErrAuthInvalidRequest)
	}

	// Find medical record
	var medicalRecord clinical_models.MedicalRecord
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Medical record not found", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	// Update or create prescription
	if medicalRecord.PrescriptionID == nil {
		// Create new prescription
		newPrescription := clinical_models.Prescription{
			Content:     prescriptionData.Content,
			Indications: prescriptionData.Indications,
		}
		if err := h.db.Create(&newPrescription).Error; err != nil {
			h.logger.Error("Failed to create prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordUpdateError)
		}
		// Link prescription to medical record
		medicalRecord.PrescriptionID = &newPrescription.ID
		if err := h.db.Save(&medicalRecord).Error; err != nil {
			h.logger.Error("Failed to link new prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordUpdateError)
		}
	} else {
		// Update existing prescription
		if err := h.db.Model(&clinical_models.Prescription{}).Where("id = ?", *medicalRecord.PrescriptionID).Updates(map[string]interface{}{
			"content":     prescriptionData.Content,
			"indications": prescriptionData.Indications,
		}).Error; err != nil {
			h.logger.Error("Failed to update prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordUpdateError)
		}
	}

	// Reload medical record with updated prescription
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated medical record after prescription update", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalRecordNotFound)
	}

	h.logger.Info("Prescription updated successfully", zap.Uint("record_id", medicalRecord.ID))
	return envelope.SuccessResponse(medicalRecord, "clinical.prescription.update.success")
}
