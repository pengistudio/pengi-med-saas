package doctor_services

import (
	"errors"
	"net/http"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"

	doctor_models "pengi-med-saas/features/doctors/models"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Errors of doctor resolution and signing. Callers map them to responses
// with ErrorResponse.
var (
	// ErrNoDoctors: the tenant has no active doctor and the row needs one.
	ErrNoDoctors = errors.New("doctors: register a doctor first")
	// ErrDoctorRequired: several active doctors and no default applies.
	ErrDoctorRequired = errors.New("doctors: pick the doctor")
	// ErrDoctorInvalid: the doctor sent is not an active doctor of the tenant.
	ErrDoctorInvalid = errors.New("doctors: invalid doctor")
	// ErrSignNotDoctor: the signer is not the user linked to the document's doctor.
	ErrSignNotDoctor = errors.New("doctors: only the document's doctor can sign it")
	// ErrSignNoAccount: the document's doctor has no account, so nobody can sign it.
	ErrSignNoAccount = errors.New("doctors: the document's doctor has no account")
)

// Policy says how strictly a kind of row needs a doctor.
type Policy int

const (
	// Optional: validated when sent and auto-assigned when the tenant has
	// exactly one active doctor, never required (patient's cabecera).
	Optional Policy = iota
	// RequiredWhenAny: required once the tenant has an active doctor; with
	// none the row is saved without one (appointments).
	RequiredWhenAny
	// Required: like RequiredWhenAny, but with no active doctor the row can't
	// be created (medical records and signable documents).
	Required
)

// Choice describes the doctor of a new row.
type Choice struct {
	Requested *uint   // doctor_id sent by the client, if any
	Fallbacks []*uint // defaults tried after the current user's doctor, in order
	Policy    Policy
}

// Resolve picks the doctor of a new row on a tenant-bound db:
//  1. the doctor sent, which must be an active doctor of the tenant;
//  2. with no active doctor: nil, or ErrNoDoctors when Policy is Required;
//  3. with exactly one active doctor: that one;
//  4. with several (not for Optional): the current user's doctor, then each
//     fallback that is an active doctor; else ErrDoctorRequired.
func Resolve(c *gin.Context, db *gorm.DB, ch Choice) (*uint, error) {
	var active []doctor_models.Doctor
	if err := db.Model(&doctor_models.Doctor{}).Select("id", "user_id").Where("active = ?", true).Order("id").Find(&active).Error; err != nil {
		return nil, err
	}
	isActive := func(id *uint) bool {
		if id == nil {
			return false
		}
		for _, d := range active {
			if d.ID == *id {
				return true
			}
		}
		return false
	}
	if ch.Requested != nil {
		if !isActive(ch.Requested) {
			return nil, ErrDoctorInvalid
		}
		id := *ch.Requested
		return &id, nil
	}
	switch len(active) {
	case 0:
		if ch.Policy == Required {
			return nil, ErrNoDoctors
		}
		return nil, nil
	case 1:
		id := active[0].ID
		return &id, nil
	}
	if ch.Policy == Optional {
		return nil, nil
	}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		for _, d := range active {
			if d.UserID != nil && *d.UserID == uint(uid) {
				id := d.ID
				return &id, nil
			}
		}
	}
	for _, fb := range ch.Fallbacks {
		if isActive(fb) {
			id := *fb
			return &id, nil
		}
	}
	return nil, ErrDoctorRequired
}

// ValidateChange checks the doctor_id sent on an update: keeping the current
// doctor is always fine (even if deactivated since); a different one must be
// an active doctor of the tenant.
func ValidateChange(db *gorm.DB, requested uint, current *uint) error {
	if current != nil && *current == requested {
		return nil
	}
	var n int64
	if err := db.Model(&doctor_models.Doctor{}).Where("id = ? AND active = ?", requested, true).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrDoctorInvalid
	}
	return nil
}

// First returns the first of ids that is a doctor of the tenant (active or
// not), or nil: the doctor a document prints, following its fallback chain.
func First(db *gorm.DB, ids ...*uint) *doctor_models.Doctor {
	for _, id := range ids {
		if id == nil {
			continue
		}
		var d doctor_models.Doctor
		if err := db.Limit(1).Find(&d, *id).Error; err == nil && d.ID != 0 {
			return &d
		}
	}
	return nil
}

// CanSign checks that userID may sign a document whose doctor is the first
// of doctorIDs that is set. Documents without a doctor (created before
// doctors existed) can be signed by any user linked to a doctor profile.
func CanSign(db *gorm.DB, userID uint, doctorIDs ...*uint) error {
	var doctorID *uint
	for _, id := range doctorIDs {
		if id != nil {
			doctorID = id
			break
		}
	}
	if doctorID == nil {
		// Legacy document: the signer must at least be a doctor of the
		// tenant, so a receptionist with the route permission can't sign it.
		var n int64
		if err := db.Model(&doctor_models.Doctor{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrSignNotDoctor
		}
		return nil
	}
	d := First(db, doctorID)
	if d == nil {
		return ErrDoctorInvalid
	}
	if d.UserID == nil {
		return ErrSignNoAccount
	}
	if *d.UserID != userID {
		return ErrSignNotDoctor
	}
	return nil
}

// ErrorResponse maps the errors above to their response; ok is false for
// any other error (the caller logs it and answers 500).
func ErrorResponse(err error) (resp envelope.Response, ok bool) {
	switch {
	case errors.Is(err, ErrNoDoctors):
		return envelope.ErrorResponse(http.StatusConflict, "doctors.error.register_first", core_errors.ErrDoctorRegisterFirst), true
	case errors.Is(err, ErrDoctorRequired):
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.selection_required", core_errors.ErrDoctorSelectionRequired), true
	case errors.Is(err, ErrDoctorInvalid):
		return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.not_available", core_errors.ErrDoctorNotAvailable), true
	case errors.Is(err, ErrSignNotDoctor):
		return envelope.ErrorResponse(http.StatusForbidden, "doctors.error.sign_not_owner", core_errors.ErrDoctorSignNotOwner), true
	case errors.Is(err, ErrSignNoAccount):
		return envelope.ErrorResponse(http.StatusConflict, "doctors.error.sign_no_account", core_errors.ErrDoctorSignNoAccount), true
	}
	return envelope.Response{}, false
}
