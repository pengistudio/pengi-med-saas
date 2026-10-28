package clinical_models

import (
	"time"

	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

// Appointment statuses that end an appointment without the patient being
// attended, plus the ones before arrival (the only ones a no-show can follow).
const (
	AppointmentStatusScheduled = "scheduled"
	AppointmentStatusConfirmed = "confirmed"
	AppointmentStatusCompleted = "completed"
	AppointmentStatusCancelled = "cancelled"
	// AppointmentStatusNoShow: the patient didn't come. Set by reception or,
	// when the clinic turns it on, by clinical_workers.NoShowScheduler.
	AppointmentStatusNoShow = "no_show"
)

// IsClosedStatus reports whether status takes the appointment off the
// calendars (its Google Calendar event is removed): completed, cancelled or
// no_show.
func IsClosedStatus(status string) bool {
	return status == AppointmentStatusCompleted || status == AppointmentStatusCancelled || status == AppointmentStatusNoShow
}

type Appointment struct {
	gorm.Model
	TenantID      uint                  `json:"tenant_id"`
	PatientID     uint                  `json:"patient_id"`
	Title         string                `json:"title"`
	Date          time.Time             `json:"date"`
	StartTime     string                `json:"start_time"`
	EndTime       string                `json:"end_time"`
	Location      string                `json:"location,omitempty"`
	Notes         string                `json:"notes,omitempty"`
	Status        string                `json:"status" gorm:"default:scheduled"`
	ColorID       string                `json:"color_id,omitempty" gorm:"default:''"`
	GoogleEventID string                `json:"google_event_id,omitempty"`
	Patient       Patient               `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	DoctorID      *uint                 `json:"doctor_id" gorm:"index"`
	Doctor        *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	// AppointmentTypeID is the optional agenda_models.AppointmentType (no
	// relation field, so models migrating Appointment don't need the type).
	AppointmentTypeID *uint `json:"appointment_type_id" gorm:"index"`
	// VitalSigns taken at triage; loaded by the waiting room list.
	VitalSigns *VitalSigns `json:"vital_signs,omitempty" gorm:"foreignKey:AppointmentID"`
}

func (Appointment) IsAuditable() bool { return true }
