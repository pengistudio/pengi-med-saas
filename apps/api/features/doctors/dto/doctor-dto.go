package doctor_dto

import doctor_models "pengi-med-saas/features/doctors/models"

// DoctorProfileRequest holds the profile fields: what an admin sets on create
// and what the linked user can edit on their own profile. It has no Active or
// UserID: those are only changed through their own admin endpoints.
type DoctorProfileRequest struct {
	FullName             string `json:"full_name" binding:"required"`
	Specialty            string `json:"specialty" binding:"required"`
	SpecialtyOther       string `json:"specialty_other"`
	IDNumber             string `json:"id_number"`
	ProfessionalRegistry string `json:"professional_registry"`
	Phone                string `json:"phone"`
	Email                string `json:"email" binding:"omitempty,email"`
	Color                string `json:"color"` // empty → next palette color
}

// CreateDoctorRequest is the admin create: the profile plus an optional link
// to a user of the tenant.
type CreateDoctorRequest struct {
	DoctorProfileRequest
	UserID *uint `json:"user_id"`
}

// UpdateDoctorRequest is a partial profile update (admin or own profile).
type UpdateDoctorRequest struct {
	FullName             *string `json:"full_name"`
	Specialty            *string `json:"specialty"`
	SpecialtyOther       *string `json:"specialty_other"`
	IDNumber             *string `json:"id_number"`
	ProfessionalRegistry *string `json:"professional_registry"`
	Phone                *string `json:"phone"`
	Email                *string `json:"email" binding:"omitempty,email"`
	Color                *string `json:"color"`
}

// LinkUserRequest links a doctor to a user of the tenant, or unlinks it when
// user_id is null.
type LinkUserRequest struct {
	UserID *uint `json:"user_id"`
}

// SpecialtyResponse is one entry of the specialty catalog.
type SpecialtyResponse struct {
	Code     string `json:"code"`
	LabelKey string `json:"label_key"`
}

// DoctorStatusResponse tells the frontend what onboarding/notices to show.
type DoctorStatusResponse struct {
	HasProfile    bool                  `json:"has_profile"`
	Doctor        *doctor_models.Doctor `json:"doctor"` // the current user's profile, or null
	ActiveDoctors int64                 `json:"active_doctors"`
	NeedsReview   bool                  `json:"needs_review"` // some doctor still has needs_review
}
