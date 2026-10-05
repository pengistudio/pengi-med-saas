package tenant_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantfiles"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_dto "pengi-med-saas/features/tenants/dto"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func newPublicDisplayHandler(t *testing.T) *TenantHandler {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{})
	return NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))
}

func publicContext(token string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/public/appointments/today?token="+token, nil)
	return c
}

// seedDisplayAppointment creates a patient with every sensitive field filled and
// one appointment for today, also with title, notes and location.
func seedDisplayAppointment(t *testing.T, h *TenantHandler, tenantID uint) clinical_models.Appointment {
	t.Helper()
	patient := clinical_models.Patient{
		TenantID:  tenantID,
		FirstName: "Juan Carlos",
		LastName:  "pérez Gómez",
		Document:  "SECRET-DOCUMENT-1712345678",
		Phone:     "SECRET-PHONE-0991234567",
		Email:     "secret-email@example.com",
		BirthDate: time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC),
		Notes:     "SECRET-PATIENT-NOTES",
		Insurance: "SECRET-INSURANCE",
		Diagnosis: "SECRET-DIAGNOSIS",
		APP:       "SECRET-APP",
		APF:       "SECRET-APF",
		APQX:      "SECRET-APQX",
		Allergies: `["SECRET-ALLERGY"]`,
	}
	if err := h.db.Create(&patient).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}
	appt := clinical_models.Appointment{
		TenantID:  tenantID,
		PatientID: patient.ID,
		Title:     "SECRET-TITLE",
		// Midday, so the day is the same in local time and in UTC.
		Date:      time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 12, 0, 0, 0, time.Local),
		StartTime: "09:00",
		EndTime:   "09:30",
		Location:  "SECRET-LOCATION",
		Notes:     "SECRET-APPOINTMENT-NOTES",
		Status:    "arrived",
	}
	if err := h.db.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	return appt
}

func TestGetTodayAppointmentsPublic_ReturnsOnlyTheMinimalDTO(t *testing.T) {
	h := newPublicDisplayHandler(t)
	token := tenant_models.NewDisplayToken()
	tenant := createDisplayTenant(t, h, token)
	appt := seedDisplayAppointment(t, h, tenant.ID)

	resp := h.GetTodayAppointmentsPublic(publicContext(token))
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%s), want 200", resp.Code, resp.Message)
	}
	items, ok := resp.Data.([]tenant_dto.PublicAppointment)
	if !ok {
		t.Fatalf("data is %T, want []tenant_dto.PublicAppointment", resp.Data)
	}
	want := tenant_dto.PublicAppointment{ID: appt.ID, StartTime: "09:00", EndTime: "09:30", Status: "arrived", PatientName: "Juan P."}
	if len(items) != 1 || items[0] != want {
		t.Fatalf("items = %+v, want [%+v]", items, want)
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for key := range decoded[0] {
		switch key {
		case "id", "start_time", "end_time", "status", "patient_name":
		default:
			t.Errorf("unexpected field %q in public response: %s", key, body)
		}
	}
	for _, forbidden := range []string{
		"document", "phone", "email", "birth_date", "diagnosis", "allergies",
		`"app"`, `"apf"`, `"apqx"`, "notes", "insurance", "patient_id", "tenant_id",
		"title", "location", "patient\"", "SECRET", "1980", "Carlos", "rez",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("public response contains %q: %s", forbidden, body)
		}
	}
}

func TestGetTodayAppointmentsPublic_RejectsEveryBadTokenWithTheSame404(t *testing.T) {
	h := newPublicDisplayHandler(t)
	valid := tenant_models.NewDisplayToken()
	createDisplayTenant(t, h, valid)
	legacy := fmt.Sprintf("%08d", time.Now().UnixNano()%100_000_000)
	// A legacy 8-digit code stored on a tenant still never opens the display.
	createDisplayTenant(t, h, legacy)

	var first envelope.Response
	for i, token := range []string{"", legacy, "not-a-token", tenant_models.NewDisplayToken(), valid[:31] + "!"} {
		resp := h.GetTodayAppointmentsPublic(publicContext(token))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("token %q: code = %d, want 404", token, resp.Code)
		}
		if i == 0 {
			first = resp
			continue
		}
		if fmt.Sprintf("%#v", resp) != fmt.Sprintf("%#v", first) {
			t.Fatalf("token %q: response %+v differs from %+v", token, resp, first)
		}
	}
}

func TestGenerateDisplayToken_ReplacesWithLongToken(t *testing.T) {
	h := newDisplayHandler(t)
	old := tenant_models.NewDisplayToken()
	tenant := createDisplayTenant(t, h, old)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	resp := h.GenerateDisplayToken(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", resp.Code)
	}
	got, _ := resp.Data.(gin.H)["token"].(string)
	if !displayTokenRe.MatchString(got) || got == old {
		t.Fatalf("token = %q, want a new 32-char token", got)
	}
	if stored := storedDisplayToken(t, h, tenant.ID); stored != got {
		t.Fatalf("stored = %q, want %q", stored, got)
	}
	if url := resp.Data.(gin.H)["display_url"]; url != testFrontendURL+"/display/waiting-room?token="+got {
		t.Fatalf("display_url = %v, want the new token's link", url)
	}
}

func TestGenerateDisplayToken_KeepsTokenWithoutFrontendURL(t *testing.T) {
	h := newDisplayHandler(t)
	t.Setenv("FRONTEND_URL", "")
	old := tenant_models.NewDisplayToken()
	tenant := createDisplayTenant(t, h, old)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	if resp := h.GenerateDisplayToken(c); resp.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", resp.Code)
	}
	if stored := storedDisplayToken(t, h, tenant.ID); stored != old {
		t.Fatalf("token rotated to %q although the link could not be built", stored)
	}
}

func TestPatientDisplayName(t *testing.T) {
	cases := map[[2]string]string{
		{"Juan Carlos", "Pérez"}: "Juan P.",
		{"ángel", "ñuñez"}:       "ángel Ñ.",
		{"  Ana ", ""}:           "Ana",
		{"", "Zapata"}:           "Z.",
		{"", ""}:                 "",
	}
	for in, want := range cases {
		if got := tenant_dto.PatientDisplayName(in[0], in[1]); got != want {
			t.Errorf("PatientDisplayName(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
