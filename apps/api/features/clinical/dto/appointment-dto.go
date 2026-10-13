package dto

import "time"

type CreateAppointmentDTO struct {
	PatientID uint      `json:"patient_id" binding:"required"`
	Title     string    `json:"title" binding:"required"`
	Date      time.Time `json:"date" binding:"required"`
	StartTime string    `json:"start_time" binding:"required"`
	EndTime   string    `json:"end_time" binding:"required"`
	Location  string    `json:"location,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	ColorID   string    `json:"color_id,omitempty" binding:"omitempty,oneof=1 2 3 4 5 6 7 8 9 10 11"`
	DoctorID  *uint     `json:"doctor_id,omitempty"` // defaults per doctor_services.Resolve
	// AppointmentTypeID is optional; it must be an active type of the tenant
	// that the doctor attends (agenda_services.ValidateAppointmentType).
	AppointmentTypeID *uint `json:"appointment_type_id,omitempty"`
}

type UpdateAppointmentDTO struct {
	PatientID *uint      `json:"patient_id,omitempty"`
	Title     *string    `json:"title,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	StartTime *string    `json:"start_time,omitempty"`
	EndTime   *string    `json:"end_time,omitempty"`
	Location  *string    `json:"location,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
	ColorID   *string    `json:"color_id,omitempty" binding:"omitempty,oneof=1 2 3 4 5 6 7 8 9 10 11"`
	DoctorID  *uint      `json:"doctor_id,omitempty"` // omitted keeps the current doctor
	// AppointmentTypeID: omitted keeps the current type, 0 removes it.
	AppointmentTypeID *uint `json:"appointment_type_id,omitempty"`
}

type UpdateAppointmentStatusDTO struct {
	Status string `json:"status" binding:"required"`
}
