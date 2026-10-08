package clinical_handlers

import (
	"encoding/json"
	"net/http"
	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MedicalRecordHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewMedicalRecordHandler(db *gorm.DB, logger *zap.Logger) *MedicalRecordHandler {
	return &MedicalRecordHandler{db: db, logger: logger}
}

// syncPatientClinicalHistoryFromFirstVisit copies the first-visit clinical
// fields (APP, APF, APQX, allergies, and the CIE-10/11 diagnosis) onto the
// Patient record so they keep surfacing in patient-card/list/PDF exports
// that read from Patient directly, even though secretaries no longer edit
// them via the patient forms.
func (h *MedicalRecordHandler) syncPatientClinicalHistoryFromFirstVisit(c *gin.Context, record *clinical_models.MedicalRecord) {
	if record.VisitType != "first" {
		return
	}

	updates := map[string]interface{}{}
	if record.APP != "" {
		updates["app"] = record.APP
	}
	if record.APF != "" {
		updates["apf"] = record.APF
	}
	if record.APQX != "" {
		updates["apqx"] = record.APQX
	}
	if record.Allergies != "" {
		updates["allergies"] = record.Allergies
	}
	if diagnosis := firstVisitDiagnosisText(record.Diagnoses); diagnosis != "" {
		updates["diagnosis"] = diagnosis
	}

	if len(updates) == 0 {
		return
	}

	if err := tenantdb.For(c, h.db).
		Model(&clinical_models.Patient{}).
		Where("id = ?", record.PatientID).
		Updates(updates).Error; err != nil {
		h.logger.Error("Failed to sync patient clinical history from first visit", zap.Error(err))
	}
}

// stringOrEmpty dereferences an optional string pointer, defaulting to "".
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// firstVisitDiagnosisText renders the CIE-10/11 diagnoses selected on a
// first-visit consultation into a human-readable string for Patient.Diagnosis.
func firstVisitDiagnosisText(diagnoses datatypes.JSON) string {
	if len(diagnoses) == 0 {
		return ""
	}
	var items []clinical_models.DiagnosisItem
	if err := json.Unmarshal(diagnoses, &items); err != nil || len(items) == 0 {
		return ""
	}
	titles := make([]string, 0, len(items))
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return strings.Join(titles, ", ")
}

func (h *MedicalRecordHandler) GetMedicalRecords(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	baseQuery := tenantdb.For(c, h.db).Model(&clinical_models.MedicalRecord{}).Where("patient_id = ?", id)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count medical records", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	var records []clinical_models.MedicalRecord
	if err := baseQuery.Preload("SOAPRecord").Preload("Prescription").Preload("Prescription.Items").Preload("VitalSigns").Order("created_at desc").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		h.logger.Error("Failed to fetch medical records", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	patientID := uint(id)
	audit.RecordAccess(h.db, c, "patients", patientID, &patientID)

	return envelope.PagedSuccessResponse(records, int(total), page, limit, "clinical.medical_record.list.success")
}

func (h *MedicalRecordHandler) GetMedicalRecord(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var record clinical_models.MedicalRecord
	if err := tenantdb.For(c, h.db).Preload("SOAPRecord").Preload("Prescription").Preload("Prescription.Items").Preload("VitalSigns").Preload("Patient").First(&record, id).Error; err != nil {
		h.logger.Error("Failed to fetch medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	audit.RecordAccess(h.db, c, "medical_records", record.ID, &record.PatientID)

	return envelope.SuccessResponse(record, "clinical.medical_record.found")
}

func (h *MedicalRecordHandler) CreateMedicalRecord(c *gin.Context) envelope.Response {
	var newRecord clinical_dto.CreateMedicalRecordDTO
	if err := c.ShouldBind(&newRecord); err != nil {
		h.logger.Error("Invalid create medical record request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	if !h.inTenant(c, &clinical_models.Patient{}, newRecord.PatientID) {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	if newRecord.AppointmentID != nil && !h.inTenant(c, &clinical_models.Appointment{}, *newRecord.AppointmentID) {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
	}

	nextAppointmentDate := (*time.Time)(nil)
	if newRecord.NextAppointmentDate != nil {
		t := time.Time(*newRecord.NextAppointmentDate)
		nextAppointmentDate = &t
	}

	nextAppointmentStatus := "pending"
	if newRecord.NextAppointmentStatus != nil {
		nextAppointmentStatus = *newRecord.NextAppointmentStatus
	}
	if nextAppointmentStatus == "not_required" {
		nextAppointmentDate = nil
	}

	record := &clinical_models.MedicalRecord{
		Date:                  time.Time(newRecord.Date),
		Motive:                newRecord.Motive,
		Observation:           *newRecord.Observation,
		PatientID:             newRecord.PatientID,
		AppointmentID:         newRecord.AppointmentID,
		NextAppointmentDate:   nextAppointmentDate,
		NextAppointmentStatus: nextAppointmentStatus,
		SOAPRecord:            newRecord.SOAPRecord,
		Diagnoses:             newRecord.Diagnoses,
		VisitType:             newRecord.VisitType,
		APP:                   stringOrEmpty(newRecord.APP),
		APF:                   stringOrEmpty(newRecord.APF),
		APQX:                  stringOrEmpty(newRecord.APQX),
		Allergies:             stringOrEmpty(newRecord.Allergies),
	}

	// Create prescription if provided
	if newRecord.Prescription != nil && (newRecord.Prescription.Content != "" || newRecord.Prescription.Indications != "" || len(newRecord.Prescription.Items) > 0) {
		record.Prescription = newRecord.Prescription
	}

	tenantID, exists := c.Get("tenant_id")
	if exists {
		record.TenantID = tenantID.(uint)
	}

	if err := tenantdb.For(c, h.db).Create(record).Error; err != nil {
		h.logger.Error("Failed to create medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordCreateError)
	}

	// Vital signs: the ones taken at triage are linked to the record; otherwise
	// the ones sent with it are created.
	if vitalSigns, err := h.saveRecordVitalSigns(c, record, newRecord.VitalSigns); err != nil {
		h.logger.Error("Failed to save vital signs", zap.Error(err))
		// Non-fatal: record was created, just log the error
	} else {
		record.VitalSigns = vitalSigns
	}

	h.syncPatientClinicalHistoryFromFirstVisit(c, record)

	h.logger.Info("Medical record created successfully", zap.Uint("id", record.ID))
	return envelope.SuccessResponse(record, "clinical.medical_record.create.success")
}

func (h *MedicalRecordHandler) UpdateMedicalRecord(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var medicalRecord clinical_models.MedicalRecord
	if err := tenantdb.For(c, h.db).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	var updatedRecord clinical_dto.UpdateMedicalRecordDTO
	if err := c.ShouldBind(&updatedRecord); err != nil {
		h.logger.Error("Invalid update medical record request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	// Build update map only with provided fields for MedicalRecord
	record := make(map[string]interface{})
	if updatedRecord.Date != nil {
		record["date"] = time.Time(*updatedRecord.Date)
	}
	if updatedRecord.AppointmentID != nil {
		if !h.inTenant(c, &clinical_models.Appointment{}, *updatedRecord.AppointmentID) {
			return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
		}
		record["appointment_id"] = *updatedRecord.AppointmentID
		// Auto-complete the linked appointment
		tenantdb.For(c, h.db).Model(&clinical_models.Appointment{}).Where("id = ?", *updatedRecord.AppointmentID).Update("status", "completed")
	}
	if updatedRecord.Motive != nil {
		record["motive"] = *updatedRecord.Motive
	}
	if updatedRecord.Observation != nil {
		record["observation"] = *updatedRecord.Observation
	}
	if updatedRecord.NextAppointmentDate != nil {
		record["next_appointment_date"] = time.Time(*updatedRecord.NextAppointmentDate)
	}
	if updatedRecord.NextAppointmentStatus != nil {
		record["next_appointment_status"] = *updatedRecord.NextAppointmentStatus
		if *updatedRecord.NextAppointmentStatus == "not_required" {
			record["next_appointment_date"] = nil
		}
	}
	if updatedRecord.Diagnoses != nil {
		record["diagnoses"] = updatedRecord.Diagnoses
	}
	if updatedRecord.APP != nil {
		record["app"] = *updatedRecord.APP
	}
	if updatedRecord.APF != nil {
		record["apf"] = *updatedRecord.APF
	}
	if updatedRecord.APQX != nil {
		record["apqx"] = *updatedRecord.APQX
	}
	if updatedRecord.Allergies != nil {
		record["allergies"] = *updatedRecord.Allergies
	}

	// Update MedicalRecord fields
	if len(record) > 0 {
		if err := tenantdb.For(c, h.db).Model(&medicalRecord).Updates(record).Error; err != nil {
			h.logger.Error("Failed to update medical record fields", zap.Error(err))
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
		}
	}

	// Update SOAPRecord if provided
	if updatedRecord.SOAPRecord != nil {
		if medicalRecord.SOAPRecordID == 0 {
			// Create new SOAP record if it doesn't exist
			newSOAP := *updatedRecord.SOAPRecord
			if err := tenantdb.For(c, h.db).Create(&newSOAP).Error; err != nil {
				h.logger.Error("Failed to create SOAP record during update", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
			}
			medicalRecord.SOAPRecordID = newSOAP.ID
			if err := tenantdb.For(c, h.db).Save(&medicalRecord).Error; err != nil {
				h.logger.Error("Failed to link new SOAP record to medical record", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
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
				if err := tenantdb.For(c, h.db).Model(&clinical_models.SOAPRecord{}).Where("id = ?", medicalRecord.SOAPRecordID).Updates(soapUpdates).Error; err != nil {
					h.logger.Error("Failed to update SOAP record", zap.Error(err))
					return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
				}
			}
		}
	}

	// Update Prescription if provided
	if updatedRecord.Prescription != nil {
		if medicalRecord.PrescriptionID == nil {
			// Create new prescription if it doesn't exist
			newPrescription := *updatedRecord.Prescription
			if err := tenantdb.For(c, h.db).Create(&newPrescription).Error; err != nil {
				h.logger.Error("Failed to create prescription during update", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
			}
			medicalRecord.PrescriptionID = &newPrescription.ID
			if err := tenantdb.For(c, h.db).Save(&medicalRecord).Error; err != nil {
				h.logger.Error("Failed to link new prescription to medical record", zap.Error(err))
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
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
				if prescriptionChanged(medicalRecord.Prescription, prescriptionUpdates) {
					for k, v := range clearedSignature() {
						prescriptionUpdates[k] = v
					}
				}
				if err := tenantdb.For(c, h.db).Model(&clinical_models.Prescription{}).Where("id = ?", *medicalRecord.PrescriptionID).Updates(prescriptionUpdates).Error; err != nil {
					h.logger.Error("Failed to update prescription", zap.Error(err))
					return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalRecordUpdateError)
				}
			}
		}
	}

	// Reload the medical record with updated SOAP and Prescription data
	if err := tenantdb.For(c, h.db).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated medical record", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	h.logger.Info("Medical record updated successfully", zap.Uint("id", medicalRecord.ID))
	return envelope.SuccessResponse(medicalRecord, "clinical.medical_record.update.success")
}

func (h *MedicalRecordHandler) UpdatePrescription(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid medical record ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var prescriptionData clinical_dto.UpdatePrescriptionDTO
	if err := c.ShouldBindJSON(&prescriptionData); err != nil {
		h.logger.Error("Invalid update prescription request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	// Find medical record
	var medicalRecord clinical_models.MedicalRecord
	if err := tenantdb.For(c, h.db).Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Medical record not found", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}

	// Update or create prescription
	if medicalRecord.PrescriptionID == nil {
		// Create new prescription
		newPrescription := clinical_models.Prescription{
			Content:     prescriptionData.Content,
			Indications: prescriptionData.Indications,
		}
		if err := tenantdb.For(c, h.db).Create(&newPrescription).Error; err != nil {
			h.logger.Error("Failed to create prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordUpdateError)
		}
		// Link prescription to medical record
		medicalRecord.PrescriptionID = &newPrescription.ID
		if err := tenantdb.For(c, h.db).Save(&medicalRecord).Error; err != nil {
			h.logger.Error("Failed to link new prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordUpdateError)
		}
	} else {
		// Update existing prescription
		updates := map[string]interface{}{
			"content":     prescriptionData.Content,
			"indications": prescriptionData.Indications,
		}
		if prescriptionChanged(medicalRecord.Prescription, updates) {
			for k, v := range clearedSignature() {
				updates[k] = v
			}
		}
		if err := tenantdb.For(c, h.db).Model(&clinical_models.Prescription{}).Where("id = ?", *medicalRecord.PrescriptionID).Updates(updates).Error; err != nil {
			h.logger.Error("Failed to update prescription", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordUpdateError)
		}
	}

	// Reload medical record with updated prescription
	if err := tenantdb.For(c, h.db).Preload("SOAPRecord").Preload("Prescription").First(&medicalRecord, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated medical record after prescription update", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalRecordNotFound)
	}

	h.logger.Info("Prescription updated successfully", zap.Uint("record_id", medicalRecord.ID))
	return envelope.SuccessResponse(medicalRecord, "clinical.prescription.update.success")
}

// inTenant reports whether the row with this ID in model's table belongs to the
// caller's tenant; client-supplied references (patient, appointment) must never
// reach another clinic's data.
// saveRecordVitalSigns stores a new record's vital signs. When the record
// comes from an appointment whose vital signs were taken at triage, that same
// row is linked to the record, updated with the measurements sent (the doctor
// may correct them); otherwise the measurements sent, if any, are created.
// Returns the record's vital signs, or nil when there are none.
func (h *MedicalRecordHandler) saveRecordVitalSigns(c *gin.Context, record *clinical_models.MedicalRecord, input *clinical_dto.VitalSignsInput) (*clinical_models.VitalSigns, error) {
	db := tenantdb.For(c, h.db)
	var measurements clinical_models.VitalSigns
	if input != nil {
		measurements = input.Measurements()
	}
	measurements.MedicalRecordID = &record.ID

	if record.AppointmentID != nil {
		var triage clinical_models.VitalSigns
		err := db.Where("appointment_id = ? AND medical_record_id IS NULL", *record.AppointmentID).First(&triage).Error
		if err == nil {
			if err := db.Model(&triage).Updates(&measurements).Error; err != nil {
				return nil, err
			}
			return &triage, db.First(&triage, triage.ID).Error
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	if input == nil {
		return nil, nil
	}
	measurements.AppointmentID = record.AppointmentID
	return &measurements, db.Create(&measurements).Error
}

func (h *MedicalRecordHandler) inTenant(c *gin.Context, model any, id uint) bool {
	var count int64
	tenantdb.For(c, h.db).Model(model).Where("id = ?", id).Count(&count)
	return count == 1
}
