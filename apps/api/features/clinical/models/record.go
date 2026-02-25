package clinical_models

import (
	"time"

	"gorm.io/gorm"
)

type MedicalRecord struct {
	gorm.Model
	TenantID                 uint          `json:"tenant_id"`
	Date                     time.Time     `json:"date"`
	NextAppointmentDate      *time.Time    `json:"next_appointment_date,omitempty"`
	NextAppointmentStartTime *string       `json:"next_appointment_start_time,omitempty"`
	NextAppointmentEndTime   *string       `json:"next_appointment_end_time,omitempty"`
	Motive                   string        `json:"motive"`
	Observation              string        `json:"observation"`
	PatientID                uint          `json:"patient_id"`
	SOAPRecordID             uint          `json:"soap_record_id"`
	SOAPRecord               SOAPRecord    `json:"soap_record" gorm:"foreignKey:SOAPRecordID"`
	PrescriptionID           *uint         `json:"prescription_id"`
	Prescription             *Prescription `json:"prescription,omitempty" gorm:"foreignKey:PrescriptionID;constraint:OnDelete:SET NULL;"`
}

type SOAPRecord struct {
	gorm.Model
	Subjective string `json:"subjective"`
	Objective  string `json:"objective"`
	Assessment string `json:"assessment"`
	Plan       string `json:"plan"`
}
