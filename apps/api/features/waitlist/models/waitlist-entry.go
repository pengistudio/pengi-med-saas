// Package waitlist_models holds the appointment waitlist.
package waitlist_models

import (
	"time"

	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"

	"gorm.io/gorm"
)

// Statuses of a WaitlistEntry. waiting and offered are active (on the list);
// the rest are final.
const (
	StatusWaiting = "waiting" // on the list
	StatusOffered = "offered" // a freed slot was offered (WhatsApp or phone); still on the list
	StatusBooked  = "booked"  // the patient got an appointment after joining
	StatusRemoved = "removed" // taken off by reception or by the patient (portal)
	StatusExpired = "expired" // the preferred range ended (or 30 days passed without one)
)

// ActiveStatuses are the statuses still on the list.
var ActiveStatuses = []string{StatusWaiting, StatusOffered}

// Sources of an entry.
const (
	SourceStaff  = "staff"  // reception, from the agenda
	SourcePortal = "portal" // the patient, from the portal's booking flow
)

// Offer channels.
const (
	ChannelWhatsApp = "whatsapp"
	ChannelPhone    = "phone"
)

// WaitlistEntry is a patient waiting for a slot: with a doctor (or any,
// DoctorID nil), optionally of an appointment type and inside a preferred
// date range, optionally to move an existing appointment (AppointmentID)
// earlier. When a matching slot frees, reception is notified and offers it
// (semi-automatic: nothing is offered on its own). Dates are "YYYY-MM-DD" in
// the clinic's time zone ("" = open), like appointment requests. ExpiresAt
// is when the entry leaves the list on its own (end of DateTo, else 30 days).
// The Offer* fields describe the last slot offered; OfferMessageID is the
// WhatsApp message (kind slot_offer) when it was offered that way.
type WaitlistEntry struct {
	gorm.Model
	TenantID          uint                           `gorm:"not null;index" json:"tenant_id"`
	PatientID         uint                           `gorm:"not null;index" json:"patient_id"`
	Patient           *clinical_models.Patient       `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	DoctorID          *uint                          `gorm:"index" json:"doctor_id"`
	Doctor            *doctor_models.Doctor          `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	AppointmentTypeID *uint                          `json:"appointment_type_id"`
	AppointmentType   *agenda_models.AppointmentType `gorm:"foreignKey:AppointmentTypeID" json:"appointment_type,omitempty"`
	DateFrom          string                         `gorm:"type:varchar(10);not null;default:''" json:"date_from"`
	DateTo            string                         `gorm:"type:varchar(10);not null;default:''" json:"date_to"`
	AppointmentID     *uint                          `json:"appointment_id"`
	Notes             string                         `gorm:"type:varchar(500);not null;default:''" json:"notes"`
	Source            string                         `gorm:"type:varchar(20);not null;default:staff" json:"source"`
	CreatedByUserID   *uint                          `json:"created_by_user_id"`
	Status            string                         `gorm:"type:varchar(20);not null;index" json:"status"`
	ExpiresAt         time.Time                      `gorm:"not null;index" json:"expires_at"`
	ClosedAt          *time.Time                     `json:"closed_at"`

	OfferedAt        *time.Time `json:"offered_at"`
	OfferedDoctorID  *uint      `json:"offered_doctor_id"`
	OfferedDate      string     `gorm:"type:varchar(10);not null;default:''" json:"offered_date"`
	OfferedStartTime string     `gorm:"type:varchar(5);not null;default:''" json:"offered_start_time"`
	OfferedEndTime   string     `gorm:"type:varchar(5);not null;default:''" json:"offered_end_time"`
	OfferChannel     string     `gorm:"type:varchar(20);not null;default:''" json:"offer_channel"`
	OfferMessageID   *uint      `gorm:"index" json:"offer_message_id"`
	OfferAcceptedAt  *time.Time `json:"offer_accepted_at"`
}

func (WaitlistEntry) TableName() string { return "waitlist_entries" }

func (WaitlistEntry) IsAuditable() bool { return true }

// BeforeSave stores ExpiresAt in UTC, the zone every query compares it in
// (SQLite compares timestamps as text).
func (e *WaitlistEntry) BeforeSave(*gorm.DB) error {
	e.ExpiresAt = e.ExpiresAt.UTC()
	return nil
}

// Active reports whether the entry is still on the list.
func (e WaitlistEntry) Active() bool {
	return e.Status == StatusWaiting || e.Status == StatusOffered
}

// WaitlistOffer is one slot offered to an entry: the offer history. Its
// unique index makes the rule strict: a slot (day, start, doctor; 0 = none)
// is offered to an entry at most once, whatever was offered in between.
type WaitlistOffer struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	TenantID  uint      `gorm:"not null;index" json:"tenant_id"`
	EntryID   uint      `gorm:"not null;uniqueIndex:idx_waitlist_offer_slot" json:"entry_id"`
	Date      string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_waitlist_offer_slot" json:"date"`
	StartTime string    `gorm:"type:varchar(5);not null;uniqueIndex:idx_waitlist_offer_slot" json:"start_time"`
	DoctorID  uint      `gorm:"not null;default:0;uniqueIndex:idx_waitlist_offer_slot" json:"doctor_id"`
	Channel   string    `gorm:"type:varchar(20);not null" json:"channel"`
	MessageID *uint     `json:"message_id"`
}

func (WaitlistOffer) TableName() string { return "waitlist_offers" }
