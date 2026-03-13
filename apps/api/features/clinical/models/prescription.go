package clinical_models

import "gorm.io/gorm"

type Prescription struct {
	gorm.Model
	Content         string             `json:"content"`
	Indications     string             `json:"indications"`
	MedicalRecordID uint               `json:"medical_record_id"`
	Items           []PrescriptionItem `json:"items" gorm:"foreignKey:PrescriptionID;constraint:OnDelete:CASCADE;"`
}

type PrescriptionItem struct {
	gorm.Model
	PrescriptionID uint   `json:"prescription_id"`
	Medication     string `json:"medication"` // e.g. "Amoxicilina 500mg"
	Dose           string `json:"dose"`       // e.g. "1 tableta"
	Frequency      string `json:"frequency"`  // e.g. "cada 8 horas"
	Duration       string `json:"duration"`   // e.g. "7 días"
	Notes          string `json:"notes"`
}
