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
}

type UpdateAppointmentDTO struct {
	PatientID *uint      `json:"patient_id,omitempty"`
	Title     *string    `json:"title,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	StartTime *string    `json:"start_time,omitempty"`
	EndTime   *string    `json:"end_time,omitempty"`
	Location  *string    `json:"location,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
}

type UpdateAppointmentStatusDTO struct {
	Status string `json:"status" binding:"required"`
}
