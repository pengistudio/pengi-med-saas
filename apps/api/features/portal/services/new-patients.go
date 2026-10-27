package portal_services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/ecid"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_models "pengi-med-saas/features/portal/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

// New-patient (first appointment) limits.
const (
	// MaxPendingNewPatientRequests is how many pending new-patient requests a
	// clinic may have at once (unverified people: a cap on abuse).
	MaxPendingNewPatientRequests = 20
	maxNameRunes                 = 100
	maxAgeYears                  = 120
)

// ErrInvalidPersonalData: the new patient's form breaks a rule (names,
// document, birth date, phone, email, or the verified contact is not in it).
var ErrInvalidPersonalData = errors.New("invalid new patient data")

// NormalizeNewPatientForm validates f and returns it normalized: names
// trimmed (1 to 100 characters), a cédula with a valid check digit or a
// passport (5 to 20 letters or digits, uppercased), a birth date in the past
// at most 120 years ago (clinic dates), the phone in E.164 without "+", the
// email lowercased. The verified contact (channel, contact; normalized) must
// be the form's email or phone. Consent to WhatsApp needs a phone.
func NormalizeNewPatientForm(f portal_dto.NewPatientForm, channel, contact string, now time.Time) (portal_dto.NewPatientForm, error) {
	out := portal_dto.NewPatientForm{
		FirstName:    strings.TrimSpace(f.FirstName),
		LastName:     strings.TrimSpace(f.LastName),
		DocumentType: f.DocumentType,
		BirthDate:    strings.TrimSpace(f.BirthDate),
	}
	for _, name := range []string{out.FirstName, out.LastName} {
		if n := utf8.RuneCountInString(name); n < 1 || n > maxNameRunes {
			return out, ErrInvalidPersonalData
		}
	}
	switch f.DocumentType {
	case portal_dto.DocumentTypeCedula:
		out.Document = strings.TrimSpace(f.Document)
		if !ecid.ValidCedula(out.Document) {
			return out, ErrInvalidPersonalData
		}
	case portal_dto.DocumentTypePassport:
		doc, ok := ecid.NormalizePassport(f.Document)
		if !ok {
			return out, ErrInvalidPersonalData
		}
		out.Document = doc
	default:
		return out, ErrInvalidPersonalData
	}
	birth, ok := agenda_services.ParseDate(out.BirthDate)
	if !ok {
		return out, ErrInvalidPersonalData
	}
	today := now.In(Location())
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	birthDate := time.Date(birth.Year(), birth.Month(), birth.Day(), 0, 0, 0, 0, time.UTC)
	if !birthDate.Before(todayDate) || birthDate.Before(todayDate.AddDate(-maxAgeYears, 0, 0)) {
		return out, ErrInvalidPersonalData
	}
	if raw := strings.TrimSpace(f.Phone); raw != "" {
		phone, err := whatsapp_services.NormalizePhone(raw)
		if err != nil {
			return out, ErrInvalidPersonalData
		}
		out.Phone = phone
	}
	if raw := strings.TrimSpace(f.Email); raw != "" {
		email, ok := NormalizeEmail(raw)
		if !ok {
			return out, ErrInvalidPersonalData
		}
		out.Email = email
	}
	switch channel {
	case portal_models.ChannelEmail:
		if out.Email != contact {
			return out, ErrInvalidPersonalData
		}
	case portal_models.ChannelWhatsApp:
		if out.Phone != contact {
			return out, ErrInvalidPersonalData
		}
	default:
		return out, ErrInvalidPersonalData
	}
	out.WhatsAppOptIn = f.WhatsAppOptIn && out.Phone != ""
	return out, nil
}

// LockNewPatients serializes the clinic's new-patient requests until tx ends
// (their per-contact, per-document and per-clinic limits).
func LockNewPatients(tx *gorm.DB, tenantID uint) error {
	return advisoryLock(tx, fmt.Sprintf("portal-new-patient:%d", tenantID))
}

// PendingNewPatientForms are the clinic's pending, unexpired new-patient
// requests (tx bound to the tenant) with their decoded forms; at most
// MaxPendingNewPatientRequests rows, the cap.
func PendingNewPatientForms(tx *gorm.DB, now time.Time) ([]portal_models.AppointmentRequest, []portal_dto.NewPatientForm, error) {
	var reqs []portal_models.AppointmentRequest
	if err := tx.Where("new_patient = ? AND status = ? AND expires_at > ?", true, portal_models.RequestPending, now.UTC()).
		Order("id").Find(&reqs).Error; err != nil {
		return nil, nil, err
	}
	forms := make([]portal_dto.NewPatientForm, len(reqs))
	for i, r := range reqs {
		_ = json.Unmarshal([]byte(r.NewPatientData), &forms[i])
	}
	return reqs, forms, nil
}

// ResolveNewPatient gives an approved new-patient request its patient, on tx
// (bound to the tenant): the tenant's patient with the form's document (the
// oldest), whose data is kept (only an empty phone or email is filled), or
// else a new patient from the form, with the request's doctor as médico de
// cabecera and the form's WhatsApp consent (source portal).
func ResolveNewPatient(tx *gorm.DB, req portal_models.AppointmentRequest, now time.Time) (uint, error) {
	var form portal_dto.NewPatientForm
	if err := json.Unmarshal([]byte(req.NewPatientData), &form); err != nil {
		return 0, fmt.Errorf("new patient data of request %d: %w", req.ID, err)
	}
	if strings.TrimSpace(form.Document) == "" {
		return 0, fmt.Errorf("new patient data of request %d has no document", req.ID)
	}
	if err := advisoryLock(tx, fmt.Sprintf("portal-new-patient-document:%d:%s", req.TenantID, form.Document)); err != nil {
		return 0, err
	}
	var existing clinical_models.Patient
	if err := tx.Where("document = ?", form.Document).Order("id ASC").Limit(1).Find(&existing).Error; err != nil {
		return 0, err
	}
	if existing.ID != 0 {
		fill := map[string]any{}
		if strings.TrimSpace(existing.Phone) == "" && form.Phone != "" {
			fill["phone"] = form.Phone
		}
		if strings.TrimSpace(existing.Email) == "" && form.Email != "" {
			fill["email"] = form.Email
		}
		if len(fill) > 0 {
			if err := tx.Model(&clinical_models.Patient{}).Where("id = ?", existing.ID).Updates(fill).Error; err != nil {
				return 0, err
			}
		}
		return existing.ID, nil
	}
	birth, ok := agenda_services.ParseDate(form.BirthDate)
	if !ok {
		return 0, fmt.Errorf("new patient data of request %d has an invalid birth date", req.ID)
	}
	fullName := strings.TrimSpace(form.FirstName + " " + form.LastName)
	doctorID := req.DoctorID
	patient := clinical_models.Patient{
		TenantID:  req.TenantID,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		FullName:  &fullName,
		Document:  form.Document,
		BirthDate: time.Date(birth.Year(), birth.Month(), birth.Day(), 0, 0, 0, 0, time.UTC),
		Phone:     form.Phone,
		Email:     form.Email,
		DoctorID:  &doctorID,
	}
	if form.WhatsAppOptIn && form.Phone != "" {
		at := now.UTC()
		patient.WhatsAppOptIn, patient.WhatsAppOptInAt = true, &at
		patient.WhatsAppOptInSource = clinical_models.WhatsAppOptInSourcePortal
	}
	if err := tx.Omit("Doctor", "MedicalRecords", "Appointments").Create(&patient).Error; err != nil {
		return 0, err
	}
	return patient.ID, nil
}
