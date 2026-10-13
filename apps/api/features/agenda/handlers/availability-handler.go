package agenda_handlers

import (
	"net/http"
	"strconv"
	"strings"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_services "pengi-med-saas/features/agenda/services"
	doctor_models "pengi-med-saas/features/doctors/models"

	"github.com/gin-gonic/gin"
)

// GetAvailability checks a slot of a doctor against the schedule and blocks:
// ?doctor_id=&date=YYYY-MM-DD&start_time=HH:MM&end_time=HH:MM. Booking outside
// the schedule is allowed; the frontend only warns.
func (h *AgendaHandler) GetAvailability(c *gin.Context) envelope.Response {
	doctorID, err := strconv.ParseUint(c.Query("doctor_id"), 10, 64)
	if err != nil || doctorID == 0 {
		return invalidRequest()
	}
	date := c.Query("date")
	if _, ok := agenda_services.ParseDate(date); !ok {
		return badRequest("agenda.error.invalid_date_range", core_errors.ErrAgendaInvalidDateRange)
	}
	// Existing appointments may use any minute, so no 5-minute step here.
	start, end, ok := agenda_services.ValidRange(c.Query("start_time"), c.Query("end_time"), 0)
	if !ok {
		return badRequest("agenda.error.invalid_time", core_errors.ErrAgendaInvalidTime)
	}

	db := tenantdb.For(c, h.db)
	var doctor doctor_models.Doctor
	if err := db.Select("id").Limit(1).Find(&doctor, uint(doctorID)).Error; err != nil {
		return h.internal("failed to fetch doctor", err)
	}
	if doctor.ID == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "doctors.error.not_found", core_errors.ErrDoctorNotFound)
	}
	result, err := agenda_services.Availability(db, doctor.ID, date, start, end)
	if err != nil {
		return h.internal("failed to check availability", err)
	}
	return envelope.SuccessResponse(result, "agenda.availability.success")
}

// GetAgendaRange returns the shading data of the agenda:
// ?from=YYYY-MM-DD&to=YYYY-MM-DD[&doctor_ids=1,2]. Without doctor_ids it
// covers the active doctors; with them, any doctor of the tenant (ids of
// other tenants are dropped).
func (h *AgendaHandler) GetAgendaRange(c *gin.Context) envelope.Response {
	from, ok1 := agenda_services.ParseDate(c.Query("from"))
	to, ok2 := agenda_services.ParseDate(c.Query("to"))
	if !ok1 || !ok2 || to.Before(from) {
		return badRequest("agenda.error.invalid_date_range", core_errors.ErrAgendaInvalidDateRange)
	}
	if to.Sub(from).Hours()/24 >= agenda_services.MaxRangeDays {
		return badRequest("agenda.error.range_too_long", core_errors.ErrAgendaRangeTooLong)
	}

	db := tenantdb.For(c, h.db)
	q := db.Model(&doctor_models.Doctor{}).Select("id", "active").Order("id")
	if raw := strings.TrimSpace(c.Query("doctor_ids")); raw != "" {
		var ids []uint
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
			if err != nil {
				return invalidRequest()
			}
			ids = append(ids, uint(id))
		}
		q = q.Where("id IN ?", ids)
	} else {
		q = q.Where("active = ?", true)
	}
	var doctors []doctor_models.Doctor
	if err := q.Find(&doctors).Error; err != nil {
		return h.internal("failed to list doctors for the agenda", err)
	}
	refs := make([]agenda_services.DoctorRef, 0, len(doctors))
	for _, d := range doctors {
		refs = append(refs, agenda_services.DoctorRef{ID: d.ID, Active: d.Active})
	}
	result, err := agenda_services.Agenda(db, refs, from, to)
	if err != nil {
		return h.internal("failed to build the agenda range", err)
	}
	return envelope.SuccessResponse(result, "agenda.range.success")
}
