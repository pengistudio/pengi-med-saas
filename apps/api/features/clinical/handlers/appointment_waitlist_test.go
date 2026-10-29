package clinical_handlers

import (
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"

	"go.uber.org/zap"
)

// Cancelling or deleting a future appointment frees its slot: reception is
// told when the waitlist wants it. Closed or past appointments free nothing.
func TestAppointment_CancelOrDeleteFreesSlotForWaitlist(t *testing.T) {
	s := newIsolation(t)
	if err := s.db.AutoMigrate(&waitlist_models.WaitlistEntry{}, &notifications_models.Notification{}, &company_models.Company{},
		&user_models.Environment{}, &user_models.Role{}, &permission_models.Permission{}); err != nil {
		t.Fatal(err)
	}
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "P", TenantID: s.own.ID}
	role := user_models.Role{Role: "agenda", Permissions: []permission_models.Permission{{
		BaseStringID: database.BaseStringID{ID: waitlist_services.ManagePermission}, Name: "x"}}}
	for _, row := range []any{&company, &role} {
		if err := s.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.db.Create(&user_models.Environment{UserID: 50, CompanyID: company.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	waiting := s.patient(s.own.ID, "WAIT")
	if err := s.db.Create(&waitlist_models.WaitlistEntry{TenantID: s.own.ID, PatientID: waiting.ID, Status: waitlist_models.StatusWaiting,
		ExpiresAt: time.Now().Add(400 * 24 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	count := func() int {
		var n int64
		s.db.Model(&notifications_models.Notification{}).Where("user_id = ? AND type = ?", 50, waitlist_services.NotificationTypeSlotFreed).Count(&n)
		return int(n)
	}
	h := NewAppointmentHandler(s.db, zap.NewNop())
	future := time.Now().Add(72 * time.Hour)

	appt := s.appointmentOn(s.own.ID, s.ownPatient.ID, future, "scheduled")
	if code := h.UpdateStatus(s.ctx(appt.ID, map[string]string{"status": "cancelled"})).Code; code != http.StatusOK {
		t.Fatalf("cancel = %d", code)
	}
	if count() != 1 {
		t.Fatalf("after cancel: %d notifications", count())
	}
	// Deleting it now (already cancelled) frees nothing more.
	if code := h.DeleteAppointment(s.ctx(appt.ID, nil)).Code; code != http.StatusOK {
		t.Fatalf("delete = %d", code)
	}
	if count() != 1 {
		t.Fatalf("deleting a cancelled appointment notified again: %d", count())
	}
	// Deleting a scheduled one frees it.
	other := s.appointmentOn(s.own.ID, s.ownPatient.ID, future.Add(24*time.Hour), "scheduled")
	h.DeleteAppointment(s.ctx(other.ID, nil))
	if count() != 2 {
		t.Fatalf("after delete: %d", count())
	}
	// Past appointments and no-shows corrected to cancelled free nothing.
	past := s.appointmentOn(s.own.ID, s.ownPatient.ID, time.Now().Add(-72*time.Hour), "scheduled")
	h.UpdateStatus(s.ctx(past.ID, map[string]string{"status": "cancelled"}))
	noShow := s.appointmentOn(s.own.ID, s.ownPatient.ID, future.Add(48*time.Hour), clinical_models.AppointmentStatusNoShow)
	h.UpdateStatus(s.ctx(noShow.ID, map[string]string{"status": "cancelled"}))
	if count() != 2 {
		t.Fatalf("past or closed appointments notified: %d", count())
	}
}
