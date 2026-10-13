package agenda_dto

import (
	"time"

	agenda_models "pengi-med-saas/features/agenda/models"
)

// ScheduleSlot is one working range of the weekly schedule.
type ScheduleSlot struct {
	Weekday   int    `json:"weekday"` // 0 = Sunday … 6 = Saturday
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// ReplaceScheduleRequest replaces a doctor's whole weekly schedule; an empty
// list clears it (the doctor goes back to "no schedule").
type ReplaceScheduleRequest struct {
	Slots []ScheduleSlot `json:"slots"`
}

// BlockRequest creates or replaces a schedule block. StartTime/EndTime both
// empty block whole days.
type BlockRequest struct {
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Reason    string `json:"reason" binding:"max=255"`
}

// AffectedAppointment is an appointment that falls inside a block, so the UI
// can list it for rescheduling. Nothing is cancelled automatically.
type AffectedAppointment struct {
	ID          uint      `json:"id"`
	Date        time.Time `json:"date"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	Status      string    `json:"status"`
	DoctorID    *uint     `json:"doctor_id"`
	PatientID   uint      `json:"patient_id"`
	PatientName string    `json:"patient_name"`
}

// BlockResponse is a saved block plus the appointments it overlaps.
type BlockResponse struct {
	Block                agenda_models.ScheduleBlock `json:"block"`
	AffectedAppointments []AffectedAppointment       `json:"affected_appointments"`
}

// CreateAppointmentTypeRequest creates a type. DoctorIDs empty = all doctors.
type CreateAppointmentTypeRequest struct {
	Name            string `json:"name" binding:"required,max=100"`
	DurationMinutes int    `json:"duration_minutes"`
	Color           string `json:"color"`
	Active          *bool  `json:"active"` // omitted = true
	DoctorIDs       []uint `json:"doctor_ids"`
}

// UpdateAppointmentTypeRequest is a partial update; doctor_ids omitted keeps
// the restriction, [] removes it.
type UpdateAppointmentTypeRequest struct {
	Name            *string `json:"name" binding:"omitempty,max=100"`
	DurationMinutes *int    `json:"duration_minutes"`
	Color           *string `json:"color"`
	Active          *bool   `json:"active"`
	DoctorIDs       *[]uint `json:"doctor_ids"`
}

// TimeRange is a working range of one day.
type TimeRange struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// AvailabilityResponse answers whether a time slot of a doctor is inside the
// schedule. Status: "inside", "outside", "blocked" or "no_schedule".
type AvailabilityResponse struct {
	Status string                        `json:"status"`
	Ranges []TimeRange                   `json:"ranges"` // the doctor's working ranges that day
	Blocks []agenda_models.ScheduleBlock `json:"blocks"` // blocks overlapping the slot
}

// AgendaDay is one day of one doctor: working ranges and the blocks that
// apply (the doctor's and the clinic's).
type AgendaDay struct {
	Date   string                        `json:"date"`
	Ranges []TimeRange                   `json:"ranges"`
	Blocks []agenda_models.ScheduleBlock `json:"blocks"`
}

// DoctorAgenda is the agenda shading data of one doctor for a date range.
type DoctorAgenda struct {
	DoctorID    uint        `json:"doctor_id"`
	Active      bool        `json:"active"`
	HasSchedule bool        `json:"has_schedule"`
	Days        []AgendaDay `json:"days"`
}

// ClinicDay lists the clinic-wide blocks of one day.
type ClinicDay struct {
	Date   string                        `json:"date"`
	Blocks []agenda_models.ScheduleBlock `json:"blocks"`
}

// AgendaRangeResponse is the shading data of the agenda for a date range.
type AgendaRangeResponse struct {
	From    string         `json:"from"`
	To      string         `json:"to"`
	Doctors []DoctorAgenda `json:"doctors"`
	Clinic  []ClinicDay    `json:"clinic"`
}
