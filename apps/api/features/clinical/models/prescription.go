package clinical_models

import "gorm.io/gorm"

type Prescription struct {
	gorm.Model
	Content         string `json:"content"`
	Indications     string `json:"indications"`
	MedicalRecordID uint   `json:"medical_record_id"`
}
