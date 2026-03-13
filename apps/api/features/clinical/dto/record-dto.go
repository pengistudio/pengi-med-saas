package dto

import (
	clinical_models "pengi-med-saas/features/clinical/models"
	"time"

	"gorm.io/datatypes"
)

type CreateMedicalRecordDTO struct {
	Date          time.Time                     `json:"date" binding:"required"`
	Motive        string                        `json:"motive" binding:"required"`
	Observation   *string                       `json:"observation"`
	PatientID     uint                          `json:"patient_id" binding:"required"`
	AppointmentID *uint                         `json:"appointment_id,omitempty"`
	SOAPRecord    clinical_models.SOAPRecord    `json:"soap_record"`
	Prescription  *clinical_models.Prescription `json:"prescription,omitempty"`
	VitalSigns    *clinical_models.VitalSigns   `json:"vital_signs,omitempty"`
	Diagnoses     datatypes.JSON                `json:"diagnoses,omitempty"`
}

type UpdateMedicalRecordDTO struct {
	Date          *time.Time                    `json:"date,omitempty"`
	Motive        *string                       `json:"motive,omitempty"`
	Observation   *string                       `json:"observation,omitempty"`
	AppointmentID *uint                         `json:"appointment_id,omitempty"`
	SOAPRecord    *clinical_models.SOAPRecord   `json:"soap_record,omitempty"`
	Prescription  *clinical_models.Prescription `json:"prescription,omitempty"`
	Diagnoses     datatypes.JSON                `json:"diagnoses,omitempty"`
}

type UpdatePrescriptionDTO struct {
	Content     string `json:"content" binding:"required"`
	Indications string `json:"indications" binding:"required"`
}
