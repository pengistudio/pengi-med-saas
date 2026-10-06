package whatsapp_workers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
	"pengi-med-saas/testutils"
)

type fakePublisher struct {
	mu  sync.Mutex
	ids []uint
}

func (p *fakePublisher) Publish(queue string, body []byte) error {
	var task whatsapp_services.SendTask
	if err := json.Unmarshal(body, &task); err != nil || queue != whatsapp_services.SendQueue {
		return fmt.Errorf("bad publish %s %s", queue, body)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, task.MessageID)
	return nil
}

type schedFixture struct {
	t      *testing.T
	db     *gorm.DB
	loc    *time.Location
	now    time.Time
	pub    *fakePublisher
	sched  *ReminderScheduler
	tenant uint
}

func newSchedFixture(t *testing.T, offsets []int) *schedFixture {
	t.Helper()
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppMessage{}, &whatsapp_models.WhatsAppConversation{},
		&company_models.Company{}, &company_models.Feature{}, &company_models.Plan{}, &company_models.Subscription{},
		&notifications_models.Notification{}, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{})
	f := &schedFixture{t: t, db: db, loc: loc, pub: &fakePublisher{},
		// 2026-10-05 08:00 in Guayaquil
		now: time.Date(2026, 10, 5, 13, 0, 0, 0, time.UTC)}
	f.tenant = f.newTenant("A", offsets, whatsapp_models.TemplateStatusApproved)
	f.sched = NewReminderScheduler(db, zap.NewNop(), f.pub).WithClock(func() time.Time { return f.now }, loc)
	return f
}

func (f *schedFixture) newTenant(tag string, offsets []int, templateStatus string) uint {
	slug := fmt.Sprintf("wa-%s-%d", tag, time.Now().UnixNano())
	tenant := tenant_models.Tenant{Name: "Clinic " + tag, Slug: slug, DisplayToken: "tok-" + slug}
	if err := f.db.Create(&tenant).Error; err != nil {
		f.t.Fatal(err)
	}
	account := whatsapp_models.WhatsAppAccount{
		TenantID: tenant.ID, WabaID: "WABA-" + slug, PhoneNumberID: "PHONE-" + slug, Mode: whatsapp_models.ModeManual,
		Status: whatsapp_models.AccountStatusConnected, TemplateStatus: templateStatus,
		RemindersEnabled: true, ReminderOffsets: offsets,
	}
	box, _ := whatsapp_models.Box()
	_ = account.SealAccessToken(box, "token-"+tag)
	if err := f.db.Create(&account).Error; err != nil {
		f.t.Fatal(err)
	}
	return tenant.ID
}

// appointment creates an appointment starting `in` from now (clinic time).
func (f *schedFixture) appointment(tenantID uint, in time.Duration, status, phone string) clinical_models.Appointment {
	f.t.Helper()
	patient := clinical_models.Patient{TenantID: tenantID, FirstName: "Ana", LastName: "Pérez", Phone: phone, WhatsAppOptIn: true, Document: fmt.Sprintf("D%d", time.Now().UnixNano())}
	if err := f.db.Create(&patient).Error; err != nil {
		f.t.Fatal(err)
	}
	start := f.now.Add(in).In(f.loc)
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, f.loc).UTC() // what the frontend sends
	appt := clinical_models.Appointment{TenantID: tenantID, PatientID: patient.ID, Title: "Control", Date: day,
		StartTime: start.Format("15:04"), EndTime: start.Add(30 * time.Minute).Format("15:04"), Status: status}
	if err := f.db.Create(&appt).Error; err != nil {
		f.t.Fatal(err)
	}
	return appt
}

func (f *schedFixture) messages() []whatsapp_models.WhatsAppMessage {
	var msgs []whatsapp_models.WhatsAppMessage
	f.db.Order("id").Find(&msgs)
	return msgs
}

func TestTick_QueuesDueReminderOnce(t *testing.T) {
	f := newSchedFixture(t, []int{24, 2})
	due := f.appointment(f.tenant, 20*time.Hour, "scheduled", "0991234567")
	f.appointment(f.tenant, 30*time.Hour, "scheduled", "0991234567")    // not due yet
	f.appointment(f.tenant, 3*time.Hour, "cancelled", "0991234567")     // cancelled
	f.appointment(f.tenant, -1*time.Hour, "scheduled", "0991234567")    // already started
	confirmed := f.appointment(f.tenant, 10*time.Hour, "confirmed", "") // confirmed still gets reminders

	f.sched.Tick()
	f.sched.Tick() // idempotent across ticks

	msgs := f.messages()
	if len(msgs) != 2 {
		t.Fatalf("queued %d messages, want 2: %+v", len(msgs), msgs)
	}
	got := map[uint]int{}
	for _, m := range msgs {
		if m.Status != whatsapp_models.MessageStatusQueued || m.TenantID != f.tenant || m.Kind != whatsapp_models.KindReminder {
			t.Fatalf("unexpected message %+v", m)
		}
		got[*m.AppointmentID] = m.OffsetHours
	}
	if got[due.ID] != 24 || got[confirmed.ID] != 24 {
		t.Fatalf("offsets = %v", got)
	}
	if len(f.pub.ids) != 2 {
		t.Fatalf("published %v, want each message once", f.pub.ids)
	}

	// 19 hours later the 2h window opens for the first appointment.
	f.now = f.now.Add(19 * time.Hour)
	f.sched.Tick()
	var second whatsapp_models.WhatsAppMessage
	f.db.Where("appointment_id = ? AND offset_hours = ?", due.ID, 2).First(&second)
	if second.ID == 0 {
		t.Fatal("2h reminder not queued")
	}
}

func TestTick_LateCreatedAppointmentGetsOnlySmallestOffset(t *testing.T) {
	f := newSchedFixture(t, []int{24, 2})
	appt := f.appointment(f.tenant, 90*time.Minute, "scheduled", "0991234567")

	f.sched.Tick()
	f.now = f.now.Add(30 * time.Minute)
	f.sched.Tick()

	msgs := f.messages()
	if len(msgs) != 1 || msgs[0].OffsetHours != 2 || *msgs[0].AppointmentID != appt.ID {
		t.Fatalf("messages = %+v, want only the 2h reminder", msgs)
	}
}

func TestTick_TimeZoneBoundary(t *testing.T) {
	// 2026-10-05 20:00 Guayaquil = 2026-10-06 01:00 UTC. An appointment at
	// 07:00 local on Oct 6 is 11h away and inside a 12h window; computed in
	// UTC (Oct 6 07:00 UTC) it would be only 6h away — both due, so use a 8h
	// offset to tell them apart: due in UTC (6h ≤ 8h), not due in Guayaquil (11h > 8h).
	f := newSchedFixture(t, []int{8})
	f.now = time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC)
	f.appointment(f.tenant, 11*time.Hour, "scheduled", "0991234567")

	f.sched.Tick()
	if n := len(f.messages()); n != 0 {
		t.Fatalf("queued %d messages; the appointment is 11h away in clinic time", n)
	}
	f.now = f.now.Add(3*time.Hour + time.Minute)
	f.sched.Tick()
	if n := len(f.messages()); n != 1 {
		t.Fatalf("queued %d messages after entering the 8h window, want 1", n)
	}
}

func TestTick_SkipsAccountsThatCannotSend(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	pending := f.newTenant("P", []int{24}, whatsapp_models.TemplateStatusPending)
	f.appointment(pending, time.Hour, "scheduled", "0991234567")
	disabled := f.newTenant("D", []int{24}, whatsapp_models.TemplateStatusApproved)
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", disabled).Update("reminders_enabled", false)
	f.appointment(disabled, time.Hour, "scheduled", "0991234567")

	f.sched.Tick()
	if n := len(f.messages()); n != 0 {
		t.Fatalf("queued %d messages for tenants that cannot send", n)
	}
}

func TestQueueReminder_UniqueIndexIsIdempotent(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	appt := f.appointment(f.tenant, time.Hour, "scheduled", "0991234567")
	db := f.db

	id1, created1, err := QueueReminder(db, appt, 24)
	if err != nil || !created1 || id1 == 0 {
		t.Fatalf("first insert: %d %v %v", id1, created1, err)
	}
	_, created2, err := QueueReminder(db, appt, 24)
	if err != nil || created2 {
		t.Fatalf("second insert must be a no-op: created=%v err=%v", created2, err)
	}
	if _, created3, err := QueueReminder(db, appt, 2); err != nil || !created3 {
		t.Fatalf("another offset is another reminder: %v %v", created3, err)
	}
	var count int64
	db.Model(&whatsapp_models.WhatsAppMessage{}).Where("appointment_id = ?", appt.ID).Count(&count)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

// --- Sender (consumer) ---

type graphStub struct {
	mu       sync.Mutex
	status   int
	body     string
	requests []string
}

func (g *graphStub) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		g.mu.Lock()
		g.requests = append(g.requests, r.URL.Path+" "+r.Header.Get("Authorization")+" "+string(raw))
		status, body := g.status, g.body
		g.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *schedFixture) sender(g *graphStub) *whatsapp_services.Sender {
	client := whatsapp.New(g.server(f.t).URL, "v23.0", "", "")
	return whatsapp_services.NewSender(f.db, zap.NewNop(), client).WithClock(func() time.Time { return f.now }, f.loc)
}

func (f *schedFixture) queued(appt clinical_models.Appointment) uint {
	id, _, err := QueueReminder(f.db, appt, 24)
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *schedFixture) message(id uint) whatsapp_models.WhatsAppMessage {
	var m whatsapp_models.WhatsAppMessage
	f.db.First(&m, id)
	return m
}

func TestSender_SendsAndRecordsWamID(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.OK"}]}`}
	id := f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "099 123 4567"))

	if err := f.sender(g).Send(id); err != nil {
		t.Fatal(err)
	}
	m := f.message(id)
	if m.Status != whatsapp_models.MessageStatusSent || m.WamID != "wamid.OK" || m.ToPhone != "593991234567" || m.SentAt == nil {
		t.Fatalf("message = %+v", m)
	}
	if len(g.requests) != 1 || !strings.Contains(g.requests[0], "Bearer token-A") || !strings.Contains(g.requests[0], fmt.Sprintf(`"payload":"confirm:%d"`, id)) {
		t.Fatalf("requests = %v", g.requests)
	}
	// A duplicate task does nothing.
	if err := f.sender(g).Send(id); err != nil || len(g.requests) != 1 {
		t.Fatalf("duplicate send: err=%v requests=%d", err, len(g.requests))
	}
}

func TestSender_FailuresAndSkips(t *testing.T) {
	f := newSchedFixture(t, []int{24})

	// 4xx from Meta: failed with Meta's code, not retried.
	g := &graphStub{status: 400, body: `{"error":{"code":131026,"message":"undeliverable"}}`}
	id := f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567"))
	if err := f.sender(g).Send(id); err == nil || whatsapp_services.IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusFailed || m.ErrorCode != "131026" {
		t.Fatalf("message = %+v", m)
	}

	// 5xx: back to queued, retryable.
	g = &graphStub{status: 503, body: `down`}
	id = f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567"))
	if err := f.sender(g).Send(id); err == nil || !whatsapp_services.IsRetryable(err) {
		t.Fatalf("err = %v, want retryable", err)
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusQueued {
		t.Fatalf("status = %s, want queued", m.Status)
	}

	// Invalid phone: failed with our own code, Meta never called.
	g = &graphStub{status: 200, body: `{"messages":[{"id":"x"}]}`}
	id = f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "12345"))
	if err := f.sender(g).Send(id); err != nil {
		t.Fatal(err)
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusFailed || m.ErrorCode != whatsapp_models.ErrorCodeInvalidPhone || len(g.requests) != 0 {
		t.Fatalf("message = %+v, requests = %d", m, len(g.requests))
	}

	// Cancelled after queueing: skipped.
	appt := f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567")
	id = f.queued(appt)
	f.db.Model(&appt).Update("status", "cancelled")
	if err := f.sender(g).Send(id); err != nil {
		t.Fatal(err)
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusSkipped || len(g.requests) != 0 {
		t.Fatalf("message = %+v", m)
	}

	// Token rejected: failed and the account is flagged.
	g = &graphStub{status: 401, body: `{"error":{"code":190,"message":"expired"}}`}
	id = f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567"))
	_ = f.sender(g).Send(id)
	var account whatsapp_models.WhatsAppAccount
	f.db.Where("tenant_id = ?", f.tenant).First(&account)
	if account.Status != whatsapp_models.AccountStatusTokenInvalid {
		t.Fatalf("account status = %s", account.Status)
	}
}

func TestBox_SealedTokenIsNotPlaintext(t *testing.T) {
	box, err := whatsapp_models.Box() // the cipher OpenAccessToken uses
	if err != nil {
		t.Fatal(err)
	}
	var a whatsapp_models.WhatsAppAccount
	if err := a.SealAccessToken(box, "EAAG-secret"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(a.AccessTokenEncrypted, "EAAG") {
		t.Fatal("token stored in plaintext")
	}
	if got, err := a.OpenAccessToken(); err != nil || got != "EAAG-secret" {
		t.Fatalf("open = %q, %v", got, err)
	}
}

func TestTick_RecoversMessagesStuckInSending(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	stuckID := f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567"))
	freshID := f.queued(f.appointment(f.tenant, 6*time.Hour, "scheduled", "0991234567"))
	f.db.Model(&whatsapp_models.WhatsAppMessage{}).Where("id = ?", stuckID).
		Updates(map[string]any{"status": whatsapp_models.MessageStatusSending, "updated_at": f.now.Add(-time.Hour)})
	f.db.Model(&whatsapp_models.WhatsAppMessage{}).Where("id = ?", freshID).
		Updates(map[string]any{"status": whatsapp_models.MessageStatusSending, "updated_at": f.now.Add(-time.Minute)})
	test := whatsapp_models.WhatsAppMessage{TenantID: f.tenant, Kind: whatsapp_models.KindTest, Status: whatsapp_models.MessageStatusSending, ToPhone: "593991234567"}
	f.db.Create(&test)
	f.db.Model(&test).Update("updated_at", f.now.Add(-time.Hour))

	f.sched.Tick()

	if m := f.message(stuckID); m.Status != whatsapp_models.MessageStatusQueued {
		t.Fatalf("stuck reminder status = %s, want queued", m.Status)
	}
	if m := f.message(freshID); m.Status != whatsapp_models.MessageStatusSending {
		t.Fatalf("a recent claim must be left alone, got %s", m.Status)
	}
	if m := f.message(test.ID); m.Status != whatsapp_models.MessageStatusFailed {
		t.Fatalf("stuck test message status = %s, want failed", m.Status)
	}
	republished := false
	for _, id := range f.pub.ids {
		if id == stuckID {
			republished = true
		}
		if id == freshID {
			t.Fatal("a recent claim must not be republished")
		}
	}
	if !republished {
		t.Fatalf("recovered reminder not republished: %v", f.pub.ids)
	}

	// The recovered message is then sent normally (once).
	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.R"}]}`}
	if err := f.sender(g).Send(stuckID); err != nil {
		t.Fatal(err)
	}
	if m := f.message(stuckID); m.Status != whatsapp_models.MessageStatusSent || len(g.requests) != 1 {
		t.Fatalf("message = %+v, requests = %d", m, len(g.requests))
	}
}

func TestTick_SkipsPatientsWithoutOptIn(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	optedIn := f.appointment(f.tenant, 20*time.Hour, "scheduled", "0991234567")
	noConsent := f.appointment(f.tenant, 20*time.Hour, "scheduled", "0991234568")
	f.db.Model(&clinical_models.Patient{}).Where("id = ?", noConsent.PatientID).Update("whatsapp_opt_in", false)

	f.sched.Tick()

	msgs := f.messages()
	if len(msgs) != 1 || *msgs[0].AppointmentID != optedIn.ID {
		t.Fatalf("messages = %+v, want only the opted-in patient's reminder", msgs)
	}
}

func TestSender_SkipsWhenConsentWithdrawnAfterQueueing(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	appt := f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567")
	id := f.queued(appt)
	f.db.Model(&clinical_models.Patient{}).Where("id = ?", appt.PatientID).Update("whatsapp_opt_in", false)

	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.x"}]}`}
	if err := f.sender(g).Send(id); err != nil {
		t.Fatal(err)
	}
	if m := f.message(id); m.Status != whatsapp_models.MessageStatusSkipped || m.ErrorCode != whatsapp_models.ErrorCodeNoOptIn {
		t.Fatalf("message = %+v", m)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.requests) != 0 {
		t.Fatalf("sent %d requests to Meta, want 0", len(g.requests))
	}
}

// A sent reminder lands in the thread of its phone, with the text the patient
// read, linked to the appointment's patient; a retried one is not added twice.
func TestSender_ReminderLandsInConversationThread(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	appt := f.appointment(f.tenant, 5*time.Hour, "scheduled", "099 123 4567")
	id := f.queued(appt)

	if err := f.sender(&graphStub{status: 503, body: "down"}).Send(id); err == nil {
		t.Fatal("want a retryable error")
	}
	if err := f.sender(&graphStub{status: 200, body: `{"messages":[{"id":"wamid.OK"}]}`}).Send(id); err != nil {
		t.Fatal(err)
	}
	m := f.message(id)
	if m.ConversationID == nil || m.Direction != whatsapp_models.DirectionOutbound ||
		!strings.Contains(m.Body, "Hola Ana Pérez") || !strings.Contains(m.Body, "a las "+f.now.Add(5*time.Hour).In(f.loc).Format("15:04")) {
		t.Fatalf("message = %+v", m)
	}
	var convs []whatsapp_models.WhatsAppConversation
	f.db.Find(&convs)
	if len(convs) != 1 {
		t.Fatalf("conversations = %d, want 1", len(convs))
	}
	c := convs[0]
	if c.ID != *m.ConversationID || c.Phone != "593991234567" || c.TenantID != f.tenant || c.PatientID == nil || *c.PatientID != appt.PatientID ||
		c.LastDirection != whatsapp_models.DirectionOutbound || !strings.HasPrefix(c.LastMessagePreview, "Hola Ana") || c.UnreadCount != 0 || c.LastInboundAt != nil {
		t.Fatalf("conversation = %+v", c)
	}
}

func TestSender_SkipsWhenMonthlyLimitReached(t *testing.T) {
	f := newSchedFixture(t, []int{24})
	code := fmt.Sprintf("P-%d", time.Now().UnixNano())
	plan := company_models.Plan{Name: code, Code: code, Properties: datatypes.JSONMap{whatsapp_services.PlanLimitKey: 1}}
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: code, TenantID: f.tenant}
	for _, row := range []any{&plan, &company} {
		if err := f.db.Omit(clause.Associations).Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	sub := company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().AddDate(1, 0, 0), CompanyID: company.ID}
	if err := f.db.Omit(clause.Associations).Create(&sub).Error; err != nil {
		t.Fatal(err)
	}

	g := &graphStub{status: 200, body: `{"messages":[{"id":"wamid.OK"}]}`}
	first := f.queued(f.appointment(f.tenant, 5*time.Hour, "scheduled", "0991234567"))
	if err := f.sender(g).Send(first); err != nil {
		t.Fatal(err)
	}
	second := f.queued(f.appointment(f.tenant, 6*time.Hour, "scheduled", "0991234568"))
	if err := f.sender(g).Send(second); err != nil {
		t.Fatal(err)
	}
	if m := f.message(first); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("first = %+v", m)
	}
	if m := f.message(second); m.Status != whatsapp_models.MessageStatusSkipped || m.ErrorCode != whatsapp_models.ErrorCodeMonthlyLimit || m.SentAt != nil {
		t.Fatalf("second = %+v", m)
	}
	if len(g.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(g.requests))
	}
}
