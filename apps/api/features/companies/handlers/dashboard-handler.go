package company_handlers

import (
	"net/http"
	"pengi-med-saas/core/tenantdb"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	kanban_models "pengi-med-saas/features/kanban/models"
	auth_middleware "pengi-med-saas/features/users/middleware"

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

// dashboardListLimit caps every "needs attention" list; the counts next to
// them say how many there are in total.
const dashboardListLimit = 5

type DashboardPatientRef struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type DashboardDraft struct {
	PatientID   uint      `json:"patient_id"`
	PatientName string    `json:"patient_name"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DashboardTask struct {
	ID      uint       `json:"id"`
	Title   string     `json:"title"`
	Status  string     `json:"status"`
	DueDate *time.Time `json:"due_date"`
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
	TodayAgenda           []UpcomingAppointment      `json:"today_agenda"`
	CriticalPatientList   []DashboardPatientRef      `json:"critical_patient_list"`
	PendingDrafts         []DashboardDraft           `json:"pending_drafts"`
	PendingDraftsCount    int64                      `json:"pending_drafts_count"`
	FailedInvoices        int64                      `json:"failed_invoices"`
	OpenTasks             []DashboardTask            `json:"open_tasks"`
	OpenTasksCount        int64                      `json:"open_tasks_count"`
	Subscription          *DashboardSubscriptionInfo `json:"subscription"`
}

func patientDisplayName(p clinical_models.Patient) string {
	if p.FullName != nil && *p.FullName != "" {
		return *p.FullName
	}
	return p.FirstName + " " + p.LastName
}

// GetDashboardStats returns aggregated statistics for the dashboard.
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	prevMonthStart := monthStart.AddDate(0, -1, 0)

	// 1. Total patients
	var totalPatients int64
	if err := db.Model(&clinical_models.Patient{}).Count(&totalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// 2. Critical patients
	var criticalPatients int64
	if err := db.Model(&clinical_models.Patient{}).Where("critical = ?", true).Count(&criticalPatients).Error; err != nil {
		h.logger.Error("Dashboard: failed to count critical patients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// 3. Today's appointments
	var todayAppointments int64
	if err := db.Model(&clinical_models.Appointment{}).
		Where("date >= ? AND date < ?", todayStart, todayEnd).
		Count(&todayAppointments).Error; err != nil {
		h.logger.Error("Dashboard: failed to count today appointments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// 4. Monthly completed appointments
	var monthlyCompleted int64
	if err := db.Model(&clinical_models.Appointment{}).
		Where("status = ? AND date >= ?", "completed", monthStart).
		Count(&monthlyCompleted).Error; err != nil {
		h.logger.Error("Dashboard: failed to count monthly completed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	// 4b. Delta queries (best-effort — don't fail the whole response on error)
	var newPatientsThisMonth int64
	db.Model(&clinical_models.Patient{}).
		Where("created_at >= ?", monthStart).
		Count(&newPatientsThisMonth)

	var prevMonthCompleted int64
	db.Model(&clinical_models.Appointment{}).
		Where("status = ? AND date >= ? AND date < ?", "completed", prevMonthStart, monthStart).
		Count(&prevMonthCompleted)

	var yesterdayAppointments int64
	db.Model(&clinical_models.Appointment{}).
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
		db.Model(&clinical_models.Appointment{}).
			Where("date >= ? AND date < ?", day, dayEnd).
			Count(&count)

		weeklyStats[i] = WeekDayStat{
			Date:  day.Format("2006-01-02"),
			Day:   dayNames[i],
			Count: count,
		}
	}

	// 6. Today's agenda: every appointment of the day but cancelled ones, so
	// the timeline shows who already came and who is next.
	var agendaRaw []clinical_models.Appointment
	db.
		Where("date >= ? AND date < ? AND status <> ?", todayStart, todayEnd, "cancelled").
		Preload("Patient").
		Order("start_time ASC").
		Find(&agendaRaw)

	agenda := make([]UpcomingAppointment, 0, len(agendaRaw))
	for _, a := range agendaRaw {
		agenda = append(agenda, UpcomingAppointment{
			ID:          a.ID,
			Title:       a.Title,
			StartTime:   a.StartTime,
			EndTime:     a.EndTime,
			PatientName: patientDisplayName(a.Patient),
			PatientID:   a.PatientID,
			Status:      a.Status,
		})
	}

	// 6b. Needs attention (best-effort, like the deltas above).
	var criticalRaw []clinical_models.Patient
	db.Where("critical = ?", true).Order("updated_at DESC").Limit(dashboardListLimit).Find(&criticalRaw)
	criticalList := make([]DashboardPatientRef, 0, len(criticalRaw))
	for _, p := range criticalRaw {
		criticalList = append(criticalList, DashboardPatientRef{ID: p.ID, Name: patientDisplayName(p)})
	}

	// Unfinished medical records of the current user (drafts are per user).
	pendingDrafts := make([]DashboardDraft, 0)
	var pendingDraftsCount int64
	if userID, _, ok := auth_middleware.GetUserFromContext(c); ok {
		draftQuery := db.Model(&clinical_models.MedicalRecordDraft{}).Where("user_id = ?", userID)
		draftQuery.Count(&pendingDraftsCount)

		var draftsRaw []clinical_models.MedicalRecordDraft
		db.Where("user_id = ?", userID).Order("updated_at DESC").Limit(dashboardListLimit).Find(&draftsRaw)
		for _, d := range draftsRaw {
			var patient clinical_models.Patient
			if err := db.First(&patient, d.PatientID).Error; err != nil {
				continue
			}
			pendingDrafts = append(pendingDrafts, DashboardDraft{
				PatientID:   d.PatientID,
				PatientName: patientDisplayName(patient),
				UpdatedAt:   d.UpdatedAt,
			})
		}
	}

	// Invoices the SRI refused or that failed: they need someone to act.
	var failedInvoices int64
	db.Model(&billing_models.Invoice{}).Where("status IN ?", []string{"failed", "rejected"}).Count(&failedInvoices)

	// Open kanban tasks, the ones with the nearest due date first.
	var openTasksCount int64
	openTaskQuery := db.Model(&kanban_models.Task{}).Where("status <> ? AND archived_at IS NULL", "done")
	openTaskQuery.Count(&openTasksCount)
	var tasksRaw []kanban_models.Task
	db.Where("status <> ? AND archived_at IS NULL", "done").
		Order("due_date IS NULL, due_date ASC, position ASC").
		Limit(dashboardListLimit).
		Find(&tasksRaw)
	openTasks := make([]DashboardTask, 0, len(tasksRaw))
	for _, t := range tasksRaw {
		openTasks = append(openTasks, DashboardTask{ID: t.ID, Title: t.Title, Status: t.Status, DueDate: t.DueDate})
	}

	// 7. Subscription + enabled features for this tenant's company
	var subscriptionInfo *DashboardSubscriptionInfo
	var company company_models.Company
	if err := tenantdb.For(c, h.db).First(&company).Error; err == nil {
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

			if ef, err := company_services.EnabledFeaturesForPlanCode(h.db, sub.PlanCode); err == nil {
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
		TodayAgenda:           agenda,
		CriticalPatientList:   criticalList,
		PendingDrafts:         pendingDrafts,
		PendingDraftsCount:    pendingDraftsCount,
		FailedInvoices:        failedInvoices,
		OpenTasks:             openTasks,
		OpenTasksCount:        openTasksCount,
		Subscription:          subscriptionInfo,
	}

	return envelope.SuccessResponse(stats, "dashboard.stats.success")
}
