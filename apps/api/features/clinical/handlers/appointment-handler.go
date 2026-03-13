package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type AppointmentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAppointmentHandler(db *gorm.DB, logger *zap.Logger) *AppointmentHandler {
	return &AppointmentHandler{db: db, logger: logger}
}

// GetAppointments returns all appointments for the tenant, optionally filtered by date range
func (h *AppointmentHandler) GetAppointments(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")
	query := h.db.Where("tenant_id = ?", tenantID).Preload("Patient")

	// Optional date range filter
	start := c.Query("start")
	end := c.Query("end")
	if start != "" && end != "" {
		query = query.Where("date >= ? AND date <= ?", start, end)
	}

	var appointments []clinical_models.Appointment
	if err := query.Order("date ASC, start_time ASC").Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetTodayAppointments returns all appointments for today grouped by status (for waiting room board)
func (h *AppointmentHandler) GetTodayAppointments(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")
	today := time.Now().Format("2006-01-02")

	var appointments []clinical_models.Appointment
	if err := h.db.Where("tenant_id = ? AND DATE(date) = ?", tenantID, today).
		Preload("Patient").
		Order("start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get today's appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetPatientAppointments returns pending appointments for a specific patient
func (h *AppointmentHandler) GetPatientAppointments(c *gin.Context) envelope.Response {
	patientID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	tenantID, _ := c.Get("tenant_id")
	status := c.DefaultQuery("status", "scheduled")

	var appointments []clinical_models.Appointment
	if err := h.db.Where("tenant_id = ? AND patient_id = ? AND status = ?", tenantID, patientID, status).
		Order("date ASC, start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get patient appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetAppointment returns a single appointment by ID
func (h *AppointmentHandler) GetAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.Preload("Patient").First(&appointment, id).Error; err != nil {
		h.logger.Error("Failed to get appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	return envelope.SuccessResponse(appointment, "appointments.get.success")
}

// CreateAppointment creates a new appointment
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) envelope.Response {
	var dto clinical_dto.CreateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid create appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	appointment := &clinical_models.Appointment{
		PatientID: dto.PatientID,
		Title:     dto.Title,
		Date:      dto.Date,
		StartTime: dto.StartTime,
		EndTime:   dto.EndTime,
		Location:  dto.Location,
		Notes:     dto.Notes,
		Status:    "scheduled",
	}

	tenantID, exists := c.Get("tenant_id")
	if exists {
		appointment.TenantID = tenantID.(uint)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(appointment).Error; err != nil {
		h.logger.Error("Failed to create appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// Reload with patient
	h.db.Preload("Patient").First(appointment, appointment.ID)

	h.logger.Info("Appointment created successfully", zap.Uint("id", appointment.ID))
	return envelope.SuccessResponse(appointment, "appointments.create.success")
}

// UpdateAppointment updates an existing appointment
func (h *AppointmentHandler) UpdateAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		h.logger.Error("Appointment not found", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var dto clinical_dto.UpdateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid update appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	updates := map[string]interface{}{}
	if dto.PatientID != nil {
		updates["patient_id"] = *dto.PatientID
	}
	if dto.Title != nil {
		updates["title"] = *dto.Title
	}
	if dto.Date != nil {
		updates["date"] = *dto.Date
	}
	if dto.StartTime != nil {
		updates["start_time"] = *dto.StartTime
	}
	if dto.EndTime != nil {
		updates["end_time"] = *dto.EndTime
	}
	if dto.Location != nil {
		updates["location"] = *dto.Location
	}
	if dto.Notes != nil {
		updates["notes"] = *dto.Notes
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Model(&appointment).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	h.db.Preload("Patient").First(&appointment, id)

	h.logger.Info("Appointment updated successfully", zap.Int("id", id))
	return envelope.SuccessResponse(appointment, "appointments.update.success")
}

// UpdateStatus changes the status of an appointment
func (h *AppointmentHandler) UpdateStatus(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var dto clinical_dto.UpdateAppointmentStatusDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid status update request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// Validate status
	validStatuses := map[string]bool{
		"scheduled":       true,
		"arrived":         true,
		"in_consultation": true,
		"completed":       true,
		"cancelled":       true,
	}
	if !validStatuses[dto.Status] {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid status. Must be: scheduled, arrived, in_consultation, completed, or cancelled", core_errors.ErrAuthInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Model(&appointment).Update("status", dto.Status).Error; err != nil {
		h.logger.Error("Failed to update appointment status", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	h.db.Preload("Patient").First(&appointment, id)

	h.logger.Info("Appointment status updated", zap.Int("id", id), zap.String("status", dto.Status))
	return envelope.SuccessResponse(appointment, "appointments.status.update.success")
}

// DeleteAppointment deletes an appointment (only if status is scheduled)
func (h *AppointmentHandler) DeleteAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	if appointment.Status != "scheduled" && appointment.Status != "cancelled" {
		return envelope.ErrorResponse(http.StatusBadRequest, "Only scheduled or cancelled appointments can be deleted", core_errors.ErrAuthInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Delete(&appointment).Error; err != nil {
		h.logger.Error("Failed to delete appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	h.logger.Info("Appointment deleted", zap.Int("id", id))
	return envelope.SuccessResponse(nil, "appointments.delete.success")
}
