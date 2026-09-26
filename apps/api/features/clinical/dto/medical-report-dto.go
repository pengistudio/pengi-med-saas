package dto

type ConsultationEntryDTO struct {
	MedicalRecordID uint   `json:"medical_record_id" binding:"required"`
	Date            string `json:"date" binding:"required"`
	Motive          string `json:"motive"`
	Summary         string `json:"summary"`
}

type CreateMedicalReportDTO struct {
	Consultations []ConsultationEntryDTO `json:"consultations" binding:"required,dive"`
	Plan          string                 `json:"plan"`
}

type EmailMedicalDocumentDTO struct {
	Email string `json:"email" binding:"required,email"`
}
