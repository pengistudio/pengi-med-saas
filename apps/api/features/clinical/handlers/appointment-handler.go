package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	google_calendar "pengi-med-saas/core/google"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	integration_models "pengi-med-saas/features/integrations/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type AppointmentHandler struct {
	db        *gorm.DB
	logger    *zap.Logger
	googleSvc *google_calendar.CalendarService
}

func NewAppointmentHandler(db *gorm.DB, logger *zap.Logger) *AppointmentHandler {
	return &AppointmentHandler{
		db:        db,
		logger:    logger,
		googleSvc: google_calendar.NewCalendarService(),
	}
}

// syncCreate creates a Google Calendar event for the appointment and saves the event ID.
// Errors are logged but never propagate to the caller.
func (h *AppointmentHandler) syncCreate(tenantID uint, appointment *clinical_models.Appointment) {
	if !h.googleSvc.IsConfigured() {
		return
	}
	token, calendarID, ok := h.getValidToken(tenantID)
	if !ok {
		return
	}
	patientName := appointment.Patient.FirstName + " " + appointment.Patient.LastName
	event := google_calendar.BuildEvent(appointment.Title, appointment.Location, appointment.Notes, patientName, appointment.Date, appointment.StartTime, appointment.EndTime)
	eventID, err := h.googleSvc.CreateEvent(token, calendarID, event)
	if err != nil {
		h.logger.Warn("Google Calendar: failed to create event", zap.Error(err), zap.Uint("appointment_id", appointment.ID))
		return
	}
	h.db.Model(appointment).Update("google_event_id", eventID)
	h.logger.Info("Google Calendar: event created", zap.String("event_id", eventID), zap.Uint("appointment_id", appointment.ID))
}

// syncUpdate updates the Google Calendar event for the appointment.
func (h *AppointmentHandler) syncUpdate(tenantID uint, appointment *clinical_models.Appointment) {
	if !h.googleSvc.IsConfigured() || appointment.GoogleEventID == "" {
		return
	}
	token, calendarID, ok := h.getValidToken(tenantID)
	if !ok {
		return
	}
	patientName := appointment.Patient.FirstName + " " + appointment.Patient.LastName
	event := google_calendar.BuildEvent(appointment.Title, appointment.Location, appointment.Notes, patientName, appointment.Date, appointment.StartTime, appointment.EndTime)
	if err := h.googleSvc.UpdateEvent(token, calendarID, appointment.GoogleEventID, event); err != nil {
		h.logger.Warn("Google Calendar: failed to update event", zap.Error(err), zap.Uint("appointment_id", appointment.ID))
	}
}

// syncDelete deletes the Google Calendar event for the appointment.
func (h *AppointmentHandler) syncDelete(tenantID uint, appointment *clinical_models.Appointment) {
	if !h.googleSvc.IsConfigured() || appointment.GoogleEventID == "" {
		return
	}
	token, calendarID, ok := h.getValidToken(tenantID)
	if !ok {
		return
	}
	if err := h.googleSvc.DeleteEvent(token, calendarID, appointment.GoogleEventID); err != nil {
		h.logger.Warn("Google Calendar: failed to delete event", zap.Error(err), zap.Uint("appointment_id", appointment.ID))
	}
}

// getValidToken returns a valid access token and calendar ID for the tenant.
// Returns false if not connected or on any error.
func (h *AppointmentHandler) getValidToken(tenantID uint) (accessToken, calendarID string, ok bool) {
	var integration integration_models.TenantIntegration
	if err := h.db.Where("tenant_id = ? AND google_connected = true", tenantID).First(&integration).Error; err != nil {
		return "", "", false
	}

	if google_calendar.IsExpired(integration.GoogleTokenExpiry) {
		if integration.GoogleRefreshToken == "" {
			return "", "", false
		}
		newToken, err := h.googleSvc.RefreshAccessToken(integration.GoogleRefreshToken)
		if err != nil {
			h.logger.Warn("Google Calendar: token refresh failed", zap.Error(err), zap.Uint("tenant_id", tenantID))
			return "", "", false
		}
		expiry := google_calendar.TokenExpiry(newToken.ExpiresIn)
		h.db.Model(&integration).Updates(map[string]interface{}{
			"google_access_token": newToken.AccessToken,
			"google_token_expiry": &expiry,
		})
		integration.GoogleAccessToken = newToken.AccessToken
	}

	return integration.GoogleAccessToken, integration.GoogleCalendarID, true
}

// GetAppointments returns all appointments for the tenant, optionally filtered by date range
func (h *AppointmentHandler) GetAppointments(c *gin.Context) envelope.Response {
	tenantID, _ := c.Get("tenant_id")
	query := h.db.Where("tenant_id = ?", tenantID).Preload("Patient")

	// Optional date range filter
	start := c.Query("start")
	end := c.Query("end")
	if start != "" && end != "" {
		query = query.Where("DATE(date) >= DATE(?) AND DATE(date) <= DATE(?)", start, end)
	}

	var appointments []clinical_models.Appointment
	if err := query.Order("date ASC, start_time ASC").Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetPatientAppointments returns pending appointments for a specific patient
func (h *AppointmentHandler) GetPatientAppointments(c *gin.Context) envelope.Response {
	patientID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	tenantID, _ := c.Get("tenant_id")
	status := c.DefaultQuery("status", "scheduled")

	var appointments []clinical_models.Appointment
	if err := h.db.Where("tenant_id = ? AND patient_id = ? AND status = ?", tenantID, patientID, status).
		Order("date ASC, start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get patient appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetAppointment returns a single appointment by ID
func (h *AppointmentHandler) GetAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.Preload("Patient").First(&appointment, id).Error; err != nil {
		h.logger.Error("Failed to get appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointment, "appointments.get.success")
}

// CreateAppointment creates a new appointment
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) envelope.Response {
	var dto clinical_dto.CreateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid create appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	tenantID, exists := c.Get("tenant_id")

	// Check for overlapping appointments on the same date and tenant
	if exists {
		var count int64
		h.db.Model(&clinical_models.Appointment{}).
			Where("tenant_id = ? AND DATE(date) = DATE(?) AND status != 'cancelled' AND start_time < ? AND end_time > ?",
				tenantID, dto.Date, dto.EndTime, dto.StartTime).
			Count(&count)
		if count > 0 {
			return envelope.ErrorResponse(http.StatusConflict, "appointments.overlap.error", core_errors.ErrClinicalAppointmentOverlap)
		}
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

	if exists {
		appointment.TenantID = tenantID.(uint)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(appointment).Error; err != nil {
		h.logger.Error("Failed to create appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	// Reload with patient
	h.db.Preload("Patient").First(appointment, appointment.ID)

	// Sync to Google Calendar (non-blocking, errors are logged)
	if exists {
		go h.syncCreate(tenantID.(uint), appointment)
	}

	h.logger.Info("Appointment created successfully", zap.Uint("id", appointment.ID))
	return envelope.SuccessResponse(appointment, "appointments.create.success")
}

// UpdateAppointment updates an existing appointment
func (h *AppointmentHandler) UpdateAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		h.logger.Error("Appointment not found", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.UpdateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid update appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
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

	// Check for overlap only when time or date fields are being changed
	newDate := appointment.Date
	if dto.Date != nil {
		newDate = *dto.Date
	}
	newStart := appointment.StartTime
	if dto.StartTime != nil {
		newStart = *dto.StartTime
	}
	newEnd := appointment.EndTime
	if dto.EndTime != nil {
		newEnd = *dto.EndTime
	}

	tenantID, _ := c.Get("tenant_id")
	var count int64
	h.db.Model(&clinical_models.Appointment{}).
		Where("tenant_id = ? AND DATE(date) = DATE(?) AND status != 'cancelled' AND start_time < ? AND end_time > ? AND id != ?",
			tenantID, newDate, newEnd, newStart, id).
		Count(&count)
	if count > 0 {
		return envelope.ErrorResponse(http.StatusConflict, "appointments.overlap.error", core_errors.ErrClinicalAppointmentOverlap)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Model(&appointment).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	h.db.Preload("Patient").First(&appointment, id)

	// Sync to Google Calendar (non-blocking, errors are logged)
	go h.syncUpdate(tenantID.(uint), &appointment)

	h.logger.Info("Appointment updated successfully", zap.Int("id", id))
	return envelope.SuccessResponse(appointment, "appointments.update.success")
}

// UpdateStatus changes the status of an appointment
func (h *AppointmentHandler) UpdateStatus(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.UpdateAppointmentStatusDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid status update request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
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
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid status. Must be: scheduled, arrived, in_consultation, completed, or cancelled", core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Model(&appointment).Update("status", dto.Status).Error; err != nil {
		h.logger.Error("Failed to update appointment status", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	h.db.Preload("Patient").First(&appointment, id)

	tenantID, _ := c.Get("tenant_id")
	if tenantID != nil {
		if dto.Status == "cancelled" || dto.Status == "completed" {
			go h.syncDelete(tenantID.(uint), &appointment)
		} else {
			go h.syncUpdate(tenantID.(uint), &appointment)
		}
	}

	h.logger.Info("Appointment status updated", zap.Int("id", id), zap.String("status", dto.Status))
	return envelope.SuccessResponse(appointment, "appointments.status.update.success")
}

// DeleteAppointment deletes an appointment (only if status is scheduled)
func (h *AppointmentHandler) DeleteAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := h.db.First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	if appointment.Status != "scheduled" && appointment.Status != "cancelled" && appointment.Status != "completed" {
		return envelope.ErrorResponse(http.StatusBadRequest, "Only scheduled, cancelled or completed appointments can be deleted", core_errors.ErrClinicalInvalidRequest)
	}

	if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Delete(&appointment).Error; err != nil {
		h.logger.Error("Failed to delete appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	tenantID, _ := c.Get("tenant_id")
	if tenantID != nil {
		go h.syncDelete(tenantID.(uint), &appointment)
	}

	h.logger.Info("Appointment deleted", zap.Int("id", id))
	return envelope.SuccessResponse(nil, "appointments.delete.success")
}
