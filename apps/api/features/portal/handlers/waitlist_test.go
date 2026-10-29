package portal_handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_models "pengi-med-saas/features/portal/models"
	portal_workers "pengi-med-saas/features/portal/workers"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"

	"go.uber.org/zap"
)

// waitlistManager is a user with MANAGE_APPOINTMENT: told when a slot frees.
const waitlistManager = 8

func newWaitlistFixture(t *testing.T) *bookingFixture {
	t.Helper()
	b := newBookingFixture(t)
	if err := b.db.AutoMigrate(&waitlist_models.WaitlistEntry{}); err != nil {
		t.Fatal(err)
	}
	// Like registerPortalRoutes: session + booking on.
	bh := NewBookingHandler(b.db, zap.NewNop(), b.delivery, b.calendar).WithSyncAfterCommit()
	booking := b.router.Group("/portal", portal_middleware.RequirePortalSession(b.db), portal_middleware.RequireBooking())
	booking.GET("/waitlist", envelope.Handle(bh.ListWaitlist))
	booking.POST("/waitlist", envelope.Handle(bh.JoinWaitlist))
	booking.DELETE("/waitlist/:id", envelope.Handle(bh.LeaveWaitlist))

	var company company_models.Company
	if err := tenantdb.System(b.db).Where("tenant_id = ?", b.own.ID).First(&company).Error; err != nil {
		t.Fatal(err)
	}
	role := user_models.Role{Role: "agenda", Permissions: []permission_models.Permission{{
		BaseStringID: database.BaseStringID{ID: waitlist_services.ManagePermission}, Name: "x", Category: "CLINICAL"}}}
	b.mustCreate(&role)
	b.mustCreate(&user_models.Environment{UserID: waitlistManager, RoleID: role.ID, CompanyID: company.ID})
	return b
}

func (b *bookingFixture) slotNotifications() []notifications_models.Notification {
	var rows []notifications_models.Notification
	tenantdb.System(b.db).Where("user_id = ? AND type = ?", waitlistManager, waitlist_services.NotificationTypeSlotFreed).Order("id").Find(&rows)
	return rows
}

func TestPortalWaitlist_JoinListLeave(t *testing.T) {
	b := newWaitlistFixture(t)
	token := b.token(b.own, b.ownPat)

	// Only a signed-in patient.
	if r := b.do(http.MethodPost, "/portal/waitlist", "", map[string]any{"type_id": b.typ.ID}); r.Code != http.StatusUnauthorized {
		t.Fatalf("without a session: %d", r.Code)
	}

	r := b.do(http.MethodPost, "/portal/waitlist", token, map[string]any{"type_id": b.typ.ID, "doctor_id": b.doctor.ID})
	if r.Code != http.StatusCreated {
		t.Fatalf("join: %d %s", r.Code, r.Raw)
	}
	data := r.Body["data"].(map[string]any)
	id := uint(data["id"].(float64))
	if data["status"] != waitlist_models.StatusWaiting || data["type_name"] != "Consulta" || data["doctor_name"] != b.doctor.FullName {
		t.Fatalf("entry = %v", data)
	}
	var stored waitlist_models.WaitlistEntry
	tenantdb.System(b.db).First(&stored, id)
	if stored.TenantID != b.own.ID || stored.PatientID != b.ownPat.ID || stored.Source != waitlist_models.SourcePortal ||
		!stored.ExpiresAt.Equal(bookingNow.Add(waitlist_services.DefaultTTL)) {
		t.Fatalf("stored = %+v", stored)
	}

	// Joining again returns the same entry.
	again := okData(t, b.do(http.MethodPost, "/portal/waitlist", token, map[string]any{"type_id": b.typ.ID, "doctor_id": b.doctor.ID}))
	if uint(again["id"].(float64)) != id {
		t.Fatalf("second join created another entry: %v", again)
	}
	// A type the portal doesn't book, or an unknown doctor: refused.
	if r := b.do(http.MethodPost, "/portal/waitlist", token, map[string]any{"type_id": 999999}); r.Code != http.StatusBadRequest {
		t.Fatalf("unknown type: %d", r.Code)
	}
	if r := b.do(http.MethodPost, "/portal/waitlist", token, map[string]any{"type_id": b.typ.ID, "doctor_id": 999999}); r.Code != http.StatusBadRequest {
		t.Fatalf("unknown doctor: %d", r.Code)
	}

	list := b.do(http.MethodGet, "/portal/waitlist", token, nil)
	if items := list.Body["data"].([]any); list.Code != http.StatusOK || len(items) != 1 {
		t.Fatalf("list = %d %s", list.Code, list.Raw)
	}
	// Another patient sees nothing and can't remove it.
	other := b.token(b.own, b.ownPat2)
	if items := b.do(http.MethodGet, "/portal/waitlist", other, nil).Body["data"].([]any); len(items) != 0 {
		t.Fatalf("other patient sees %v", items)
	}
	if r := b.do(http.MethodDelete, fmt.Sprintf("/portal/waitlist/%d", id), other, nil); r.Code != http.StatusNotFound {
		t.Fatalf("other patient leave: %d", r.Code)
	}
	left := okData(t, b.do(http.MethodDelete, fmt.Sprintf("/portal/waitlist/%d", id), token, nil))
	if left["status"] != waitlist_models.StatusRemoved {
		t.Fatalf("left = %v", left)
	}

	// Gated like booking: with booking off the endpoint is not available.
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = false })
	if r := b.do(http.MethodPost, "/portal/waitlist", token, map[string]any{"type_id": b.typ.ID}); r.Code != http.StatusNotFound {
		t.Fatalf("booking off: %d", r.Code)
	}
}

func TestPortalWaitlist_ReleasedRequestNotifies(t *testing.T) {
	b := newWaitlistFixture(t)
	// ownPat2 waits for any doctor; ownPat asks for a slot, then it is rejected.
	b.mustCreate(&waitlist_models.WaitlistEntry{TenantID: b.own.ID, PatientID: b.ownPat2.ID, Status: waitlist_models.StatusWaiting,
		ExpiresAt: bookingNow.Add(24 * time.Hour)})
	token := b.token(b.own, b.ownPat)

	created := okData(t, b.request(token, "09:00", nil))
	if len(b.slotNotifications()) != 0 {
		t.Fatal("asking for a slot frees nothing")
	}
	okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/reject", int(created["id"].(float64))), b.own, ""))
	ns := b.slotNotifications()
	if len(ns) != 1 || ns[0].TenantID != b.own.ID {
		t.Fatalf("after reject: %+v", ns)
	}

	// The patient cancelling a pending request frees it too.
	created = okData(t, b.request(token, "09:30", nil))
	okData(t, b.do(http.MethodPost, fmt.Sprintf("/portal/booking/requests/%d/cancel", int(created["id"].(float64))), token, nil))
	if len(b.slotNotifications()) != 2 {
		t.Fatalf("after cancel: %d", len(b.slotNotifications()))
	}

	// And so does a request nobody decided in time.
	created = okData(t, b.request(token, "10:00", nil))
	id := uint(created["id"].(float64))
	if err := tenantdb.System(b.db).Model(&portal_models.AppointmentRequest{}).Where("id = ?", id).
		Update("expires_at", bookingNow.Add(-time.Minute).UTC()).Error; err != nil {
		t.Fatal(err)
	}
	if n := portal_workers.NewRequestExpiryScheduler(b.db, zap.NewNop(), nil).WithClock(func() time.Time { return bookingNow }).Tick(); n != 1 {
		t.Fatalf("expired = %d", n)
	}
	if len(b.slotNotifications()) != 3 {
		t.Fatalf("after expiry: %d", len(b.slotNotifications()))
	}

	// Nobody waiting: no notification.
	tenantdb.System(b.db).Model(&waitlist_models.WaitlistEntry{}).Where("tenant_id = ?", b.own.ID).Update("status", waitlist_models.StatusRemoved)
	created = okData(t, b.request(token, "10:30", nil))
	okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/reject", int(created["id"].(float64))), b.own, ""))
	if len(b.slotNotifications()) != 3 {
		t.Fatalf("notified without matches: %d", len(b.slotNotifications()))
	}
}
