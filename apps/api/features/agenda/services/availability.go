package agenda_services

import (
	"strings"
	"time"

	"gorm.io/gorm"

	agenda_dto "pengi-med-saas/features/agenda/dto"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// Availability statuses of a time slot.
const (
	StatusInside     = "inside"      // within a working range of the day
	StatusOutside    = "outside"     // the doctor has a schedule, the slot is not in it
	StatusBlocked    = "blocked"     // a doctor or clinic block overlaps the slot
	StatusNoSchedule = "no_schedule" // the doctor has no weekly schedule at all (no warning)
)

// MaxRangeDays bounds the agenda range endpoint.
const MaxRangeDays = 62

// All functions take a tenant-bound db (tenantdb.For / ForTenant).

// BlocksBetween returns the blocks overlapping the dates from..to (inclusive)
// of the given doctors plus, with clinic, the clinic-wide ones.
func BlocksBetween(db *gorm.DB, doctorIDs []uint, clinic bool, from, to string) ([]agenda_models.ScheduleBlock, error) {
	q := db.Where("start_date <= ? AND end_date >= ?", to, from)
	switch {
	case len(doctorIDs) > 0 && clinic:
		q = q.Where("doctor_id IN ? OR doctor_id IS NULL", doctorIDs)
	case len(doctorIDs) > 0:
		q = q.Where("doctor_id IN ?", doctorIDs)
	case clinic:
		q = q.Where("doctor_id IS NULL")
	default:
		return []agenda_models.ScheduleBlock{}, nil
	}
	blocks := []agenda_models.ScheduleBlock{}
	err := q.Order("start_date, start_time, id").Find(&blocks).Error
	return blocks, err
}

// blockOverlaps reports whether block covers part of [start, end) on date.
func blockOverlaps(b agenda_models.ScheduleBlock, date string, start, end int) bool {
	if date < b.StartDate || date > b.EndDate {
		return false
	}
	if b.FullDay() {
		return true
	}
	bs, be, ok := ValidRange(b.StartTime, b.EndTime, 0)
	if !ok {
		return true // malformed rows block conservatively
	}
	return bs < end && be > start
}

// dayRanges is the doctor's merged working ranges on weekday.
func dayRanges(schedules []agenda_models.DoctorSchedule, weekday time.Weekday) []span {
	var spans []span
	for _, s := range schedules {
		if s.Weekday != int(weekday) {
			continue
		}
		if st, en, ok := ValidRange(s.StartTime, s.EndTime, 0); ok {
			spans = append(spans, span{st, en})
		}
	}
	return mergeRanges(spans)
}

// Availability checks the slot start-end ("HH:MM") of doctorID on date
// ("YYYY-MM-DD"). A block wins over everything (a clinic holiday also blocks
// a doctor without schedule); then a doctor without any schedule is
// "no_schedule"; else "inside" when one working range contains the slot.
func Availability(db *gorm.DB, doctorID uint, date string, start, end int) (agenda_dto.AvailabilityResponse, error) {
	resp := agenda_dto.AvailabilityResponse{Ranges: []agenda_dto.TimeRange{}, Blocks: []agenda_models.ScheduleBlock{}}
	day, _ := ParseDate(date)

	var schedules []agenda_models.DoctorSchedule
	if err := db.Where("doctor_id = ?", doctorID).Find(&schedules).Error; err != nil {
		return resp, err
	}
	ranges := dayRanges(schedules, day.Weekday())
	resp.Ranges = toTimeRanges(ranges)

	blocks, err := BlocksBetween(db, []uint{doctorID}, true, date, date)
	if err != nil {
		return resp, err
	}
	for _, b := range blocks {
		if blockOverlaps(b, date, start, end) {
			resp.Blocks = append(resp.Blocks, b)
		}
	}

	switch {
	case len(resp.Blocks) > 0:
		resp.Status = StatusBlocked
	case len(schedules) == 0:
		resp.Status = StatusNoSchedule
	default:
		resp.Status = StatusOutside
		for _, r := range ranges {
			if r.start <= start && end <= r.end {
				resp.Status = StatusInside
				break
			}
		}
	}
	return resp, nil
}

// Agenda returns, for each doctor and each day from..to, the working ranges
// and the blocks that apply (the doctor's and the clinic's), plus the
// clinic-wide blocks per day.
func Agenda(db *gorm.DB, doctors []DoctorRef, from, to time.Time) (agenda_dto.AgendaRangeResponse, error) {
	resp := agenda_dto.AgendaRangeResponse{
		From:    from.Format(DateLayout),
		To:      to.Format(DateLayout),
		Doctors: []agenda_dto.DoctorAgenda{},
		Clinic:  []agenda_dto.ClinicDay{},
	}
	ids := make([]uint, 0, len(doctors))
	for _, d := range doctors {
		ids = append(ids, d.ID)
	}

	var schedules []agenda_models.DoctorSchedule
	if len(ids) > 0 {
		if err := db.Where("doctor_id IN ?", ids).Find(&schedules).Error; err != nil {
			return resp, err
		}
	}
	byDoctor := map[uint][]agenda_models.DoctorSchedule{}
	for _, s := range schedules {
		byDoctor[s.DoctorID] = append(byDoctor[s.DoctorID], s)
	}
	blocks, err := BlocksBetween(db, ids, true, resp.From, resp.To)
	if err != nil {
		return resp, err
	}

	var days []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		days = append(days, d)
	}
	onDay := func(date string, doctorID *uint) []agenda_models.ScheduleBlock {
		out := []agenda_models.ScheduleBlock{}
		for _, b := range blocks {
			if date < b.StartDate || date > b.EndDate {
				continue
			}
			if b.DoctorID == nil || (doctorID != nil && *b.DoctorID == *doctorID) {
				out = append(out, b)
			}
		}
		return out
	}

	for _, d := range days {
		date := d.Format(DateLayout)
		resp.Clinic = append(resp.Clinic, agenda_dto.ClinicDay{Date: date, Blocks: onDay(date, nil)})
	}
	for _, doc := range doctors {
		id := doc.ID
		agenda := agenda_dto.DoctorAgenda{DoctorID: id, Active: doc.Active, HasSchedule: len(byDoctor[id]) > 0, Days: []agenda_dto.AgendaDay{}}
		for _, d := range days {
			date := d.Format(DateLayout)
			agenda.Days = append(agenda.Days, agenda_dto.AgendaDay{
				Date:   date,
				Ranges: toTimeRanges(dayRanges(byDoctor[id], d.Weekday())),
				Blocks: onDay(date, &id),
			})
		}
		resp.Doctors = append(resp.Doctors, agenda)
	}
	return resp, nil
}

// DoctorRef is the part of a doctor the agenda needs.
type DoctorRef struct {
	ID     uint
	Active bool
}

// AffectedAppointments lists the appointments (not cancelled or completed)
// that fall inside block: the block's doctor, or every doctor for a clinic
// block.
func AffectedAppointments(db *gorm.DB, block agenda_models.ScheduleBlock) ([]agenda_dto.AffectedAppointment, error) {
	q := db.Model(&clinical_models.Appointment{}).Preload("Patient").
		Where("DATE(date) >= DATE(?) AND DATE(date) <= DATE(?)", block.StartDate, block.EndDate).
		Where("status NOT IN ?", []string{"cancelled", "completed"})
	if block.DoctorID != nil {
		q = q.Where("doctor_id = ?", *block.DoctorID)
	}
	if !block.FullDay() {
		q = q.Where("start_time < ? AND end_time > ?", block.EndTime, block.StartTime)
	}
	var appointments []clinical_models.Appointment
	if err := q.Order("date ASC, start_time ASC").Find(&appointments).Error; err != nil {
		return nil, err
	}
	out := make([]agenda_dto.AffectedAppointment, 0, len(appointments))
	for _, a := range appointments {
		out = append(out, agenda_dto.AffectedAppointment{
			ID:          a.ID,
			Date:        a.Date,
			StartTime:   a.StartTime,
			EndTime:     a.EndTime,
			Status:      a.Status,
			DoctorID:    a.DoctorID,
			PatientID:   a.PatientID,
			PatientName: patientName(a.Patient),
		})
	}
	return out, nil
}

func patientName(p clinical_models.Patient) string {
	if p.FullName != nil && strings.TrimSpace(*p.FullName) != "" {
		return strings.TrimSpace(*p.FullName)
	}
	return strings.TrimSpace(p.FirstName + " " + p.LastName)
}
