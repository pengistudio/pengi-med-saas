package clinical_models

import (
	"time"

	"gorm.io/gorm"
)

type Patient struct {
	gorm.Model
	TenantID       uint            `json:"tenant_id"`
	Document       string          `json:"document"`
	Phone          string          `json:"phone"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	FullName       *string         `json:"full_name"`
	BirthDate      time.Time       `json:"birth_date"`
	Institution    string          `json:"institution"`
	Gender         string          `json:"gender"`
	Notes          string          `json:"notes"`
	Insurance      string          `json:"insurance"`
	Medic          string          `json:"medic"`
	Diagnosis      string          `json:"diagnosis"`
	Critical       bool            `json:"critical"`
	APP            string          `json:"app"`  // Antecedentes Personales Patológicos
	APF            string          `json:"apf"`  // Antecedentes Patológicos Familiares
	APQX           string          `json:"apqx"` // Antecedentes Patológicos Quirúrgicos
	MedicalRecords []MedicalRecord `json:"medical_records" gorm:"foreignKey:PatientID;constraint:OnDelete:CASCADE;"`
	Appointments   []Appointment   `json:"appointments,omitempty" gorm:"foreignKey:PatientID"`
}
