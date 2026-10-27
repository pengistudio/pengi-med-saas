package portal_dto

import "time"

// BookingInfo is what the portal needs to show the "book" entry point.
type BookingInfo struct {
	Enabled  bool   `json:"enabled"`
	ChooseBy string `json:"choose_by"` // "doctor" | "specialty"
	// NewPatients: people without a patient record can request a first
	// appointment (booking effective and the setting on).
	NewPatients bool `json:"new_patients"`
}

// BookingDoctor is a doctor patients can book.
type BookingDoctor struct {
	ID             uint   `json:"id"`
	FullName       string `json:"full_name"`
	Specialty      string `json:"specialty"`
	SpecialtyOther string `json:"specialty_other"`
	TypeIDs        []uint `json:"type_ids"` // bookable types this doctor attends
}

// BookingSpecialty is a specialty of the bookable doctors.
type BookingSpecialty struct {
	Specialty      string `json:"specialty"`
	SpecialtyOther string `json:"specialty_other"`
	TypeIDs        []uint `json:"type_ids"` // bookable types some doctor of it attends
}

// BookingType is a bookable appointment type.
type BookingType struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
}

// BookingOptions is GET /portal/booking/options. Doctors is empty in
// specialty mode.
type BookingOptions struct {
	ChooseBy    string             `json:"choose_by"`
	MinDate     string             `json:"min_date"` // YYYY-MM-DD
	MaxDate     string             `json:"max_date"`
	Doctors     []BookingDoctor    `json:"doctors"`
	Specialties []BookingSpecialty `json:"specialties"`
	Types       []BookingType      `json:"types"`
}

// SlotDay is one day with free start times.
type SlotDay struct {
	Date  string   `json:"date"`
	Times []string `json:"times"` // "HH:MM", ascending
}

// SlotsResponse is GET /portal/booking/slots.
type SlotsResponse struct {
	Days []SlotDay `json:"days"`
}

// CreateRequestBody is POST /portal/booking/requests. DoctorID in doctor
// mode; Specialty (+ SpecialtyOther when "other") in specialty mode.
type CreateRequestBody struct {
	TypeID         uint   `json:"type_id"`
	DoctorID       uint   `json:"doctor_id"`
	Specialty      string `json:"specialty"`
	SpecialtyOther string `json:"specialty_other"`
	Date           string `json:"date"`       // YYYY-MM-DD
	StartTime      string `json:"start_time"` // HH:MM
	Reason         string `json:"reason"`     // optional, at most 500 characters
}

// RequestDoctor is the doctor of a request, as the patient sees it.
type RequestDoctor struct {
	FullName       string `json:"full_name"`
	Specialty      string `json:"specialty"`
	SpecialtyOther string `json:"specialty_other"`
}

// BookingRequest is one of the patient's appointment requests.
type BookingRequest struct {
	ID            uint          `json:"id"`
	Status        string        `json:"status"` // pending | approved | rejected | cancelled | expired
	Date          string        `json:"date"`
	StartTime     string        `json:"start_time"`
	EndTime       string        `json:"end_time"`
	Doctor        RequestDoctor `json:"doctor"`
	TypeName      string        `json:"type_name"`
	Reason        string        `json:"reason"`
	RejectReason  string        `json:"reject_reason"`
	ExpiresAt     time.Time     `json:"expires_at"`
	AppointmentID *uint         `json:"appointment_id"` // set when approved
}

// RejectRequestBody is POST /appointment-requests/:id/reject.
type RejectRequestBody struct {
	Reason string `json:"reason"`
}

// RequestPatient is the patient of a request, as staff see it.
type RequestPatient struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Document  string `json:"document"`
	Phone     string `json:"phone"`
}

// ClinicRequestDoctor is the doctor of a request, as staff see it.
type ClinicRequestDoctor struct {
	ID             uint   `json:"id"`
	FullName       string `json:"full_name"`
	Specialty      string `json:"specialty"`
	SpecialtyOther string `json:"specialty_other"`
}

// ClinicRequest is an appointment request in the clinic's inbox.
type ClinicRequest struct {
	ID                uint       `json:"id"`
	Status            string     `json:"status"`
	Date              string     `json:"date"`
	StartTime         string     `json:"start_time"`
	EndTime           string     `json:"end_time"`
	Reason            string     `json:"reason"`
	RejectReason      string     `json:"reject_reason"`
	ExpiresAt         time.Time  `json:"expires_at"`
	CreatedAt         time.Time  `json:"created_at"`
	DecidedAt         *time.Time `json:"decided_at"`
	AppointmentID     *uint      `json:"appointment_id"`
	AppointmentTypeID uint       `json:"appointment_type_id"`
	TypeName          string     `json:"type_name"`
	// Patient is nil for a new-patient request not approved yet.
	Patient *RequestPatient     `json:"patient"`
	Doctor  ClinicRequestDoctor `json:"doctor"`
	// NewPatient is the form of a new-patient request (nil otherwise).
	NewPatient *NewPatientInfo `json:"new_patient"`
	// MatchingPatient is the tenant's patient with the same document as a
	// pending new-patient request: approving links the request to it.
	MatchingPatient *MatchingPatient `json:"matching_patient"`
}

// NewPatientInfo is a new-patient request's form, as staff see it.
type NewPatientInfo struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	DocumentType   string `json:"document_type"`
	Document       string `json:"document"`
	BirthDate      string `json:"birth_date"` // YYYY-MM-DD
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	ContactChannel string `json:"contact_channel"` // the verified one: email | whatsapp
}

// MatchingPatient is an existing patient a new-patient request will be linked to.
type MatchingPatient struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// PendingCountResponse is GET /appointment-requests/pending-count.
type PendingCountResponse struct {
	Count int64 `json:"count"`
}
