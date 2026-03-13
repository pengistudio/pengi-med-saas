package clinical_models

import "gorm.io/gorm"

type VitalSigns struct {
	gorm.Model
	MedicalRecordID uint     `json:"medical_record_id"`
	Weight          *float64 `json:"weight"`         // kg
	Height          *float64 `json:"height"`         // cm
	BloodPressure   string   `json:"blood_pressure"` // e.g. "120/80"
	Temperature     *float64 `json:"temperature"`    // °C
	HeartRate       *uint    `json:"heart_rate"`     // bpm
	O2Saturation    *uint    `json:"o2_saturation"`  // %
}
