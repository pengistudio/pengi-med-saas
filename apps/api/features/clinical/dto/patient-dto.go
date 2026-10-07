package dto

import "time"

type CreatePatientDTO struct {
	Document           string     `json:"document" binding:"required"`
	Phone              string     `json:"phone"`
	WhatsAppOptIn      bool       `json:"whatsapp_opt_in"`
	Email              string     `json:"email"`
	FirstName          string     `json:"first_name" binding:"required"`
	LastName           string     `json:"last_name" binding:"required"`
	BirthDate          *time.Time `json:"birth_date"`
	BirthDateEstimated *bool      `json:"birth_date_estimated"` // BirthDate derived from an age; ignored without BirthDate
	Institution        string     `json:"institution" binding:"required"`
	Gender             string     `json:"gender"`
	Notes              string     `json:"notes"`
	Insurance          string     `json:"insurance"`
	Medic              string     `json:"medic"`
}

type UpdatePatientDTO struct {
	Document           *string    `json:"document"`
	Phone              *string    `json:"phone"`
	WhatsAppOptIn      *bool      `json:"whatsapp_opt_in"`
	Email              *string    `json:"email"`
	FirstName          *string    `json:"first_name"`
	LastName           *string    `json:"last_name"`
	BirthDate          *time.Time `json:"birth_date"`
	BirthDateEstimated *bool      `json:"birth_date_estimated"` // BirthDate derived from an age; ignored without BirthDate
	Institution        *string    `json:"institution"`
	Gender             *string    `json:"gender"`
	Notes              *string    `json:"notes"`
	Insurance          *string    `json:"insurance"`
	Medic              *string    `json:"medic"`
}

type DeletePatientsDTO struct {
	IdList []uint `json:"id_list" binding:"required"`
}

// BulkWhatsAppOptInDTO sets (opt_in true) or clears WhatsApp consent on the
// patients in ids.
type BulkWhatsAppOptInDTO struct {
	IDs   []uint `json:"ids" binding:"required"`
	OptIn *bool  `json:"opt_in" binding:"required"`
}

// BulkWhatsAppOptInResponse is how many patients changed, and how many were
// left out because they opted out by replying STOP.
type BulkWhatsAppOptInResponse struct {
	Updated         int64 `json:"updated"`
	SkippedOptedOut int64 `json:"skipped_opted_out"`
}
