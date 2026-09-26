package dto

import "time"

type CreateMedicalCertificateDTO struct {
	Diagnosis    string     `json:"diagnosis" binding:"required"`
	Observations string     `json:"observations"`
	RestDays     *int       `json:"rest_days,omitempty"`
	RestFrom     *time.Time `json:"rest_from,omitempty"`
	RestTo       *time.Time `json:"rest_to,omitempty"`
}
