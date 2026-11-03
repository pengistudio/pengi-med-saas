package agenda_services

import (
	"strings"
	"time"

	agenda_models "pengi-med-saas/features/agenda/models"

	"gorm.io/gorm"
)

// ClinicHours are the clinic's office hours: the union of the weekly
// schedules of its active doctors, merged per weekday (Days[0] is Sunday,
// like time.Weekday). The clinic has no hours of its own; the agenda is the
// one place they are loaded. ClosedDates are the dates a clinic-wide
// full-day block (holiday) covers within the window loaded.
type ClinicHours struct {
	Days        [7][]ClockRange
	ClosedDates map[string]bool
}

// ClockRange is a working range in minutes after midnight, [Start, End).
type ClockRange struct{ Start, End int }

// LoadClinicHours reads the hours (db bound to the tenant) and the
// clinic-wide full-day blocks of the dates from..to ("YYYY-MM-DD").
func LoadClinicHours(db *gorm.DB, from, to string) (ClinicHours, error) {
	h := ClinicHours{ClosedDates: map[string]bool{}}
	var schedules []agenda_models.DoctorSchedule
	err := db.Model(&agenda_models.DoctorSchedule{}).
		Joins("JOIN doctors ON doctors.id = doctor_schedules.doctor_id AND doctors.deleted_at IS NULL").
		Where("doctors.active = ?", true).
		Select("doctor_schedules.weekday", "doctor_schedules.start_time", "doctor_schedules.end_time").
		Find(&schedules).Error
	if err != nil {
		return h, err
	}
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		for _, s := range dayRanges(schedules, wd) {
			h.Days[wd] = append(h.Days[wd], ClockRange{s.start, s.end})
		}
	}
	// dayRanges merges per doctor list; merge again across doctors.
	for wd := range h.Days {
		spans := make([]span, 0, len(h.Days[wd]))
		for _, r := range h.Days[wd] {
			spans = append(spans, span{r.Start, r.End})
		}
		h.Days[wd] = nil
		for _, s := range mergeRanges(spans) {
			h.Days[wd] = append(h.Days[wd], ClockRange{s.start, s.end})
		}
	}
	var blocks []agenda_models.ScheduleBlock
	if err := db.Where("doctor_id IS NULL AND start_date <= ? AND end_date >= ?", to, from).Find(&blocks).Error; err != nil {
		return h, err
	}
	for _, b := range blocks {
		if !b.FullDay() {
			continue
		}
		start, ok1 := ParseDate(maxDate(b.StartDate, from))
		end, ok2 := ParseDate(minDate(b.EndDate, to))
		if !ok1 || !ok2 {
			continue
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			h.ClosedDates[d.Format(DateLayout)] = true
		}
	}
	return h, nil
}

// Empty reports whether no weekday has hours.
func (h ClinicHours) Empty() bool {
	for _, d := range h.Days {
		if len(d) > 0 {
			return false
		}
	}
	return true
}

// Open reports whether the clinic is attending at t (in loc): t's weekday
// has a range containing its clock time and the date is not a clinic-wide
// holiday.
func (h ClinicHours) Open(t time.Time, loc *time.Location) bool {
	local := t.In(loc)
	if h.ClosedDates[local.Format(DateLayout)] {
		return false
	}
	minute := local.Hour()*60 + local.Minute()
	for _, r := range h.Days[local.Weekday()] {
		if minute >= r.Start && minute < r.End {
			return true
		}
	}
	return false
}

// spanishDays are the weekday names of Text, Monday first.
var spanishDays = []struct {
	day  time.Weekday
	name string
}{
	{time.Monday, "Lunes"}, {time.Tuesday, "Martes"}, {time.Wednesday, "Miércoles"}, {time.Thursday, "Jueves"},
	{time.Friday, "Viernes"}, {time.Saturday, "Sábado"}, {time.Sunday, "Domingo"},
}

// Text is the weekly hours in Spanish, one day per line ("Lunes: 08:00 a
// 13:00 y 15:00 a 19:00"), days without hours as "cerrado". It is what the
// WhatsApp assistant may quote: patients are answered in Spanish only.
func (h ClinicHours) Text() string {
	lines := make([]string, 0, 7)
	for _, d := range spanishDays {
		ranges := h.Days[d.day]
		if len(ranges) == 0 {
			lines = append(lines, d.name+": cerrado")
			continue
		}
		parts := make([]string, 0, len(ranges))
		for _, r := range ranges {
			parts = append(parts, FormatClock(r.Start)+" a "+FormatClock(r.End))
		}
		lines = append(lines, d.name+": "+strings.Join(parts, " y "))
	}
	return strings.Join(lines, "\n")
}

func maxDate(a, b string) string {
	if a > b {
		return a
	}
	return b
}

func minDate(a, b string) string {
	if a < b {
		return a
	}
	return b
}
