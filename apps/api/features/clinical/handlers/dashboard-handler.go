package clinical_handlers

import (
	"net/http"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDashboardHandler(db *gorm.DB, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{db: db, logger: logger}
}

// ─── Response DTOs ───────────────────────────────────────────────────────────

type WeekDayStat struct {
	Date  string `json:"date"`
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

type UpcomingAppointment struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	PatientName string `json:"patient_name"`
	PatientID   uint   `json:"patient_id"`
	Status      string `json:"status"`
}

type DashboardStats struct {
	TotalPatients        int64                 `json:"total_patients"`
	CriticalPatients     int64                 `json:"critical_patients"`
	TodayAppointments    int64                 `json:"today_appointments"`
	MonthlyCompleted     int64                 `json:"monthly_completed"`
	WeeklyAppointments   []WeekDayStat         `json:"weekly_appointments"`
	UpcomingAppointments []UpcomingAppointment `json:"upcoming_appointments"`
}

// GetDashboardStats returns aggregated statistics for the dashboard
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) envelope.Response {
	scope := tenant_middleware.TenantScope(c)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 1. Total patients
	var totalPatients int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Patient{}).Count(&totalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// 2. Critical patients
	var criticalPatients int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Patient{}).Where("critical = ?", true).Count(&criticalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count critical patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// 3. Today's appointments
	var todayAppointments int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("date >= ? AND date < ?", todayStart, todayEnd).
		Count(&todayAppointments).Error; err != nil {
		h.logger.Error("Dashboard: failed to count today appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// 4. Monthly completed appointments
	var monthlyCompleted int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("status = ? AND date >= ?", "completed", monthStart).
		Count(&monthlyCompleted).Error; err != nil {
		h.logger.Error("Dashboard: failed to count monthly completed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// 5. Weekly appointments (current week Mon-Sun)
	weekday := now.Weekday()
	offset := (int(weekday) + 6) % 7 // Monday = 0
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-offset, 0, 0, 0, 0, now.Location())

	dayNames := []string{"Lun", "Mar", "Mié", "Jue", "Vie", "Sáb", "Dom"}
	weeklyStats := make([]WeekDayStat, 7)
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		dayEnd := day.Add(24 * time.Hour)

		var count int64
		h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
			Where("date >= ? AND date < ?", day, dayEnd).
			Count(&count)

		weeklyStats[i] = WeekDayStat{
			Date:  day.Format("2006-01-02"),
			Day:   dayNames[i],
			Count: count,
		}
	}

	// 6. Upcoming appointments today (next 5, ordered by start_time)
	var upcomingRaw []clinical_models.Appointment
	h.db.Scopes(scope).
		Where("date >= ? AND date < ? AND status = ?", todayStart, todayEnd, "scheduled").
		Preload("Patient").
		Order("start_time ASC").
		Limit(5).
		Find(&upcomingRaw)

	upcoming := make([]UpcomingAppointment, 0, len(upcomingRaw))
	for _, a := range upcomingRaw {
		name := ""
		if a.Patient.FullName != nil {
			name = *a.Patient.FullName
		} else {
			name = a.Patient.FirstName + " " + a.Patient.LastName
		}
		upcoming = append(upcoming, UpcomingAppointment{
			ID:          a.ID,
			Title:       a.Title,
			StartTime:   a.StartTime,
			EndTime:     a.EndTime,
			PatientName: name,
			PatientID:   a.PatientID,
			Status:      a.Status,
		})
	}

	stats := DashboardStats{
		TotalPatients:        totalPatients,
		CriticalPatients:     criticalPatients,
		TodayAppointments:    todayAppointments,
		MonthlyCompleted:     monthlyCompleted,
		WeeklyAppointments:   weeklyStats,
		UpcomingAppointments: upcoming,
	}

	return envelope.SuccessResponse(stats, "dashboard.stats.success")
}
