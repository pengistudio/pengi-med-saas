package portal_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	portal_workers "pengi-med-saas/features/portal/workers"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
)

// --- contact codes (real clock) ---

// newContactFixture is the base fixture with the contact-code endpoints and
// the own clinic taking new patients (the other one does not).
func newContactFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.setSettings(f.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled, s.Booking.NewPatients = true, true })
	server := httptest.NewServer(f.graph)
	t.Cleanup(server.Close)
	delivery := portal_services.NewDelivery(f.db, zap.NewNop(), f.mail, whatsapp.New(server.URL, "v23.0", "", ""))
	h := NewFirstAppointmentHandler(f.db, zap.NewNop(), delivery).WithSync().WithCaptcha(nil)
	f.router.POST("/portal/clinics/:slug/new-patient/code", envelope.Handle(h.RequestCode))
	f.router.POST("/portal/clinics/:slug/new-patient/verify", envelope.Handle(h.Verify))
	return f
}

func (f *fixture) contactCode(tenant tenant_models.Tenant, channel, contact string) result {
	return f.do(http.MethodPost, "/portal/clinics/"+tenant.Slug+"/new-patient/code", "", map[string]string{"channel": channel, "contact": contact})
}

func (f *fixture) verifyContact(tenant tenant_models.Tenant, channel, contact, code string) result {
	return f.do(http.MethodPost, "/portal/clinics/"+tenant.Slug+"/new-patient/verify", "", map[string]string{"channel": channel, "contact": contact, "code": code})
}

func (f *fixture) contactCodes(tenantID uint) []portal_models.PortalContactCode {
	var rows []portal_models.PortalContactCode
	tenantdb.System(f.db).Where("tenant_id = ?", tenantID).Order("id").Find(&rows)
	return rows
}

func (f *fixture) seedContactCodes(tenantID uint, contact string, n int, at time.Time) {
	f.t.Helper()
	hash := portal_services.ContactHash(tenantID, portal_models.ChannelEmail, contact)
	rows := make([]portal_models.PortalContactCode, n)
	for i := range rows {
		rows[i] = portal_models.PortalContactCode{TenantID: tenantID, Channel: portal_models.ChannelEmail, ContactHash: hash,
			CodeHash: "x", ExpiresAt: at.Add(portal_services.CodeTTL).UTC()}
		rows[i].CreatedAt = at.UTC()
	}
	if err := tenantdb.System(f.db).Create(&rows).Error; err != nil {
		f.t.Fatal(err)
	}
}

func TestNewPatientCode_IssueAndVerify(t *testing.T) {
	f := newContactFixture(t)
	const email = "maria@example.com"

	r := f.contactCode(f.own, "email", "  Maria@Example.COM ")
	if r.Code != http.StatusOK || r.Body["message"] != "portal.new_patient.code_sent" || string(mustJSON(r.Body["data"])) != `{"expires_in":600}` {
		t.Fatalf("code: %d %s", r.Code, r.Raw)
	}
	code := f.mail.last(email)
	if code == "" {
		t.Fatal("the code must be emailed to the normalized address")
	}
	rows := f.contactCodes(f.own.ID)
	if len(rows) != 1 || strings.Contains(rows[0].ContactHash, "maria") || rows[0].Channel != "email" {
		t.Fatalf("rows = %+v", rows)
	}

	v := f.verifyContact(f.own, "email", "MARIA@example.com", code)
	if v.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", v.Code, v.Raw)
	}
	data := v.Body["data"].(map[string]any)
	claims, err := auth.ParsePortalGuestToken(data["guest_token"].(string))
	if err != nil || claims.TenantID != f.own.ID || claims.Channel != "email" || claims.Contact != email || data["expires_in"] != float64(1800) {
		t.Fatalf("guest token = %+v %v (%v)", claims, err, data)
	}

	t.Run("single use", func(t *testing.T) {
		if r := f.verifyContact(f.own, "email", email, code); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("second use: %d %s", r.Code, r.Raw)
		}
	})
	t.Run("expires", func(t *testing.T) {
		f.contactCode(f.own, "email", "exp@example.com")
		c := f.mail.last("exp@example.com")
		tenantdb.System(f.db).Model(&portal_models.PortalContactCode{}).Where("tenant_id = ?", f.own.ID).Update("expires_at", time.Now().Add(-time.Second).UTC())
		if r := f.verifyContact(f.own, "email", "exp@example.com", c); r.Code != http.StatusUnauthorized || errorCode(r) != "E-PORTAL-002" {
			t.Fatalf("expired: %d %s", r.Code, r.Raw)
		}
	})
	t.Run("max attempts", func(t *testing.T) {
		f.contactCode(f.own, "email", "try@example.com")
		c := f.mail.last("try@example.com")
		wrong := "000000"
		if c == wrong {
			wrong = "111111"
		}
		for i := 0; i < portal_services.MaxAttempts; i++ {
			f.verifyContact(f.own, "email", "try@example.com", wrong)
		}
		if r := f.verifyContact(f.own, "email", "try@example.com", c); r.Code != http.StatusUnauthorized {
			t.Fatalf("right code after %d tries: %d", portal_services.MaxAttempts, r.Code)
		}
	})
	t.Run("another contact or clinic", func(t *testing.T) {
		f.contactCode(f.own, "email", "one@example.com")
		c := f.mail.last("one@example.com")
		if r := f.verifyContact(f.own, "email", "two@example.com", c); r.Code != http.StatusUnauthorized {
			t.Fatalf("other contact: %d", r.Code)
		}
		f.setSettings(f.other, func(s *tenant_models.PortalSettings) { s.Booking.Enabled, s.Booking.NewPatients = true, true })
		if r := f.verifyContact(f.other, "email", "one@example.com", c); r.Code != http.StatusUnauthorized {
			t.Fatalf("other clinic: %d", r.Code)
		}
		f.setSettings(f.other, func(s *tenant_models.PortalSettings) { s.Booking.NewPatients = false })
	})
	t.Run("bad input", func(t *testing.T) {
		for _, tc := range [][2]string{{"email", "not-an-email"}, {"email", "a@b"}, {"sms", "0991234567"}, {"whatsapp", "123"}} {
			if r := f.contactCode(f.own, tc[0], tc[1]); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-005" {
				t.Fatalf("%v: %d %s", tc, r.Code, r.Raw)
			}
		}
		// WhatsApp needs the clinic's channel.
		if r := f.contactCode(f.own, "whatsapp", "0991234567"); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-005" {
			t.Fatalf("whatsapp unavailable: %d %s", r.Code, r.Raw)
		}
		if r := f.verifyContact(f.own, "email", "bad", "123456"); r.Code != http.StatusUnauthorized {
			t.Fatalf("verify malformed: %d", r.Code)
		}
	})
	t.Run("honeypot answers the same and sends nothing", func(t *testing.T) {
		before := len(f.contactCodes(f.own.ID))
		h := f.do(http.MethodPost, "/portal/clinics/"+f.own.Slug+"/new-patient/code", "", map[string]string{"channel": "email", "contact": "bot@example.com", "website": "x"})
		normal := f.contactCode(f.own, "email", "human@example.com")
		if string(h.Raw) != string(normal.Raw) || len(f.contactCodes(f.own.ID)) != before+1 || f.mail.last("bot@example.com") != "" {
			t.Fatalf("honeypot: %s vs %s", h.Raw, normal.Raw)
		}
	})
	t.Run("new patients off: 404", func(t *testing.T) {
		if r := f.contactCode(f.other, "email", email); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-015" {
			t.Fatalf("code: %d %s", r.Code, r.Raw)
		}
		if r := f.verifyContact(f.other, "email", email, "123456"); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-015" {
			t.Fatalf("verify: %d %s", r.Code, r.Raw)
		}
		// Booking off makes the setting ineffective.
		f.setSettings(f.own, func(s *tenant_models.PortalSettings) { s.Booking.Enabled = false })
		if r := f.contactCode(f.own, "email", email); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-015" {
			t.Fatalf("booking off: %d %s", r.Code, r.Raw)
		}
	})
}

func TestNewPatientCode_PerContactBudget(t *testing.T) {
	f := newContactFixture(t)
	for i := 0; i < portal_services.MaxCodesPerWindow+2; i++ {
		if r := f.contactCode(f.own, "email", "ana.n@example.com"); r.Code != http.StatusOK {
			t.Fatalf("code %d: %d", i, r.Code)
		}
	}
	if n := len(f.contactCodes(f.own.ID)); n != portal_services.MaxCodesPerWindow {
		t.Fatalf("codes in the window = %d", n)
	}
	// Daily cap: spread over the day, outside the window.
	f.seedContactCodes(f.own.ID, "day@example.com", portal_services.MaxCodesPerContactPerDay-1, time.Now().Add(-3*time.Hour))
	f.contactCode(f.own, "email", "day@example.com")
	f.contactCode(f.own, "email", "day@example.com")
	if len(f.mail.sent["day@example.com"]) != 1 {
		t.Fatalf("daily cap: %d emails", len(f.mail.sent["day@example.com"]))
	}
}

func TestNewPatientCode_SharesClinicAndGlobalBudgets(t *testing.T) {
	t.Run("sign-in codes fill the clinic budget", func(t *testing.T) {
		f := newContactFixture(t)
		f.seedCodes(f.own.ID, f.ownPat2.ID, portal_services.MaxCodesPerTenantPerHour, time.Now().Add(-10*time.Minute))
		if r := f.contactCode(f.own, "email", "x@example.com"); r.Code != http.StatusOK {
			t.Fatalf("answer: %d", r.Code)
		}
		if len(f.contactCodes(f.own.ID)) != 0 || f.mail.calls != 0 {
			t.Fatal("nothing must be sent over the clinic budget")
		}
	})
	t.Run("contact codes fill the clinic budget", func(t *testing.T) {
		f := newContactFixture(t)
		f.seedContactCodes(f.own.ID, "y@example.com", portal_services.MaxCodesPerTenantPerHour, time.Now().Add(-10*time.Minute))
		f.login(f.own, f.ownPat.Document)
		if len(f.codes(f.ownPat.ID)) != 0 || f.mail.calls != 0 {
			t.Fatal("contact codes must count toward the clinic's sign-in budget")
		}
	})
	t.Run("global", func(t *testing.T) {
		f := newContactFixture(t)
		t.Setenv("PORTAL_MAX_CODES_PER_DAY", "4")
		f.seedContactCodes(f.other.ID, "z@example.com", 4, time.Now().Add(-time.Hour))
		f.contactCode(f.own, "email", "w@example.com")
		f.login(f.own, f.ownPat.Document)
		if f.mail.calls != 0 {
			t.Fatal("the global budget counts contact codes")
		}
	})
}

func TestNewPatientCode_WhatsApp(t *testing.T) {
	f := newContactFixture(t)
	f.connectWhatsApp(f.own, whatsapp_models.TemplateStatusApproved)
	if r := f.contactCode(f.own, "whatsapp", "099 123 4567"); r.Code != http.StatusOK {
		t.Fatalf("code: %d %s", r.Code, r.Raw)
	}
	sends := f.graph.sends()
	if len(sends) != 1 || !strings.Contains(sends[0], `"593991234567"`) || f.mail.calls != 0 {
		t.Fatalf("sends = %v, emails %d", sends, f.mail.calls)
	}
	var msg whatsapp_models.WhatsAppMessage
	tenantdb.System(f.db).Where("tenant_id = ? AND kind = ?", f.own.ID, whatsapp_models.KindPortalCode).First(&msg)
	if msg.PatientID != nil || strings.ContainsAny(msg.Body, "0123456789") {
		t.Fatalf("message = %+v", msg)
	}
	if rows := f.contactCodes(f.own.ID); len(rows) != 1 || rows[0].Channel != "whatsapp" {
		t.Fatalf("rows = %+v", rows)
	}
}

// --- new-patient requests (frozen clock) ---

const (
	guestEmail  = "maria@example.com"
	validCedula = "1710034065"
)

func newGuestFixture(t *testing.T) *bookingFixture {
	t.Helper()
	b := newBookingFixture(t)
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.NewPatients = true })
	return b
}

func (b *bookingFixture) guest(tenant tenant_models.Tenant, channel, contact string) string {
	b.t.Helper()
	token, err := auth.GeneratePortalGuestToken(tenant.ID, channel, contact)
	if err != nil {
		b.t.Fatal(err)
	}
	return token
}

func newPatientForm(over map[string]any) map[string]any {
	p := map[string]any{
		"first_name": "María José", "last_name": "Pérez", "document_type": "cedula", "document": validCedula,
		"birth_date": "1990-05-01", "phone": "0991234567", "email": guestEmail, "whatsapp_opt_in": true,
	}
	for k, v := range over {
		p[k] = v
	}
	return p
}

func (b *bookingFixture) newPatientRequest(token, start string, patient map[string]any) result {
	return b.do(http.MethodPost, "/portal/new-patient/requests", token, map[string]any{
		"type_id": b.typ.ID, "doctor_id": b.doctor.ID, "date": tomorrow, "start_time": start, "reason": "Primera vez",
		"patient": patient,
	})
}

func TestNewPatientRequest_GuestOpensOnlyTheCatalog(t *testing.T) {
	b := newGuestFixture(t)
	g := b.guest(b.own, "email", guestEmail)
	if r := b.do(http.MethodGet, "/portal/booking/options", g, nil); r.Code != http.StatusOK {
		t.Fatalf("options: %d %s", r.Code, r.Raw)
	}
	if got := b.times(g, tomorrow, fmt.Sprintf("doctor_id=%d", b.doctor.ID)); len(got) == 0 {
		t.Fatal("slots must answer a guest")
	}
	for _, path := range []string{"/portal/booking/requests", "/portal/me", "/portal/appointments"} {
		if r := b.do(http.MethodGet, path, g, nil); r.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, r.Code)
		}
	}
	// A patient session token can't create a new-patient request.
	if r := b.newPatientRequest(b.token(b.own, b.ownPat), "09:00", newPatientForm(nil)); r.Code != http.StatusUnauthorized {
		t.Fatalf("session token: %d", r.Code)
	}
	// Guest of a clinic not taking new patients.
	if r := b.do(http.MethodGet, "/portal/booking/options", b.guest(b.other, "email", guestEmail), nil); r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-015" {
		t.Fatalf("other clinic guest: %d %s", r.Code, r.Raw)
	}
}

func TestNewPatientRequest_AlwaysPending(t *testing.T) {
	b := newGuestFixture(t)
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.AutoConfirm = true })
	r := b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil))
	got := okData(t, r)
	if r.Body["message"] != "portal.new_patient.request.create.success" || got["status"] != "pending" || got["contact_channel"] != "email" ||
		got["type_name"] != "Consulta" || got["end_time"] != "09:30" || got["doctor"].(map[string]any)["full_name"] != b.doctor.FullName {
		t.Fatalf("response = %s", r.Raw)
	}
	keys := []string{}
	for k := range got {
		keys = append(keys, k)
	}
	if len(keys) != 9 {
		t.Fatalf("response fields = %v", keys)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, numID(got["id"]))
	if row.PatientID != 0 || !row.NewPatient || row.ContactValue != guestEmail || row.AppointmentID != nil {
		t.Fatalf("row = %+v", row)
	}
	var form portal_dto.NewPatientForm
	if err := json.Unmarshal([]byte(row.NewPatientData), &form); err != nil || form.Phone != "593991234567" || form.Document != validCedula {
		t.Fatalf("form = %+v %v", form, err)
	}
	var appts int64
	tenantdb.System(b.db).Model(&clinical_models.Appointment{}).Count(&appts)
	if appts != 0 {
		t.Fatal("no appointment before approval")
	}
	if n := b.notifications(); len(n) != 1 || n[0] != "notification.portal.booking.new_patient_request" {
		t.Fatalf("notifications = %v", n)
	}
	// It holds the slot: a patient can't take it, nor another new patient.
	if r := b.request(b.token(b.own, b.ownPat), "09:00", nil); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-010" {
		t.Fatalf("held slot: %d %s", r.Code, r.Raw)
	}
	other := newPatientForm(map[string]any{"email": "otro@example.com", "document": "1700000001"})
	if r := b.newPatientRequest(b.guest(b.own, "email", "otro@example.com"), "09:00", other); r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-010" {
		t.Fatalf("slot taken: %d %s", r.Code, r.Raw)
	}
}

func TestNewPatientRequest_InvalidPersonalData(t *testing.T) {
	b := newGuestFixture(t)
	g := b.guest(b.own, "email", guestEmail)
	cases := map[string]map[string]any{
		"bad cédula":        {"document": "1710034066"},
		"short passport":    {"document_type": "passport", "document": "AB12"},
		"passport symbols":  {"document_type": "passport", "document": "AB-12345"},
		"unknown doc type":  {"document_type": "ruc"},
		"empty first name":  {"first_name": "  "},
		"long last name":    {"last_name": strings.Repeat("a", 101)},
		"birth in future":   {"birth_date": "2030-02-01"},
		"born today":        {"birth_date": "2030-01-07"},
		"older than 120":    {"birth_date": "1909-01-01"},
		"bad birth date":    {"birth_date": "01/05/1990"},
		"bad phone":         {"phone": "12"},
		"bad email":         {"email": "maria@"},
		"contact mismatch":  {"email": "someone@example.com"},
		"no verified email": {"email": ""},
	}
	for name, over := range cases {
		if r := b.newPatientRequest(g, "09:00", newPatientForm(over)); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-016" {
			t.Fatalf("%s: %d %s", name, r.Code, r.Raw)
		}
	}
	wa := b.guest(b.own, "whatsapp", "593991234567")
	if r := b.newPatientRequest(wa, "09:00", newPatientForm(map[string]any{"phone": "0997654321"})); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-016" {
		t.Fatalf("whatsapp mismatch: %d %s", r.Code, r.Raw)
	}
	// The verified phone in another format matches; a passport is uppercased.
	r := b.newPatientRequest(wa, "09:00", newPatientForm(map[string]any{"phone": "+593 99 123 4567", "email": "", "document_type": "passport", "document": "ab123456"}))
	okData(t, r)
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).Order("id DESC").First(&row)
	if !strings.Contains(row.NewPatientData, `"document":"AB123456"`) || row.ContactChannel != "whatsapp" {
		t.Fatalf("row = %+v", row)
	}
	// Bad slot data stays E-PORTAL-005.
	bad := b.do(http.MethodPost, "/portal/new-patient/requests", b.guest(b.own, "email", "z@example.com"), map[string]any{
		"type_id": 999, "doctor_id": b.doctor.ID, "date": tomorrow, "start_time": "10:00",
		"patient": newPatientForm(map[string]any{"email": "z@example.com", "document": "1700000001"}),
	})
	if bad.Code != http.StatusBadRequest || errorCode(bad) != "E-PORTAL-005" {
		t.Fatalf("bad slot: %d %s", bad.Code, bad.Raw)
	}
}

func TestNewPatientRequest_Limits(t *testing.T) {
	b := newGuestFixture(t)
	okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil)))

	// Same contact, another document.
	r := b.newPatientRequest(b.guest(b.own, "email", guestEmail), "10:00", newPatientForm(map[string]any{"document": "1700000001"}))
	if r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-017" {
		t.Fatalf("same contact: %d %s", r.Code, r.Raw)
	}
	// Same document, another contact.
	r = b.newPatientRequest(b.guest(b.own, "email", "b@example.com"), "10:00", newPatientForm(map[string]any{"email": "b@example.com"}))
	if r.Code != http.StatusConflict || errorCode(r) != "E-PORTAL-017" {
		t.Fatalf("same document: %d %s", r.Code, r.Raw)
	}
	// Clinic cap.
	for i := 0; i < portal_services.MaxPendingNewPatientRequests-1; i++ {
		data, _ := json.Marshal(portal_dto.NewPatientForm{FirstName: "X", LastName: "Y", DocumentType: "passport", Document: fmt.Sprintf("PASS%04d", i)})
		b.mustCreate(&portal_models.AppointmentRequest{TenantID: b.own.ID, DoctorID: b.doctor2.ID, AppointmentTypeID: b.typ.ID,
			Date: "2030-01-20", StartTime: fmt.Sprintf("%02d:00", i%24), EndTime: "00:30", Status: portal_models.RequestPending,
			ExpiresAt: bookingNow.Add(24 * time.Hour), NewPatient: true, NewPatientData: string(data),
			ContactChannel: "email", ContactValue: fmt.Sprintf("seed%d@example.com", i)})
	}
	r = b.newPatientRequest(b.guest(b.own, "email", "c@example.com"), "10:00", newPatientForm(map[string]any{"email": "c@example.com", "document": "1700000001"}))
	if r.Code != http.StatusTooManyRequests || errorCode(r) != "E-PORTAL-018" {
		t.Fatalf("clinic cap: %d %s", r.Code, r.Raw)
	}
	// New patients turned off after the token was issued.
	b.set(b.own, func(s *tenant_models.PortalSettings) { s.Booking.NewPatients = false })
	r = b.newPatientRequest(b.guest(b.own, "email", "d@example.com"), "11:00", newPatientForm(map[string]any{"email": "d@example.com"}))
	if r.Code != http.StatusNotFound || errorCode(r) != "E-PORTAL-015" {
		t.Fatalf("off: %d %s", r.Code, r.Raw)
	}
}

// The guest never learns whether the document is a patient of the clinic.
func TestNewPatientRequest_SameAnswerWhetherTheDocumentExists(t *testing.T) {
	b := newGuestFixture(t)
	b.patient(b.own.ID, "1700000001", "", "")
	strip := func(r result) string {
		d := okData(t, r)
		for _, k := range []string{"id", "start_time", "end_time", "expires_at"} {
			delete(d, k)
		}
		r.Body["data"] = d
		return string(mustJSON(r.Body))
	}
	known := b.newPatientRequest(b.guest(b.own, "email", "k@example.com"), "09:00", newPatientForm(map[string]any{"email": "k@example.com", "document": "1700000001"}))
	unknown := b.newPatientRequest(b.guest(b.own, "email", "u@example.com"), "10:00", newPatientForm(map[string]any{"email": "u@example.com"}))
	if strip(known) != strip(unknown) {
		t.Fatalf("answers differ:\n%s\n%s", known.Raw, unknown.Raw)
	}
}

func (b *bookingFixture) inboxItem(id uint) map[string]any {
	b.t.Helper()
	r := b.staff(http.MethodGet, "/appointment-requests?status=all", b.own, "")
	for _, it := range r.Body["data"].([]any) {
		item := it.(map[string]any)
		if numID(item["id"]) == id {
			return item
		}
	}
	b.t.Fatalf("request %d not in the inbox: %s", id, r.Raw)
	return nil
}

func TestNewPatientRequest_ApproveCreatesThePatient(t *testing.T) {
	b := newGuestFixture(t)
	got := okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil)))
	id := numID(got["id"])

	item := b.inboxItem(id)
	np, _ := item["new_patient"].(map[string]any)
	if item["patient"] != nil || item["matching_patient"] != nil || np == nil || np["document"] != validCedula ||
		np["document_type"] != "cedula" || np["birth_date"] != "1990-05-01" || np["contact_channel"] != "email" || np["email"] != guestEmail {
		t.Fatalf("inbox item = %v", item)
	}

	approved := okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/approve", id), b.own, ""))
	if approved["status"] != "approved" || approved["patient"] == nil {
		t.Fatalf("approve = %v", approved)
	}
	var p clinical_models.Patient
	if err := tenantdb.System(b.db).Where("tenant_id = ? AND document = ?", b.own.ID, validCedula).First(&p).Error; err != nil {
		t.Fatal(err)
	}
	if p.FirstName != "María José" || p.LastName != "Pérez" || p.Email != guestEmail || p.Phone != "593991234567" ||
		p.BirthDate.UTC().Format("2006-01-02") != "1990-05-01" || p.DoctorID == nil || *p.DoctorID != b.doctor.ID ||
		!p.WhatsAppOptIn || p.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourcePortal || p.WhatsAppOptInAt == nil {
		t.Fatalf("patient = %+v", p)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, id)
	var appt clinical_models.Appointment
	tenantdb.System(b.db).First(&appt, *row.AppointmentID)
	if row.PatientID != p.ID || appt.PatientID != p.ID {
		t.Fatalf("request %+v appointment %+v", row, appt)
	}
	if n := b.mail.noticesTo(guestEmail); len(n) != 1 || !strings.Contains(n[0].Subject, "confirmada") {
		t.Fatalf("approval notice = %+v", n)
	}
}

func TestNewPatientRequest_ApproveLinksAnExistingPatient(t *testing.T) {
	b := newGuestFixture(t)
	existing := clinical_models.Patient{TenantID: b.own.ID, Document: validCedula, FirstName: "Maria", LastName: "Original", Phone: "022345678"}
	b.mustCreate(&existing)
	// A patient of another clinic with the same document is not a match.
	b.patient(b.other.ID, "1700000001", "", "")

	got := okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(map[string]any{"whatsapp_opt_in": false})))
	id := numID(got["id"])
	item := b.inboxItem(id)
	m, _ := item["matching_patient"].(map[string]any)
	if m == nil || numID(m["id"]) != existing.ID || m["last_name"] != "Original" {
		t.Fatalf("matching = %v", item)
	}
	okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/approve", id), b.own, ""))

	var patients []clinical_models.Patient
	tenantdb.System(b.db).Where("tenant_id = ? AND document = ?", b.own.ID, validCedula).Find(&patients)
	if len(patients) != 1 {
		t.Fatalf("patients = %d", len(patients))
	}
	p := patients[0]
	if p.ID != existing.ID || p.FirstName != "Maria" || p.LastName != "Original" || p.Phone != "022345678" || p.Email != guestEmail || p.WhatsAppOptIn {
		t.Fatalf("linked patient = %+v", p)
	}
	var row portal_models.AppointmentRequest
	tenantdb.System(b.db).First(&row, id)
	if row.PatientID != existing.ID {
		t.Fatalf("request patient = %d", row.PatientID)
	}
	if item := b.inboxItem(id); item["matching_patient"] != nil || item["patient"] == nil || item["new_patient"] == nil {
		t.Fatalf("approved item = %v", item)
	}
}

func TestNewPatientRequest_RejectAndExpiryNotifyTheContact(t *testing.T) {
	b := newGuestFixture(t)
	got := okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil)))
	okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/reject", numID(got["id"])), b.own, `{"reason":"Sin cupo"}`))
	n := b.mail.noticesTo(guestEmail)
	if len(n) != 1 || !strings.Contains(strings.Join(n[0].Lines, " "), "María José") || !strings.Contains(strings.Join(n[0].Lines, " "), "Sin cupo") {
		t.Fatalf("reject notice = %+v", n)
	}
	var patients int64
	tenantdb.System(b.db).Model(&clinical_models.Patient{}).Where("document = ?", validCedula).Count(&patients)
	if patients != 0 {
		t.Fatal("a rejected request creates no patient")
	}

	// Verified by WhatsApp, no approved template: the form's email.
	wa := b.guest(b.own, "whatsapp", "593987654321")
	okData(t, b.newPatientRequest(wa, "10:00", newPatientForm(map[string]any{"phone": "0987654321", "email": "wa@example.com", "document": "1700000001"})))
	worker := portal_workers.NewRequestExpiryScheduler(b.db, zap.NewNop(), b.delivery).
		WithClock(func() time.Time { return bookingNow.Add(25 * time.Hour) })
	if n := worker.Tick(); n != 1 {
		t.Fatalf("expired = %d", n)
	}
	if n := b.mail.noticesTo("wa@example.com"); len(n) != 1 || !strings.Contains(n[0].Subject, "no pudo confirmar") {
		t.Fatalf("expiry notice = %+v", n)
	}
}

func TestNewPatientRequest_TenantIsolation(t *testing.T) {
	b := newGuestFixture(t)
	b.set(b.other, func(s *tenant_models.PortalSettings) { s.Booking.NewPatients = true })
	got := okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil)))
	id := numID(got["id"])
	if r := b.staff(http.MethodGet, "/appointment-requests?status=all", b.other, ""); listLen(t, r) != 0 {
		t.Fatalf("other inbox = %s", r.Raw)
	}
	if r := b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/approve", id), b.other, ""); r.Code != http.StatusNotFound {
		t.Fatalf("other approve: %d", r.Code)
	}
	// A guest of the other clinic can't book this clinic's doctor, and its
	// limits don't see this clinic's request.
	r := b.newPatientRequest(b.guest(b.other, "email", guestEmail), "10:00", newPatientForm(nil))
	if r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-005" {
		t.Fatalf("cross-clinic booking: %d %s", r.Code, r.Raw)
	}
}

// The approval notice goes to the verified contact, not to the record's
// contact (a linked patient may have another email, or none).
func TestNewPatientRequest_ApprovalNoticeGoesToTheVerifiedContact(t *testing.T) {
	b := newGuestFixture(t)
	b.mustCreate(&clinical_models.Patient{TenantID: b.own.ID, Document: validCedula, FirstName: "Maria", LastName: "Original", Email: "old@example.com"})
	got := okData(t, b.newPatientRequest(b.guest(b.own, "email", guestEmail), "09:00", newPatientForm(nil)))
	okData(t, b.staff(http.MethodPost, fmt.Sprintf("/appointment-requests/%d/approve", numID(got["id"])), b.own, ""))
	n := b.mail.noticesTo(guestEmail)
	if len(n) != 1 || !strings.Contains(n[0].Subject, "confirmada") || !strings.Contains(strings.Join(n[0].Lines, " "), "María José") {
		t.Fatalf("approval notice = %+v", n)
	}
	if len(b.mail.noticesTo("old@example.com")) != 0 {
		t.Fatal("the record's email must not get the notice")
	}
}
