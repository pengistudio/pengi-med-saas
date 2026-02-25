package clinical_models

import (
	"time"

	"gorm.io/gorm"
)

type MedicalRecord struct {
	gorm.Model
	TenantID       uint          `json:"tenant_id"`
	Date           time.Time     `json:"date"`
	Motive         string        `json:"motive"`
	Observation    string        `json:"observation"`
	PatientID      uint          `json:"patient_id"`
	AppointmentID  *uint         `json:"appointment_id,omitempty"`
	Appointment    *Appointment  `json:"appointment,omitempty" gorm:"foreignKey:AppointmentID"`
	SOAPRecordID   uint          `json:"soap_record_id"`
	SOAPRecord     SOAPRecord    `json:"soap_record" gorm:"foreignKey:SOAPRecordID"`
	PrescriptionID *uint         `json:"prescription_id"`
	Prescription   *Prescription `json:"prescription,omitempty" gorm:"foreignKey:PrescriptionID;constraint:OnDelete:SET NULL;"`
}

type SOAPRecord struct {
	gorm.Model
	Subjective string `json:"subjective"`
	Objective  string `json:"objective"`
	Assessment string `json:"assessment"`
	Plan       string `json:"plan"`
}
