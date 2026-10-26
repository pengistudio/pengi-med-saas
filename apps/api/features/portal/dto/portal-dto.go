package portal_dto

import (
	"time"

	tenant_models "pengi-med-saas/features/tenants/models"
)

// SettingsResponse is the clinic's portal configuration plus the link
// patients use (PORTAL_URL/<slug>).
type SettingsResponse struct {
	tenant_models.PortalSettings
	PortalURL string `json:"portal_url"`
}

// LoginRequest asks for a sign-in code. Website is a honeypot: people never
// fill it (the field is hidden), bots do. CaptchaToken is the Cloudflare
// Turnstile token, required when TURNSTILE_SECRET_KEY is set.
type LoginRequest struct {
	Document     string `json:"document"`
	Website      string `json:"website"`
	CaptchaToken string `json:"captcha_token"`
}

// LoginResponse is the same whether or not a code was sent.
type LoginResponse struct {
	ExpiresIn int `json:"expires_in"`
}

// VerifyRequest exchanges a sign-in code for a session.
type VerifyRequest struct {
	Document string `json:"document"`
	Code     string `json:"code"`
}

// TokenResponse carries a portal access token; the refresh token travels in
// the portal_refresh cookie.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

// ClinicInfo is what the portal shows about the clinic.
type ClinicInfo struct {
	Name     string                       `json:"name"`
	Slug     string                       `json:"slug"`
	HasLogo  bool                         `json:"has_logo"`
	Sections tenant_models.PortalSections `json:"sections"`
}

// Channels are the ways the clinic can deliver a sign-in code.
type Channels struct {
	WhatsApp bool `json:"whatsapp"`
	Email    bool `json:"email"`
}

// PublicClinicResponse is GET /portal/clinics/:slug.
type PublicClinicResponse struct {
	ClinicInfo
	Channels Channels `json:"channels"`
}

// PortalPatient is the signed-in patient, as the portal shows them.
type PortalPatient struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Document  string `json:"document"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// MeResponse is GET /portal/me.
type MeResponse struct {
	Patient PortalPatient `json:"patient"`
	Clinic  ClinicInfo    `json:"clinic"`
}

// Appointment is one of the patient's appointments: never notes or internal fields.
type Appointment struct {
	ID              uint   `json:"id"`
	Date            string `json:"date"` // YYYY-MM-DD
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	Status          string `json:"status"`
	DoctorName      string `json:"doctor_name"`
	Specialty       string `json:"specialty"`       // doctor_data specialty code
	SpecialtyOther  string `json:"specialty_other"` // the name when Specialty is "other"
	AppointmentType string `json:"appointment_type"`
	Location        string `json:"location"`
}

// ResultFile is a result file of a reviewed exam.
type ResultFile struct {
	ID          uint   `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

// ResultItem is a reviewed exam of an order.
type ResultItem struct {
	ID         uint         `json:"id"`
	Name       string       `json:"name"`
	ReviewedAt time.Time    `json:"reviewed_at"`
	Files      []ResultFile `json:"files"`
}

// ResultOrder is an exam order with at least one reviewed exam.
type ResultOrder struct {
	ID         uint         `json:"id"`
	Number     string       `json:"number"` // ORD-000001
	Date       string       `json:"date"`   // YYYY-MM-DD
	DoctorName string       `json:"doctor_name"`
	Items      []ResultItem `json:"items"`
}

// Document types of GET /portal/documents.
const (
	DocumentPrescription = "prescription"
	DocumentCertificate  = "certificate"
	DocumentReport       = "report"
)

// Document is an electronically signed document of the patient.
type Document struct {
	Type       string    `json:"type"`
	ID         uint      `json:"id"`
	Date       string    `json:"date"` // YYYY-MM-DD
	DoctorName string    `json:"doctor_name"`
	SignedAt   time.Time `json:"signed_at"`
}

// Invoice is an SRI-authorized invoice issued to the patient.
type Invoice struct {
	ID        uint    `json:"id"`
	Number    string  `json:"number"` // 001-001-000000001
	Date      string  `json:"date"`   // YYYY-MM-DD
	Total     float64 `json:"total"`
	AccessKey string  `json:"access_key"`
}
