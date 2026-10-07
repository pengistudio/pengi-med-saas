package dto

import clinical_models "pengi-med-saas/features/clinical/models"

// VitalSignsInput is what a client may send for a consultation's vital signs.
// Binding the model instead would let the client set ID, timestamps and the
// soft-delete column (gorm.Model).
type VitalSignsInput struct {
	Weight        *float64 `json:"weight"`         // kg
	Height        *float64 `json:"height"`         // cm
	BloodPressure string   `json:"blood_pressure"` // e.g. "120/80"
	Temperature   *float64 `json:"temperature"`    // °C
	HeartRate     *uint    `json:"heart_rate"`     // bpm
	O2Saturation  *uint    `json:"o2_saturation"`  // %
}

// Model returns the vital signs of the medical record recordID.
func (v VitalSignsInput) Model(recordID uint) clinical_models.VitalSigns {
	return clinical_models.VitalSigns{
		MedicalRecordID: recordID,
		Weight:          v.Weight,
		Height:          v.Height,
		BloodPressure:   v.BloodPressure,
		Temperature:     v.Temperature,
		HeartRate:       v.HeartRate,
		O2Saturation:    v.O2Saturation,
	}
}
