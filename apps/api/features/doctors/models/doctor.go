package doctor_models

import "gorm.io/gorm"

// Doctor is a professional who attends patients in a tenant. It is
// independent of the user's role: an admin can have a profile, and a doctor
// may have no account at all (UserID nil: listed and printed, cannot sign).
// Doctors are deactivated, not deleted; a hard delete is only allowed while no
// record references them (doctor_services.IsReferenced).
type Doctor struct {
	gorm.Model
	TenantID uint `gorm:"not null;uniqueIndex:idx_doctor_tenant_user" json:"tenant_id"`
	// UserID links the profile to an account; unique per tenant when set.
	UserID               *uint  `gorm:"uniqueIndex:idx_doctor_tenant_user" json:"user_id"`
	FullName             string `gorm:"not null" json:"full_name"`
	Specialty            string `gorm:"type:varchar(50);not null" json:"specialty"` // doctor_data.Specialties code
	SpecialtyOther       string `json:"specialty_other"`                            // name when Specialty is "other"
	IDNumber             string `json:"id_number"`                                  // cédula
	ProfessionalRegistry string `json:"professional_registry"`                      // Senescyt / MSP registry
	Phone                string `json:"phone"`
	Email                string `json:"email"`
	Color                string `gorm:"type:varchar(7)" json:"color"` // #RRGGBB, agenda color
	// Active is written with Update("active", ...): GORM skips a false zero
	// value on create and the column default would win.
	Active bool `gorm:"not null;default:true" json:"active"`
	// NeedsReview marks profiles created by the data migration from existing
	// users, so the admin is asked once to review them. Cleared on edit or ack.
	NeedsReview bool `gorm:"not null;default:false" json:"needs_review"`
}

func (Doctor) IsAuditable() bool { return true }
