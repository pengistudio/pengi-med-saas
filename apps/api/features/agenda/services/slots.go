package agenda_services

import (
	"sort"
	"time"

	"gorm.io/gorm"

	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_models "pengi-med-saas/features/portal/models"
)

// slotGrid is the minute grid every offered start time sits on.
const slotGrid = 5

// SlotQuery asks for the free start times of some doctors. Dates are
// "YYYY-MM-DD" (From..To inclusive, already clamped to the booking window by
// the caller); times are clock times in Loc.
type SlotQuery struct {
	DoctorIDs []uint
	Duration  int // minutes of the appointment type
	From, To  string
	Now       time.Time
	Loc       *time.Location
	// MinNotice drops start times earlier than Now + MinNotice.
	MinNotice time.Duration
	// IgnoreRequestID is a request whose hold does not count (0 = none).
	IgnoreRequestID uint
}

// FreeSlotMap maps doctor id → date → free start times ("HH:MM", ascending).
// Days without free times are absent.
type FreeSlotMap map[uint]map[string][]string

// Has reports whether start is a free time of doctorID on date.
func (m FreeSlotMap) Has(doctorID uint, date, start string) bool {
	for _, t := range m[doctorID][date] {
		if t == start {
			return true
		}
	}
	return false
}

// busy is a taken span of one doctor on one date.
type busy struct {
	doctorID uint
	date     string
	span     span
}

// FreeSlots returns, on a tenant-bound db, the start times each doctor can
// take a new appointment of q.Duration minutes: inside the doctor's weekly
// schedule, outside blocks (the doctor's or the clinic's), not overlapping an
// appointment of the doctor (any status but cancelled) nor a pending,
// unexpired portal request. Candidates step by the duration from the start of
// each working range (on the 5-minute grid) and end within it. A doctor
// without a weekly schedule gets nothing.
func FreeSlots(db *gorm.DB, q SlotQuery) (FreeSlotMap, error) {
	out := FreeSlotMap{}
	if len(q.DoctorIDs) == 0 || q.Duration <= 0 || q.Loc == nil {
		return out, nil
	}
	from, ok1 := ParseDate(q.From)
	to, ok2 := ParseDate(q.To)
	if !ok1 || !ok2 || from.After(to) {
		return out, nil
	}

	var schedules []agenda_models.DoctorSchedule
	if err := db.Where("doctor_id IN ?", q.DoctorIDs).Find(&schedules).Error; err != nil {
		return nil, err
	}
	byDoctor := map[uint][]agenda_models.DoctorSchedule{}
	for _, s := range schedules {
		byDoctor[s.DoctorID] = append(byDoctor[s.DoctorID], s)
	}
	blocks, err := BlocksBetween(db, q.DoctorIDs, true, q.From, q.To)
	if err != nil {
		return nil, err
	}
	taken, err := takenSpans(db, q)
	if err != nil {
		return nil, err
	}

	earliest := q.Now.Add(q.MinNotice)
	for _, doctorID := range q.DoctorIDs {
		if len(byDoctor[doctorID]) == 0 {
			continue
		}
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			date := d.Format(DateLayout)
			times := daySlots(byDoctor[doctorID], blocks, taken, doctorID, date, d.Weekday(), q.Duration, func(start int) bool {
				instant := time.Date(d.Year(), d.Month(), d.Day(), start/60, start%60, 0, 0, q.Loc)
				return !instant.Before(earliest)
			})
			if len(times) == 0 {
				continue
			}
			if out[doctorID] == nil {
				out[doctorID] = map[string][]string{}
			}
			out[doctorID][date] = times
		}
	}
	return out, nil
}

// daySlots is one doctor's free start times on date.
func daySlots(schedules []agenda_models.DoctorSchedule, blocks []agenda_models.ScheduleBlock, taken []busy,
	doctorID uint, date string, weekday time.Weekday, duration int, notTooSoon func(int) bool) []string {
	var spans []span
	for _, b := range blocks {
		if b.DoctorID != nil && *b.DoctorID != doctorID {
			continue
		}
		if date < b.StartDate || date > b.EndDate {
			continue
		}
		if b.FullDay() {
			return nil
		}
		bs, be, ok := ValidRange(b.StartTime, b.EndTime, 0)
		if !ok {
			return nil // malformed rows block conservatively
		}
		spans = append(spans, span{bs, be})
	}
	for _, t := range taken {
		if t.doctorID == doctorID && t.date == date {
			spans = append(spans, t.span)
		}
	}
	var times []string
	for _, r := range dayRanges(schedules, weekday) {
		start := r.start
		if rem := start % slotGrid; rem != 0 {
			start += slotGrid - rem
		}
		for s := start; s+duration <= r.end; s += duration {
			e := s + duration
			free := true
			for _, b := range spans {
				if b.start < e && b.end > s {
					free = false
					break
				}
			}
			if free && notTooSoon(s) {
				times = append(times, FormatClock(s))
			}
		}
	}
	sort.Strings(times)
	return times
}

// takenSpans loads the appointments (not cancelled) and the pending,
// unexpired portal requests of q's doctors on q's dates.
func takenSpans(db *gorm.DB, q SlotQuery) ([]busy, error) {
	var out []busy
	appts, err := DoctorAppointments(db, q.DoctorIDs, q.From, q.To, q.Loc)
	if err != nil {
		return nil, err
	}
	for _, a := range appts {
		if s, e, ok := ValidRange(a.StartTime, a.EndTime, 0); ok && a.DoctorID != nil {
			out = append(out, busy{doctorID: *a.DoctorID, date: a.Date.In(q.Loc).Format(DateLayout), span: span{s, e}})
		}
	}

	var requests []portal_models.AppointmentRequest
	rq := db.Select("id", "doctor_id", "date", "start_time", "end_time").
		Where("doctor_id IN ? AND status = ? AND expires_at > ? AND date >= ? AND date <= ?",
			q.DoctorIDs, portal_models.RequestPending, q.Now.UTC(), q.From, q.To)
	if q.IgnoreRequestID != 0 {
		rq = rq.Where("id <> ?", q.IgnoreRequestID)
	}
	if err := rq.Find(&requests).Error; err != nil {
		return nil, err
	}
	for _, r := range requests {
		if s, e, ok := ValidRange(r.StartTime, r.EndTime, 0); ok {
			out = append(out, busy{doctorID: r.DoctorID, date: r.Date, span: span{s, e}})
		}
	}
	return out, nil
}

// DoctorAppointments returns the appointments (any status but cancelled) of
// the doctors whose clinic date (Date in loc, the frontend stores local
// midnight) is within from..to.
func DoctorAppointments(db *gorm.DB, doctorIDs []uint, from, to string, loc *time.Location) ([]clinical_models.Appointment, error) {
	f, ok1 := ParseDate(from)
	t, ok2 := ParseDate(to)
	if !ok1 || !ok2 || len(doctorIDs) == 0 {
		return nil, nil
	}
	// Coarse window on the stored instant (a day of margin each side), exact
	// clinic date filtered in Go.
	lo := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
	hi := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 2)
	var rows []clinical_models.Appointment
	if err := db.Select("id", "doctor_id", "date", "start_time", "end_time", "status").
		Where("doctor_id IN ? AND status <> ? AND date >= ? AND date < ?", doctorIDs, "cancelled", lo, hi).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, a := range rows {
		d := a.Date.In(loc).Format(DateLayout)
		if d >= from && d <= to {
			out = append(out, a)
		}
	}
	return out, nil
}

// AppointmentOverlaps reports whether doctorID has an appointment (not
// cancelled) on date overlapping start-end ("HH:MM").
func AppointmentOverlaps(db *gorm.DB, doctorID uint, date, start, end string, loc *time.Location) (bool, error) {
	s, e, ok := ValidRange(start, end, 0)
	if !ok {
		return false, nil
	}
	appts, err := DoctorAppointments(db, []uint{doctorID}, date, date, loc)
	if err != nil {
		return false, err
	}
	for _, a := range appts {
		if as, ae, ok := ValidRange(a.StartTime, a.EndTime, 0); ok && as < e && ae > s {
			return true, nil
		}
	}
	return false, nil
}
