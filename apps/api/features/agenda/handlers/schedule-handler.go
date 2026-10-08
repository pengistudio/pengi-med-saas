package agenda_handlers

import (
	"sort"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_dto "pengi-med-saas/features/agenda/dto"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	doctor_models "pengi-med-saas/features/doctors/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// maxScheduleSlots bounds a weekly schedule (12 ranges a day is plenty).
const maxScheduleSlots = 7 * 12

// stepMinutes is the granularity of schedules, blocks and type durations.
const stepMinutes = 5

// GetDoctorSchedule returns the weekly schedule of doctor :id.
func (h *AgendaHandler) GetDoctorSchedule(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.getSchedule(db, doctor)
}

// GetMySchedule returns the weekly schedule of the current user's doctor.
func (h *AgendaHandler) GetMySchedule(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.getSchedule(db, doctor)
}

// ReplaceDoctorSchedule replaces the weekly schedule of doctor :id (admin).
func (h *AgendaHandler) ReplaceDoctorSchedule(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.replaceSchedule(c, db, doctor)
}

// ReplaceMySchedule replaces the weekly schedule of the current user's doctor.
func (h *AgendaHandler) ReplaceMySchedule(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.replaceSchedule(c, db, doctor)
}

func loadSchedule(db *gorm.DB, doctorID uint) ([]agenda_models.DoctorSchedule, error) {
	slots := []agenda_models.DoctorSchedule{}
	err := db.Where("doctor_id = ?", doctorID).Order("weekday, start_time").Find(&slots).Error
	return slots, err
}

func (h *AgendaHandler) getSchedule(db *gorm.DB, doctor *doctor_models.Doctor) envelope.Response {
	slots, err := loadSchedule(db, doctor.ID)
	if err != nil {
		return h.internal("failed to fetch doctor schedule", err)
	}
	return envelope.SuccessResponse(slots, "agenda.schedule.fetch.success")
}

// validateSlots checks weekday, 5-minute times, start < end, and that ranges
// of the same weekday do not overlap (touching is fine).
func validateSlots(slots []agenda_dto.ScheduleSlot) *envelope.Response {
	if len(slots) > maxScheduleSlots {
		resp := invalidRequest()
		return &resp
	}
	type rng struct{ start, end int }
	byDay := map[int][]rng{}
	for _, s := range slots {
		if s.Weekday < 0 || s.Weekday > 6 {
			resp := invalidRequest()
			return &resp
		}
		start, end, ok := agenda_services.ValidRange(s.StartTime, s.EndTime, stepMinutes)
		if !ok {
			resp := badRequest("agenda.error.invalid_time", core_errors.ErrAgendaInvalidTime)
			return &resp
		}
		byDay[s.Weekday] = append(byDay[s.Weekday], rng{start, end})
	}
	for _, ranges := range byDay {
		sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
		for i := 1; i < len(ranges); i++ {
			if ranges[i].start < ranges[i-1].end {
				resp := badRequest("agenda.error.schedule_overlap", core_errors.ErrAgendaScheduleOverlap)
				return &resp
			}
		}
	}
	return nil
}

func (h *AgendaHandler) replaceSchedule(c *gin.Context, db *gorm.DB, doctor *doctor_models.Doctor) envelope.Response {
	var req agenda_dto.ReplaceScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequest()
	}
	if errResp := validateSlots(req.Slots); errResp != nil {
		return *errResp
	}
	tenantID := tenantdb.TenantID(c)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("doctor_id = ?", doctor.ID).Delete(&agenda_models.DoctorSchedule{}).Error; err != nil {
			return err
		}
		if len(req.Slots) == 0 {
			return nil
		}
		rows := make([]agenda_models.DoctorSchedule, 0, len(req.Slots))
		for _, s := range req.Slots {
			rows = append(rows, agenda_models.DoctorSchedule{
				TenantID: tenantID, DoctorID: doctor.ID, Weekday: s.Weekday, StartTime: s.StartTime, EndTime: s.EndTime,
			})
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return h.internal("failed to replace doctor schedule", err)
	}
	slots, err := loadSchedule(db, doctor.ID)
	if err != nil {
		return h.internal("failed to fetch doctor schedule", err)
	}
	return envelope.SuccessResponse(slots, "agenda.schedule.update.success")
}
