package clinical_models

import (
	"time"

	"gorm.io/gorm"
)

type Appointment struct {
	gorm.Model
	TenantID  uint      `json:"tenant_id"`
	PatientID uint      `json:"patient_id"`
	Title     string    `json:"title"`
	Date      time.Time `json:"date"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	Location  string    `json:"location,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Status    string    `json:"status" gorm:"default:scheduled"`
	Patient   Patient   `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
}

func (Appointment) IsAuditable() bool { return true }
