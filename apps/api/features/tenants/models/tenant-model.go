package tenant_models

import (
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

// UISettings holds per-tenant UI configuration stored as a JSON string.
type UISettings struct {
	Clinical ClinicalSettings `json:"clinical"`
	Kanban   KanbanSettings   `json:"kanban"`
	Agenda   AgendaSettings   `json:"agenda"`
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
	// Patient form
	PatientAgeInput bool `json:"patient_age_input"` // use age number instead of birth date picker
}

type KanbanSettings struct {
	AutoArchiveDelay string `json:"auto_archive_delay"` // "never", "1_day", "1_week", "2_weeks", "1_month"
}

// No-show marking limits (minutes after an appointment's end time).
const (
	MinNoShowAfterMinutes     = 5
	MaxNoShowAfterMinutes     = 240
	DefaultNoShowAfterMinutes = 30
)

// AgendaSettings configures the agenda. AutoNoShow marks as no_show the
// appointments still scheduled or confirmed NoShowAfterMinutes after their
// end (clinical_workers.NoShowScheduler). AutoNoShowSince is when the clinic
// turned it on: only appointments ending after it are marked (no backfill).
// It is set by the server (UpdateUISettings), never taken from the client.
type AgendaSettings struct {
	AutoNoShow         bool       `json:"auto_no_show"`
	NoShowAfterMinutes int        `json:"no_show_after_minutes"`
	AutoNoShowSince    *time.Time `json:"auto_no_show_since"`
}

// ValidNoShowAfterMinutes reports whether m is inside the allowed range.
func ValidNoShowAfterMinutes(m int) bool {
	return m >= MinNoShowAfterMinutes && m <= MaxNoShowAfterMinutes
}

// ParseUISettings reads the stored JSON over the defaults, so sections and
// keys that are absent keep their default value. Unparseable JSON yields the
// defaults.
func ParseUISettings(raw string) UISettings {
	settings := DefaultUISettings()
	if strings.TrimSpace(raw) == "" {
		return settings
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return DefaultUISettings()
	}
	return settings
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
		Kanban: KanbanSettings{
			AutoArchiveDelay: "never",
		},
		Agenda: AgendaSettings{
			NoShowAfterMinutes: DefaultNoShowAfterMinutes,
		},
	}
}

type EnabledFeatures struct {
	Clinical bool `json:"clinical"`
	Billing  bool `json:"billing"`
	Team     bool `json:"team"`
	Kanban   bool `json:"kanban"`
	WhatsApp bool `json:"whatsapp"`
	// PatientPortal: the plan includes the patient portal (category
	// PATIENT_PORTAL). Off by default, like WhatsApp.
	PatientPortal bool `json:"patient_portal"`
}

// DefaultEnabledFeatures returns the flags of a tenant without a subscription:
// the core modules on, WhatsApp (a paid add-on with Meta costs) and the
// patient portal off.
func DefaultEnabledFeatures() EnabledFeatures {
	return EnabledFeatures{
		Clinical: true,
		Billing:  true,
		Team:     true,
		Kanban:   true,
	}
}

// PortalSettings is the clinic's patient portal configuration, stored as JSON
// in Tenant.PortalSettings. The portal answers only when the plan includes it
// and Enabled is set; each section can be hidden on its own.
type PortalSettings struct {
	Enabled  bool           `json:"enabled"`
	Sections PortalSections `json:"sections"`
	Booking  PortalBooking  `json:"booking"`
}

// Booking "choose by" modes: the patient picks a doctor, or a specialty and
// the clinic assigns the doctor.
const (
	BookingChooseByDoctor    = "doctor"
	BookingChooseBySpecialty = "specialty"
)

// PortalBooking configures booking from the portal (Entrega 2). It is
// effective only when the portal is enabled, the appointments section is
// shown and Enabled is set (PortalSettings.BookingEnabled).
type PortalBooking struct {
	Enabled            bool   `json:"enabled"`
	ChooseBy           string `json:"choose_by"`            // BookingChooseBy*
	AutoConfirm        bool   `json:"auto_confirm"`         // false: the clinic approves each request
	RequestExpiryHours int    `json:"request_expiry_hours"` // 12 | 24 | 48
	MinNoticeHours     int    `json:"min_notice_hours"`     // 0..72
	MaxDaysAhead       int    `json:"max_days_ahead"`       // 1..180
	CancelNoticeHours  int    `json:"cancel_notice_hours"`  // 0..168
	// SlotIntervalMinutes is the gap between offered start times: 0 steps by
	// the appointment type's duration, else 10 | 15 | 20 | 30 | 60.
	SlotIntervalMinutes int `json:"slot_interval_minutes"`
	// NewPatients lets people without a patient record request a first
	// appointment (Entrega 3). Those requests always need approval.
	NewPatients bool `json:"new_patients"`
}

// DefaultPortalBooking: off, pick a doctor, clinic approval, 24 h expiry,
// 2 h notice, 60 days ahead, cancel up to 24 h before.
func DefaultPortalBooking() PortalBooking {
	return PortalBooking{
		ChooseBy: BookingChooseByDoctor, RequestExpiryHours: 24, MinNoticeHours: 2, MaxDaysAhead: 60, CancelNoticeHours: 24,
	}
}

// Valid reports whether every value is inside its allowed range.
func (b PortalBooking) Valid() bool {
	if b.ChooseBy != BookingChooseByDoctor && b.ChooseBy != BookingChooseBySpecialty {
		return false
	}
	switch b.RequestExpiryHours {
	case 12, 24, 48:
	default:
		return false
	}
	switch b.SlotIntervalMinutes {
	case 0, 10, 15, 20, 30, 60:
	default:
		return false
	}
	return b.MinNoticeHours >= 0 && b.MinNoticeHours <= 72 &&
		b.MaxDaysAhead >= 1 && b.MaxDaysAhead <= 180 &&
		b.CancelNoticeHours >= 0 && b.CancelNoticeHours <= 168
}

// BookingEnabled reports whether patients can book from the portal.
func (s PortalSettings) BookingEnabled() bool {
	return s.Enabled && s.Sections.Appointments && s.Booking.Enabled
}

// NewPatientsEnabled reports whether people without a patient record can
// request a first appointment: booking is effective and NewPatients is set.
func (s PortalSettings) NewPatientsEnabled() bool {
	return s.BookingEnabled() && s.Booking.NewPatients
}

// PortalSections are the parts of the portal a clinic can show or hide.
type PortalSections struct {
	Appointments bool `json:"appointments"`
	Results      bool `json:"results"`
	Documents    bool `json:"documents"`
	Invoices     bool `json:"invoices"`
}

// DefaultPortalSettings: portal off, every section on once enabled.
func DefaultPortalSettings() PortalSettings {
	return PortalSettings{
		Sections: PortalSections{Appointments: true, Results: true, Documents: true, Invoices: true},
		Booking:  DefaultPortalBooking(),
	}
}

// ParsePortalSettings reads the stored JSON over the defaults, so keys that
// are absent keep their default value.
func (t Tenant) ParsePortalSettings() PortalSettings {
	settings := DefaultPortalSettings()
	if strings.TrimSpace(t.PortalSettings) != "" {
		if err := json.Unmarshal([]byte(t.PortalSettings), &settings); err != nil {
			return DefaultPortalSettings()
		}
	}
	return settings
}

// DisplayName is the name patients know the clinic by.
func (t Tenant) DisplayName() string {
	if strings.TrimSpace(t.TradeName) != "" {
		return t.TradeName
	}
	return t.Name
}

type Tenant struct {
	gorm.Model
	Name              string `gorm:"not null" json:"name"`
	Slug              string `gorm:"not null;unique" json:"slug"`
	TradeName         string `json:"trade_name"`         // Nombre Comercial
	CorporateName     string `json:"corporate_name"`     // Razón Social
	TaxID             string `json:"tax_id"`             // RUC
	Address           string `json:"address"`            // Dirección Matriz
	AccountingObliged bool   `json:"accounting_obliged"` // Obligado a llevar contabilidad (SRI)
	// Clasificación tributaria opcional del SRI — se llenan en el XML solo si están seteados.
	SpecialContributorNumber *string    `json:"special_contributor_number"`                     // Nº resolución contribuyente especial
	MicroenterpriseRegime    *string    `json:"microenterprise_regime"`                         // "SI" si aplica régimen de microempresas
	WithholdingAgent         *string    `json:"withholding_agent"`                              // Nº resolución agente de retención
	RimpeTaxpayer            *string    `json:"rimpe_taxpayer"`                                 // Leyenda RIMPE (ej. "CONTRIBUYENTE RÉGIMEN RIMPE")
	SriPassword              string     `json:"-"`                                              // Legacy plaintext P12 password; see sri-password.go
	SriPasswordEncrypted     string     `gorm:"type:text" json:"-"`                             // P12 password sealed with core/secretbox
	SriP12Path               string     `json:"-"`                                              // Local storage path to the uploaded signature
	SriCertExpiration        *time.Time `json:"sri_cert_expiration"`                            // Date the certificate expires
	LogoPath                 *string    `json:"-"`                                              // Local storage path to the uploaded company logo
	UISettings               string     `gorm:"type:text;default:'{}'" json:"-"`                // JSON-encoded UISettings
	EnabledFeatures          string     `gorm:"type:text;default:'{}'" json:"enabled_features"` // JSON-encoded EnabledFeatures
	DisplayToken             string     `gorm:"type:varchar(36);uniqueIndex" json:"-"`          // Token for public TV display access
	PortalSettings           string     `gorm:"type:text;default:'{}'" json:"-"`                // JSON-encoded PortalSettings
}

func NewTenant(name string) *Tenant {
	return &Tenant{
		Name: name,
	}
}

func (t *Tenant) Save(db *gorm.DB) error {
	return db.Save(t).Error
}
