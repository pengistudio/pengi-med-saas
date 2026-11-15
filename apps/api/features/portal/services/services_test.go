package portal_services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

// --- captcha ---

func TestTurnstile_Verify(t *testing.T) {
	var got struct{ secret, response, remoteIP string }
	answer, statusCode := `{"success":true}`, http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got.secret, got.response, got.remoteIP = r.PostForm.Get("secret"), r.PostForm.Get("response"), r.PostForm.Get("remoteip")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()
	v := portal_services.NewTurnstile("s3cret", srv.URL, srv.Client())
	ctx := context.Background()

	if ok, err := v.Verify(ctx, " tok ", "1.2.3.4"); !ok || err != nil {
		t.Fatalf("success: %v %v", ok, err)
	}
	if got.secret != "s3cret" || got.response != "tok" || got.remoteIP != "1.2.3.4" {
		t.Fatalf("form sent = %+v", got)
	}

	answer = `{"success":false}`
	if ok, err := v.Verify(ctx, "tok", ""); ok || err != nil {
		t.Fatalf("rejected token: %v %v", ok, err)
	}
	answer, statusCode = `{"success":true}`, http.StatusInternalServerError
	if ok, _ := v.Verify(ctx, "tok", ""); ok {
		t.Fatal("non-200 must fail")
	}
	answer, statusCode = `not json`, http.StatusOK
	if ok, err := v.Verify(ctx, "tok", ""); ok || err == nil {
		t.Fatalf("bad body: %v %v", ok, err)
	}
	// Empty or oversized tokens never reach Cloudflare.
	got.response = "untouched"
	for _, tok := range []string{"", "   ", strings.Repeat("x", 2049)} {
		if ok, err := v.Verify(ctx, tok, ""); ok || err != nil {
			t.Fatalf("token len %d: %v %v", len(tok), ok, err)
		}
	}
	if got.response != "untouched" {
		t.Fatal("an invalid token was sent to the verifier")
	}
}

func TestTurnstile_UnreachableFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	if ok, err := portal_services.NewTurnstile("s", url, nil).Verify(context.Background(), "tok", ""); ok || err == nil {
		t.Fatalf("unreachable: %v %v", ok, err)
	}
}

// --- new patients ---

const validCedula = "1710034065"

var formNow = time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC)

func baseForm() portal_dto.NewPatientForm {
	return portal_dto.NewPatientForm{
		FirstName: "  Ana ", LastName: " Ruiz ", DocumentType: portal_dto.DocumentTypeCedula, Document: validCedula,
		BirthDate: "1990-05-01", Email: " Ana@Example.COM ", Phone: "0991234567", WhatsAppOptIn: true,
	}
}

func TestNormalizeNewPatientForm_Valid(t *testing.T) {
	out, err := portal_services.NormalizeNewPatientForm(baseForm(), portal_models.ChannelEmail, "ana@example.com", formNow)
	if err != nil {
		t.Fatal(err)
	}
	if out.FirstName != "Ana" || out.LastName != "Ruiz" || out.Email != "ana@example.com" || out.Phone != "593991234567" || !out.WhatsAppOptIn {
		t.Fatalf("normalized = %+v", out)
	}
	// WhatsApp channel: the verified phone must be the form's.
	if _, err := portal_services.NormalizeNewPatientForm(baseForm(), portal_models.ChannelWhatsApp, "593991234567", formNow); err != nil {
		t.Fatalf("whatsapp channel: %v", err)
	}
	// Passport: uppercased; opt-in dropped without a phone.
	f := baseForm()
	f.DocumentType, f.Document, f.Phone = portal_dto.DocumentTypePassport, " ab12345 ", ""
	out, err = portal_services.NormalizeNewPatientForm(f, portal_models.ChannelEmail, "ana@example.com", formNow)
	if err != nil || out.Document != "AB12345" || out.WhatsAppOptIn {
		t.Fatalf("passport = %+v %v", out, err)
	}
}

func TestNormalizeNewPatientForm_Rejects(t *testing.T) {
	cases := map[string]func(*portal_dto.NewPatientForm) (channel, contact string){
		"empty first name": func(f *portal_dto.NewPatientForm) (string, string) { f.FirstName = "  "; return "", "" },
		"name too long": func(f *portal_dto.NewPatientForm) (string, string) {
			f.LastName = strings.Repeat("a", 101)
			return "", ""
		},
		"bad cedula": func(f *portal_dto.NewPatientForm) (string, string) { f.Document = "0102030405"; return "", "" },
		"short passport": func(f *portal_dto.NewPatientForm) (string, string) {
			f.DocumentType, f.Document = portal_dto.DocumentTypePassport, "A1"
			return "", ""
		},
		"unknown doc type": func(f *portal_dto.NewPatientForm) (string, string) { f.DocumentType = "ruc"; return "", "" },
		"bad birth date":   func(f *portal_dto.NewPatientForm) (string, string) { f.BirthDate = "01/05/1990"; return "", "" },
		"born today":       func(f *portal_dto.NewPatientForm) (string, string) { f.BirthDate = "2026-11-14"; return "", "" },
		"older than 120":   func(f *portal_dto.NewPatientForm) (string, string) { f.BirthDate = "1900-01-01"; return "", "" },
		"bad phone":        func(f *portal_dto.NewPatientForm) (string, string) { f.Phone = "12"; return "", "" },
		"bad email":        func(f *portal_dto.NewPatientForm) (string, string) { f.Email = "nope"; return "", "" },
		"contact not in form": func(f *portal_dto.NewPatientForm) (string, string) {
			return portal_models.ChannelEmail, "otra@example.com"
		},
		"phone not in form": func(f *portal_dto.NewPatientForm) (string, string) {
			return portal_models.ChannelWhatsApp, "593990000000"
		},
		"unknown channel": func(f *portal_dto.NewPatientForm) (string, string) { return "sms", "ana@example.com" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := baseForm()
			channel, contact := mutate(&f)
			if channel == "" {
				channel, contact = portal_models.ChannelEmail, "ana@example.com"
			}
			if _, err := portal_services.NormalizeNewPatientForm(f, channel, contact, formNow); err != portal_services.ErrInvalidPersonalData {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func newPatientRequest(t *testing.T, tenantID uint, form portal_dto.NewPatientForm) portal_models.AppointmentRequest {
	t.Helper()
	raw, _ := json.Marshal(form)
	return portal_models.AppointmentRequest{TenantID: tenantID, DoctorID: 7, NewPatient: true, NewPatientData: string(raw)}
}

func TestResolveNewPatient(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &doctor_models.Doctor{}, &clinical_models.Patient{},
		&portal_models.AppointmentRequest{}, &audit.AuditLog{})
	sys := tenantdb.System(db)
	form, err := portal_services.NormalizeNewPatientForm(baseForm(), portal_models.ChannelEmail, "ana@example.com", formNow)
	if err != nil {
		t.Fatal(err)
	}

	// A patient with the same document at another clinic is never linked.
	foreign := clinical_models.Patient{TenantID: 2, Document: form.Document, FirstName: "Otra", LastName: "Clínica"}
	if err := sys.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	tx := tenantdb.ForTenant(db, 1)
	id, err := portal_services.ResolveNewPatient(tx, newPatientRequest(t, 1, form), formNow)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 || id == foreign.ID {
		t.Fatalf("resolved to %d (foreign %d)", id, foreign.ID)
	}
	var created clinical_models.Patient
	sys.First(&created, id)
	if created.TenantID != 1 || created.Document != form.Document || created.DoctorID == nil || *created.DoctorID != 7 ||
		!created.WhatsAppOptIn || created.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourcePortal {
		t.Fatalf("created = %+v", created)
	}

	// Again: the clinic's patient is reused; only empty contacts are filled.
	sys.Model(&clinical_models.Patient{}).Where("id = ?", id).Updates(map[string]any{"phone": "", "email": "keep@example.com"})
	again, err := portal_services.ResolveNewPatient(tx, newPatientRequest(t, 1, form), formNow)
	if err != nil || again != id {
		t.Fatalf("reuse: %d %v", again, err)
	}
	var reused clinical_models.Patient
	sys.First(&reused, id)
	if reused.Phone != form.Phone || reused.Email != "keep@example.com" {
		t.Fatalf("reused = %+v", reused)
	}
	var foreignAfter clinical_models.Patient
	sys.First(&foreignAfter, foreign.ID)
	if foreignAfter.Phone != "" || foreignAfter.Email != "" {
		t.Fatalf("foreign patient touched: %+v", foreignAfter)
	}

	// Broken data is an error, not a patient.
	bad := portal_models.AppointmentRequest{TenantID: 1, NewPatientData: "{"}
	if _, err := portal_services.ResolveNewPatient(tx, bad, formNow); err == nil {
		t.Fatal("broken data must fail")
	}
	noDoc := form
	noDoc.Document = ""
	if _, err := portal_services.ResolveNewPatient(tx, newPatientRequest(t, 1, noDoc), formNow); err == nil {
		t.Fatal("missing document must fail")
	}
}

func TestPendingNewPatientForms_OnlyThisClinicsPendingUnexpired(t *testing.T) {
	db := testutils.SetupTestDB(t, &portal_models.AppointmentRequest{}, &audit.AuditLog{})
	sys := tenantdb.System(db)
	form := baseForm()
	raw, _ := json.Marshal(form)
	mk := func(tenantID uint, newPatient bool, status string, expires time.Time) {
		r := portal_models.AppointmentRequest{TenantID: tenantID, DoctorID: 1, AppointmentTypeID: 1, Date: "2026-12-01", StartTime: "09:00", EndTime: "09:30",
			Status: status, ExpiresAt: expires, NewPatient: newPatient, NewPatientData: string(raw)}
		if err := sys.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
	}
	later, earlier := formNow.Add(time.Hour), formNow.Add(-time.Hour)
	mk(1, true, portal_models.RequestPending, later)   // counted
	mk(1, true, portal_models.RequestPending, earlier) // expired
	mk(1, true, portal_models.RequestRejected, later)  // decided
	mk(1, false, portal_models.RequestPending, later)  // a patient's request
	mk(2, true, portal_models.RequestPending, later)   // another clinic

	reqs, forms, err := portal_services.PendingNewPatientForms(tenantdb.ForTenant(db, 1), formNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || len(forms) != 1 || reqs[0].TenantID != 1 || forms[0].Document != form.Document {
		t.Fatalf("pending = %+v / %+v", reqs, forms)
	}
}
