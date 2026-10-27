package portal_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	portal_workers "pengi-med-saas/features/portal/workers"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeCalendar records the Google Calendar syncs.
type fakeCalendar struct {
	mu      sync.Mutex
	created []uint
	changed []string // "<id>:<status>"
}

func (c *fakeCalendar) SyncCreated(_ uint, a *clinical_models.Appointment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.created = append(c.created, a.ID)
}

func (c *fakeCalendar) SyncStatusChange(_ uint, a *clinical_models.Appointment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changed = append(c.changed, fmt.Sprintf("%d:%s", a.ID, a.Status))
}

// bookingNow is the frozen clock of the booking tests: Monday 2030-01-07,
// 07:00 in the clinic. Min notice 2 h: the first free time today is 09:00.
var bookingNow = time.Date(2030, 1, 7, 7, 0, 0, 0, portal_services.Location())

const tomorrow = "2030-01-08"

type bookingFixture struct {
	*fixture
	calendar *fakeCalendar
	delivery *portal_services.Delivery
	typ      agenda_models.AppointmentType
	doctor2  doctor_models.Doctor // same specialty as f.doctor, bookable
	first    *FirstAppointmentHandler
}

func newBookingFixture(t *testing.T) *bookingFixture {
	t.Helper()
	f := newFixture(t)
	if err := f.db.AutoMigrate(&agenda_models.DoctorSchedule{}, &agenda_models.ScheduleBlock{},
		&portal_models.AppointmentRequest{}, &notifications_models.Notification{},
		&user_models.Role{}, &user_models.Environment{}); err != nil {
		t.Fatal(err)
	}
	prev := timeNow
	timeNow = func() time.Time { return bookingNow }
	t.Cleanup(func() { timeNow = prev })

	b := &bookingFixture{fixture: f, calendar: &fakeCalendar{}}
	if err := tenantdb.System(f.db).Model(&f.doctor).Update("portal_bookable", true).Error; err != nil {
		t.Fatal(err)
	}
	b.doctor2 = doctor_models.Doctor{TenantID: f.own.ID, FullName: "Dr. Luis Paz", Specialty: "cardiology", Active: true, PortalBookable: true}
	f.mustCreate(&b.doctor2)
	for _, d := range []uint{f.doctor.ID, b.doctor2.ID} {
		for wd := 0; wd < 7; wd++ {
			f.mustCreate(&agenda_models.DoctorSchedule{TenantID: f.own.ID, DoctorID: d, Weekday: wd, StartTime: "08:00", EndTime: "12:00"})
		}
	}
	b.typ = agenda_models.AppointmentType{TenantID: f.own.ID, Name: "Consulta", DurationMinutes: 30, Active: true, PortalBookable: true}
	f.mustCreate(&b.typ)
	b.set(f.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = true })
	b.set(f.other, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = true })

	// A receptionist of the clinic with MANAGE_APPOINTMENT_REQUESTS gets the
	// staff notifications.
	company := company_models.Company{}
	if err := tenantdb.System(f.db).Where("tenant_id = ?", f.own.ID).First(&company).Error; err != nil {
		t.Fatal(err)
	}
	role := user_models.Role{Role: "recepcionista", Permissions: []permission_models.Permission{{BaseStringID: database.BaseStringID{ID: portal_services.ManageRequestsPermission}, Name: "x", Category: "PATIENT_PORTAL"}}}
	f.mustCreate(&role)
	f.mustCreate(&user_models.Environment{UserID: 7, RoleID: role.ID, CompanyID: company.ID})

	b.delivery = portal_services.NewDelivery(f.db, zap.NewNop(), f.mail, nil)
	bh := NewBookingHandler(f.db, zap.NewNop(), b.delivery, b.calendar).WithSyncAfterCommit()
	rh := NewRequestHandler(f.db, zap.NewNop(), b.delivery, b.calendar).WithSyncAfterCommit()
	// Like registerPortalRoutes: the catalog takes a session or a guest token.
	catalog := f.router.Group("/portal", portal_middleware.RequirePortalSessionOrGuest(f.db), portal_middleware.RequireBooking())
	catalog.GET("/booking/options", envelope.Handle(bh.GetOptions))
	catalog.GET("/booking/slots", envelope.Handle(bh.GetSlots))
	b.first = NewFirstAppointmentHandler(f.db, zap.NewNop(), b.delivery).WithSync().WithCaptcha(nil)
	f.router.POST("/portal/clinics/:slug/new-patient/code", envelope.Handle(b.first.RequestCode))
	f.router.POST("/portal/clinics/:slug/new-patient/verify", envelope.Handle(b.first.Verify))
	guest := f.router.Group("/portal/new-patient", portal_middleware.RequirePortalGuest(f.db), portal_middleware.RequireBooking())
	guest.POST("/requests", envelope.Handle(b.first.CreateRequest))
	authed := f.router.Group("/portal", portal_middleware.RequirePortalSession(f.db))
	booking := authed.Group("", portal_middleware.RequireBooking())
	booking.GET("/booking/requests", envelope.Handle(bh.ListRequests))
	booking.POST("/booking/requests", envelope.Handle(bh.CreateRequest))
	booking.POST("/booking/requests/:id/cancel", envelope.Handle(bh.CancelRequest))
	booking.POST("/appointments/:id/cancel", envelope.Handle(bh.CancelAppointment))

	// Staff inbox: the tenant comes from a test header (TenantMiddleware's job).
	staff := f.router.Group("/appointment-requests", func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.GetHeader("X-Test-Tenant"), 10, 32)
		c.Set("tenant_id", uint(id))
		c.Set("user_id", int64(7))
	})
	staff.GET("", envelope.Handle(rh.ListRequests))
	staff.GET("/pending-count", envelope.Handle(rh.PendingCount))
	staff.POST("/:id/approve", envelope.Handle(rh.Approve))
	staff.POST("/:id/reject", envelope.Handle(rh.Reject))
	return b
}

// set changes a clinic's portal settings, read fresh from the database.
func (b *bookingFixture) set(tenant tenant_models.Tenant, mutate func(*tenant_models.PortalSettings)) {
	b.t.Helper()
	var fresh tenant_models.Tenant
	if err := tenantdb.System(b.db).First(&fresh, tenant.ID).Error; err != nil {
		b.t.Fatal(err)
	}
	b.setSettings(fresh, mutate)
}

// notifications returns the message keys of user 7's notifications.
func (b *bookingFixture) notifications() []string {
	var rows []notifications_models.Notification
	tenantdb.System(b.db).Where("user_id = ?", 7).Order("id").Find(&rows)
	out := []string{}
	for _, r := range rows {
		out = append(out, r.MessageKey)
	}
	return out
}

func (b *bookingFixture) token(tenant tenant_models.Tenant, p clinical_models.Patient) string {
	b.t.Helper()
	token, err := auth.GeneratePortalAccessToken(tenant.ID, p.ID)
	if err != nil {
		b.t.Fatal(err)
	}
	return token
}

func (b *bookingFixture) staff(method, path string, tenant tenant_models.Tenant, body string) result {
	b.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", strconv.FormatUint(uint64(tenant.ID), 10))
	w := httptest.NewRecorder()
	b.router.ServeHTTP(w, req)
	return b.decode(w)
}

func (b *bookingFixture) decode(w *httptest.ResponseRecorder) result {
	res := result{Code: w.Code, Raw: w.Body.Bytes(), Headers: w.Header()}
	_ = json.Unmarshal(w.Body.Bytes(), &res.Body)
	return res
}

func (b *bookingFixture) request(token string, start string, extra map[string]any) result {
	body := map[string]any{"type_id": b.typ.ID, "doctor_id": b.doctor.ID, "date": tomorrow, "start_time": start}
	for k, v := range extra {
		body[k] = v
	}
	return b.do(http.MethodPost, "/portal/booking/requests", token, body)
}

func okData(t *testing.T, r result) map[string]any {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("status %d: %s", r.Code, r.Raw)
	}
	return r.Body["data"].(map[string]any)
}

func numID(v any) uint { return uint(v.(float64)) }

func (b *bookingFixture) times(token, date string, query string) []string {
	b.t.Helper()
	r := b.do(http.MethodGet, "/portal/booking/slots?type_id="+strconv.Itoa(int(b.typ.ID))+"&"+query+"&from="+date+"&to="+date, token, nil)
	days := okData(b.t, r)["days"].([]any)
	out := []string{}
	for _, d := range days {
		day := d.(map[string]any)
		if day["date"] == date {
			for _, t := range day["times"].([]any) {
				out = append(out, t.(string))
			}
		}
	}
	return out
}

func hasTime(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestBooking_OptionsAndSlots(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	// Not offered: a doctor not bookable, and a bookable one without schedule.
	b.mustCreate(&doctor_models.Doctor{TenantID: b.own.ID, FullName: "No portal", Specialty: "general", Active: true})
	b.mustCreate(&doctor_models.Doctor{TenantID: b.own.ID, FullName: "No schedule", Specialty: "general", Active: true, PortalBookable: true})

	opts := okData(t, b.do(http.MethodGet, "/portal/booking/options", token, nil))
	if opts["choose_by"] != "doctor" || opts["min_date"] != "2030-01-07" || opts["max_date"] != "2030-03-08" {
		t.Fatalf("options = %v", opts)
	}
	doctors := opts["doctors"].([]any)
	if len(doctors) != 2 || len(opts["types"].([]any)) != 1 || len(opts["specialties"].([]any)) != 1 {
		t.Fatalf("options = %s", mustJSON(opts))
	}
	first := doctors[0].(map[string]any)
	if numID(first["id"]) != b.doctor.ID || len(first["type_ids"].([]any)) != 1 {
		t.Fatalf("doctor = %v", first)
	}

	// Today: min notice (2 h from 07:00) leaves 09:00..11:30.
	today := b.times(token, "2030-01-07", "doctor_id="+strconv.Itoa(int(b.doctor.ID)))
	if len(today) != 6 || today[0] != "09:00" {
		t.Fatalf("today = %v", today)
	}

	// Specialty mode: no doctors listed, slots merged.
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.ChooseBy = tenant_models.BookingChooseBySpecialty })
	opts = okData(t, b.do(http.MethodGet, "/portal/booking/options", token, nil))
	if len(opts["doctors"].([]any)) != 0 || len(opts["specialties"].([]any)) != 1 {
		t.Fatalf("specialty options = %s", mustJSON(opts))
	}
	if got := b.times(token, tomorrow, "specialty=cardiology"); len(got) != 8 {
		t.Fatalf("specialty slots = %v", got)
	}
}

func TestBooking_PendingRequestHoldsSlot(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	req := okData(t, b.request(token, "09:00", map[string]any{"reason": "Dolor de cabeza"}))
	if req["status"] != "pending" || req["end_time"] != "09:30" || req["type_name"] != "Consulta" || req["appointment_id"] != nil {
		t.Fatalf("request = %v", req)
	}
	exp, _ := time.Parse(time.RFC3339, req["expires_at"].(string))
	if !exp.Equal(bookingNow.Add(24 * time.Hour)) {
		t.Fatalf("expires_at = %v", exp)
	}
	doctorQ := "doctor_id=" + strconv.Itoa(int(b.doctor.ID))
	if hasTime(b.times(token, tomorrow, doctorQ), "09:00") {
		t.Fatal("a pending request must hold its slot")
	}
	// Another patient asking for the same slot: 409 slot taken.
	other := b.token(b.own, b.ownPat2)
	if r := b.request(other, "09:00", nil); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-010" {
		t.Fatalf("slot taken: %d %s", r.Code, r.Raw)
	}
	// Times outside the grid or the schedule are not slots either.
	if r := b.request(other, "09:10", nil); r.Code != http.StatusConflict {
		t.Fatalf("off-grid: %d %s", r.Code, r.Raw)
	}
	var count int64
	tenantdb.System(b.db).Model(&clinical_models.Appointment{}).Count(&count)
	if count != 0 {
		t.Fatal("a pending request creates no appointment")
	}
	// The clinic sees it in its inbox and counter.
	if r := b.staff(http.MethodGet, "/appointment-requests/pending-count", b.own, ""); string(mustJSON(r.Body["data"])) != `{"count":1}` {
		t.Fatalf("pending count = %s", r.Raw)
	}
	list := b.staff(http.MethodGet, "/appointment-requests", b.own, "")
	items := list.Body["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("inbox = %s", list.Raw)
	}
	item := items[0].(map[string]any)
	patient := item["patient"].(map[string]any)
	if numID(patient["id"]) != b.ownPat.ID || patient["document"] != sharedDocument || item["reason"] != "Dolor de cabeza" ||
		item["doctor"].(map[string]any)["full_name"] != b.doctor.FullName || item["type_name"] != "Consulta" {
		t.Fatalf("inbox item = %v", item)
	}
	if got := b.notifications(); len(got) != 1 || got[0] != "notification.portal.booking.request" {
		t.Fatalf("staff notifications = %v", got)
	}
	// The patient sees it in their list.
	mine := b.do(http.MethodGet, "/portal/booking/requests", token, nil)
	if listLen(t, mine) != 1 {
		t.Fatalf("my requests = %s", mine.Raw)
	}
}

func TestBooking_AutoConfirmCreatesAppointment(t *testing.T) {
	b := newBookingFixture(t)
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.AutoConfirm = true })
	token := b.token(b.own, b.ownPat)
	req := okData(t, b.request(token, "10:00", nil))
	if req["status"] != "approved" || req["appointment_id"] == nil {
		t.Fatalf("request = %v", req)
	}
	var appt clinical_models.Appointment
	tenantdb.System(b.db).First(&appt, numID(req["appointment_id"]))
	if appt.Status != "scheduled" || appt.Title != "Consulta" || appt.PatientID != b.ownPat.ID || appt.TenantID != b.own.ID ||
		appt.DoctorID == nil || *appt.DoctorID != b.doctor.ID || appt.AppointmentTypeID == nil || *appt.AppointmentTypeID != b.typ.ID ||
		appt.StartTime != "10:00" || appt.EndTime != "10:30" || appt.Date.In(portal_services.Location()).Format("2006-01-02") != tomorrow {
		t.Fatalf("appointment = %+v", appt)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, numID(req["id"]))
	if row.DecidedByUserID != nil || row.DecidedAt == nil {
		t.Fatalf("auto-confirm row = %+v", row)
	}
	if len(b.calendar.created) != 1 || b.calendar.created[0] != appt.ID {
		t.Fatalf("calendar = %v", b.calendar.created)
	}
	notices := b.mail.noticesTo(b.ownPat.Email)
	if len(notices) != 1 || !strings.Contains(notices[0].Lines[0], b.doctor.FullName) || !strings.Contains(notices[0].Lines[0], "10:00") {
		t.Fatalf("notices = %+v", notices)
	}
	// The appointment takes the slot.
	if hasTime(b.times(token, tomorrow, "doctor_id="+strconv.Itoa(int(b.doctor.ID))), "10:00") {
		t.Fatal("the appointment must take its slot")
	}
}

func TestBooking_MaxThreePending(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	for _, start := range []string{"08:00", "08:30", "09:00"} {
		okData(t, b.request(token, start, nil))
	}
	if r := b.request(token, "09:30", nil); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-011" {
		t.Fatalf("4th pending: %d %s", r.Code, r.Raw)
	}
}

func TestBooking_OffAnswers404(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = false })
	for _, path := range []string{"/portal/booking/options", "/portal/booking/slots", "/portal/booking/requests"} {
		if r := b.do(http.MethodGet, path, token, nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-009" {
			t.Fatalf("%s: %d %s", path, r.Code, r.Raw)
		}
	}
	if r := b.request(token, "09:00", nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-009" {
		t.Fatalf("create: %d %s", r.Code, r.Raw)
	}
	// Booking on but the appointments section hidden: still off.
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = true; s.Sections.Appointments = false })
	if r := b.do(http.MethodGet, "/portal/booking/options", token, nil); r.Code != http.StatusNotFound {
		t.Fatalf("section hidden: %d", r.Code)
	}
	// The clinic card tells the portal.
	r := b.do(http.MethodGet, "/portal/clinics/"+b.own.Slug, "", nil)
	if bk := okData(t, r)["booking"].(map[string]any); bk["enabled"] != false || bk["choose_by"] != "doctor" {
		t.Fatalf("clinic booking = %v", bk)
	}
}

func TestBooking_CancelRequest(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	req := okData(t, b.request(token, "09:00", nil))
	path := fmt.Sprintf("/portal/booking/requests/%d/cancel", numID(req["id"]))
	// Another patient of the clinic, or of another clinic: not found.
	if r := b.do(http.MethodPost, path, b.token(b.own, b.ownPat2), nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-004" {
		t.Fatalf("other patient: %d %s", r.Code, r.Raw)
	}
	if r := b.do(http.MethodPost, path, b.token(b.other, b.otherPat), nil); r.Code != http.StatusNotFound {
		t.Fatalf("other clinic: %d %s", r.Code, r.Raw)
	}
	if got := okData(t, b.do(http.MethodPost, path, token, nil)); got["status"] != "cancelled" {
		t.Fatalf("cancel = %v", got)
	}
	if r := b.do(http.MethodPost, path, token, nil); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-013" {
		t.Fatalf("cancel twice: %d %s", r.Code, r.Raw)
	}
	// The slot is free again.
	if !hasTime(b.times(token, tomorrow, "doctor_id="+strconv.Itoa(int(b.doctor.ID))), "09:00") {
		t.Fatal("a cancelled request frees its slot")
	}
}

func TestBooking_ApproveAndReject(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	req := okData(t, b.request(token, "09:00", nil))
	approve := fmt.Sprintf("/appointment-requests/%d/approve", numID(req["id"]))

	// Staff booked over it meanwhile: 409 with the appointments overlap code.
	doctorID := b.doctor.ID
	staffAppt := clinical_models.Appointment{TenantID: b.own.ID, PatientID: b.ownPat2.ID, Title: "x", Status: "scheduled",
		Date: time.Date(2030, 1, 8, 0, 0, 0, 0, portal_services.Location()), StartTime: "09:15", EndTime: "09:45", DoctorID: &doctorID}
	b.mustCreate(&staffAppt)
	if r := b.staff(http.MethodPost, approve, b.own, ""); r.Code != http.StatusConflict || errorCode(r) != "E-CLIN-009" {
		t.Fatalf("overlap: %d %s", r.Code, r.Raw)
	}
	tenantdb.System(b.db).Model(&staffAppt).Update("status", "cancelled")

	got := okData(t, b.staff(http.MethodPost, approve, b.own, ""))
	if got["status"] != "approved" || got["appointment_id"] == nil {
		t.Fatalf("approve = %v", got)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, numID(req["id"]))
	if row.DecidedByUserID == nil || *row.DecidedByUserID != 7 || row.AppointmentID == nil {
		t.Fatalf("approved row = %+v", row)
	}
	var appt clinical_models.Appointment
	tenantdb.System(b.db).First(&appt, *row.AppointmentID)
	if appt.Status != "scheduled" || appt.StartTime != "09:00" || appt.Title != "Consulta" {
		t.Fatalf("appointment = %+v", appt)
	}
	if len(b.calendar.created) != 1 || len(b.mail.noticesTo(b.ownPat.Email)) != 1 {
		t.Fatalf("calendar %v, notices %v", b.calendar.created, b.mail.noticesTo(b.ownPat.Email))
	}
	if r := b.staff(http.MethodPost, approve, b.own, ""); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-013" {
		t.Fatalf("approve twice: %d %s", r.Code, r.Raw)
	}

	// Reject another one, with a reason the patient reads.
	req2 := okData(t, b.request(token, "11:00", nil))
	reject := fmt.Sprintf("/appointment-requests/%d/reject", numID(req2["id"]))
	got = okData(t, b.staff(http.MethodPost, reject, b.own, `{"reason":"El doctor no atiende ese día"}`))
	if got["status"] != "rejected" || got["reject_reason"] != "El doctor no atiende ese día" {
		t.Fatalf("reject = %v", got)
	}
	notices := b.mail.noticesTo(b.ownPat.Email)
	last := notices[len(notices)-1]
	if len(notices) != 2 || !strings.Contains(strings.Join(last.Lines, " "), "El doctor no atiende ese día") {
		t.Fatalf("reject notice = %+v", last)
	}
	if r := b.staff(http.MethodPost, reject, b.own, ""); r.Code != http.StatusConflict {
		t.Fatalf("reject twice: %d", r.Code)
	}
	// Ended requests stay in the patient's list (rejected); approved ones are appointments.
	mine := b.do(http.MethodGet, "/portal/booking/requests", token, nil)
	if listLen(t, mine) != 1 || mine.Body["data"].([]any)[0].(map[string]any)["status"] != "rejected" {
		t.Fatalf("my requests = %s", mine.Raw)
	}
}

func TestBooking_PatientCancelsAppointment(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	loc := portal_services.Location()
	mk := func(p clinical_models.Patient, day int, start string) clinical_models.Appointment {
		doctorID := b.doctor.ID
		a := clinical_models.Appointment{TenantID: b.own.ID, PatientID: p.ID, Title: "x", Status: "scheduled",
			Date: time.Date(2030, 1, day, 0, 0, 0, 0, loc), StartTime: start, EndTime: "23:30", DoctorID: &doctorID}
		b.mustCreate(&a)
		return a
	}
	soon := mk(b.ownPat, 7, "20:00")      // 13 h ahead: inside the 24 h cancel notice
	later := mk(b.ownPat, 10, "10:00")    // days ahead
	notMine := mk(b.ownPat2, 10, "11:00") // another patient's

	if r := b.do(http.MethodPost, fmt.Sprintf("/portal/appointments/%d/cancel", soon.ID), token, nil); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-012" {
		t.Fatalf("too late: %d %s", r.Code, r.Raw)
	}
	if r := b.do(http.MethodPost, fmt.Sprintf("/portal/appointments/%d/cancel", notMine.ID), token, nil); r.Code != http.StatusNotFound {
		t.Fatalf("not mine: %d %s", r.Code, r.Raw)
	}
	// The list says which ones can be cancelled.
	list := b.do(http.MethodGet, "/portal/appointments", token, nil)
	canCancel := map[uint]bool{}
	for _, it := range list.Body["data"].([]any) {
		m := it.(map[string]any)
		canCancel[numID(m["id"])] = m["can_cancel"].(bool)
	}
	if canCancel[soon.ID] || !canCancel[later.ID] {
		t.Fatalf("can_cancel = %v", canCancel)
	}

	got := okData(t, b.do(http.MethodPost, fmt.Sprintf("/portal/appointments/%d/cancel", later.ID), token, nil))
	if got["status"] != "cancelled" {
		t.Fatalf("cancel = %v", got)
	}
	var row clinical_models.Appointment
	tenantdb.System(b.db).First(&row, later.ID)
	if row.Status != "cancelled" || len(b.calendar.changed) != 1 || b.calendar.changed[0] != fmt.Sprintf("%d:cancelled", later.ID) {
		t.Fatalf("row %s, calendar %v", row.Status, b.calendar.changed)
	}
	if r := b.do(http.MethodPost, fmt.Sprintf("/portal/appointments/%d/cancel", later.ID), token, nil); r.Code != http.StatusConflict {
		t.Fatalf("cancel twice: %d", r.Code)
	}
	if got := b.notifications(); len(got) != 1 || got[0] != "notification.portal.booking.appointment_cancelled" {
		t.Fatalf("staff notifications = %v", got)
	}
}

func TestBooking_ExpiryWorkerIsIdempotent(t *testing.T) {
	b := newBookingFixture(t)
	token := b.token(b.own, b.ownPat)
	req := okData(t, b.request(token, "09:00", nil))
	worker := portal_workers.NewRequestExpiryScheduler(b.db, zap.NewNop(), b.delivery).
		WithClock(func() time.Time { return bookingNow.Add(time.Hour) })
	if n := worker.Tick(); n != 0 {
		t.Fatalf("expired too early: %d", n)
	}
	worker.WithClock(func() time.Time { return bookingNow.Add(25 * time.Hour) })
	if n := worker.Tick(); n != 1 {
		t.Fatalf("expired = %d", n)
	}
	if n := worker.Tick(); n != 0 {
		t.Fatalf("second tick expired %d again", n)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, numID(req["id"]))
	if row.Status != portal_models.RequestExpired {
		t.Fatalf("status = %s", row.Status)
	}
	if n := len(b.mail.noticesTo(b.ownPat.Email)); n != 1 {
		t.Fatalf("expiry notices = %d", n)
	}
	// An expired request cannot be approved.
	if r := b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/approve", row.ID), b.own, ""); r.Code != http.StatusConflict {
		t.Fatalf("approve expired: %d %s", r.Code, r.Raw)
	}
}

func TestBooking_TenantIsolation(t *testing.T) {
	b := newBookingFixture(t)
	req := okData(t, b.request(b.token(b.own, b.ownPat), "09:00", nil))
	// Clinic B's staff see nothing of clinic A's requests and cannot decide them.
	if r := b.staff(http.MethodGet, "/appointment-requests?status=all", b.other, ""); listLen(t, r) != 0 {
		t.Fatalf("other clinic inbox = %s", r.Raw)
	}
	if r := b.staff(http.MethodGet, "/appointment-requests/pending-count", b.other, ""); string(mustJSON(r.Body["data"])) != `{"count":0}` {
		t.Fatalf("other clinic count = %s", r.Raw)
	}
	for _, action := range []string{"approve", "reject"} {
		if r := b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/%s", numID(req["id"]), action), b.other, ""); r.Code != http.StatusNotFound {
			t.Fatalf("other clinic %s: %d %s", action, r.Code, r.Raw)
		}
	}
	// Clinic B's patient (same document) sees none of them either.
	if r := b.do(http.MethodGet, "/portal/booking/requests", b.token(b.other, b.otherPat), nil); listLen(t, r) != 0 {
		t.Fatalf("other clinic patient = %s", r.Raw)
	}
	// And cannot book clinic A's doctor.
	r := b.do(http.MethodPost, "/portal/booking/requests", b.token(b.other, b.otherPat),
		map[string]any{"type_id": b.typ.ID, "doctor_id": b.doctor.ID, "date": tomorrow, "start_time": "10:00"})
	if r.Code != http.StatusBadRequest {
		t.Fatalf("cross-clinic booking: %d %s", r.Code, r.Raw)
	}
}

func TestBooking_SpecialtyModeAssignsAFreeDoctor(t *testing.T) {
	b := newBookingFixture(t)
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.ChooseBy = tenant_models.BookingChooseBySpecialty })
	// f.doctor (lower id) has a busier day: the request goes to doctor2.
	doctorID := b.doctor.ID
	b.mustCreate(&clinical_models.Appointment{TenantID: b.own.ID, PatientID: b.ownPat2.ID, Title: "x", Status: "scheduled",
		Date: time.Date(2030, 1, 8, 0, 0, 0, 0, portal_services.Location()), StartTime: "11:00", EndTime: "11:30", DoctorID: &doctorID})
	token := b.token(b.own, b.ownPat)
	got := okData(t, b.do(http.MethodPost, "/portal/booking/requests", token,
		map[string]any{"type_id": b.typ.ID, "specialty": "cardiology", "date": tomorrow, "start_time": "09:00"}))
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, numID(got["id"]))
	if row.DoctorID != b.doctor2.ID {
		t.Fatalf("assigned doctor %d, want %d", row.DoctorID, b.doctor2.ID)
	}
	// Same time again: doctor2 is held, f.doctor is free.
	got = okData(t, b.do(http.MethodPost, "/portal/booking/requests", b.token(b.own, b.ownPat2),
		map[string]any{"type_id": b.typ.ID, "specialty": "cardiology", "date": tomorrow, "start_time": "09:00"}))
	var second portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&second, numID(got["id"]))
	if second.DoctorID != b.doctor.ID {
		t.Fatalf("second request doctor %d", second.DoctorID)
	}
}

func TestSettings_BookingValidation(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	tenant := tenant_models.Tenant{Name: "A", Slug: "booking-settings", DisplayToken: "tok-booking-settings"}
	if err := tenantdb.System(db).Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	h := NewSettingsHandler(db, zap.NewNop(), nil)
	put := func(body string) envelope.Response {
		c, _ := testutils.NewGinContext(tenant.ID, 1)
		c.Request = httptest.NewRequest(http.MethodPut, "/portal-settings", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return h.UpdateSettings(c)
	}
	for _, bad := range []string{
		`{"booking":{"request_expiry_hours":36}}`, `{"booking":{"choose_by":"room"}}`, `{"booking":{"min_notice_hours":73}}`,
		`{"booking":{"max_days_ahead":0}}`, `{"booking":{"max_days_ahead":181}}`, `{"booking":{"cancel_notice_hours":-1}}`,
		`{"booking":{"slot_interval_minutes":25}}`,
	} {
		if resp := put(bad); resp.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", bad, resp.Code)
		}
	}
	resp := put(`{"booking":{"enabled":true,"choose_by":"specialty","auto_confirm":true,"request_expiry_hours":48,"min_notice_hours":0,"max_days_ahead":180,"cancel_notice_hours":168,"slot_interval_minutes":30}}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("valid: %+v", resp)
	}
	var saved tenant_models.Tenant
	tenantdb.System(db).First(&saved, tenant.ID)
	got := saved.ParsePortalSettings().Booking
	if !got.Enabled || got.ChooseBy != "specialty" || !got.AutoConfirm || got.RequestExpiryHours != 48 || got.MaxDaysAhead != 180 ||
		got.SlotIntervalMinutes != 30 {
		t.Fatalf("saved = %+v", got)
	}
	// Defaults for clinics that never saved booking settings.
	if d := tenant_models.DefaultPortalSettings().Booking; d.Enabled || d.ChooseBy != "doctor" || d.RequestExpiryHours != 24 ||
		d.MinNoticeHours != 2 || d.MaxDaysAhead != 60 || d.CancelNoticeHours != 24 || d.SlotIntervalMinutes != 0 {
		t.Fatalf("defaults = %+v", d)
	}
}
