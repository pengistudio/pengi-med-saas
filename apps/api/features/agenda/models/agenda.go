package agenda_models

import (
	"gorm.io/gorm"

	doctor_models "pengi-med-saas/features/doctors/models"
)

// Times are local clock times "HH:MM" (America/Guayaquil) in 5-minute steps;
// an end time may be "24:00". Dates are "YYYY-MM-DD" strings, so they compare
// as text and never shift with a time zone.

// DoctorSchedule is one working range of a doctor's weekly schedule. A day can
// have several ranges (morning and afternoon); ranges of the same doctor and
// weekday never overlap. The whole week is replaced at once, so rows are hard
// deleted and not audited one by one.
type DoctorSchedule struct {
	gorm.Model
	TenantID  uint   `gorm:"not null;index:idx_doctor_schedule_doctor,priority:1" json:"tenant_id"`
	DoctorID  uint   `gorm:"not null;index:idx_doctor_schedule_doctor,priority:2" json:"doctor_id"`
	Weekday   int    `gorm:"not null" json:"weekday"` // 0 = Sunday … 6 = Saturday (time.Weekday)
	StartTime string `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime   string `gorm:"type:varchar(5);not null" json:"end_time"`
}

// ScheduleBlock is an exception to the schedule: vacation, a congress, a
// holiday. DoctorID nil blocks the whole clinic. With StartTime/EndTime empty
// it covers whole days; otherwise that time range on each day from StartDate
// to EndDate (both inclusive).
type ScheduleBlock struct {
	gorm.Model
	TenantID  uint   `gorm:"not null;index" json:"tenant_id"`
	DoctorID  *uint  `gorm:"index" json:"doctor_id"`
	StartDate string `gorm:"type:varchar(10);not null;index" json:"start_date"`
	EndDate   string `gorm:"type:varchar(10);not null;index" json:"end_date"`
	StartTime string `gorm:"type:varchar(5);not null;default:''" json:"start_time"`
	EndTime   string `gorm:"type:varchar(5);not null;default:''" json:"end_time"`
	Reason    string `gorm:"type:varchar(255)" json:"reason"`
}

func (ScheduleBlock) IsAuditable() bool { return true }

// FullDay reports whether the block covers whole days.
func (b ScheduleBlock) FullDay() bool { return b.StartTime == "" && b.EndTime == "" }

// AppointmentType is a kind of appointment with its usual duration (first
// visit, control, procedure). Doctors restricts which doctors attend it;
// empty means every doctor.
type AppointmentType struct {
	gorm.Model
	TenantID        uint   `gorm:"not null;index" json:"tenant_id"`
	Name            string `gorm:"type:varchar(100);not null" json:"name"`
	DurationMinutes int    `gorm:"not null" json:"duration_minutes"`
	Color           string `gorm:"type:varchar(7)" json:"color"` // #RRGGBB or empty
	// Active is written with Update("active", ...): GORM skips a false zero
	// value on create and the column default would win.
	Active    bool                   `gorm:"not null;default:true" json:"active"`
	Doctors   []doctor_models.Doctor `gorm:"many2many:appointment_type_doctors;" json:"-"`
	DoctorIDs []uint                 `gorm:"-" json:"doctor_ids"` // filled from Doctors when served
}

func (AppointmentType) IsAuditable() bool { return true }

// AllowsDoctor reports whether doctorID may attend this type (Doctors loaded).
func (t AppointmentType) AllowsDoctor(doctorID uint) bool {
	if len(t.Doctors) == 0 {
		return true
	}
	for _, d := range t.Doctors {
		if d.ID == doctorID {
			return true
		}
	}
	return false
}

// FillDoctorIDs copies the loaded Doctors into DoctorIDs (never null in JSON).
func (t *AppointmentType) FillDoctorIDs() {
	t.DoctorIDs = make([]uint, 0, len(t.Doctors))
	for _, d := range t.Doctors {
		t.DoctorIDs = append(t.DoctorIDs, d.ID)
	}
}
