package dto

import clinical_models "pengi-med-saas/features/clinical/models"

type ConsultationEntryDTO struct {
	MedicalRecordID uint                                       `json:"medical_record_id" binding:"required"`
	Date            string                                     `json:"date" binding:"required"`
	Motive          string                                     `json:"motive"`
	VisitType       string                                     `json:"visit_type"`
	Observation     string                                     `json:"observation"`
	Subjective      string                                     `json:"subjective"`
	Objective       string                                     `json:"objective"`
	Assessment      string                                     `json:"assessment"`
	Plan            string                                     `json:"plan"`
	APP             string                                     `json:"app"`
	APF             string                                     `json:"apf"`
	APQX            string                                     `json:"apqx"`
	Allergies       string                                     `json:"allergies"`
	Diagnoses       []clinical_models.DiagnosisItem            `json:"diagnoses"`
	VitalSigns      *clinical_models.MedicalReportVitalSigns   `json:"vital_signs"`
	Prescription    *clinical_models.MedicalReportPrescription `json:"prescription"`
}

type CreateMedicalReportDTO struct {
	Consultations []ConsultationEntryDTO `json:"consultations" binding:"required,dive"`
	Plan          string                 `json:"plan"`
}

type EmailMedicalDocumentDTO struct {
	Email string `json:"email" binding:"required,email"`
}
