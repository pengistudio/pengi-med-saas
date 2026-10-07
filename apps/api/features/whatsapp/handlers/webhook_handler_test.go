package whatsapp_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// seedReminder gives tenantID a connected account on phoneID and a sent
// reminder for a new scheduled appointment.
func (f *fixture) seedReminder(tenantID uint, phoneID, wabaID string) (clinical_models.Appointment, whatsapp_models.WhatsAppMessage) {
	f.t.Helper()
	if f.account(tenantID).ID == 0 {
		acc := whatsapp_models.WhatsAppAccount{TenantID: tenantID, WabaID: wabaID, PhoneNumberID: phoneID,
			Status: whatsapp_models.AccountStatusConnected, TemplateStatus: "PENDING", ReminderOffsets: []int{24}}
		if err := f.db.Create(&acc).Error; err != nil {
			f.t.Fatal(err)
		}
	}
	patient := clinical_models.Patient{TenantID: tenantID, FirstName: "Luis", LastName: "Mora", Phone: "0991234567", WhatsAppOptIn: true, Document: fmt.Sprintf("D%d", time.Now().UnixNano())}
	if err := f.db.Create(&patient).Error; err != nil {
		f.t.Fatal(err)
	}
	appt := clinical_models.Appointment{TenantID: tenantID, PatientID: patient.ID, Title: "Control",
		Date: time.Now().Add(24 * time.Hour), StartTime: "10:00", EndTime: "10:30", Status: "scheduled"}
	if err := f.db.Create(&appt).Error; err != nil {
		f.t.Fatal(err)
	}
	apptID, patientID := appt.ID, patient.ID
	msg := whatsapp_models.WhatsAppMessage{TenantID: tenantID, Kind: whatsapp_models.KindReminder, AppointmentID: &apptID, PatientID: &patientID,
		OffsetHours: 24, Template: whatsapp_models.ReminderTemplate, Status: whatsapp_models.MessageStatusSent,
		WamID: fmt.Sprintf("wamid.%d", time.Now().UnixNano()), ToPhone: "593991234567"}
	if err := f.db.Create(&msg).Error; err != nil {
		f.t.Fatal(err)
	}
	return appt, msg
}

func (f *fixture) post(payload any, signature func([]byte) string) int {
	raw, _ := json.Marshal(payload)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", bytes.NewReader(raw))
	c.Request.Header.Set(whatsapp.SignatureHeader, signature(raw))
	return f.webhook.Receive(c).Code
}

func (f *fixture) signed(payload any) int {
	return f.post(payload, func(b []byte) string { return whatsapp.SignatureFor(testAppSecret, b) })
}

var wamSeq atomic.Int64

// nextWamID is a fresh inbound wamid: inbound messages are deduplicated by it.
func nextWamID() string { return fmt.Sprintf("wamid.in.%d", wamSeq.Add(1)) }

func messagesEvent(phoneID string, value map[string]any) map[string]any {
	value["messaging_product"] = "whatsapp"
	value["metadata"] = map[string]string{"phone_number_id": phoneID, "display_phone_number": "15550100"}
	return map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{
		"id": "WABA", "changes": []any{map[string]any{"field": "messages", "value": value}},
	}}}
}

func buttonEvent(phoneID, payload string) map[string]any {
	return messagesEvent(phoneID, map[string]any{"messages": []any{map[string]any{
		"from": "593991234567", "id": nextWamID(), "timestamp": "1760000000", "type": "button",
		"button": map[string]string{"payload": payload, "text": "Confirmar"},
	}}})
}

func (f *fixture) appointmentStatus(id uint) string {
	var a clinical_models.Appointment
	f.db.Unscoped().First(&a, id)
	return a.Status
}

func (f *fixture) message(id uint) whatsapp_models.WhatsAppMessage {
	var m whatsapp_models.WhatsAppMessage
	f.db.First(&m, id)
	return m
}

func TestWebhook_RejectsBadSignature(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	event := buttonEvent("PHONE-OWN", fmt.Sprintf("confirm:%d", msg.ID))

	if code := f.post(event, func([]byte) string { return "sha256=00" }); code != http.StatusUnauthorized {
		t.Fatalf("bad signature: %d", code)
	}
	if code := f.post(event, func(b []byte) string { return whatsapp.SignatureFor("other-secret", b) }); code != http.StatusUnauthorized {
		t.Fatalf("wrong secret: %d", code)
	}
	if f.appointmentStatus(appt.ID) != "scheduled" {
		t.Fatal("an unsigned event must change nothing")
	}
}

func TestWebhook_ConfirmButton(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	if code := f.signed(buttonEvent("PHONE-OWN", fmt.Sprintf("confirm:%d", msg.ID))); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if s := f.appointmentStatus(appt.ID); s != "confirmed" {
		t.Fatalf("appointment status = %s", s)
	}
	m := f.message(msg.ID)
	if m.Status != whatsapp_models.MessageStatusReplied || m.Reply != "confirm" || m.RepliedAt == nil {
		t.Fatalf("message = %+v", m)
	}
	var notes []notifications_models.Notification
	f.db.Where("tenant_id = ?", f.own).Find(&notes)
	if len(notes) != 2 || notes[0].Type != NotificationTypeConfirmed || notes[0].ResourceID != appt.ID ||
		notes[0].MessageKey != "notification.whatsapp.appointment.confirmed" {
		t.Fatalf("notifications = %+v", notes)
	}

	// A duplicate delivery changes nothing and notifies no one again.
	f.signed(buttonEvent("PHONE-OWN", fmt.Sprintf("confirm:%d", msg.ID)))
	var count int64
	f.db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 2 {
		t.Fatalf("notifications after duplicate = %d", count)
	}

	time.Sleep(20 * time.Millisecond) // calendar sync runs in a goroutine
	f.calendar.mu.Lock()
	defer f.calendar.mu.Unlock()
	if len(f.calendar.calls) != 1 || f.calendar.calls[0] != fmt.Sprintf("%d:%d:confirmed", f.own, appt.ID) {
		t.Fatalf("calendar calls = %v", f.calendar.calls)
	}
}

func TestWebhook_CancelButton(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.db.Model(&appt).Update("status", "confirmed") // cancelling after confirming still works

	f.signed(buttonEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", msg.ID)))
	if s := f.appointmentStatus(appt.ID); s != "cancelled" {
		t.Fatalf("appointment status = %s", s)
	}
	var note notifications_models.Notification
	f.db.Where("tenant_id = ?", f.own).First(&note)
	if note.Type != NotificationTypeCancelled {
		t.Fatalf("notification = %+v", note)
	}
}

func TestWebhook_ButtonLeavesLaterStatusesAlone(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.db.Model(&appt).Update("status", "arrived")

	f.signed(buttonEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", msg.ID)))
	if s := f.appointmentStatus(appt.ID); s != "arrived" {
		t.Fatalf("appointment status = %s, want arrived", s)
	}
	var count int64
	f.db.Model(&notifications_models.Notification{}).Count(&count)
	if count != 0 {
		t.Fatal("no change, no notification")
	}
}

func TestWebhook_TenantIsolation(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	otherAppt, otherMsg := f.seedReminder(f.other, "PHONE-OTHER", "WABA-OTHER")

	// The other clinic's message id arriving on our number must not touch it.
	f.signed(buttonEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", otherMsg.ID)))
	if s := f.appointmentStatus(otherAppt.ID); s != "scheduled" {
		t.Fatalf("other tenant's appointment changed to %s", s)
	}
	if m := f.message(otherMsg.ID); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("other tenant's message changed to %s", m.Status)
	}
	// Unknown number: ignored, still 200.
	if code := f.signed(buttonEvent("PHONE-NOBODY", fmt.Sprintf("cancel:%d", otherMsg.ID))); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if s := f.appointmentStatus(otherAppt.ID); s != "scheduled" {
		t.Fatalf("other tenant's appointment changed to %s", s)
	}
	// Garbage payloads are ignored.
	for _, p := range []string{"confirm:", "delete:1", "confirm:abc", "nonsense"} {
		if code := f.signed(buttonEvent("PHONE-OTHER", p)); code != http.StatusOK {
			t.Fatalf("payload %q: %d", p, code)
		}
	}
}

func TestWebhook_StatusUpdates(t *testing.T) {
	f := newFixture(t)
	_, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	status := func(s string, errs ...map[string]any) {
		st := map[string]any{"id": msg.WamID, "status": s, "timestamp": "1760000000", "recipient_id": "593991234567"}
		if len(errs) > 0 {
			st["errors"] = errs
		}
		f.signed(messagesEvent("PHONE-OWN", map[string]any{"statuses": []any{st}}))
	}

	status("read")
	m := f.message(msg.ID)
	if m.Status != "read" || m.ReadAt == nil || m.DeliveredAt == nil {
		t.Fatalf("after read: %+v", m)
	}
	status("delivered") // late, must not move it back
	if m := f.message(msg.ID); m.Status != "read" {
		t.Fatalf("status went back to %s", m.Status)
	}

	_, msg2 := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	msg = msg2
	status("failed", map[string]any{"code": 131026, "title": "Message undeliverable", "error_data": map[string]string{"details": "not on whatsapp"}})
	if m := f.message(msg.ID); m.Status != "failed" || m.ErrorCode != "131026" || m.ErrorDetail == "" {
		t.Fatalf("after failed: %+v", m)
	}
}

func TestWebhook_TemplateStatusUpdate(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.seedReminder(f.other, "PHONE-OTHER", "WABA-OTHER")
	event := func(name, status string) map[string]any {
		return map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{
			"id": "WABA-OWN", "changes": []any{map[string]any{"field": "message_template_status_update", "value": map[string]any{
				"event": status, "message_template_id": 123, "message_template_name": name, "message_template_language": "es", "reason": "NONE",
			}}},
		}}}
	}
	f.signed(event("some_other_template", "REJECTED"))
	if s := f.account(f.own).TemplateStatus; s != "PENDING" {
		t.Fatalf("other template changed ours: %s", s)
	}
	f.signed(event(whatsapp_models.ReminderTemplate, "APPROVED"))
	if s := f.account(f.own).TemplateStatus; s != "APPROVED" {
		t.Fatalf("template status = %s", s)
	}
	if s := f.account(f.other).TemplateStatus; s != "PENDING" {
		t.Fatalf("other WABA changed: %s", s)
	}
}

func TestWebhook_VerifyHandshake(t *testing.T) {
	f := newFixture(t)
	verify := func(query string) *httptest.ResponseRecorder {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/webhooks/whatsapp?"+query, nil)
		f.webhook.Verify(c)
		return w
	}
	if w := verify("hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=12345"); w.Code != http.StatusOK || w.Body.String() != "12345" {
		t.Fatalf("handshake: %d %q", w.Code, w.Body.String())
	}
	if w := verify("hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=12345"); w.Code != http.StatusForbidden {
		t.Fatalf("wrong token: %d", w.Code)
	}
}

// replyEvent is a button tap from `from`, optionally quoting message contextID.
func replyEvent(phoneID, payload, from, contextID string) map[string]any {
	m := map[string]any{
		"from": from, "id": nextWamID(), "timestamp": "1760000000", "type": "button",
		"button": map[string]string{"payload": payload, "text": "Cancelar"},
	}
	if contextID != "" {
		m["context"] = map[string]string{"id": contextID, "from": "15550100"}
	}
	return messagesEvent(phoneID, map[string]any{"messages": []any{m}})
}

func TestWebhook_ReplyMustComeFromRecipient(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	f.signed(replyEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", msg.ID), "593987654321", msg.WamID))
	if s := f.appointmentStatus(appt.ID); s != "scheduled" {
		t.Fatalf("reply from another number changed the appointment to %s", s)
	}
	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("message marked %s", m.Status)
	}

	// The recipient quoting the right message is accepted.
	f.signed(replyEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", msg.ID), "593991234567", msg.WamID))
	if s := f.appointmentStatus(appt.ID); s != "cancelled" {
		t.Fatalf("appointment status = %s, want cancelled", s)
	}
}

func TestWebhook_ReplyMustQuoteTheMessage(t *testing.T) {
	f := newFixture(t)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	f.signed(replyEvent("PHONE-OWN", fmt.Sprintf("confirm:%d", msg.ID), "593991234567", "wamid.SOMETHING-ELSE"))
	if s := f.appointmentStatus(appt.ID); s != "scheduled" {
		t.Fatalf("reply quoting another message changed the appointment to %s", s)
	}
	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("message marked %s", m.Status)
	}
}

func textEvent(phoneID, from, body string) map[string]any {
	return messagesEvent(phoneID, map[string]any{"messages": []any{map[string]any{
		"from": from, "id": nextWamID(), "timestamp": "1760000000", "type": "text",
		"text": map[string]string{"body": body},
	}}})
}

func (f *fixture) optedIn(patientID uint) bool {
	var p clinical_models.Patient
	f.db.Where("id = ?", patientID).First(&p)
	return p.WhatsAppOptIn
}

func TestWebhook_StopWithdrawsConsent(t *testing.T) {
	f := newFixture(t)
	appt, _ := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	otherAppt, _ := f.seedReminder(f.other, "PHONE-OTHER", "WABA-OTHER") // same phone, other clinic

	// Ordinary text is not an opt-out.
	f.signed(textEvent("PHONE-OWN", "593991234567", "hola, gracias"))
	if !f.optedIn(appt.PatientID) {
		t.Fatal("plain text withdrew consent")
	}
	// Another sender's STOP doesn't touch this patient.
	f.signed(textEvent("PHONE-OWN", "593990000000", "STOP"))
	if !f.optedIn(appt.PatientID) {
		t.Fatal("STOP from another number withdrew consent")
	}

	if code := f.signed(textEvent("PHONE-OWN", "593991234567", " baja. ")); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if f.optedIn(appt.PatientID) {
		t.Fatal("BAJA did not withdraw consent")
	}
	// Only the clinic whose number received it.
	if !f.optedIn(otherAppt.PatientID) {
		t.Fatal("opt-out leaked to another tenant")
	}
}
