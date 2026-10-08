package clinical_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	google_calendar "pengi-med-saas/core/google"
	"pengi-med-saas/core/tenantdb"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_services "pengi-med-saas/features/doctors/services"
	integration_models "pengi-med-saas/features/integrations/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	event := google_calendar.BuildEvent(h.calendarTitle(tenantID, appointment), appointment.Location, appointment.Notes, patientName, appointment.ColorID, appointment.Date, appointment.StartTime, appointment.EndTime)
	eventID, err := h.googleSvc.CreateEvent(token, calendarID, event)
	if err != nil {
		h.logger.Warn("Google Calendar: failed to create event", zap.Error(err), zap.Uint("appointment_id", appointment.ID))
		return
	}
	tenantdb.ForTenant(h.db, tenantID).Model(appointment).Update("google_event_id", eventID)
	h.logger.Info("Google Calendar: event created", zap.String("event_id", eventID), zap.Uint("appointment_id", appointment.ID))
}

// calendarTitle is the Google Calendar event title: the appointment title,
// plus the doctor's name when the appointment has one (one calendar per tenant).
func (h *AppointmentHandler) calendarTitle(tenantID uint, appointment *clinical_models.Appointment) string {
	if appointment.DoctorID == nil {
		return appointment.Title
	}
	doctor := appointment.Doctor
	if doctor == nil {
		doctor = doctor_services.First(tenantdb.ForTenant(h.db, tenantID), appointment.DoctorID)
	}
	if doctor == nil || doctor.FullName == "" {
		return appointment.Title
	}
	return appointment.Title + " · " + doctor.FullName
}

// overlapQuery counts the appointments that overlap [start, end) on date,
// for the same doctor when the appointment has one (several doctors attend
// at once), else across the tenant as before doctors existed.
func overlapQuery(db *gorm.DB, doctorID *uint, date time.Time, start, end string) *gorm.DB {
	q := db.Model(&clinical_models.Appointment{}).
		Where("DATE(date) = DATE(?) AND status != 'cancelled' AND start_time < ? AND end_time > ?", date, end, start)
	if doctorID != nil {
		q = q.Where("doctor_id = ?", *doctorID)
	}
	return q
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
	event := google_calendar.BuildEvent(h.calendarTitle(tenantID, appointment), appointment.Location, appointment.Notes, patientName, appointment.ColorID, appointment.Date, appointment.StartTime, appointment.EndTime)
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

// SyncStatusChange mirrors a status change made outside this handler (e.g. a
// WhatsApp reply) to Google Calendar, like UpdateStatus does: cancelled and
// completed remove the event, anything else updates it. Errors are logged.
func (h *AppointmentHandler) SyncStatusChange(tenantID uint, appointment *clinical_models.Appointment) {
	if appointment.Status == "cancelled" || appointment.Status == "completed" {
		h.syncDelete(tenantID, appointment)
		return
	}
	h.syncUpdate(tenantID, appointment)
}

// getValidToken returns a valid access token and calendar ID for the tenant.
// Returns false if not connected or on any error.
func (h *AppointmentHandler) getValidToken(tenantID uint) (accessToken, calendarID string, ok bool) {
	var integration integration_models.TenantIntegration
	if err := tenantdb.ForTenant(h.db, tenantID).Where("google_connected = true").First(&integration).Error; err != nil {
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
		tenantdb.ForTenant(h.db, tenantID).Model(&integration).Updates(map[string]interface{}{
			"google_access_token": newToken.AccessToken,
			"google_token_expiry": &expiry,
		})
		integration.GoogleAccessToken = newToken.AccessToken
	}

	return integration.GoogleAccessToken, integration.GoogleCalendarID, true
}

// GetAppointments returns all appointments for the tenant, optionally filtered by date range
func (h *AppointmentHandler) GetAppointments(c *gin.Context) envelope.Response {
	query := tenantdb.For(c, h.db).Preload("Patient").Preload("Doctor")

	// Optional date range filter
	start := c.Query("start")
	end := c.Query("end")
	if start != "" && end != "" {
		query = query.Where("DATE(date) >= DATE(?) AND DATE(date) <= DATE(?)", start, end)
	}

	var appointments []clinical_models.Appointment
	if err := query.Order("date ASC, start_time ASC").Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetTodayAppointments returns all appointments for today grouped by status (for waiting room board)
func (h *AppointmentHandler) GetTodayAppointments(c *gin.Context) envelope.Response {
	today := time.Now().Format("2006-01-02")

	var appointments []clinical_models.Appointment
	if err := tenantdb.For(c, h.db).Where("DATE(date) = ?", today).
		Preload("Patient").Preload("Doctor").
		// Only whether triage took the vital signs (the card shows a check): the
		// measurements need RECORD_VITAL_SIGNS or READ_MEDICAL_RECORD, not just
		// READ_APPOINTMENT.
		Preload("VitalSigns", func(db *gorm.DB) *gorm.DB { return db.Select("id", "appointment_id") }).
		Order("start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get today's appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetPatientAppointments returns pending appointments for a specific patient
func (h *AppointmentHandler) GetPatientAppointments(c *gin.Context) envelope.Response {
	patientID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	// Without ?status, "pending" means upcoming: scheduled or confirmed by the
	// patient (e.g. a WhatsApp reminder reply).
	statuses := []string{"scheduled", "confirmed"}
	if status := c.Query("status"); status != "" {
		statuses = []string{status}
	}

	var appointments []clinical_models.Appointment
	if err := tenantdb.For(c, h.db).Where("patient_id = ? AND status IN ?", patientID, statuses).
		Order("date ASC, start_time ASC").
		Find(&appointments).Error; err != nil {
		h.logger.Error("Failed to get patient appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalInvalidRequest)
	}

	return envelope.SuccessResponse(appointments, "appointments.get.success")
}

// GetAppointment returns a single appointment by ID
func (h *AppointmentHandler) GetAppointment(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&appointment, id).Error; err != nil {
		h.logger.Error("Failed to get appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
	}

	return envelope.SuccessResponse(appointment, "appointments.get.success")
}

// CreateAppointment creates a new appointment
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) envelope.Response {
	var dto clinical_dto.CreateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid create appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	if !h.patientInTenant(c, dto.PatientID) {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	doctorID, err := doctor_services.Resolve(c, tenantdb.For(c, h.db), doctor_services.Choice{
		Requested: dto.DoctorID,
		Fallbacks: []*uint{patientDoctorID(tenantdb.For(c, h.db), dto.PatientID)},
		Policy:    doctor_services.RequiredWhenAny,
	})
	if err != nil {
		return doctorErrorResponse(h.logger, err)
	}
	typeID := dto.AppointmentTypeID
	if typeID != nil && *typeID == 0 {
		typeID = nil
	}
	if typeID != nil {
		if err := agenda_services.ValidateAppointmentType(tenantdb.For(c, h.db), *typeID, doctorID, nil); err != nil {
			return appointmentTypeErrorResponse(h.logger, err)
		}
	}

	tenantID, exists := c.Get("tenant_id")

	// Check for overlapping appointments on the same date (and doctor)
	if exists {
		var count int64
		overlapQuery(tenantdb.For(c, h.db), doctorID, dto.Date, dto.StartTime, dto.EndTime).Count(&count)
		if count > 0 {
			return envelope.ErrorResponse(http.StatusConflict, "appointments.overlap.error", core_errors.ErrClinicalAppointmentOverlap)
		}
	}

	appointment := &clinical_models.Appointment{
		PatientID:         dto.PatientID,
		Title:             dto.Title,
		Date:              dto.Date,
		StartTime:         dto.StartTime,
		EndTime:           dto.EndTime,
		Location:          dto.Location,
		Notes:             dto.Notes,
		ColorID:           dto.ColorID,
		Status:            "scheduled",
		DoctorID:          doctorID,
		AppointmentTypeID: typeID,
	}

	if exists {
		appointment.TenantID = tenantID.(uint)
	}

	if err := tenantdb.For(c, h.db).Create(appointment).Error; err != nil {
		h.logger.Error("Failed to create appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	// Reload with patient and doctor
	tenantdb.For(c, h.db).Preload("Patient").Preload("Doctor").First(appointment, appointment.ID)

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
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := tenantdb.For(c, h.db).First(&appointment, id).Error; err != nil {
		h.logger.Error("Appointment not found", zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
	}

	var dto clinical_dto.UpdateAppointmentDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid update appointment request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	updates := map[string]interface{}{}
	if dto.PatientID != nil {
		if !h.patientInTenant(c, *dto.PatientID) {
			return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
		}
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
	if dto.ColorID != nil {
		updates["color_id"] = *dto.ColorID
	}
	newDoctorID := appointment.DoctorID
	if dto.DoctorID != nil {
		if err := doctor_services.ValidateChange(tenantdb.For(c, h.db), *dto.DoctorID, appointment.DoctorID); err != nil {
			return doctorErrorResponse(h.logger, err)
		}
		updates["doctor_id"] = *dto.DoctorID
		newDoctorID = dto.DoctorID
	}
	// The type is checked when it or the doctor changes: active (unless kept)
	// and attended by the appointment's doctor.
	newTypeID := appointment.AppointmentTypeID
	if dto.AppointmentTypeID != nil {
		if *dto.AppointmentTypeID == 0 {
			updates["appointment_type_id"] = nil
			newTypeID = nil
		} else {
			updates["appointment_type_id"] = *dto.AppointmentTypeID
			newTypeID = dto.AppointmentTypeID
		}
	}
	if newTypeID != nil && (dto.AppointmentTypeID != nil || dto.DoctorID != nil) {
		if err := agenda_services.ValidateAppointmentType(tenantdb.For(c, h.db), *newTypeID, newDoctorID, appointment.AppointmentTypeID); err != nil {
			return appointmentTypeErrorResponse(h.logger, err)
		}
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
	overlapQuery(tenantdb.For(c, h.db), newDoctorID, newDate, newStart, newEnd).Where("id != ?", id).Count(&count)
	if count > 0 {
		return envelope.ErrorResponse(http.StatusConflict, "appointments.overlap.error", core_errors.ErrClinicalAppointmentOverlap)
	}

	if err := tenantdb.For(c, h.db).Model(&appointment).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	tenantdb.For(c, h.db).Preload("Patient").Preload("Doctor").First(&appointment, id)

	// Sync to Google Calendar (non-blocking, errors are logged)
	go h.syncUpdate(tenantID.(uint), &appointment)

	h.logger.Info("Appointment updated successfully", zap.Int("id", id))
	return envelope.SuccessResponse(appointment, "appointments.update.success")
}

// UpdateStatus changes the status of an appointment
func (h *AppointmentHandler) UpdateStatus(c *gin.Context) envelope.Response {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.UpdateAppointmentStatusDTO
	if err := c.ShouldBind(&dto); err != nil {
		h.logger.Error("Invalid status update request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	// Validate status
	validStatuses := map[string]bool{
		"scheduled":       true,
		"confirmed":       true, // the patient confirmed (e.g. WhatsApp reminder reply)
		"arrived":         true,
		"in_consultation": true,
		"completed":       true,
		"cancelled":       true,
	}
	if !validStatuses[dto.Status] {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := tenantdb.For(c, h.db).First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
	}

	if err := tenantdb.For(c, h.db).Model(&appointment).Update("status", dto.Status).Error; err != nil {
		h.logger.Error("Failed to update appointment status", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	tenantdb.For(c, h.db).Preload("Patient").First(&appointment, id)

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
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var appointment clinical_models.Appointment
	if err := tenantdb.For(c, h.db).First(&appointment, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAppointmentNotFound)
	}

	if appointment.Status != "scheduled" && appointment.Status != "confirmed" && appointment.Status != "cancelled" && appointment.Status != "completed" {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.appointment.delete.invalid_status", core_errors.ErrClinicalInvalidRequest)
	}

	if err := tenantdb.For(c, h.db).Delete(&appointment).Error; err != nil {
		h.logger.Error("Failed to delete appointment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	tenantID, _ := c.Get("tenant_id")
	if tenantID != nil {
		go h.syncDelete(tenantID.(uint), &appointment)
	}

	h.logger.Info("Appointment deleted", zap.Int("id", id))
	return envelope.SuccessResponse(nil, "appointments.delete.success")
}

// patientInTenant reports whether the patient belongs to the caller's tenant, so
// an appointment can never point at (and later preload) another clinic's patient.
func (h *AppointmentHandler) patientInTenant(c *gin.Context, patientID uint) bool {
	var count int64
	tenantdb.For(c, h.db).Model(&clinical_models.Patient{}).Where("id = ?", patientID).Count(&count)
	return count == 1
}
