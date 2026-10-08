package clinical_models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

type MedicalReportVitalSigns struct {
	Weight        *float64 `json:"weight,omitempty"`
	Height        *float64 `json:"height,omitempty"`
	BloodPressure string   `json:"blood_pressure,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	HeartRate     *uint    `json:"heart_rate,omitempty"`
	O2Saturation  *uint    `json:"o2_saturation,omitempty"`
}

type MedicalReportPrescriptionItem struct {
	Medication string `json:"medication"`
	Dose       string `json:"dose"`
	Frequency  string `json:"frequency"`
	Duration   string `json:"duration"`
	Notes      string `json:"notes"`
}

type MedicalReportPrescription struct {
	Indications string                          `json:"indications"`
	Items       []MedicalReportPrescriptionItem `json:"items"`
}

// MedicalReportConsultationEntry is the snapshot of one consultation inside a
// report. Summary is only set on reports created before the full snapshot.
type MedicalReportConsultationEntry struct {
	MedicalRecordID uint                       `json:"medical_record_id"`
	Date            time.Time                  `json:"date"`
	Motive          string                     `json:"motive"`
	VisitType       string                     `json:"visit_type,omitempty"`
	Observation     string                     `json:"observation,omitempty"`
	Subjective      string                     `json:"subjective,omitempty"`
	Objective       string                     `json:"objective,omitempty"`
	Assessment      string                     `json:"assessment,omitempty"`
	Plan            string                     `json:"plan,omitempty"`
	APP             string                     `json:"app,omitempty"`
	APF             string                     `json:"apf,omitempty"`
	APQX            string                     `json:"apqx,omitempty"`
	Allergies       string                     `json:"allergies,omitempty"`
	Diagnoses       []DiagnosisItem            `json:"diagnoses,omitempty"`
	VitalSigns      *MedicalReportVitalSigns   `json:"vital_signs,omitempty"`
	Prescription    *MedicalReportPrescription `json:"prescription,omitempty"`
	Summary         string                     `json:"summary,omitempty"`
}

// MedicalReport is an editable snapshot of a patient's consultation history,
// generated on demand for printing/emailing. It does not mutate the
// underlying MedicalRecord/SOAPRecord data it was seeded from.
type MedicalReport struct {
	gorm.Model
	TenantID      uint                  `json:"tenant_id"`
	PatientID     uint                  `json:"patient_id"`
	Patient       *Patient              `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	Consultations datatypes.JSON        `json:"consultations" gorm:"type:jsonb;default:'[]'"`
	Plan          string                `json:"plan"`
	GeneratedByID uint                  `json:"generated_by_id"`
	DoctorID      *uint                 `json:"doctor_id" gorm:"index"`
	Doctor        *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	DocumentSignature
}

func (MedicalReport) IsAuditable() bool { return true }
