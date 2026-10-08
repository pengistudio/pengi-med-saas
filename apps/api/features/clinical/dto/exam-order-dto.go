package dto

import clinical_models "pengi-med-saas/features/clinical/models"

// ── Exam catalog ────────────────────────────────────────────────────────────

type CreateExamCatalogItemRequest struct {
	Name               string `json:"name" binding:"required"`
	Category           string `json:"category" binding:"required,oneof=laboratory imaging other"`
	Subgroup           string `json:"subgroup"`
	DefaultIndications string `json:"default_indications"`
}

type UpdateExamCatalogItemRequest struct {
	Name               *string `json:"name"`
	Category           *string `json:"category" binding:"omitempty,oneof=laboratory imaging other"`
	Subgroup           *string `json:"subgroup"`
	DefaultIndications *string `json:"default_indications"`
	Active             *bool   `json:"active"`
}

type CreateExamProfileRequest struct {
	Name    string `json:"name" binding:"required"`
	ItemIDs []uint `json:"item_ids"`
}

type UpdateExamProfileRequest struct {
	Name    *string `json:"name"`
	Active  *bool   `json:"active"`
	ItemIDs *[]uint `json:"item_ids"`
}

type RestoreExamCatalogResponse struct {
	RestoredItems    int `json:"restored_items"`
	RestoredProfiles int `json:"restored_profiles"`
}

// ── Exam orders ─────────────────────────────────────────────────────────────

// ExamOrderItemInput is one exam of an order. ID identifies an existing item
// on update (omit it for new exams). With CatalogItemID, Name/Category/
// Subgroup come from the catalog and are ignored here; without it (exam
// written by hand) Name is required and Category defaults to "other".
// Indications nil = the catalog's default (new item) or unchanged (existing).
type ExamOrderItemInput struct {
	ID            *uint   `json:"id"`
	CatalogItemID *uint   `json:"catalog_item_id"`
	Name          string  `json:"name"`
	Category      string  `json:"category" binding:"omitempty,oneof=laboratory imaging other"`
	Subgroup      string  `json:"subgroup"`
	Indications   *string `json:"indications"`
}

type CreateExamOrderRequest struct {
	PatientID       uint                            `json:"patient_id" binding:"required"`
	MedicalRecordID *uint                           `json:"medical_record_id"`
	DoctorID        *uint                           `json:"doctor_id"` // create: defaults per doctor_services.Resolve; update: omitted keeps it
	Diagnoses       []clinical_models.DiagnosisItem `json:"diagnoses"`
	Priority        string                          `json:"priority" binding:"omitempty,oneof=routine urgent"`
	Notes           string                          `json:"notes"`
	DestinationLab  string                          `json:"destination_lab"`
	Items           []ExamOrderItemInput            `json:"items" binding:"required,min=1,dive"`
}

// UpdateExamOrderRequest replaces the order's editable content. Once any exam
// has a result, only new items may be added: header fields and existing
// items must be sent unchanged.
type UpdateExamOrderRequest struct {
	MedicalRecordID *uint                           `json:"medical_record_id"`
	DoctorID        *uint                           `json:"doctor_id"` // create: defaults per doctor_services.Resolve; update: omitted keeps it
	Diagnoses       []clinical_models.DiagnosisItem `json:"diagnoses"`
	Priority        string                          `json:"priority" binding:"omitempty,oneof=routine urgent"`
	Notes           string                          `json:"notes"`
	DestinationLab  string                          `json:"destination_lab"`
	Items           []ExamOrderItemInput            `json:"items" binding:"required,min=1,dive"`
}

type VoidExamOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ReviewExamResultsRequest struct {
	ItemIDs []uint `json:"item_ids" binding:"required,min=1"`
}
