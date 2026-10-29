package waitlist_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	waitlist_dto "pengi-med-saas/features/waitlist/dto"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var loc = time.FixedZone("ECT", -5*3600)

// now is Monday 2 November 2026, 09:00 clinic time.
var now = time.Date(2026, 11, 2, 9, 0, 0, 0, loc)

// graph is a fake Graph API: every message send succeeds; sends are recorded.
type graph struct {
	mu    sync.Mutex
	sends []string
}

func (g *graph) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages") {
			g.mu.Lock()
			g.sends = append(g.sends, string(raw))
			n := len(g.sends)
			g.mu.Unlock()
			_, _ = fmt.Fprintf(w, `{"messages":[{"id":"wamid.offer.%d"}]}`, n)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":100,"message":"unknown path"}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (g *graph) count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.sends)
}

type fixture struct {
	t          *testing.T
	db         *gorm.DB
	h          *WaitlistHandler
	graph      *graph
	own, other uint
	doctor     doctor_models.Doctor
	apptType   agenda_models.AppointmentType
	seq        int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &company_models.Plan{},
		&company_models.Feature{}, &company_models.Subscription{}, &permission_models.Permission{},
		&clinical_models.Patient{}, &clinical_models.Appointment{}, &doctor_models.Doctor{}, &agenda_models.AppointmentType{},
		&waitlist_models.WaitlistEntry{}, &waitlist_models.WaitlistOffer{}, &whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppMessage{},
		&whatsapp_models.WhatsAppConversation{}, &whatsapp_models.WhatsAppTemplate{}, &notifications_models.Notification{})
	g := &graph{}
	client := whatsapp.New(g.server(t).URL, "v23.0", "app-id", "secret")
	f := &fixture{t: t, db: db, graph: g}
	f.h = NewWaitlistHandler(db, zap.NewNop(), client).WithClock(func() time.Time { return now }, loc)
	stamp := time.Now().UnixNano()
	for i, id := range []*uint{&f.own, &f.other} {
		slug := fmt.Sprintf("wlh-%d-%d", i, stamp)
		tenant := tenant_models.Tenant{Name: slug, Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		*id = tenant.ID
		if err := db.Create(&company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "P", TenantID: tenant.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.doctor = doctor_models.Doctor{TenantID: f.own, FullName: "Dra. Ana Torres", Specialty: "general"}
	f.apptType = agenda_models.AppointmentType{TenantID: f.own, Name: "Consulta", DurationMinutes: 30, Active: true}
	if err := db.Create(&f.doctor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&f.apptType).Error; err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) call(action envelope.Action, tenantID uint, method string, body any, id uint, query string) envelope.Response {
	c, _ := testutils.NewGinContext(tenantID, 1)
	c.Set("user_id", int64(7))
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest(method, "/?"+query, bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	if id != 0 {
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	}
	return action(c)
}

func (f *fixture) patient(tenantID uint, optIn bool) clinical_models.Patient {
	f.t.Helper()
	f.seq++
	p := clinical_models.Patient{TenantID: tenantID, FirstName: fmt.Sprintf("María%d", f.seq), LastName: "Pérez",
		Phone: "0991234567", WhatsAppOptIn: optIn, Document: fmt.Sprintf("D%d-%d", f.seq, time.Now().UnixNano())}
	if err := tenantdb.System(f.db).Create(&p).Error; err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) create(tenantID uint, body waitlist_dto.EntryBody) envelope.Response {
	return f.call(f.h.Create, tenantID, http.MethodPost, body, 0, "")
}

func (f *fixture) entry(id uint) waitlist_models.WaitlistEntry {
	var e waitlist_models.WaitlistEntry
	tenantdb.System(f.db).First(&e, id)
	return e
}

func (f *fixture) offer(tenantID, id uint, channel, date, start string) envelope.Response {
	return f.call(f.h.Offer, tenantID, http.MethodPost, waitlist_dto.OfferBody{
		Channel: channel, DoctorID: &f.doctor.ID, AppointmentTypeID: &f.apptType.ID, Date: date, StartTime: start,
	}, id, "")
}

// whatsAppPlan gives tenantID an active plan with the WHATSAPP feature
// (monthly cap limit, -1 = none).
func (f *fixture) whatsAppPlan(tenantID uint, limit int) {
	f.t.Helper()
	perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: "USE_WHATSAPP_INBOX"}, Name: "Inbox", Category: "WHATSAPP"}
	if err := f.db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
		f.t.Fatal(err)
	}
	code := fmt.Sprintf("WA-%d", time.Now().UnixNano())
	feature := company_models.Feature{Code: code, Name: "WhatsApp", Permissions: []permission_models.Permission{perm}}
	if err := f.db.Create(&feature).Error; err != nil {
		f.t.Fatal(err)
	}
	plan := company_models.Plan{Name: code, Code: code, Price: 10, Features: []company_models.Feature{feature}}
	if limit >= 0 {
		plan.Properties = datatypes.JSONMap{"max_whatsapp_messages": limit}
	}
	if err := f.db.Create(&plan).Error; err != nil {
		f.t.Fatal(err)
	}
	var company company_models.Company
	if err := tenantdb.System(f.db).Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	if err := f.db.Create(&company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().Add(30 * 24 * time.Hour), CompanyID: company.ID}).Error; err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) connect(tenantID uint) {
	f.t.Helper()
	box, err := whatsapp_models.Box()
	if err != nil {
		f.t.Fatal(err)
	}
	sealed, err := box.Seal("token")
	if err != nil {
		f.t.Fatal(err)
	}
	acc := whatsapp_models.WhatsAppAccount{TenantID: tenantID, WabaID: "WABA-1", PhoneNumberID: fmt.Sprintf("PHONE-%d", tenantID),
		AccessTokenEncrypted: sealed, Status: whatsapp_models.AccountStatusConnected, ReminderOffsets: []int{24}}
	if err := f.db.Create(&acc).Error; err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) approve(tenantID uint) {
	f.t.Helper()
	if err := f.db.Create(&whatsapp_models.WhatsAppTemplate{TenantID: tenantID, Name: whatsapp_models.TemplateSlotOffer,
		Status: whatsapp_models.TemplateStatusApproved}).Error; err != nil {
		f.t.Fatal(err)
	}
}

func TestWaitlist_CRUDAndValidation(t *testing.T) {
	f := newFixture(t)
	p := f.patient(f.own, true)

	resp := f.create(f.own, waitlist_dto.EntryBody{PatientID: p.ID, DoctorID: &f.doctor.ID, AppointmentTypeID: &f.apptType.ID,
		DateFrom: "2026-11-03", DateTo: "2026-11-20", Notes: " mañanas "})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create: %d %v", resp.Code, resp.Data)
	}
	created := resp.Data.(waitlist_dto.Entry)
	if created.Status != waitlist_models.StatusWaiting || created.Doctor == nil || created.Doctor.ID != f.doctor.ID ||
		created.AppointmentType == nil || created.Notes != "mañanas" || created.Source != waitlist_models.SourceStaff {
		t.Fatalf("created = %+v", created)
	}
	stored := f.entry(created.ID)
	if stored.TenantID != f.own || stored.CreatedByUserID == nil || *stored.CreatedByUserID != 7 ||
		!stored.ExpiresAt.Equal(time.Date(2026, 11, 21, 0, 0, 0, 0, loc)) {
		t.Fatalf("stored = %+v", stored)
	}

	// The same patient, doctor and type again: duplicate.
	if r := f.create(f.own, waitlist_dto.EntryBody{PatientID: p.ID, DoctorID: &f.doctor.ID, AppointmentTypeID: &f.apptType.ID}); r.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", r.Code)
	}
	// Invalid: dates in the wrong order, a range already over, another clinic's patient or doctor.
	other := f.patient(f.other, true)
	// A separate patient: an appointment created after joining books the entry.
	other2 := f.patient(f.own, false)
	for name, body := range map[string]waitlist_dto.EntryBody{
		"order":         {PatientID: p.ID, DateFrom: "2026-11-10", DateTo: "2026-11-05"},
		"over":          {PatientID: p.ID, DateTo: "2026-11-01"},
		"bad date":      {PatientID: p.ID, DateFrom: "11/03/2026"},
		"other patient": {PatientID: other.ID},
		"other doctor": {PatientID: f.patient(f.own, false).ID, DoctorID: func() *uint {
			d := doctor_models.Doctor{TenantID: f.other, FullName: "X", Specialty: "general"}
			f.db.Create(&d)
			return &d.ID
		}()},
		"missing": {},
		"past appt link": {PatientID: other2.ID, AppointmentID: func() *uint {
			a := clinical_models.Appointment{TenantID: f.own, PatientID: other2.ID, Title: "x", Date: time.Date(2026, 10, 30, 0, 0, 0, 0, loc), StartTime: "10:00", EndTime: "10:30", Status: "scheduled"}
			f.db.Create(&a)
			return &a.ID
		}()},
	} {
		if r := f.create(f.own, body); r.Code != http.StatusBadRequest {
			t.Errorf("%s: code = %d, want 400", name, r.Code)
		}
	}

	// Update: any doctor, no range.
	upd := f.call(f.h.Update, f.own, http.MethodPut, waitlist_dto.EntryBody{Notes: "tardes"}, created.ID, "")
	if upd.Code != http.StatusOK {
		t.Fatalf("update: %d", upd.Code)
	}
	if e := upd.Data.(waitlist_dto.Entry); e.Doctor != nil || e.DateTo != "" || e.Notes != "tardes" || e.Patient.ID != p.ID {
		t.Fatalf("updated = %+v", e)
	}

	// Another clinic neither sees nor touches it.
	if r := f.call(f.h.List, f.other, http.MethodGet, nil, 0, ""); len(r.Data.([]waitlist_dto.Entry)) != 0 {
		t.Fatalf("other tenant sees %v", r.Data)
	}
	if r := f.call(f.h.Update, f.other, http.MethodPut, waitlist_dto.EntryBody{}, created.ID, ""); r.Code != http.StatusNotFound {
		t.Fatalf("other tenant update: %d", r.Code)
	}
	if r := f.call(f.h.Remove, f.other, http.MethodDelete, nil, created.ID, ""); r.Code != http.StatusNotFound {
		t.Fatalf("other tenant remove: %d", r.Code)
	}

	list := f.call(f.h.List, f.own, http.MethodGet, nil, 0, "")
	if entries := list.Data.([]waitlist_dto.Entry); len(entries) != 1 || entries[0].ID != created.ID {
		t.Fatalf("list = %+v", list.Data)
	}

	// Remove: it leaves the active list but stays in "all".
	if r := f.call(f.h.Remove, f.own, http.MethodDelete, nil, created.ID, ""); r.Code != http.StatusOK {
		t.Fatalf("remove: %d", r.Code)
	}
	if r := f.call(f.h.Remove, f.own, http.MethodDelete, nil, created.ID, ""); r.Code != http.StatusConflict {
		t.Fatalf("remove twice: %d", r.Code)
	}
	if r := f.call(f.h.List, f.own, http.MethodGet, nil, 0, ""); len(r.Data.([]waitlist_dto.Entry)) != 0 {
		t.Fatal("removed entry still active")
	}
	if r := f.call(f.h.List, f.own, http.MethodGet, nil, 0, "status=all"); len(r.Data.([]waitlist_dto.Entry)) != 1 {
		t.Fatal("removed entry missing from all")
	}
	if r := f.call(f.h.List, f.own, http.MethodGet, nil, 0, "status=bogus"); r.Code != http.StatusBadRequest {
		t.Fatalf("bad status filter: %d", r.Code)
	}
}

func TestWaitlist_ExpiredEntriesLeaveTheList(t *testing.T) {
	f := newFixture(t)
	p := f.patient(f.own, true)
	e := waitlist_models.WaitlistEntry{TenantID: f.own, PatientID: p.ID, Status: waitlist_models.StatusWaiting, ExpiresAt: now.Add(-time.Minute)}
	if err := f.db.Create(&e).Error; err != nil {
		t.Fatal(err)
	}
	if r := f.call(f.h.List, f.own, http.MethodGet, nil, 0, ""); len(r.Data.([]waitlist_dto.Entry)) != 0 {
		t.Fatal("an expired entry shows as waiting")
	}
	if s := f.entry(e.ID).Status; s != waitlist_models.StatusExpired {
		t.Fatalf("status = %s", s)
	}
	if r := f.offer(f.own, e.ID, waitlist_models.ChannelPhone, "2026-11-04", "10:00"); r.Code != http.StatusConflict {
		t.Fatalf("offering an expired entry: %d", r.Code)
	}
}

func TestOffer_PhoneMarksOfferedOncePerSlot(t *testing.T) {
	f := newFixture(t)
	p := f.patient(f.own, false)
	created := f.create(f.own, waitlist_dto.EntryBody{PatientID: p.ID, DoctorID: &f.doctor.ID}).Data.(waitlist_dto.Entry)

	cands := f.call(f.h.Candidates, f.own, http.MethodGet, nil, 0,
		fmt.Sprintf("date=2026-11-04&start_time=10:00&end_time=10:30&doctor_id=%d&appointment_type_id=%d", f.doctor.ID, f.apptType.ID))
	data := cands.Data.(waitlist_dto.Candidates)
	if cands.Code != http.StatusOK || len(data.Candidates) != 1 || data.WhatsAppAvailable || data.Candidates[0].CanWhatsApp ||
		data.Doctor != f.doctor.FullName {
		t.Fatalf("candidates = %d %+v", cands.Code, data)
	}

	r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-04", "10:00")
	if r.Code != http.StatusOK {
		t.Fatalf("offer: %d %v", r.Code, r.Data)
	}
	e := f.entry(created.ID)
	if e.Status != waitlist_models.StatusOffered || e.OfferChannel != waitlist_models.ChannelPhone || e.OfferedDate != "2026-11-04" ||
		e.OfferedStartTime != "10:00" || e.OfferedAt == nil || e.OfferMessageID != nil {
		t.Fatalf("entry = %+v", e)
	}
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-04", "10:00"); r.Code != http.StatusConflict {
		t.Fatalf("same slot twice: %d", r.Code)
	}
	cands = f.call(f.h.Candidates, f.own, http.MethodGet, nil, 0, fmt.Sprintf("date=2026-11-04&start_time=10:00&doctor_id=%d", f.doctor.ID))
	if c := cands.Data.(waitlist_dto.Candidates).Candidates; len(c) != 1 || !c[0].OfferedThisSlot {
		t.Fatalf("offered entry must stay a candidate, marked: %+v", c)
	}
	// Another slot can be offered; a past slot or another doctor's cannot.
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-05", "11:00"); r.Code != http.StatusOK {
		t.Fatalf("another slot: %d", r.Code)
	}
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-01", "11:00"); r.Code != http.StatusBadRequest {
		t.Fatalf("past slot: %d", r.Code)
	}
	otherDoctor := doctor_models.Doctor{TenantID: f.own, FullName: "Dr. Luis", Specialty: "general"}
	f.db.Create(&otherDoctor)
	r = f.call(f.h.Offer, f.own, http.MethodPost, waitlist_dto.OfferBody{Channel: "phone", DoctorID: &otherDoctor.ID, Date: "2026-11-06", StartTime: "09:00"}, created.ID, "")
	if r.Code != http.StatusConflict {
		t.Fatalf("another doctor's slot: %d", r.Code)
	}
	// Other tenants can't offer it.
	if r := f.offer(f.other, created.ID, waitlist_models.ChannelPhone, "2026-11-07", "10:00"); r.Code != http.StatusNotFound {
		t.Fatalf("other tenant offer: %d", r.Code)
	}
}

func TestOffer_WhatsAppSendConditions(t *testing.T) {
	f := newFixture(t)
	optedIn := f.patient(f.own, true)
	noConsent := f.patient(f.own, false)
	entry := f.create(f.own, waitlist_dto.EntryBody{PatientID: optedIn.ID}).Data.(waitlist_dto.Entry)
	noConsentEntry := f.create(f.own, waitlist_dto.EntryBody{PatientID: noConsent.ID}).Data.(waitlist_dto.Entry)
	wa := func(id uint, date string) envelope.Response {
		return f.offer(f.own, id, waitlist_models.ChannelWhatsApp, date, "10:00")
	}

	if r := wa(entry.ID, "2026-11-04"); r.Code != http.StatusForbidden {
		t.Fatalf("without WhatsApp in the plan: %d", r.Code)
	}
	f.whatsAppPlan(f.own, 1)
	if r := wa(entry.ID, "2026-11-04"); r.Code != http.StatusConflict {
		t.Fatalf("not connected: %d", r.Code)
	}
	f.connect(f.own)
	if r := wa(entry.ID, "2026-11-04"); r.Code != http.StatusConflict {
		t.Fatalf("template not approved: %d", r.Code)
	}
	f.approve(f.own)
	if r := wa(noConsentEntry.ID, "2026-11-04"); r.Code != http.StatusConflict {
		t.Fatalf("no consent: %d", r.Code)
	}
	if f.graph.count() != 0 || f.entry(entry.ID).Status != waitlist_models.StatusWaiting {
		t.Fatal("a refused offer must send nothing and leave the entry waiting")
	}

	cands := f.call(f.h.Candidates, f.own, http.MethodGet, nil, 0, "date=2026-11-04&start_time=10:00")
	if d := cands.Data.(waitlist_dto.Candidates); !d.WhatsAppAvailable || len(d.Candidates) != 2 || !d.Candidates[0].CanWhatsApp || d.Candidates[1].CanWhatsApp {
		t.Fatalf("candidates = %+v", d)
	}

	r := wa(entry.ID, "2026-11-04")
	if r.Code != http.StatusOK {
		t.Fatalf("offer: %d %v", r.Code, r.Data)
	}
	stored := f.entry(entry.ID)
	if stored.Status != waitlist_models.StatusOffered || stored.OfferMessageID == nil || stored.OfferChannel != waitlist_models.ChannelWhatsApp {
		t.Fatalf("entry = %+v", stored)
	}
	var msg whatsapp_models.WhatsAppMessage
	tenantdb.System(f.db).First(&msg, *stored.OfferMessageID)
	if msg.Kind != whatsapp_models.KindSlotOffer || msg.Status != whatsapp_models.MessageStatusSent || msg.SentAt == nil ||
		msg.ConversationID == nil || msg.SentByUserID == nil || !strings.Contains(msg.Body, "Dra. Ana Torres") ||
		!strings.Contains(msg.Body, "miércoles 4 de noviembre") || msg.ToPhone != "593991234567" {
		t.Fatalf("message = %+v", msg)
	}
	if sent := f.graph.sends[0]; !strings.Contains(sent, fmt.Sprintf(`"payload":"slot_offer:%d"`, msg.ID)) ||
		!strings.Contains(sent, whatsapp_models.TemplateSlotOffer) {
		t.Fatalf("graph body = %s", sent)
	}

	// The same slot again: refused, nothing sent.
	if r := wa(entry.ID, "2026-11-04"); r.Code != http.StatusConflict || f.graph.count() != 1 {
		t.Fatalf("same slot again: %d, sends = %d", r.Code, f.graph.count())
	}
	// The plan's cap is 1 and the offer counted: another slot is refused.
	if r := wa(entry.ID, "2026-11-05"); r.Code != http.StatusForbidden || f.graph.count() != 1 {
		t.Fatalf("over the monthly cap: %d, sends = %d", r.Code, f.graph.count())
	}
	if e := f.entry(entry.ID); e.OfferedDate != "2026-11-04" {
		t.Fatalf("a refused offer changed the entry: %+v", e)
	}
}

func TestOffer_NeverTheSameSlotTwice(t *testing.T) {
	f := newFixture(t)
	p := f.patient(f.own, false)
	created := f.create(f.own, waitlist_dto.EntryBody{PatientID: p.ID}).Data.(waitlist_dto.Entry)

	// A, B, then A again: A was offered already, whatever came between.
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-04", "10:00"); r.Code != http.StatusOK {
		t.Fatalf("A: %d", r.Code)
	}
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-05", "11:00"); r.Code != http.StatusOK {
		t.Fatalf("B: %d", r.Code)
	}
	if r := f.offer(f.own, created.ID, waitlist_models.ChannelPhone, "2026-11-04", "10:00"); r.Code != http.StatusConflict {
		t.Fatalf("A again: %d, want 409", r.Code)
	}
	if e := f.entry(created.ID); e.OfferedDate != "2026-11-05" {
		t.Fatalf("a refused offer changed the entry: %+v", e)
	}
	var n int64
	f.db.Model(&waitlist_models.WaitlistOffer{}).Where("entry_id = ?", created.ID).Count(&n)
	if n != 2 {
		t.Fatalf("offer history rows = %d, want 2", n)
	}
	// The candidates of A show it as already offered.
	cands := f.call(f.h.Candidates, f.own, http.MethodGet, nil, 0, fmt.Sprintf("date=2026-11-04&start_time=10:00&doctor_id=%d", f.doctor.ID))
	if c := cands.Data.(waitlist_dto.Candidates).Candidates; len(c) != 1 || !c[0].OfferedThisSlot {
		t.Fatalf("candidates of A = %+v", c)
	}
}

func TestOffer_FreeingPatientIsNotACandidate(t *testing.T) {
	f := newFixture(t)
	freer := f.patient(f.own, true)
	other := f.patient(f.own, true)
	own := f.create(f.own, waitlist_dto.EntryBody{PatientID: freer.ID}).Data.(waitlist_dto.Entry)
	f.create(f.own, waitlist_dto.EntryBody{PatientID: other.ID})

	q := fmt.Sprintf("date=2026-11-04&start_time=10:00&doctor_id=%d&patient_id=%d", f.doctor.ID, freer.ID)
	c := f.call(f.h.Candidates, f.own, http.MethodGet, nil, 0, q).Data.(waitlist_dto.Candidates).Candidates
	if len(c) != 1 || c[0].Patient.ID != other.ID {
		t.Fatalf("candidates = %+v, want only the other patient", c)
	}
	r := f.call(f.h.Offer, f.own, http.MethodPost, waitlist_dto.OfferBody{Channel: "phone", DoctorID: &f.doctor.ID,
		Date: "2026-11-04", StartTime: "10:00", PatientID: freer.ID}, own.ID, "")
	if r.Code != http.StatusConflict {
		t.Fatalf("offering the freeing patient their own slot: %d, want 409", r.Code)
	}
	if e := f.entry(own.ID); e.Status != waitlist_models.StatusWaiting {
		t.Fatalf("entry = %+v", e)
	}
}
