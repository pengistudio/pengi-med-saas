package clinical_models

import "gorm.io/gorm"

// VitalSigns are a consultation's measurements. Triage takes them on the
// appointment (AppointmentID) before the medical record exists; creating the
// record from that appointment links the same row to it (MedicalRecordID).
type VitalSigns struct {
	gorm.Model
	MedicalRecordID *uint    `json:"medical_record_id"`
	AppointmentID   *uint    `json:"appointment_id" gorm:"index"`
	Weight          *float64 `json:"weight"`         // kg
	Height          *float64 `json:"height"`         // cm
	BloodPressure   string   `json:"blood_pressure"` // e.g. "120/80"
	Temperature     *float64 `json:"temperature"`    // °C
	HeartRate       *uint    `json:"heart_rate"`     // bpm
	O2Saturation    *uint    `json:"o2_saturation"`  // %
}

func (VitalSigns) IsAuditable() bool { return true }
