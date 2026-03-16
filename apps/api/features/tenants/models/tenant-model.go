package tenant_models

import (
	"time"

	"gorm.io/gorm"
)

// UISettings holds per-tenant UI configuration stored as a JSON string.
type UISettings struct {
	Clinical ClinicalSettings `json:"clinical"`
}

type ClinicalSettings struct {
	// Patient table columns
	ShowNextAppointment bool `json:"show_next_appointment"`
	ShowDiagnosis       bool `json:"show_diagnosis"`
	ShowMedic           bool `json:"show_medic"`
	ShowInsurance       bool `json:"show_insurance"`
	// Consultation form sections
	ShowVitalSigns  bool   `json:"show_vital_signs"`
	ShowDiagnoses   bool   `json:"show_diagnoses"`
	DiagnosisSystem string `json:"diagnosis_system"` // "cie11" | "cie10"
}

// DefaultUISettings returns sensible defaults (all visible, CIE-11).
func DefaultUISettings() UISettings {
	return UISettings{
		Clinical: ClinicalSettings{
			ShowNextAppointment: true,
			ShowDiagnosis:       true,
			ShowMedic:           true,
			ShowInsurance:       true,
			ShowVitalSigns:      true,
			ShowDiagnoses:       true,
			DiagnosisSystem:     "cie11",
		},
	}
}

type Tenant struct {
	gorm.Model
	Name              string     `gorm:"not null" json:"name"`
	Slug              string     `gorm:"not null;unique" json:"slug"`
	TradeName         string     `json:"trade_name"`          // Nombre Comercial
	CorporateName     string     `json:"corporate_name"`      // Razón Social
	TaxID             string     `json:"tax_id"`              // RUC
	Address           string     `json:"address"`             // Dirección Matriz
	AccountingObliged bool       `json:"accounting_obliged"`  // Obligado a llevar contabilidad (SRI)
	SriPassword       string     `json:"-"`                   // Hidden from API responses
	SriP12Path        string     `json:"sri_p12_path"`        // Local storage path to the uploaded signature
	SriCertExpiration *time.Time `json:"sri_cert_expiration"` // Date the certificate expires
	UISettings        string     `gorm:"type:text;default:'{}'" json:"-"` // JSON-encoded UISettings
}

func NewTenant(name string) *Tenant {
	return &Tenant{
		Name: name,
	}
}

func (t *Tenant) Save(db *gorm.DB) error {
	return db.Save(t).Error
}
