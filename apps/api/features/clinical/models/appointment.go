package clinical_models

import (
	"time"

	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

type Appointment struct {
	gorm.Model
	TenantID      uint                  `json:"tenant_id"`
	PatientID     uint                  `json:"patient_id"`
	Title         string                `json:"title"`
	Date          time.Time             `json:"date"`
	StartTime     string                `json:"start_time"`
	EndTime       string                `json:"end_time"`
	Location      string                `json:"location,omitempty"`
	Notes         string                `json:"notes,omitempty"`
	Status        string                `json:"status" gorm:"default:scheduled"`
	ColorID       string                `json:"color_id,omitempty" gorm:"default:''"`
	GoogleEventID string                `json:"google_event_id,omitempty"`
	Patient       Patient               `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	DoctorID      *uint                 `json:"doctor_id" gorm:"index"`
	Doctor        *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	// VitalSigns taken at triage; loaded by the waiting room list.
	VitalSigns *VitalSigns `json:"vital_signs,omitempty" gorm:"foreignKey:AppointmentID"`
}

func (Appointment) IsAuditable() bool { return true }
