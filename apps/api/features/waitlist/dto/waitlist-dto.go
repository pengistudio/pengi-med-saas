// Package waitlist_dto holds the waitlist's request and response bodies.
package waitlist_dto

import "time"

// EntryBody is POST/PUT /agenda/waitlist. DoctorID nil or 0: any doctor;
// AppointmentTypeID nil or 0: any type; dates "YYYY-MM-DD" or "" (open);
// AppointmentID: the patient's future appointment to move earlier (optional).
// PatientID is ignored on update (an entry keeps its patient).
type EntryBody struct {
	PatientID         uint   `json:"patient_id"`
	DoctorID          *uint  `json:"doctor_id"`
	AppointmentTypeID *uint  `json:"appointment_type_id"`
	DateFrom          string `json:"date_from"`
	DateTo            string `json:"date_to"`
	AppointmentID     *uint  `json:"appointment_id"`
	Notes             string `json:"notes"`
}

// OfferBody is POST /agenda/waitlist/:id/offer: the freed slot and how it is
// offered ("whatsapp" sends the template, "phone" only records the call).
type OfferBody struct {
	Channel           string `json:"channel" binding:"required"`
	DoctorID          *uint  `json:"doctor_id"`
	AppointmentTypeID *uint  `json:"appointment_type_id"`
	Date              string `json:"date" binding:"required"`
	StartTime         string `json:"start_time" binding:"required"`
	EndTime           string `json:"end_time"`
	// PatientID: who freed the slot (from the notification link); their own
	// entries can't be offered it.
	PatientID uint `json:"patient_id"`
}

// Person is the entry's patient.
type Person struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Document  string `json:"document"`
	Phone     string `json:"phone"`
	// WhatsAppOptIn: the patient agreed to WhatsApp messages.
	WhatsAppOptIn bool `json:"whatsapp_opt_in"`
}

// Doctor is the entry's doctor.
type Doctor struct {
	ID       uint   `json:"id"`
	FullName string `json:"full_name"`
}

// Type is the entry's appointment type.
type Type struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
}

// Appointment is the appointment the patient wants to move earlier.
type Appointment struct {
	ID        uint   `json:"id"`
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	Status    string `json:"status"`
}

// Offer is the last slot offered to the entry.
type Offer struct {
	DoctorID   *uint      `json:"doctor_id"`
	Date       string     `json:"date"`
	StartTime  string     `json:"start_time"`
	EndTime    string     `json:"end_time"`
	Channel    string     `json:"channel"`
	OfferedAt  *time.Time `json:"offered_at"`
	AcceptedAt *time.Time `json:"accepted_at"`
}

// Entry is a waitlist entry as the agenda shows it. CanWhatsApp (candidates
// only): the patient can be offered the slot by WhatsApp (consent and a
// valid phone); OfferedThisSlot: the slot asked about was already offered.
type Entry struct {
	ID              uint         `json:"id"`
	Status          string       `json:"status"`
	Source          string       `json:"source"`
	Patient         Person       `json:"patient"`
	Doctor          *Doctor      `json:"doctor"`
	AppointmentType *Type        `json:"appointment_type"`
	DateFrom        string       `json:"date_from"`
	DateTo          string       `json:"date_to"`
	Appointment     *Appointment `json:"appointment"`
	Notes           string       `json:"notes"`
	CreatedAt       time.Time    `json:"created_at"`
	ExpiresAt       time.Time    `json:"expires_at"`
	Offer           *Offer       `json:"offer"`
	CanWhatsApp     bool         `json:"can_whatsapp,omitempty"`
	OfferedThisSlot bool         `json:"offered_this_slot,omitempty"`
}

// Candidates is GET /agenda/waitlist/candidates: the entries a slot matches,
// in signup order. WhatsAppAvailable: the clinic can offer by WhatsApp
// (plan, connection, approved template); otherwise the UI offers only a call.
type Candidates struct {
	WhatsAppAvailable bool    `json:"whatsapp_available"`
	Doctor            string  `json:"doctor"`
	Candidates        []Entry `json:"candidates"`
}

// PortalEntryBody is POST /portal/waitlist: the patient asks to be told
// when a slot of the type frees (with the doctor, or any when 0/nil).
type PortalEntryBody struct {
	TypeID   uint  `json:"type_id" binding:"required"`
	DoctorID *uint `json:"doctor_id"`
}

// PortalEntry is an entry as the portal shows it to its patient.
type PortalEntry struct {
	ID         uint      `json:"id"`
	Status     string    `json:"status"`
	DoctorID   *uint     `json:"doctor_id"`
	DoctorName string    `json:"doctor_name"`
	TypeID     *uint     `json:"type_id"`
	TypeName   string    `json:"type_name"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
