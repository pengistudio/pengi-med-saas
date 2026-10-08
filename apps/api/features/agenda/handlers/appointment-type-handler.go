package agenda_handlers

import (
	"net/http"
	"strconv"
	"strings"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_dto "pengi-med-saas/features/agenda/dto"
	agenda_models "pengi-med-saas/features/agenda/models"
	doctor_data "pengi-med-saas/features/doctors/data"
	doctor_models "pengi-med-saas/features/doctors/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	minTypeDuration = 5
	maxTypeDuration = 480
)

func typeNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "agenda.error.type_not_found", core_errors.ErrAgendaTypeNotFound)
}

func validDuration(m int) bool {
	return m >= minTypeDuration && m <= maxTypeDuration && m%stepMinutes == 0
}

// normalizeColor validates an optional #RRGGBB color.
func normalizeColor(color string) (string, bool) {
	color = strings.ToUpper(strings.TrimSpace(color))
	if color == "" {
		return "", true
	}
	return color, doctor_data.IsValidColor(color)
}

// typeDoctors loads the doctors of ids from the tenant; ok is false when some
// id is not a doctor of the tenant.
func typeDoctors(db *gorm.DB, ids []uint) ([]doctor_models.Doctor, bool, error) {
	unique := map[uint]bool{}
	for _, id := range ids {
		unique[id] = true
	}
	doctors := []doctor_models.Doctor{}
	if len(unique) == 0 {
		return doctors, true, nil
	}
	if err := db.Where("id IN ?", ids).Find(&doctors).Error; err != nil {
		return nil, false, err
	}
	return doctors, len(doctors) == len(unique), nil
}

func doctorNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "doctors.error.not_found", core_errors.ErrDoctorNotFound)
}

// findType loads type id with its doctors from the tenant-bound db.
func findType(db *gorm.DB, id uint) (agenda_models.AppointmentType, error) {
	var t agenda_models.AppointmentType
	err := db.Preload("Doctors").Limit(1).Find(&t, id).Error
	t.FillDoctorIDs()
	return t, err
}

// ListAppointmentTypes lists the tenant's types. ?active=true keeps active
// ones; ?doctor_id=N keeps the ones doctor N attends.
func (h *AgendaHandler) ListAppointmentTypes(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	q := db.Preload("Doctors").Order("name, id")
	if c.Query("active") == "true" {
		q = q.Where("active = ?", true)
	}
	var doctorID uint
	if raw := c.Query("doctor_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return invalidRequest()
		}
		doctorID = uint(id)
	}
	var types []agenda_models.AppointmentType
	if err := q.Find(&types).Error; err != nil {
		return h.internal("failed to list appointment types", err)
	}
	out := make([]agenda_models.AppointmentType, 0, len(types))
	for _, t := range types {
		if doctorID != 0 && !t.AllowsDoctor(doctorID) {
			continue
		}
		t.FillDoctorIDs()
		out = append(out, t)
	}
	return envelope.SuccessResponse(out, "agenda.types.list.success")
}

// CreateAppointmentType creates a type (admin).
func (h *AgendaHandler) CreateAppointmentType(c *gin.Context) envelope.Response {
	var req agenda_dto.CreateAppointmentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequest()
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return invalidRequest()
	}
	if !validDuration(req.DurationMinutes) {
		return badRequest("agenda.error.invalid_duration", core_errors.ErrAgendaInvalidDuration)
	}
	color, ok := normalizeColor(req.Color)
	if !ok {
		return badRequest("doctors.error.invalid_color", core_errors.ErrDoctorInvalidColor)
	}
	db := tenantdb.For(c, h.db)
	doctors, ok, err := typeDoctors(db, req.DoctorIDs)
	if err != nil {
		return h.internal("failed to load doctors of an appointment type", err)
	}
	if !ok {
		return doctorNotFound()
	}

	t := agenda_models.AppointmentType{
		TenantID: tenantdb.TenantID(c), Name: name, DurationMinutes: req.DurationMinutes, Color: color, Active: true,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Doctors").Create(&t).Error; err != nil {
			return err
		}
		if req.Active != nil && !*req.Active {
			if err := tx.Model(&t).Update("active", false).Error; err != nil {
				return err
			}
		}
		if len(doctors) > 0 {
			return tx.Model(&t).Omit("Doctors.*").Association("Doctors").Replace(doctors)
		}
		return nil
	})
	if err != nil {
		return h.internal("failed to create appointment type", err)
	}
	saved, err := findType(db, t.ID)
	if err != nil {
		return h.internal("failed to fetch appointment type", err)
	}
	return envelope.SuccessResponse(saved, "agenda.types.create.success")
}

// UpdateAppointmentType updates type :id (admin).
func (h *AgendaHandler) UpdateAppointmentType(c *gin.Context) envelope.Response {
	id, ok := uintParam(c, "id")
	if !ok {
		return typeNotFound()
	}
	var req agenda_dto.UpdateAppointmentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequest()
	}
	db := tenantdb.For(c, h.db)
	t, err := findType(db, id)
	if err != nil {
		return h.internal("failed to fetch appointment type", err)
	}
	if t.ID == 0 {
		return typeNotFound()
	}

	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return invalidRequest()
		}
		updates["name"] = name
	}
	if req.DurationMinutes != nil {
		if !validDuration(*req.DurationMinutes) {
			return badRequest("agenda.error.invalid_duration", core_errors.ErrAgendaInvalidDuration)
		}
		updates["duration_minutes"] = *req.DurationMinutes
	}
	if req.Color != nil {
		color, ok := normalizeColor(*req.Color)
		if !ok {
			return badRequest("doctors.error.invalid_color", core_errors.ErrDoctorInvalidColor)
		}
		updates["color"] = color
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	var doctors []doctor_models.Doctor
	if req.DoctorIDs != nil {
		var ok bool
		doctors, ok, err = typeDoctors(db, *req.DoctorIDs)
		if err != nil {
			return h.internal("failed to load doctors of an appointment type", err)
		}
		if !ok {
			return doctorNotFound()
		}
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&t).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.DoctorIDs == nil {
			return nil
		}
		if len(doctors) == 0 {
			return tx.Model(&t).Association("Doctors").Clear()
		}
		return tx.Model(&t).Omit("Doctors.*").Association("Doctors").Replace(doctors)
	})
	if err != nil {
		return h.internal("failed to update appointment type", err)
	}
	saved, err := findType(db, t.ID)
	if err != nil {
		return h.internal("failed to fetch appointment type", err)
	}
	return envelope.SuccessResponse(saved, "agenda.types.update.success")
}

// DeleteAppointmentType soft-deletes type :id (admin). Appointments keep
// their appointment_type_id.
func (h *AgendaHandler) DeleteAppointmentType(c *gin.Context) envelope.Response {
	id, ok := uintParam(c, "id")
	if !ok {
		return typeNotFound()
	}
	db := tenantdb.For(c, h.db)
	var t agenda_models.AppointmentType
	if err := db.Limit(1).Find(&t, id).Error; err != nil {
		return h.internal("failed to fetch appointment type", err)
	}
	if t.ID == 0 {
		return typeNotFound()
	}
	if err := db.Delete(&t).Error; err != nil {
		return h.internal("failed to delete appointment type", err)
	}
	return envelope.SuccessResponse(nil, "agenda.types.delete.success")
}
