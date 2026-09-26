package clinical_models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MedicalReportConsultationEntry struct {
	MedicalRecordID uint      `json:"medical_record_id"`
	Date            time.Time `json:"date"`
	Motive          string    `json:"motive"`
	Summary         string    `json:"summary"`
}

// MedicalReport is an editable snapshot of a patient's consultation history,
// generated on demand for printing/emailing. It does not mutate the
// underlying MedicalRecord/SOAPRecord data it was seeded from.
type MedicalReport struct {
	gorm.Model
	TenantID      uint           `json:"tenant_id"`
	PatientID     uint           `json:"patient_id"`
	Patient       *Patient       `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	Consultations datatypes.JSON `json:"consultations" gorm:"type:jsonb;default:'[]'"`
	Plan          string         `json:"plan"`
	GeneratedByID uint           `json:"generated_by_id"`
}

func (MedicalReport) IsAuditable() bool { return true }
