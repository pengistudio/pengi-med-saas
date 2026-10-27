package portal_dto

import "time"

// Document types of a new patient's form.
const (
	DocumentTypeCedula   = "cedula"
	DocumentTypePassport = "passport"
)

// NewPatientCodeRequest is POST /portal/clinics/:slug/new-patient/code.
// Website is a honeypot; CaptchaToken the Turnstile token (like LoginRequest).
type NewPatientCodeRequest struct {
	Channel      string `json:"channel"` // email | whatsapp
	Contact      string `json:"contact"`
	Website      string `json:"website"`
	CaptchaToken string `json:"captcha_token"`
}

// NewPatientVerifyRequest is POST /portal/clinics/:slug/new-patient/verify.
type NewPatientVerifyRequest struct {
	Channel string `json:"channel"`
	Contact string `json:"contact"`
	Code    string `json:"code"`
}

// GuestTokenResponse carries a portal guest token (kept in memory, no refresh).
type GuestTokenResponse struct {
	GuestToken string `json:"guest_token"`
	ExpiresIn  int    `json:"expires_in"` // seconds
}

// NewPatientForm is a new patient's personal data. It is also stored, once
// validated and normalized, as AppointmentRequest.NewPatientData.
type NewPatientForm struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DocumentType  string `json:"document_type"` // DocumentType*
	Document      string `json:"document"`
	BirthDate     string `json:"birth_date"` // YYYY-MM-DD
	Phone         string `json:"phone"`      // optional; stored in E.164 without "+"
	Email         string `json:"email"`      // optional; stored lowercased
	WhatsAppOptIn bool   `json:"whatsapp_opt_in"`
}

// NewPatientRequestBody is POST /portal/new-patient/requests: the slot (as in
// CreateRequestBody) plus the person's data.
type NewPatientRequestBody struct {
	CreateRequestBody
	Patient NewPatientForm `json:"patient"`
}

// NewPatientRequestResponse is the created new-patient request.
type NewPatientRequestResponse struct {
	ID             uint          `json:"id"`
	Status         string        `json:"status"` // always pending
	Date           string        `json:"date"`
	StartTime      string        `json:"start_time"`
	EndTime        string        `json:"end_time"`
	Doctor         RequestDoctor `json:"doctor"`
	TypeName       string        `json:"type_name"`
	ExpiresAt      time.Time     `json:"expires_at"`
	ContactChannel string        `json:"contact_channel"` // where the clinic's answer goes
}
