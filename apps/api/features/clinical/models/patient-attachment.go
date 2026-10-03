package clinical_models

import (
	"time"

	"gorm.io/gorm"
)

// Attachment categories: a fixed set, no tenant-defined categories.
const (
	AttachmentCategoryLabResult      = "lab_result"
	AttachmentCategoryImaging        = "imaging"
	AttachmentCategoryExternalReport = "external_report"
	AttachmentCategoryClinicalPhoto  = "clinical_photo"
	AttachmentCategoryConsent        = "consent"
	AttachmentCategoryOther          = "other"
)

// AttachmentCategories lists every valid PatientAttachment.Category.
var AttachmentCategories = []string{
	AttachmentCategoryLabResult,
	AttachmentCategoryImaging,
	AttachmentCategoryExternalReport,
	AttachmentCategoryClinicalPhoto,
	AttachmentCategoryConsent,
	AttachmentCategoryOther,
}

// IsAttachmentCategory reports whether category is one of AttachmentCategories.
func IsAttachmentCategory(category string) bool {
	for _, c := range AttachmentCategories {
		if c == category {
			return true
		}
	}
	return false
}

// PatientAttachment (Adjunto) is a file that belongs to a patient, optionally
// linked to one of the same patient's consultations. The file itself lives
// encrypted in the tenant file store under StoredName and is never replaced;
// only the metadata is editable. Deletion is soft, with who and why.
type PatientAttachment struct {
	gorm.Model
	TenantID        uint      `json:"tenant_id" gorm:"index"`
	PatientID       uint      `json:"patient_id" gorm:"index"`
	MedicalRecordID *uint     `json:"medical_record_id,omitempty" gorm:"index"`
	Category        string    `json:"category" gorm:"size:32;not null"`
	TakenAt         time.Time `json:"taken_at" gorm:"type:date;index"` // exam day (a calendar date); the upload day when not given
	Description     string    `json:"description" gorm:"size:255"`
	FileName        string    `json:"file_name" gorm:"size:255"` // as uploaded, for display and download
	MimeType        string    `json:"mime_type" gorm:"size:64"`  // detected from the content, never the client's
	Size            int64     `json:"size"`
	SHA256          string    `json:"sha256" gorm:"column:sha256;size:64;index"` // of the original content
	StoredName      string    `json:"-" gorm:"size:128"`                         // generated; never the user's file name
	UploadedByID    uint      `json:"uploaded_by_id"`
	DeletedByID     *uint     `json:"deleted_by_id,omitempty"`
	DeleteReason    string    `json:"delete_reason,omitempty"`
}

func (PatientAttachment) IsAuditable() bool { return true }
