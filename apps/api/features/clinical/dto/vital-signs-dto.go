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

// Measurements returns the vital signs without an owner; the caller sets the
// medical record or appointment they belong to.
func (v VitalSignsInput) Measurements() clinical_models.VitalSigns {
	return clinical_models.VitalSigns{
		Weight:        v.Weight,
		Height:        v.Height,
		BloodPressure: v.BloodPressure,
		Temperature:   v.Temperature,
		HeartRate:     v.HeartRate,
		O2Saturation:  v.O2Saturation,
	}
}
