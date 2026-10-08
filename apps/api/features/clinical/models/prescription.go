package clinical_models

import (
	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

type Prescription struct {
	gorm.Model
	Content         string                `json:"content"`
	Indications     string                `json:"indications"`
	MedicalRecordID uint                  `json:"medical_record_id"`
	Items           []PrescriptionItem    `json:"items" gorm:"foreignKey:PrescriptionID;constraint:OnDelete:CASCADE;"`
	DoctorID        *uint                 `json:"doctor_id" gorm:"index"` // set from the record, never from the client
	Doctor          *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	DocumentSignature
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

func (Prescription) IsAuditable() bool     { return true }
func (PrescriptionItem) IsAuditable() bool { return true }
