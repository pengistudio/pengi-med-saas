package clinical_handlers

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_services "pengi-med-saas/features/doctors/services"
)

// defaultDoctorName is printed when a document has no doctor at all.
const defaultDoctorName = "Médico Tratante"

// doctorErrorResponse answers an error from doctor_services (resolution,
// validation, signing), or a 500 for anything else.
func doctorErrorResponse(logger *zap.Logger, err error) envelope.Response {
	if resp, ok := doctor_services.ErrorResponse(err); ok {
		return resp
	}
	logger.Error("failed to resolve doctor", zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
}

// documentDoctor is the doctor name and professional registry a document
// prints: the first doctor of ids (the document's, then e.g. its record's),
// then the patient's cabecera, then the legacy patient.Medic text, then
// defaultDoctorName. db must be bound to the tenant.
func documentDoctor(db *gorm.DB, patient *clinical_models.Patient, ids ...*uint) (name, registry string) {
	if patient != nil {
		ids = append(ids, patient.DoctorID)
	}
	if d := doctor_services.First(db, ids...); d != nil {
		return d.FullName, d.ProfessionalRegistry
	}
	if patient != nil && strings.TrimSpace(patient.Medic) != "" {
		return strings.TrimSpace(patient.Medic), ""
	}
	return defaultDoctorName, ""
}

// recordDoctorID is the doctor of medical record id (tenant-bound db), or nil.
func recordDoctorID(db *gorm.DB, id *uint) *uint {
	if id == nil {
		return nil
	}
	var record clinical_models.MedicalRecord
	if err := db.Select("id", "doctor_id").Limit(1).Find(&record, *id).Error; err != nil || record.ID == 0 {
		return nil
	}
	return record.DoctorID
}

// patientDoctorID is the cabecera of patient id (tenant-bound db), or nil.
func patientDoctorID(db *gorm.DB, id uint) *uint {
	var patient clinical_models.Patient
	if err := db.Select("id", "doctor_id").Limit(1).Find(&patient, id).Error; err != nil || patient.ID == 0 {
		return nil
	}
	return patient.DoctorID
}
