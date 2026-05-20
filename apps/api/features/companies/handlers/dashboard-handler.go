package company_handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	tenant_models "pengi-med-saas/features/tenants/models"

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

type DashboardSubscriptionInfo struct {
	PlanName          string          `json:"plan_name"`
	PlanCode          string          `json:"plan_code"`
	Status            string          `json:"status"`
	ExpiresAt         time.Time       `json:"expires_at"`
	DaysLeft          int             `json:"days_left"`
	Amount            float64         `json:"amount"`
	LastPaymentAmount float64         `json:"last_payment_amount"`
	LastPaymentMonths int             `json:"last_payment_months"`
	LastPaymentDate   *string         `json:"last_payment_date"`
	EnabledFeatures   map[string]bool `json:"enabled_features"`
}

type DashboardStats struct {
	TotalPatients         int64                      `json:"total_patients"`
	NewPatientsThisMonth  int64                      `json:"new_patients_this_month"`
	CriticalPatients      int64                      `json:"critical_patients"`
	TodayAppointments     int64                      `json:"today_appointments"`
	YesterdayAppointments int64                      `json:"yesterday_appointments"`
	MonthlyCompleted      int64                      `json:"monthly_completed"`
	PrevMonthCompleted    int64                      `json:"prev_month_completed"`
	WeeklyAppointments    []WeekDayStat              `json:"weekly_appointments"`
	UpcomingAppointments  []UpcomingAppointment      `json:"upcoming_appointments"`
	Subscription          *DashboardSubscriptionInfo `json:"subscription"`
}

// GetDashboardStats returns aggregated statistics for the dashboard.
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) envelope.Response {
	scope := tenant_middleware.TenantScope(c)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	prevMonthStart := monthStart.AddDate(0, -1, 0)

	// 1. Total patients
	var totalPatients int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Patient{}).Count(&totalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	// 2. Critical patients
	var criticalPatients int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Patient{}).Where("critical = ?", true).Count(&criticalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count critical patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	// 3. Today's appointments
	var todayAppointments int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("date >= ? AND date < ?", todayStart, todayEnd).
		Count(&todayAppointments).Error; err != nil {
		h.logger.Error("Dashboard: failed to count today appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	// 4. Monthly completed appointments
	var monthlyCompleted int64
	if err := h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("status = ? AND date >= ?", "completed", monthStart).
		Count(&monthlyCompleted).Error; err != nil {
		h.logger.Error("Dashboard: failed to count monthly completed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrClinicalInvalidRequest)
	}

	// 4b. Delta queries (best-effort — don't fail the whole response on error)
	var newPatientsThisMonth int64
	h.db.Scopes(scope).Model(&clinical_models.Patient{}).
		Where("created_at >= ?", monthStart).
		Count(&newPatientsThisMonth)

	var prevMonthCompleted int64
	h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("status = ? AND date >= ? AND date < ?", "completed", prevMonthStart, monthStart).
		Count(&prevMonthCompleted)

	var yesterdayAppointments int64
	h.db.Scopes(scope).Model(&clinical_models.Appointment{}).
		Where("date >= ? AND date < ?", yesterdayStart, todayStart).
		Count(&yesterdayAppointments)

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

	// 7. Subscription + enabled features for this tenant's company
	var subscriptionInfo *DashboardSubscriptionInfo
	tenantID := c.GetUint("tenant_id")
	var company company_models.Company
	if err := h.db.Where("tenant_id = ?", tenantID).First(&company).Error; err == nil {
		var sub company_models.Subscription
		if err := h.db.Preload("Plan").
			Where("company_id = ?", company.ID).
			Order("expires_at DESC").
			First(&sub).Error; err == nil {
			daysLeft := int(time.Until(sub.ExpiresAt).Hours() / 24)
			if daysLeft < 0 {
				daysLeft = 0
			}
			if checkAndApplyPendingPlanChange(h.db, h.logger, &sub, &company) {
				h.db.Preload("Plan").First(&sub, sub.ID)
			}

			subscriptionInfo = &DashboardSubscriptionInfo{
				PlanName:  sub.Plan.Name,
				PlanCode:  sub.PlanCode,
				Status:    sub.Status,
				ExpiresAt: sub.ExpiresAt,
				DaysLeft:  daysLeft,
				Amount:    sub.Plan.Price,
			}

			var lastPayment company_models.SubscriptionPayment
			if err := h.db.Where("company_id = ? AND status = ?", company.ID, "paid").
				Order("created_at DESC").First(&lastPayment).Error; err == nil {
				subscriptionInfo.LastPaymentAmount = lastPayment.Amount
				subscriptionInfo.LastPaymentMonths = lastPayment.Months
				paidDate := lastPayment.CreatedAt.Format("2006-01-02")
				subscriptionInfo.LastPaymentDate = &paidDate
			}

			var tenant tenant_models.Tenant
			if err := h.db.First(&tenant, tenantID).Error; err == nil {
				ef := tenant_models.DefaultEnabledFeatures()
				if tenant.EnabledFeatures != "" {
					json.Unmarshal([]byte(tenant.EnabledFeatures), &ef)
				}
				subscriptionInfo.EnabledFeatures = map[string]bool{
					"clinical": ef.Clinical,
					"billing":  ef.Billing,
					"team":     ef.Team,
					"kanban":   ef.Kanban,
				}
			}
		}
	}

	stats := DashboardStats{
		TotalPatients:         totalPatients,
		NewPatientsThisMonth:  newPatientsThisMonth,
		CriticalPatients:      criticalPatients,
		TodayAppointments:     todayAppointments,
		YesterdayAppointments: yesterdayAppointments,
		MonthlyCompleted:      monthlyCompleted,
		PrevMonthCompleted:    prevMonthCompleted,
		WeeklyAppointments:    weeklyStats,
		UpcomingAppointments:  upcoming,
		Subscription:          subscriptionInfo,
	}

	return envelope.SuccessResponse(stats, "dashboard.stats.success")
}
