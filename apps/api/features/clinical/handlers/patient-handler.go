package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type PatientHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPatientHandler(db *gorm.DB, logger *zap.Logger) *PatientHandler {
	return &PatientHandler{db: db, logger: logger}
}

func (h *PatientHandler) CreatePatient(c *gin.Context) envelope.Response {
	var newPatient clinical_dto.CreatePatientDTO
	if err := c.ShouldBind(&newPatient); err != nil {
		h.logger.Error("Invalid create patient request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	birthDate := time.Time{}
	if newPatient.BirthDate != nil {
		birthDate = *newPatient.BirthDate
	}

	fullName := newPatient.FirstName + " " + newPatient.LastName

	patient := &clinical_models.Patient{
		Document:    newPatient.Document,
		Phone:       newPatient.Phone,
		FirstName:   newPatient.FirstName,
		LastName:    newPatient.LastName,
		FullName:    &fullName,
		BirthDate:   birthDate,
		Institution: newPatient.Institution,
		Gender:      newPatient.Gender,
		Notes:       newPatient.Notes,
		Insurance:   newPatient.Insurance,
		Medic:       newPatient.Medic,
		Diagnosis:   newPatient.Diagnosis,
		APP:         newPatient.APP,
		APF:         newPatient.APF,
		APQX:        newPatient.APQX,
	}

	tenantID, exists := c.Get("tenant_id")
	if exists {
		patient.TenantID = tenantID.(uint)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(patient).Error; err != nil {
		h.logger.Error("Failed to create patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientCreateError)
	}

	h.logger.Info("Patient created successfully", zap.Uint("id", patient.ID))
	return envelope.SuccessResponse(patient, "clinical.patient.create.success")
}

func (h *PatientHandler) UpdatePatient(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	var updateData clinical_dto.UpdatePatientDTO
	if err := c.ShouldBind(&updateData); err != nil {
		h.logger.Error("Invalid update patient request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to find patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	// Build update map
	updates := make(map[string]interface{})

	if updateData.Document != nil {
		updates["document"] = *updateData.Document
	}
	if updateData.Phone != nil {
		updates["phone"] = *updateData.Phone
	}
	if updateData.FirstName != nil {
		updates["first_name"] = *updateData.FirstName
	}
	if updateData.LastName != nil {
		updates["last_name"] = *updateData.LastName
	}
	if updateData.FirstName != nil || updateData.LastName != nil {
		firstName := patient.FirstName
		lastName := patient.LastName
		if updateData.FirstName != nil {
			firstName = *updateData.FirstName
		}
		if updateData.LastName != nil {
			lastName = *updateData.LastName
		}
		fullName := firstName + " " + lastName
		updates["full_name"] = fullName
	}
	if updateData.BirthDate != nil {
		updates["birth_date"] = *updateData.BirthDate
	}
	if updateData.Institution != nil {
		updates["institution"] = *updateData.Institution
	}
	if updateData.Gender != nil {
		updates["gender"] = *updateData.Gender
	}
	if updateData.Notes != nil {
		updates["notes"] = *updateData.Notes
	}
	if updateData.Insurance != nil {
		updates["insurance"] = *updateData.Insurance
	}
	if updateData.Medic != nil {
		updates["medic"] = *updateData.Medic
	}
	if updateData.Diagnosis != nil {
		updates["diagnosis"] = *updateData.Diagnosis
	}
	if updateData.APP != nil {
		updates["app"] = *updateData.APP
	}
	if updateData.APF != nil {
		updates["apf"] = *updateData.APF
	}
	if updateData.APQX != nil {
		updates["apqx"] = *updateData.APQX
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&patient).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientUpdateError)
	}

	// Reload patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	h.logger.Info("Patient updated successfully", zap.Uint("id", patient.ID))
	return envelope.SuccessResponse(patient, "clinical.patient.update.success")
}

func (h *PatientHandler) GetAllPatients(c *gin.Context) envelope.Response {
	var patients []clinical_models.Patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Find(&patients).Error; err != nil {
		h.logger.Error("Failed to fetch patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}
	return envelope.SuccessResponse(patients, "clinical.patient.list.success")
}

func (h *PatientHandler) GetAllPatientsWithLastFollowUp(c *gin.Context) envelope.Response {
	var patients []clinical_models.Patient
	err := h.db.
		Scopes(tenant_middleware.TenantScope(c)).
		Preload("MedicalRecords", func(db *gorm.DB) *gorm.DB {
			return db.Order("date DESC").Limit(1)
		}).
		Model(&clinical_models.Patient{}).
		Find(&patients).Error

	if err != nil {
		h.logger.Error("Failed to fetch patients with followup", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	sort.SliceStable(patients, func(i, j int) bool {
		return patients[i].Critical && !patients[j].Critical
	})

	return envelope.SuccessResponse(patients, "clinical.patient.followup.success")
}

func (h *PatientHandler) DeleteMultiplePatients(c *gin.Context) envelope.Response {
	var deleteJSON clinical_dto.DeletePatientsDTO
	if err := c.ShouldBind(&deleteJSON); err != nil {
		h.logger.Error("Invalid delete multiple request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&clinical_models.Patient{}).Where("id IN (?)", deleteJSON.IdList).Delete(&clinical_models.Patient{}).Error; err != nil {
		h.logger.Error("Failed to delete patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientDeleteError)
	}

	// Returning the remaining list like the original logic did
	return h.GetAllPatientsWithLastFollowUp(c)
}

func (h *PatientHandler) DeleteOnePatient(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Where("id = ?", id).Delete(&clinical_models.Patient{}).Error; err != nil {
		h.logger.Error("Failed to delete patient", zap.Uint64("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientDeleteError)
	}

	return h.GetAllPatientsWithLastFollowUp(c)
}

func (h *PatientHandler) GetPatientByID(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to find patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	return envelope.SuccessResponse(patient, "clinical.patient.found")
}

func (h *PatientHandler) UpdatePatientCritical(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to find patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&patient).Update("critical", true).Error; err != nil {
		h.logger.Error("Failed to update patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientUpdateError)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	h.logger.Info("Patient marked as critical",
		zap.Uint("patient_id", patient.ID),
		zap.String("patient_name", patient.FirstName+" "+patient.LastName),
	)

	return envelope.SuccessResponse(patient, "clinical.patient.critical.success")
}

func (h *PatientHandler) UpdatePatientCriticalRevert(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		h.logger.Error("Invalid patient ID", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to find patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&patient).Update("critical", false).Error; err != nil {
		h.logger.Error("Failed to update patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientUpdateError)
	}

	if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&patient, id).Error; err != nil {
		h.logger.Error("Failed to fetch updated patient", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalPatientNotFound)
	}

	h.logger.Info("Patient critical status reverted",
		zap.Uint("patient_id", patient.ID),
		zap.String("patient_name", patient.FirstName+" "+patient.LastName),
	)

	return envelope.SuccessResponse(patient, "clinical.patient.critical_revert.success")
}
