package whatsapp_services

import (
	"errors"
	"os"
	"sort"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
)

// DefaultTimeZone is the clinics' time zone when TZ is unset.
const DefaultTimeZone = "America/Guayaquil"

// Reminder offset limits (hours before the appointment).
const (
	MinOffsetHours = 1
	MaxOffsetHours = 168
	MaxOffsets     = 3
)

// ErrInvalidOffsets means a reminder offset list breaks the limits above.
var ErrInvalidOffsets = errors.New("invalid reminder offsets")

// ClinicLocation is the time zone appointment times are written in: TZ (as
// Google Calendar sync uses), else America/Guayaquil.
func ClinicLocation() *time.Location {
	for _, name := range []string{os.Getenv("TZ"), DefaultTimeZone} {
		if name == "" {
			continue
		}
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}

// AppointmentStart is the instant an appointment starts: the calendar day of
// Date in loc (the frontend sends local midnight) at StartTime ("HH:MM") in loc.
func AppointmentStart(a clinical_models.Appointment, loc *time.Location) (time.Time, error) {
	clock, err := time.Parse("15:04", a.StartTime)
	if err != nil {
		return time.Time{}, err
	}
	d := a.Date.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), clock.Hour(), clock.Minute(), 0, 0, loc), nil
}

// DueOffset returns the reminder offset to send now for an appointment
// starting at start, or 0 for none. An offset o is due when start falls in
// (now, now+o]. When several are due — the appointment was created (or the
// offsets configured) after a larger window had opened — only the smallest
// one is sent, so the patient gets one timely reminder instead of a burst;
// the larger ones are then never due alone again.
func DueOffset(start, now time.Time, offsets []int) int {
	if !start.After(now) {
		return 0
	}
	until := start.Sub(now)
	due := 0
	for _, o := range offsets {
		if o <= 0 {
			continue
		}
		if until <= time.Duration(o)*time.Hour && (due == 0 || o < due) {
			due = o
		}
	}
	return due
}

// NormalizeOffsets validates offsets (1–168 h, at most 3) and returns them
// de-duplicated, largest first.
func NormalizeOffsets(offsets []int) ([]int, error) {
	seen := map[int]bool{}
	out := make([]int, 0, len(offsets))
	for _, o := range offsets {
		if o < MinOffsetHours || o > MaxOffsetHours {
			return nil, ErrInvalidOffsets
		}
		if !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	if len(out) > MaxOffsets {
		return nil, ErrInvalidOffsets
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out, nil
}

// MaxOffset is the largest offset (0 for none).
func MaxOffset(offsets []int) int {
	m := 0
	for _, o := range offsets {
		if o > m {
			m = o
		}
	}
	return m
}
