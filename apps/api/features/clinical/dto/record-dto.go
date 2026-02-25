package dto

import (
	clinical_models "pengi-med-saas/features/clinical/models"
	"time"
)

type CreateMedicalRecordDTO struct {
	Date                     time.Time                     `json:"date" binding:"required"`
	NextAppointmentDate      *time.Time                    `json:"next_appointment_date,omitempty"`
	NextAppointmentStartTime *string                       `json:"next_appointment_start_time,omitempty"`
	NextAppointmentEndTime   *string                       `json:"next_appointment_end_time,omitempty"`
	Motive                   string                        `json:"motive" binding:"required"`
	Observation              string                        `json:"observation" binding:"required"`
	PatientID                uint                          `json:"patient_id" binding:"required"`
	SOAPRecord               clinical_models.SOAPRecord    `json:"soap_record"`
	Prescription             *clinical_models.Prescription `json:"prescription,omitempty"`
}

type UpdateMedicalRecordDTO struct {
	Date                     *time.Time                    `json:"date,omitempty"`
	NextAppointmentDate      *time.Time                    `json:"next_appointment_date,omitempty"`
	NextAppointmentStartTime *string                       `json:"next_appointment_start_time,omitempty"`
	NextAppointmentEndTime   *string                       `json:"next_appointment_end_time,omitempty"`
	Motive                   *string                       `json:"motive,omitempty"`
	Observation              *string                       `json:"observation,omitempty"`
	SOAPRecord               *clinical_models.SOAPRecord   `json:"soap_record,omitempty"`
	Prescription             *clinical_models.Prescription `json:"prescription,omitempty"`
}

type UpdatePrescriptionDTO struct {
	Content     string `json:"content" binding:"required"`
	Indications string `json:"indications" binding:"required"`
}
