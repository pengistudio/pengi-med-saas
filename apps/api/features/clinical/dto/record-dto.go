package dto

import (
	"encoding/json"
	clinical_models "pengi-med-saas/features/clinical/models"
	"time"

	"gorm.io/datatypes"
)

type CustomDateTime time.Time

func (ct *CustomDateTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Try multiple date formats
	for _, format := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(format, s); err == nil {
			*ct = CustomDateTime(t.UTC())
			return nil
		}
	}

	return &json.SyntaxError{}
}

type CreateMedicalRecordDTO struct {
	Date                CustomDateTime                `json:"date" binding:"required"`
	Motive              string                        `json:"motive" binding:"required"`
	Observation         *string                       `json:"observation"`
	PatientID           uint                          `json:"patient_id" binding:"required"`
	AppointmentID       *uint                         `json:"appointment_id,omitempty"`
	NextAppointmentDate *CustomDateTime               `json:"next_appointment_date,omitempty"`
	SOAPRecord          clinical_models.SOAPRecord    `json:"soap_record"`
	Prescription        *clinical_models.Prescription `json:"prescription,omitempty"`
	VitalSigns          *clinical_models.VitalSigns   `json:"vital_signs,omitempty"`
	Diagnoses           datatypes.JSON                `json:"diagnoses,omitempty"`
}

type UpdateMedicalRecordDTO struct {
	Date                *CustomDateTime               `json:"date,omitempty"`
	Motive              *string                       `json:"motive,omitempty"`
	Observation         *string                       `json:"observation,omitempty"`
	AppointmentID       *uint                         `json:"appointment_id,omitempty"`
	NextAppointmentDate *CustomDateTime               `json:"next_appointment_date,omitempty"`
	SOAPRecord          *clinical_models.SOAPRecord   `json:"soap_record,omitempty"`
	Prescription        *clinical_models.Prescription `json:"prescription,omitempty"`
	Diagnoses           datatypes.JSON                `json:"diagnoses,omitempty"`
}

type UpdatePrescriptionDTO struct {
	Content     string `json:"content" binding:"required"`
	Indications string `json:"indications" binding:"required"`
}
