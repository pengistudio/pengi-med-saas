package clinical_models

import (
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

// Exam order statuses. The server recomputes Status from the items' results;
// clients never send it (only the explicit "close as complete" action).
const (
	ExamOrderStatusIssued          = "issued"
	ExamOrderStatusPartialResults  = "partial_results"
	ExamOrderStatusCompleteResults = "complete_results"
	ExamOrderStatusVoided          = "voided"
)

// Exam order priorities.
const (
	ExamPriorityRoutine = "routine"
	ExamPriorityUrgent  = "urgent"
)

// ExamCatalogItem is one exam a tenant can order (catálogo de exámenes).
// Seeded rows carry IsDefault and the SeedKey of their seed entry, so
// "restore defaults" finds them even after the tenant renamed them.
type ExamCatalogItem struct {
	gorm.Model
	TenantID           uint   `json:"tenant_id" gorm:"index;not null"`
	Name               string `json:"name" gorm:"not null"`
	Category           string `json:"category" gorm:"not null;index"` // laboratory | imaging | other
	Subgroup           string `json:"subgroup"`
	DefaultIndications string `json:"default_indications"`
	Active             bool   `json:"active" gorm:"not null;default:true"`
	IsDefault          bool   `json:"is_default" gorm:"not null;default:false"`
	SeedKey            string `json:"-" gorm:"index"`
	SortOrder          int    `json:"sort_order" gorm:"not null;default:0"`
}

// ExamProfile is a named set of catalog exams ordered together (perfil).
type ExamProfile struct {
	gorm.Model
	TenantID  uint              `json:"tenant_id" gorm:"index;not null"`
	Name      string            `json:"name" gorm:"not null"`
	Active    bool              `json:"active" gorm:"not null;default:true"`
	IsDefault bool              `json:"is_default" gorm:"not null;default:false"`
	SeedKey   string            `json:"-" gorm:"index"`
	Items     []ExamCatalogItem `json:"items" gorm:"many2many:exam_profile_items;"`
}

// ExamOrder is a doctor's request for one or more exams for a patient. It
// always belongs to a patient and optionally to the consultation it was
// issued in (several orders per consultation). Number is a per-tenant
// correlative, printed as ORD-000123.
type ExamOrder struct {
	gorm.Model
	TenantID        uint                  `json:"tenant_id" gorm:"not null;uniqueIndex:idx_exam_order_tenant_number"`
	Number          uint                  `json:"number" gorm:"not null;uniqueIndex:idx_exam_order_tenant_number"`
	PatientID       uint                  `json:"patient_id" gorm:"not null;index"`
	Patient         *Patient              `json:"patient,omitempty" gorm:"foreignKey:PatientID"`
	MedicalRecordID *uint                 `json:"medical_record_id" gorm:"index"`
	OrderedByID     uint                  `json:"ordered_by_id" gorm:"not null;index"`
	DoctorID        *uint                 `json:"doctor_id" gorm:"index"` // OrderedByID stays the creating user
	Doctor          *doctor_models.Doctor `json:"doctor,omitempty" gorm:"foreignKey:DoctorID"`
	Diagnoses       datatypes.JSON        `json:"diagnoses" gorm:"type:jsonb;default:'[]'"`
	Priority        string                `json:"priority" gorm:"not null;default:'routine'"`
	Notes           string                `json:"notes"`
	DestinationLab  string                `json:"destination_lab"`
	Status          string                `json:"status" gorm:"not null;default:'issued';index"`
	ClosedManually  bool                  `json:"closed_manually" gorm:"not null;default:false"`
	VoidReason      string                `json:"void_reason"`
	VoidedAt        *time.Time            `json:"voided_at"`
	VoidedByID      *uint                 `json:"voided_by_id"`
	Items           []ExamOrderItem       `json:"items" gorm:"foreignKey:ExamOrderID"`
	DocumentSignature

	// Filled by the handlers, not stored.
	Code          string `json:"code" gorm:"-"`
	OrderedByName string `json:"ordered_by_name" gorm:"-"`
	PendingReview bool   `json:"pending_review" gorm:"-"`
}

// FormatExamOrderNumber is how an order number is shown and printed.
func FormatExamOrderNumber(number uint) string { return fmt.Sprintf("ORD-%06d", number) }

// ExamOrderItem is one exam requested in an order. Name, Category and
// Subgroup are copied from the catalog so the order never changes when the
// catalog is edited; CatalogItemID is nil for exams written by hand.
type ExamOrderItem struct {
	gorm.Model
	TenantID      uint       `json:"tenant_id" gorm:"index;not null"`
	ExamOrderID   uint       `json:"exam_order_id" gorm:"not null;index"`
	CatalogItemID *uint      `json:"catalog_item_id"`
	Name          string     `json:"name" gorm:"not null"`
	Category      string     `json:"category" gorm:"not null"`
	Subgroup      string     `json:"subgroup"`
	Indications   string     `json:"indications"`
	ReviewedByID  *uint      `json:"reviewed_by_id"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	// Filled by the handlers, not stored.
	ReviewedByName string `json:"reviewed_by_name,omitempty" gorm:"-"`
	// Results: the patient's attachments (Adjuntos) that cover this exam. A
	// deleted attachment drops out of the preload, so it no longer counts.
	Attachments []PatientAttachment `json:"attachments" gorm:"many2many:exam_order_item_attachments;"`
}

// HasResult reports whether at least one (non-deleted) result covers the item.
func (i ExamOrderItem) HasResult() bool { return len(i.Attachments) > 0 }

// ExamOrderCounter holds the last order number issued per tenant; it is
// incremented atomically inside the order's creation transaction.
type ExamOrderCounter struct {
	TenantID uint `gorm:"primaryKey;autoIncrement:false"`
	Value    uint `gorm:"not null;default:0"`
}

func (ExamCatalogItem) IsAuditable() bool { return false }
func (ExamProfile) IsAuditable() bool     { return false }
func (ExamOrder) IsAuditable() bool       { return true }
func (ExamOrderItem) IsAuditable() bool   { return true }
