package portal_models

import (
	"time"

	"gorm.io/gorm"
)

// Statuses of an AppointmentRequest.
const (
	RequestPending   = "pending"   // waiting for the clinic; holds its slot until ExpiresAt
	RequestApproved  = "approved"  // the clinic approved it (or auto-confirm): AppointmentID is set
	RequestRejected  = "rejected"  // the clinic rejected it (RejectReason optional)
	RequestCancelled = "cancelled" // the patient cancelled it while pending
	RequestExpired   = "expired"   // nobody decided before ExpiresAt
)

// AppointmentRequest is a patient's request for an appointment made from the
// patient portal. While pending it holds its slot (no other patient can ask
// for it and the portal doesn't offer it); it ends approved (an Appointment
// is created), rejected, cancelled by the patient or expired. Auto-confirm
// clinics still write the row, approved with no DecidedByUserID.
// A new-patient request (NewPatient) comes from a person without a patient
// record who verified a contact: PatientID is 0 until the clinic approves it
// (the patient is created or linked then), NewPatientData holds the form and
// ContactValue the verified contact. It always needs approval.
// Dates are "YYYY-MM-DD" and times "HH:MM" in the clinic's time zone, like
// the agenda.
type AppointmentRequest struct {
	gorm.Model
	TenantID          uint       `gorm:"not null;index" json:"tenant_id"`
	PatientID         uint       `gorm:"not null;index" json:"patient_id"`
	DoctorID          uint       `gorm:"not null;index" json:"doctor_id"`
	AppointmentTypeID uint       `gorm:"not null" json:"appointment_type_id"`
	Date              string     `gorm:"type:varchar(10);not null" json:"date"`
	StartTime         string     `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime           string     `gorm:"type:varchar(5);not null" json:"end_time"`
	Reason            string     `gorm:"type:varchar(500);not null;default:''" json:"reason"`
	Status            string     `gorm:"type:varchar(20);not null;index" json:"status"`
	ExpiresAt         time.Time  `gorm:"not null;index" json:"expires_at"`
	AppointmentID     *uint      `json:"appointment_id"`
	DecidedByUserID   *uint      `json:"decided_by_user_id"`
	DecidedAt         *time.Time `json:"decided_at"`
	RejectReason      string     `gorm:"type:varchar(500);not null;default:''" json:"reject_reason"`
	NewPatient        bool       `gorm:"not null;default:false" json:"new_patient"`
	NewPatientData    string     `gorm:"type:text;not null;default:''" json:"-"` // JSON of portal_dto.NewPatientForm
	ContactChannel    string     `gorm:"type:varchar(20);not null;default:''" json:"contact_channel"`
	ContactValue      string     `gorm:"type:varchar(255);not null;default:''" json:"-"` // normalized verified contact
}

func (AppointmentRequest) IsAuditable() bool { return true }

// BeforeSave stores ExpiresAt in UTC, the zone every query compares it in
// (SQLite compares timestamps as text).
func (r *AppointmentRequest) BeforeSave(*gorm.DB) error {
	r.ExpiresAt = r.ExpiresAt.UTC()
	return nil
}
