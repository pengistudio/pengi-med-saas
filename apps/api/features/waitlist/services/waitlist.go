// Package waitlist_services holds the appointment waitlist's rules: when an
// entry leaves the list on its own (expired, booked), which entries a freed
// slot matches, the staff notification when a slot frees, and offering a
// slot by WhatsApp.
package waitlist_services

import (
	"fmt"
	"net/url"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

// DateLayout is the layout of the entries' and slots' dates (clinic time).
const DateLayout = "2006-01-02"

// DefaultTTL is how long an entry without an end date stays on the list.
const DefaultTTL = 30 * 24 * time.Hour

// Location is the clinic's time zone (dates and times of slots and entries).
func Location() *time.Location { return whatsapp_services.ClinicLocation() }

// Slot is a freed (or offered) time: a doctor (nil when the clinic has none),
// an appointment type (nil: untyped), a day "YYYY-MM-DD" and "HH:MM" times.
// PatientID is the patient who freed it, never a candidate for their own
// slot (0: none).
type Slot struct {
	DoctorID          *uint
	AppointmentTypeID *uint
	Date              string
	StartTime         string
	EndTime           string
	PatientID         uint
}

// Start is the instant the slot starts, in loc.
func (s Slot) Start(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(DateLayout+" 15:04", s.Date+" "+s.StartTime, loc)
}

// SlotFromAppointment is the slot an appointment takes.
func SlotFromAppointment(appt clinical_models.Appointment, loc *time.Location) (Slot, error) {
	start, err := whatsapp_services.AppointmentStart(appt, loc)
	if err != nil {
		return Slot{}, err
	}
	return Slot{
		DoctorID: appt.DoctorID, AppointmentTypeID: appt.AppointmentTypeID,
		Date: start.Format(DateLayout), StartTime: start.Format("15:04"), EndTime: appt.EndTime,
		PatientID: appt.PatientID,
	}, nil
}

// URL is the web waitlist view filtered for the slot (its candidates).
func (s Slot) URL() string {
	q := url.Values{}
	q.Set("date", s.Date)
	q.Set("start_time", s.StartTime)
	q.Set("end_time", s.EndTime)
	if s.DoctorID != nil {
		q.Set("doctor_id", fmt.Sprint(*s.DoctorID))
	}
	if s.AppointmentTypeID != nil {
		q.Set("appointment_type_id", fmt.Sprint(*s.AppointmentTypeID))
	}
	if s.PatientID != 0 {
		// The patient who freed it: never a candidate for their own slot.
		q.Set("patient_id", fmt.Sprint(s.PatientID))
	}
	return "/clinical/waitlist?" + q.Encode()
}

// ExpiresAt is when an entry created (or edited) at now leaves the list on
// its own: the end of dateTo (clinic time); without it, DefaultTTL after
// now, or after dateFrom when that is later (so a range starting in two
// months still gets its 30 days).
func ExpiresAt(now time.Time, dateFrom, dateTo string, loc *time.Location) time.Time {
	if dateTo != "" {
		if d, err := time.ParseInLocation(DateLayout, dateTo, loc); err == nil {
			return d.AddDate(0, 0, 1)
		}
	}
	base := now
	if dateFrom != "" {
		if d, err := time.ParseInLocation(DateLayout, dateFrom, loc); err == nil && d.After(base) {
			base = d
		}
	}
	return base.Add(DefaultTTL)
}

// Refresh closes the entries (db bound to a tenant, or System for all) that
// left the list on their own:
//   - expired: ExpiresAt passed;
//   - booked: the patient got an appointment created after the entry, with
//     its doctor (any doctor for an any-doctor entry), not cancelled.
//
// It runs before every read and match, so a closed entry never shows as
// waiting nor matches a slot (no worker needed).
func Refresh(db *gorm.DB, now time.Time) error {
	utc := now.UTC()
	if err := db.Model(&waitlist_models.WaitlistEntry{}).
		Where("status IN ? AND expires_at <= ?", waitlist_models.ActiveStatuses, utc).
		Updates(map[string]any{"status": waitlist_models.StatusExpired, "closed_at": utc, "updated_at": utc}).Error; err != nil {
		return err
	}
	return db.Model(&waitlist_models.WaitlistEntry{}).
		Where("status IN ?", waitlist_models.ActiveStatuses).
		Where(`EXISTS (SELECT 1 FROM appointments a
			WHERE a.tenant_id = waitlist_entries.tenant_id AND a.patient_id = waitlist_entries.patient_id
			AND a.deleted_at IS NULL AND a.created_at > waitlist_entries.created_at AND a.status <> ?
			AND (waitlist_entries.doctor_id IS NULL OR a.doctor_id = waitlist_entries.doctor_id))`,
			clinical_models.AppointmentStatusCancelled).
		Updates(map[string]any{"status": waitlist_models.StatusBooked, "closed_at": utc, "updated_at": utc}).Error
}

// matchQuery selects the active, unexpired entries (db bound to the tenant)
// a slot matches, before the move-earlier check:
//   - doctor: the slot's, or any doctor (an untyped slot without a doctor
//     only matches any-doctor entries);
//   - type: the entry has none, the slot has none, or they are the same;
//   - date: inside the entry's range (open ends match);
//   - never the patient who freed it.
func matchQuery(db *gorm.DB, slot Slot, now time.Time) *gorm.DB {
	q := db.Model(&waitlist_models.WaitlistEntry{}).
		Where("status IN ? AND expires_at > ?", waitlist_models.ActiveStatuses, now.UTC())
	if slot.PatientID != 0 {
		q = q.Where("patient_id <> ?", slot.PatientID)
	}
	if slot.DoctorID != nil {
		q = q.Where("(doctor_id IS NULL OR doctor_id = ?)", *slot.DoctorID)
	} else {
		q = q.Where("doctor_id IS NULL")
	}
	if slot.AppointmentTypeID != nil {
		q = q.Where("(appointment_type_id IS NULL OR appointment_type_id = ?)", *slot.AppointmentTypeID)
	}
	return q.Where("(date_from = '' OR date_from <= ?)", slot.Date).
		Where("(date_to = '' OR date_to >= ?)", slot.Date)
}

// Matching returns the entries (db bound to the tenant) slot matches, in
// signup order, with their patient, doctor and type. An entry that wants to
// move an appointment earlier only matches a slot before that appointment
// (an appointment gone or closed no longer constrains it).
func Matching(db *gorm.DB, slot Slot, now time.Time, loc *time.Location) ([]waitlist_models.WaitlistEntry, error) {
	var entries []waitlist_models.WaitlistEntry
	if err := matchQuery(db, slot, now).Preload("Patient").Preload("Doctor").Preload("AppointmentType").
		Order("created_at ASC, id ASC").Find(&entries).Error; err != nil {
		return nil, err
	}
	return filterMoveEarlier(db, entries, slot, loc)
}

// OfferedSlots returns, of entryIDs, those already offered slot (db bound
// to the tenant), from the offer history.
func OfferedSlots(db *gorm.DB, entryIDs []uint, slot Slot) (map[uint]bool, error) {
	out := map[uint]bool{}
	if len(entryIDs) == 0 {
		return out, nil
	}
	var rows []waitlist_models.WaitlistOffer
	if err := db.Where("entry_id IN ? AND date = ? AND start_time = ? AND doctor_id = ?",
		entryIDs, slot.Date, slot.StartTime, uintOr0(slot.DoctorID)).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.EntryID] = true
	}
	return out, nil
}

// Matches reports whether slot matches entry (db bound to the tenant).
func Matches(db *gorm.DB, entry waitlist_models.WaitlistEntry, slot Slot, now time.Time, loc *time.Location) (bool, error) {
	var entries []waitlist_models.WaitlistEntry
	if err := matchQuery(db, slot, now).Where("id = ?", entry.ID).Find(&entries).Error; err != nil {
		return false, err
	}
	kept, err := filterMoveEarlier(db, entries, slot, loc)
	return len(kept) == 1, err
}

func filterMoveEarlier(db *gorm.DB, entries []waitlist_models.WaitlistEntry, slot Slot, loc *time.Location) ([]waitlist_models.WaitlistEntry, error) {
	var ids []uint
	for _, e := range entries {
		if e.AppointmentID != nil {
			ids = append(ids, *e.AppointmentID)
		}
	}
	if len(ids) == 0 {
		return entries, nil
	}
	slotStart, err := slot.Start(loc)
	if err != nil {
		return nil, err
	}
	var appts []clinical_models.Appointment
	if err := db.Where("id IN ?", ids).Find(&appts).Error; err != nil {
		return nil, err
	}
	starts := make(map[uint]time.Time, len(appts))
	for _, a := range appts {
		if clinical_models.IsClosedStatus(a.Status) {
			continue
		}
		if start, err := whatsapp_services.AppointmentStart(a, loc); err == nil {
			starts[a.ID] = start
		}
	}
	out := entries[:0]
	for _, e := range entries {
		if e.AppointmentID != nil {
			if start, ok := starts[*e.AppointmentID]; ok && !start.After(slotStart) {
				continue // the slot is not earlier than what the patient has
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// ActiveDuplicate returns the patient's active entry (db bound to the
// tenant) for the same doctor (or any) and type, if there is one.
func ActiveDuplicate(db *gorm.DB, patientID uint, doctorID, typeID *uint, exceptID uint) (waitlist_models.WaitlistEntry, error) {
	q := db.Where("patient_id = ? AND status IN ? AND id <> ?", patientID, waitlist_models.ActiveStatuses, exceptID)
	if doctorID == nil {
		q = q.Where("doctor_id IS NULL")
	} else {
		q = q.Where("doctor_id = ?", *doctorID)
	}
	if typeID == nil {
		q = q.Where("appointment_type_id IS NULL")
	} else {
		q = q.Where("appointment_type_id = ?", *typeID)
	}
	var entry waitlist_models.WaitlistEntry
	err := q.Order("id").Limit(1).Find(&entry).Error
	return entry, err
}
