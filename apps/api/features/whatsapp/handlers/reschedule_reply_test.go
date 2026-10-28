package whatsapp_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/database"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
)

// seedNoShow gives tenantID a connected account on phoneID (sealed token)
// and a sent no-show message for a new appointment marked no_show.
func (f *fixture) seedNoShow(tenantID uint, phoneID string) (clinical_models.Appointment, whatsapp_models.WhatsAppMessage) {
	f.t.Helper()
	if f.account(tenantID).ID == 0 {
		f.connectedAccount(tenantID, phoneID)
	}
	patient := clinical_models.Patient{TenantID: tenantID, FirstName: "Luis", LastName: "Mora", Phone: "0991234567", WhatsAppOptIn: true, Document: fmt.Sprintf("D%d", time.Now().UnixNano())}
	if err := f.db.Create(&patient).Error; err != nil {
		f.t.Fatal(err)
	}
	day := time.Date(2026, 10, 27, 0, 0, 0, 0, whatsapp_services.ClinicLocation())
	appt := clinical_models.Appointment{TenantID: tenantID, PatientID: patient.ID, Title: "Control",
		Date: day, StartTime: "10:00", EndTime: "10:30", Status: clinical_models.AppointmentStatusNoShow}
	if err := f.db.Create(&appt).Error; err != nil {
		f.t.Fatal(err)
	}
	apptID, patientID := appt.ID, patient.ID
	msg := whatsapp_models.WhatsAppMessage{TenantID: tenantID, Kind: whatsapp_models.KindNoShow, AppointmentID: &apptID, PatientID: &patientID,
		OffsetHours: whatsapp_models.NoShowOffsetHours, Template: whatsapp_models.TemplateNoShow, Status: whatsapp_models.MessageStatusSent,
		WamID: fmt.Sprintf("wamid.ns.%d", time.Now().UnixNano()), ToPhone: "593991234567"}
	if err := f.db.Create(&msg).Error; err != nil {
		f.t.Fatal(err)
	}
	return appt, msg
}

// portalBooking gives tenantID a plan with the patient portal and turns the
// portal and its online booking on.
func (f *fixture) portalBooking(tenantID uint) {
	f.t.Helper()
	perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: "MANAGE_PATIENT_PORTAL"}, Name: "Portal", Category: "PATIENT_PORTAL"}
	if err := f.db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
		f.t.Fatal(err)
	}
	code := fmt.Sprintf("PORTAL-%d", time.Now().UnixNano())
	feature := company_models.Feature{Code: code, Name: "Portal", Permissions: []permission_models.Permission{perm}}
	if err := f.db.Create(&feature).Error; err != nil {
		f.t.Fatal(err)
	}
	plan := company_models.Plan{Name: code, Code: code, Price: 10, Features: []company_models.Feature{feature}}
	if err := f.db.Create(&plan).Error; err != nil {
		f.t.Fatal(err)
	}
	var company company_models.Company
	if err := f.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	if err := f.db.Create(&company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().Add(30 * 24 * time.Hour), CompanyID: company.ID}).Error; err != nil {
		f.t.Fatal(err)
	}
	settings := tenant_models.DefaultPortalSettings()
	settings.Enabled, settings.Sections.Appointments, settings.Booking.Enabled = true, true, true
	raw, _ := json.Marshal(settings)
	if err := f.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("portal_settings", string(raw)).Error; err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) slug(tenantID uint) string {
	var tenant tenant_models.Tenant
	f.db.Select("id", "slug").First(&tenant, tenantID)
	return tenant.Slug
}

func TestWebhook_RescheduleWithPortalRepliesWithLink(t *testing.T) {
	if config.Get().PortalURL == "" {
		t.Skip("PORTAL_URL not set")
	}
	f := newFixture(t)
	f.portalBooking(f.own)
	f.grant(f.own, []uint{100}, "USE_WHATSAPP_INBOX")
	appt, msg := f.seedNoShow(f.own, "PHONE-1")

	f.signed(replyEvent("PHONE-1", fmt.Sprintf("reschedule:%d", msg.ID), "593991234567", msg.WamID))

	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusReplied || m.Reply != whatsapp_models.ReplyReschedule {
		t.Fatalf("message = %+v", m)
	}
	link := strings.TrimRight(config.Get().PortalURL, "/") + "/" + f.slug(f.own)
	sent := f.graph.bodyWith("POST /v23.0/PHONE-1/messages", `"type":"text"`)
	if !strings.Contains(sent, link) || !strings.Contains(sent, `"to":"593991234567"`) {
		t.Fatalf("graph body = %q, want the link %s", sent, link)
	}
	// Both the tap and the answer are in the thread; the answer is a free
	// system text.
	convs := f.conversations(f.own)
	if len(convs) != 1 {
		t.Fatalf("conversations = %d", len(convs))
	}
	var system []whatsapp_models.WhatsAppMessage
	for _, m := range f.thread(convs[0].ID) {
		if m.Kind == whatsapp_models.KindSystem {
			system = append(system, m)
		}
	}
	if len(system) != 1 || !strings.Contains(system[0].Body, link) || system[0].Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("system messages = %+v", system)
	}
	// No reschedule notification: the patient books by themselves.
	if ns := f.notifications(100, whatsapp_services.NotificationTypeRescheduleRequest); len(ns) != 0 {
		t.Fatalf("notifications = %+v", ns)
	}
	if s := f.appointmentStatus(appt.ID); s != clinical_models.AppointmentStatusNoShow {
		t.Fatalf("appointment status = %s", s)
	}

	// Meta delivering the tap again sends nothing more.
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("reschedule:%d", msg.ID), "593991234567", msg.WamID))
	if n := len(f.graph.allBodies["POST /v23.0/PHONE-1/messages"]); n != 1 {
		t.Fatalf("texts sent = %d, want 1", n)
	}
}

func TestWebhook_RescheduleWithoutPortalNotifiesInbox(t *testing.T) {
	f := newFixture(t)
	f.grant(f.own, []uint{100, inboxUserID}, "USE_WHATSAPP_INBOX")
	f.grant(f.other, []uint{110}, "USE_WHATSAPP_INBOX")
	_, msg := f.seedNoShow(f.own, "PHONE-1")

	f.signed(replyEvent("PHONE-1", fmt.Sprintf("reschedule:%d", msg.ID), "593991234567", msg.WamID))

	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusReplied || m.Reply != whatsapp_models.ReplyReschedule {
		t.Fatalf("message = %+v", m)
	}
	if f.graph.called("POST /v23.0/PHONE-1/messages") {
		t.Fatal("no text must be sent without online booking")
	}
	convs := f.conversations(f.own)
	if len(convs) != 1 {
		t.Fatalf("conversations = %d", len(convs))
	}
	thread := f.thread(convs[0].ID)
	if len(thread) != 1 || thread[0].Direction != whatsapp_models.DirectionInbound || thread[0].Reply != whatsapp_models.ReplyReschedule {
		t.Fatalf("thread = %+v", thread)
	}
	for _, uid := range []uint{100, inboxUserID} {
		ns := f.notifications(uid, whatsapp_services.NotificationTypeRescheduleRequest)
		if len(ns) != 1 || ns[0].MessageKey != whatsapp_services.RescheduleRequestKey || ns[0].ResourceID != convs[0].ID ||
			ns[0].ActionURL != whatsapp_services.MessageNotificationURL(convs[0].ID) || ns[0].TenantID != f.own {
			t.Fatalf("user %d notifications = %+v", uid, ns)
		}
		var p map[string]any
		if err := json.Unmarshal(ns[0].Params, &p); err != nil || p["patient_name"] != "Luis Mora" || p["date"] != "2026-10-27" || p["time"] != "10:00" {
			t.Fatalf("params = %s", ns[0].Params)
		}
	}
	// Users without the inbox permission and other clinics are not told.
	if ns := f.notifications(101, whatsapp_services.NotificationTypeRescheduleRequest); len(ns) != 0 {
		t.Fatal("user without inbox permission notified")
	}
	if ns := f.notifications(110, whatsapp_services.NotificationTypeRescheduleRequest); len(ns) != 0 {
		t.Fatal("other clinic notified")
	}
}

func TestWebhook_RescheduleTenantIsolation(t *testing.T) {
	f := newFixture(t)
	f.grant(f.own, []uint{100}, "USE_WHATSAPP_INBOX")
	f.grant(f.other, []uint{110}, "USE_WHATSAPP_INBOX")
	f.seedNoShow(f.own, "PHONE-1")
	_, otherMsg := f.seedNoShow(f.other, "PHONE-2")

	// The other clinic's message id arriving on our number does nothing.
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("reschedule:%d", otherMsg.ID), "593991234567", ""))
	if m := f.message(otherMsg.ID); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("other tenant's message changed to %s", m.Status)
	}
	for _, uid := range []uint{100, 110} {
		if ns := f.notifications(uid, whatsapp_services.NotificationTypeRescheduleRequest); len(ns) != 0 {
			t.Fatalf("user %d notified: %+v", uid, ns)
		}
	}
	// A reschedule payload on a reminder (not a no-show message) does nothing either.
	_, reminder := f.seedReminder(f.own, "PHONE-1", "WABA-PHONE-1")
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("reschedule:%d", reminder.ID), "593991234567", reminder.WamID))
	if ns := f.notifications(100, whatsapp_services.NotificationTypeRescheduleRequest); len(ns) != 0 {
		t.Fatalf("reminder answered as reschedule: %+v", ns)
	}
}

func TestUpdateSettings_NoShowMessageSwitch(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	put := func(body map[string]any) whatsapp_dto.AccountResponse {
		t.Helper()
		resp, _ := f.call(f.h.UpdateSettings, f.own, http.MethodPut, body, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("code = %d (%v)", resp.Code, resp.Data)
		}
		return resp.Data.(whatsapp_dto.AccountResponse)
	}
	if f.account(f.own).NoShowMessageEnabled {
		t.Fatal("off by default")
	}
	if got := put(map[string]any{"reminders_enabled": true, "reminder_offsets": []int{24}, "no_show_message_enabled": true}); !got.NoShowMessageEnabled {
		t.Fatalf("response = %+v", got)
	}
	if !f.account(f.own).NoShowMessageEnabled {
		t.Fatal("not stored")
	}
	// Absent: unchanged.
	if got := put(map[string]any{"reminders_enabled": false, "reminder_offsets": []int{}}); !got.NoShowMessageEnabled {
		t.Fatalf("absent switch changed it: %+v", got)
	}
	put(map[string]any{"reminders_enabled": false, "reminder_offsets": []int{}, "no_show_message_enabled": false})
	if f.account(f.own).NoShowMessageEnabled {
		t.Fatal("not turned off")
	}
	if f.account(f.other).ID != 0 {
		t.Fatal("other clinic touched")
	}
}
