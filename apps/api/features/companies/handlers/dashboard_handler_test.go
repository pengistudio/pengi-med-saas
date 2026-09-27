package company_handlers

import (
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	kanban_models "pengi-med-saas/features/kanban/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

// A tenant id unlikely to hold data in a shared local test database.
const dashboardTestTenant uint = 9301

func TestGetDashboardStats_TodayAndNeedsAttention(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&clinical_models.Patient{}, &clinical_models.Appointment{}, &clinical_models.MedicalRecordDraft{},
		&billing_models.Invoice{}, &kanban_models.Task{},
		&company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{}, &company_models.SubscriptionPayment{},
	)
	tdb := tenantdb.ForTenant(db, dashboardTestTenant)
	mustCreate := func(v any) {
		t.Helper()
		if err := tdb.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}

	ana := clinical_models.Patient{FirstName: "Ana", LastName: "Mora", Critical: true}
	luis := clinical_models.Patient{FirstName: "Luis", LastName: "Paz"}
	mustCreate(&ana)
	mustCreate(&luis)

	today := time.Now()
	mustCreate(&clinical_models.Appointment{PatientID: luis.ID, Title: "Control", Date: today, StartTime: "11:00", EndTime: "11:30", Status: "scheduled"})
	mustCreate(&clinical_models.Appointment{PatientID: ana.ID, Title: "Primera vez", Date: today, StartTime: "08:00", EndTime: "08:30", Status: "completed"})
	mustCreate(&clinical_models.Appointment{PatientID: ana.ID, Title: "Anulada", Date: today, StartTime: "09:00", EndTime: "09:30", Status: "cancelled"})

	const me, colleague = 41, 42
	mustCreate(&clinical_models.MedicalRecordDraft{UserID: me, PatientID: ana.ID})
	mustCreate(&clinical_models.MedicalRecordDraft{UserID: colleague, PatientID: luis.ID})

	mustCreate(&billing_models.Invoice{Status: "rejected"})
	mustCreate(&billing_models.Invoice{Status: "failed"})
	mustCreate(&billing_models.Invoice{Status: "authorized"})

	due := today.AddDate(0, 0, 1)
	archived := today
	mustCreate(&kanban_models.Task{Title: "Sin fecha", Status: "todo"})
	mustCreate(&kanban_models.Task{Title: "Llamar laboratorio", Status: "in_progress", DueDate: &due})
	mustCreate(&kanban_models.Task{Title: "Hecha", Status: "done"})
	mustCreate(&kanban_models.Task{Title: "Archivada", Status: "todo", ArchivedAt: &archived})

	c, _ := testutils.NewGinContext(dashboardTestTenant, me)
	c.Set("user_id", int64(me))
	c.Set("username", "doctora")

	res := NewDashboardHandler(db, zap.NewNop()).GetDashboardStats(c)
	stats, ok := res.Data.(DashboardStats)
	if !ok {
		t.Fatalf("unexpected response: %+v", res)
	}

	if len(stats.TodayAgenda) != 2 || stats.TodayAgenda[0].StartTime != "08:00" || stats.TodayAgenda[1].PatientName != "Luis Paz" {
		t.Errorf("today agenda = %+v, want 08:00 then 11:00 without the cancelled one", stats.TodayAgenda)
	}
	if len(stats.CriticalPatientList) != 1 || stats.CriticalPatientList[0].Name != "Ana Mora" {
		t.Errorf("critical patients = %+v", stats.CriticalPatientList)
	}
	if stats.PendingDraftsCount != 1 || len(stats.PendingDrafts) != 1 || stats.PendingDrafts[0].PatientID != ana.ID {
		t.Errorf("pending drafts = %d %+v, want only my draft", stats.PendingDraftsCount, stats.PendingDrafts)
	}
	if stats.FailedInvoices != 2 {
		t.Errorf("failed invoices = %d, want 2", stats.FailedInvoices)
	}
	if stats.OpenTasksCount != 2 || len(stats.OpenTasks) != 2 || stats.OpenTasks[0].Title != "Llamar laboratorio" {
		t.Errorf("open tasks = %d %+v, want the dated one first", stats.OpenTasksCount, stats.OpenTasks)
	}
}
