package clinical_models

import (
	"time"

	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

// MedicalCertificate is a standalone document (diagnosis + rest period) that
// does not derive from or mutate any MedicalRecord.
type MedicalCertificate struct {
	gorm.Model
	TenantID      uint                  `json:"tenant_id"`
	PatientID     uint                  `json:"patient_id"`
	Patient       *Patient              `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	Diagnosis     string                `json:"diagnosis"`
	Observations  string                `json:"observations"`
	RestDays      *int                  `json:"rest_days,omitempty"`
	RestFrom      *time.Time            `json:"rest_from,omitempty" gorm:"type:date"`
	RestTo        *time.Time            `json:"rest_to,omitempty" gorm:"type:date"`
	GeneratedByID uint                  `json:"generated_by_id"`
	DoctorID      *uint                 `json:"doctor_id" gorm:"index"`
	Doctor        *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	DocumentSignature
}

func (MedicalCertificate) IsAuditable() bool { return true }
