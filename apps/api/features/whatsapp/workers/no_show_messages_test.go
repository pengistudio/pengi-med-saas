package whatsapp_workers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"

	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
)

// noShowFixture is a schedFixture whose tenant turned the no-show message on
// and has the template approved on its WABA.
type noShowFixture struct {
	*schedFixture
	noShows *whatsapp_services.NoShowMessages
}

func newNoShowFixture(t *testing.T) *noShowFixture {
	t.Helper()
	f := &noShowFixture{schedFixture: newSchedFixture(t, []int{24})}
	if err := f.db.AutoMigrate(&whatsapp_models.WhatsAppTemplate{}); err != nil {
		t.Fatal(err)
	}
	f.enable(f.tenant, true, whatsapp_models.TemplateStatusApproved)
	f.noShows = whatsapp_services.NewNoShowMessages(f.db, zap.NewNop(), f.pub).WithClock(func() time.Time { return f.now }, f.loc)
	return f
}

// enable sets the tenant's no-show switch and its template's review status.
func (f *noShowFixture) enable(tenantID uint, on bool, templateStatus string) {
	f.t.Helper()
	if err := f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", tenantID).Update("no_show_message_enabled", on).Error; err != nil {
		f.t.Fatal(err)
	}
	if err := whatsapp_services.SetTemplateStatus(f.db, tenantID, whatsapp_models.TemplateNoShow, templateStatus, ""); err != nil {
		f.t.Fatal(err)
	}
}

// noShow creates an appointment that started 2 h ago, marked no_show.
func (f *noShowFixture) noShow(tenantID uint, phone string) clinical_models.Appointment {
	return f.appointment(tenantID, -2*time.Hour, clinical_models.AppointmentStatusNoShow, phone)
}

func (f *noShowFixture) noShowMessages() []whatsapp_models.WhatsAppMessage {
	var msgs []whatsapp_models.WhatsAppMessage
	f.db.Where("kind = ?", whatsapp_models.KindNoShow).Order("id").Find(&msgs)
	return msgs
}

func (f *noShowFixture) setStatus(appt clinical_models.Appointment, status string) {
	f.t.Helper()
	if err := f.db.Model(&clinical_models.Appointment{}).Where("id = ?", appt.ID).Update("status", status).Error; err != nil {
		f.t.Fatal(err)
	}
}

func TestNoShow_QueuesOnceAndPublishesInsideSendHours(t *testing.T) {
	f := newNoShowFixture(t) // 08:00 clinic time: inside the send hours
	appt := f.noShow(f.tenant, "0991234567")

	f.noShows.Marked(f.tenant, appt.ID)
	f.noShows.Marked(f.tenant, appt.ID) // marked again: still one

	msgs := f.noShowMessages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	m := msgs[0]
	if m.Status != whatsapp_models.MessageStatusQueued || m.TenantID != f.tenant || *m.AppointmentID != appt.ID ||
		m.Template != whatsapp_models.TemplateNoShow || m.SendAfter != nil || m.OffsetHours != whatsapp_models.NoShowOffsetHours {
		t.Fatalf("message = %+v", m)
	}
	if len(f.pub.ids) != 1 || f.pub.ids[0] != m.ID {
		t.Fatalf("published = %v", f.pub.ids)
	}

	// Sent with the patient's data and the button carrying the message id.
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.NS"}]}`}
	if err := f.sender(g).Send(m.ID); err != nil {
		t.Fatal(err)
	}
	sent := f.message(m.ID)
	if sent.Status != whatsapp_models.MessageStatusSent || sent.ConversationID == nil || !strings.Contains(sent.Body, "Ana Pérez") || !strings.Contains(sent.Body, "tu médico") {
		t.Fatalf("sent = %+v", sent)
	}
	if len(g.requests) != 1 || !strings.Contains(g.requests[0], `"name":"pengi_cita_no_asistio"`) ||
		!strings.Contains(g.requests[0], fmt.Sprintf(`"payload":"reschedule:%d"`, m.ID)) {
		t.Fatalf("requests = %v", g.requests)
	}
	// Counted toward the monthly cap.
	if usage, err := whatsapp_services.MonthlyUsage(f.db, f.tenant, f.now); err != nil || usage.Used != 1 {
		t.Fatalf("usage = %+v, %v", usage, err)
	}

	// Sent, then corrected and marked again: never a second one.
	f.setStatus(appt, "scheduled")
	f.noShows.Corrected(f.tenant, appt.ID)
	f.setStatus(appt, clinical_models.AppointmentStatusNoShow)
	f.noShows.Marked(f.tenant, appt.ID)
	if msgs := f.noShowMessages(); len(msgs) != 1 || msgs[0].Status != whatsapp_models.MessageStatusSent || len(f.pub.ids) != 1 {
		t.Fatalf("after re-mark: %+v, published %v", msgs, f.pub.ids)
	}
}

func TestNoShow_NotQueued(t *testing.T) {
	cases := map[string]func(f *noShowFixture, appt *clinical_models.Appointment){
		"setting off": func(f *noShowFixture, _ *clinical_models.Appointment) {
			f.enable(f.tenant, false, whatsapp_models.TemplateStatusApproved)
		},
		"template not approved": func(f *noShowFixture, _ *clinical_models.Appointment) {
			f.enable(f.tenant, true, whatsapp_models.TemplateStatusPending)
		},
		"no consent": func(f *noShowFixture, appt *clinical_models.Appointment) {
			f.db.Model(&clinical_models.Patient{}).Where("id = ?", appt.PatientID).Update("whatsapp_opt_in", false)
		},
		"no phone": func(f *noShowFixture, appt *clinical_models.Appointment) {
			f.db.Model(&clinical_models.Patient{}).Where("id = ?", appt.PatientID).Update("phone", "")
		},
		"account disconnected": func(f *noShowFixture, _ *clinical_models.Appointment) {
			f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.tenant).Update("status", whatsapp_models.AccountStatusTokenInvalid)
		},
		"not a no-show": func(f *noShowFixture, appt *clinical_models.Appointment) { f.setStatus(*appt, "scheduled") },
		"monthly limit reached": func(f *noShowFixture, _ *clinical_models.Appointment) {
			f.planLimit(0)
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := newNoShowFixture(t)
			appt := f.noShow(f.tenant, "0991234567")
			setup(f, &appt)
			f.noShows.Marked(f.tenant, appt.ID)
			if msgs := f.noShowMessages(); len(msgs) != 0 || len(f.pub.ids) != 0 {
				t.Fatalf("queued %+v, published %v", msgs, f.pub.ids)
			}
		})
	}
}

// planLimit gives the tenant a plan capping template messages at limit.
func (f *noShowFixture) planLimit(limit int) {
	f.t.Helper()
	code := fmt.Sprintf("P-%d", time.Now().UnixNano())
	plan := company_models.Plan{Name: code, Code: code, Properties: datatypes.JSONMap{whatsapp_services.PlanLimitKey: limit}}
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: code, TenantID: f.tenant}
	for _, row := range []any{&plan, &company} {
		if err := f.db.Omit(clause.Associations).Create(row).Error; err != nil {
			f.t.Fatal(err)
		}
	}
	sub := company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().AddDate(1, 0, 0), CompanyID: company.ID}
	if err := f.db.Omit(clause.Associations).Create(&sub).Error; err != nil {
		f.t.Fatal(err)
	}
}

func TestNoShow_DeferredOutsideSendHours(t *testing.T) {
	f := newNoShowFixture(t)
	f.now = time.Date(2026, 10, 5, 21, 30, 0, 0, f.loc) // after the send hours
	f.sched.noShows = f.noShows
	appt := f.noShow(f.tenant, "0991234567")

	f.noShows.Marked(f.tenant, appt.ID)
	msgs := f.noShowMessages()
	if len(msgs) != 1 || msgs[0].SendAfter == nil || len(f.pub.ids) != 0 {
		t.Fatalf("messages = %+v, published %v", msgs, f.pub.ids)
	}
	want := time.Date(2026, 10, 6, whatsapp_services.SendHourStart, 0, 0, 0, f.loc)
	if !msgs[0].SendAfter.Equal(want) {
		t.Fatalf("send_after = %v, want %v", msgs[0].SendAfter, want)
	}
	id := msgs[0].ID

	// A task arriving early sends nothing and leaves it queued.
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.D"}]}`}
	if err := f.sender(g).Send(id); err != nil || len(g.requests) != 0 || f.message(id).Status != whatsapp_models.MessageStatusQueued {
		t.Fatalf("early send: err=%v requests=%d status=%s", err, len(g.requests), f.message(id).Status)
	}
	// Still night: the tick does not publish it.
	f.now = time.Date(2026, 10, 6, 7, 59, 0, 0, f.loc)
	f.sched.Tick()
	if len(f.pub.ids) != 0 {
		t.Fatalf("published before the send hours: %v", f.pub.ids)
	}
	// Next morning: published once, then sent.
	f.now = want
	f.sched.Tick()
	f.sched.Tick()
	if len(f.pub.ids) != 1 || f.pub.ids[0] != id {
		t.Fatalf("published = %v", f.pub.ids)
	}
	if err := f.sender(g).Send(id); err != nil || f.message(id).Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("send: err=%v message=%+v", err, f.message(id))
	}
}

func TestNoShow_CorrectedBeforeSendingIsWithdrawn(t *testing.T) {
	f := newNoShowFixture(t)
	f.now = time.Date(2026, 10, 5, 22, 0, 0, 0, f.loc) // deferred to the morning
	appt := f.noShow(f.tenant, "0991234567")
	f.noShows.Marked(f.tenant, appt.ID)

	f.setStatus(appt, "arrived")
	f.noShows.Corrected(f.tenant, appt.ID)
	msgs := f.noShowMessages()
	if len(msgs) != 1 || msgs[0].Status != whatsapp_models.MessageStatusSkipped || msgs[0].ErrorCode != whatsapp_models.ErrorCodeNoShowCorrected {
		t.Fatalf("after correction: %+v", msgs)
	}
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.C"}]}`}
	f.now = time.Date(2026, 10, 6, 9, 0, 0, 0, f.loc)
	if err := f.sender(g).Send(msgs[0].ID); err != nil || len(g.requests) != 0 {
		t.Fatalf("withdrawn message sent: err=%v requests=%d", err, len(g.requests))
	}

	// Marked again (never sent): the same row is queued again, now inside
	// the send hours, so published right away.
	f.setStatus(appt, clinical_models.AppointmentStatusNoShow)
	f.noShows.Marked(f.tenant, appt.ID)
	msgs = f.noShowMessages()
	if len(msgs) != 1 || msgs[0].Status != whatsapp_models.MessageStatusQueued || msgs[0].SendAfter != nil || msgs[0].ErrorCode != "" {
		t.Fatalf("after re-mark: %+v", msgs)
	}
	if len(f.pub.ids) != 1 || f.pub.ids[0] != msgs[0].ID {
		t.Fatalf("published = %v", f.pub.ids)
	}
}

func TestNoShow_SenderSkipsWhenCorrectedWithoutWithdrawal(t *testing.T) {
	f := newNoShowFixture(t)
	appt := f.noShow(f.tenant, "0991234567")
	f.noShows.Marked(f.tenant, appt.ID)
	id := f.noShowMessages()[0].ID

	f.setStatus(appt, "completed")
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.X"}]}`}
	if err := f.sender(g).Send(id); err != nil || len(g.requests) != 0 {
		t.Fatalf("err=%v requests=%d", err, len(g.requests))
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusSkipped || m.ErrorCode != whatsapp_models.ErrorCodeNoShowCorrected {
		t.Fatalf("message = %+v", m)
	}
}

func TestNoShow_SenderSkipsWhenTurnedOffOrLimitReachedAfterQueueing(t *testing.T) {
	f := newNoShowFixture(t)
	off := f.noShow(f.tenant, "0991234567")
	f.noShows.Marked(f.tenant, off.ID)
	limited := f.noShow(f.tenant, "0991234568")
	f.noShows.Marked(f.tenant, limited.ID)
	msgs := f.noShowMessages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d", len(msgs))
	}

	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.Y"}]}`}
	f.planLimit(0)
	if err := f.sender(g).Send(msgs[1].ID); err != nil {
		t.Fatal(err)
	}
	if m := f.message(msgs[1].ID); m.Status != whatsapp_models.MessageStatusSkipped || m.ErrorCode != whatsapp_models.ErrorCodeMonthlyLimit {
		t.Fatalf("limited = %+v", m)
	}
	f.enable(f.tenant, false, whatsapp_models.TemplateStatusApproved)
	if err := f.sender(g).Send(msgs[0].ID); err != nil {
		t.Fatal(err)
	}
	if m := f.message(msgs[0].ID); m.Status != whatsapp_models.MessageStatusSkipped || m.ErrorCode != whatsapp_models.ErrorCodeNoShowDisabled {
		t.Fatalf("off = %+v", m)
	}
	if len(g.requests) != 0 {
		t.Fatalf("requests = %d", len(g.requests))
	}
}

func TestNoShow_TenantIsolation(t *testing.T) {
	f := newNoShowFixture(t)
	other := f.newTenant("B", []int{24}, whatsapp_models.TemplateStatusApproved)
	f.enable(other, true, whatsapp_models.TemplateStatusApproved)
	appt := f.noShow(f.tenant, "0991234567")

	// Another clinic's id: its handle never sees the appointment.
	f.noShows.Marked(other, appt.ID)
	if msgs := f.noShowMessages(); len(msgs) != 0 {
		t.Fatalf("queued across tenants: %+v", msgs)
	}
	f.noShows.Marked(f.tenant, appt.ID)
	f.noShows.Corrected(other, appt.ID) // another clinic can't withdraw it
	msgs := f.noShowMessages()
	if len(msgs) != 1 || msgs[0].TenantID != f.tenant || msgs[0].Status != whatsapp_models.MessageStatusQueued {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestNextSendTime(t *testing.T) {
	loc := time.FixedZone("ECT", -5*3600)
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, loc) }
	for _, c := range []struct{ now, want time.Time }{
		{at(5, 7, 59), at(5, 8, 0)},
		{at(5, 8, 0), at(5, 8, 0)},
		{at(5, 19, 59), at(5, 19, 59)},
		{at(5, 20, 0), at(6, 8, 0)},
		{at(5, 23, 30), at(6, 8, 0)},
		{at(6, 0, 10), at(6, 8, 0)},
	} {
		if got := whatsapp_services.NextSendTime(c.now, loc); !got.Equal(c.want) {
			t.Errorf("NextSendTime(%v) = %v, want %v", c.now, got, c.want)
		}
	}
}
