package clinical_models

import (
	"time"

	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

type Patient struct {
	gorm.Model
	TenantID uint   `json:"tenant_id"`
	Document string `json:"document"`
	Phone    string `json:"phone"`
	// WhatsAppOptIn: the patient agreed to receive WhatsApp reminders (Meta
	// requires opt-in). Reminders only go to patients with it set; replying
	// STOP clears it. WhatsAppOptInAt is when it was last set or cleared.
	WhatsAppOptIn   bool       `json:"whatsapp_opt_in" gorm:"column:whatsapp_opt_in;not null;default:false"`
	WhatsAppOptInAt *time.Time `json:"whatsapp_opt_in_at" gorm:"column:whatsapp_opt_in_at"`
	// WhatsAppOptInSource is how consent was last set or cleared
	// (WhatsAppOptInSource* values); empty for changes made before it existed.
	WhatsAppOptInSource string                `json:"whatsapp_opt_in_source" gorm:"column:whatsapp_opt_in_source;not null;default:''"`
	Email               string                `json:"email"`
	FirstName           string                `json:"first_name"`
	LastName            string                `json:"last_name"`
	FullName            *string               `json:"full_name"`
	BirthDate           time.Time             `json:"birth_date"`
	BirthDateEstimated  bool                  `json:"birth_date_estimated" gorm:"not null;default:false"` // derived from a reported age: only the age is meaningful
	Institution         string                `json:"institution"`
	Gender              string                `json:"gender"`
	Notes               string                `json:"notes"`
	Insurance           string                `json:"insurance"`
	Medic               string                `json:"medic"`
	DoctorID            *uint                 `json:"doctor_id" gorm:"index"` // médico de cabecera; Medic is the legacy free-text name
	Doctor              *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	Diagnosis           string                `json:"diagnosis"`
	Critical            bool                  `json:"critical"`
	APP                 string                `json:"app"`       // Antecedentes Personales Patológicos
	APF                 string                `json:"apf"`       // Antecedentes Patológicos Familiares
	APQX                string                `json:"apqx"`      // Antecedentes Patológicos Quirúrgicos
	Allergies           string                `json:"allergies"` // JSON array of allergy strings
	MedicalRecords      []MedicalRecord       `json:"medical_records" gorm:"foreignKey:PatientID;constraint:OnDelete:CASCADE;"`
	Appointments        []Appointment         `json:"appointments,omitempty" gorm:"foreignKey:PatientID"`
}

// Origins of WhatsApp consent (Patient.WhatsAppOptInSource).
const (
	WhatsAppOptInSourceRegistration = "registration"  // checked when the patient was created
	WhatsAppOptInSourceManual       = "manual"        // changed on the patient form
	WhatsAppOptInSourceBulk         = "bulk"          // set from the patient list selection
	WhatsAppOptInSourceWhatsApp     = "whatsapp"      // the patient replied SÍ when asked by WhatsApp
	WhatsAppOptInSourceStop         = "whatsapp_stop" // the patient replied STOP / BAJA
)

func (Patient) IsAuditable() bool { return true }
