package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
	clinical_models "pengi-med-saas/features/clinical/models"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// clinicalPermsFixture is one clinic on a plan that includes every permission,
// so what a request may do depends only on the caller's role.
type clinicalPermsFixture struct {
	t             *testing.T
	db            *gorm.DB
	tenant, other tenant_models.Tenant
	patient       clinical_models.Patient
	record        clinical_models.MedicalRecord
	appointment   clinical_models.Appointment
}

func newClinicalPermsFixture(t *testing.T) *clinicalPermsFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.MedicalRecord{}, &clinical_models.VitalSigns{})
	f := &clinicalPermsFixture{t: t, db: db}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&f.tenant, &f.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("perm-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-perm-%d-%d", i, now)}
		mustCreate(t, db, tenant)
	}
	f.patient = clinical_models.Patient{TenantID: f.tenant.ID, FirstName: "Ana", LastName: "Paciente", Institution: "H", Document: fmt.Sprintf("DOC-%d", now)}
	mustCreate(t, db, &f.patient)
	f.record = clinical_models.MedicalRecord{TenantID: f.tenant.ID, PatientID: f.patient.ID, Motive: "Control"}
	mustCreate(t, db, &f.record)
	f.appointment = clinical_models.Appointment{TenantID: f.tenant.ID, PatientID: f.patient.ID, Title: "Control", Date: time.Now(), StartTime: "09:00", EndTime: "09:30", Status: "scheduled"}
	mustCreate(t, db, &f.appointment)
	return f
}

func mustCreate(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	if err := db.Create(value).Error; err != nil {
		t.Fatalf("create %T: %v", value, err)
	}
}

// router mounts the appointment and vital signs routes for a caller in tenant
// whose role holds exactly permissionIDs.
func (f *clinicalPermsFixture) router(tenant tenant_models.Tenant, permissionIDs []string) *gin.Engine {
	f.t.Helper()
	role := user_models.Role{Role: fmt.Sprintf("role-%d", time.Now().UnixNano())}
	for _, id := range permissionIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(f.t, f.db, &role)
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	mustCreate(f.t, f.db, &env)

	plan := map[string]bool{}
	for _, catalog := range [][]permission_models.Permission{permission_data.ClinicalPermissions, permission_data.BillingPermissions, permission_data.KanbanPermissions} {
		for _, perm := range catalog {
			plan[perm.ID] = true
		}
	}

	router := gin.New()
	group := router.Group("/clinical", func(c *gin.Context) {
		// What AuthMiddleware, TenantMiddleware and SubscriptionMiddleware leave.
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	})
	registerAppointmentRoutes(group.Group("/appointments"), f.db, clinical_handlers.NewAppointmentHandler(f.db, zap.NewNop()))
	registerVitalSignsRoutes(group.Group("/records"), f.db, clinical_handlers.NewVitalSignsHandler(f.db, zap.NewNop()))
	registerTriageVitalSignsRoutes(group.Group("/appointments"), f.db, clinical_handlers.NewVitalSignsHandler(f.db, zap.NewNop()))
	return router
}

func serve(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestAppointmentRoutes_ContadorIsForbidden(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, role_data.RolePermissionMatrix[role_data.RoleContador])
	id := f.appointment.ID
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/clinical/appointments", ""},
		{http.MethodGet, "/clinical/appointments/today", ""},
		{http.MethodGet, fmt.Sprintf("/clinical/appointments/%d", id), ""},
		{http.MethodGet, fmt.Sprintf("/clinical/appointments/patient/%d", f.patient.ID), ""},
		{http.MethodPost, "/clinical/appointments", "{}"},
		{http.MethodPut, fmt.Sprintf("/clinical/appointments/%d", id), "{}"},
		{http.MethodPut, fmt.Sprintf("/clinical/appointments/%d/status", id), `{"status":"arrived"}`},
		{http.MethodDelete, fmt.Sprintf("/clinical/appointments/%d", id), ""},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("contador %s %s = %d, want 403", tc.method, tc.path, w.Code)
		}
	}
}

func TestAppointmentRoutes_RecepcionistaReadsAndManages(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, role_data.RolePermissionMatrix[role_data.RoleRecepcionista])

	for _, path := range []string{"/clinical/appointments", "/clinical/appointments/today", fmt.Sprintf("/clinical/appointments/%d", f.appointment.ID)} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("recepcionista GET %s = %d, want 200: %s", path, w.Code, w.Body.String())
		}
	}
	status := fmt.Sprintf("/clinical/appointments/%d/status", f.appointment.ID)
	if w := serve(router, http.MethodPut, status, `{"status":"arrived"}`); w.Code != http.StatusOK {
		t.Errorf("recepcionista PUT %s = %d, want 200: %s", status, w.Code, w.Body.String())
	}
}

func TestAppointmentRoutes_ReadOnlyRoleCannotChangeAppointments(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"READ_APPOINTMENT"})

	if w := serve(router, http.MethodGet, "/clinical/appointments", ""); w.Code != http.StatusOK {
		t.Fatalf("GET with READ_APPOINTMENT = %d, want 200", w.Code)
	}
	status := fmt.Sprintf("/clinical/appointments/%d/status", f.appointment.ID)
	if w := serve(router, http.MethodPut, status, `{"status":"arrived"}`); w.Code != http.StatusForbidden {
		t.Fatalf("PUT status without MANAGE_APPOINTMENT = %d, want 403", w.Code)
	}
}

func TestVitalSignsRoutes_EitherPermissionRecords(t *testing.T) {
	for _, perm := range []string{"RECORD_VITAL_SIGNS", "UPDATE_MEDICAL_RECORD"} {
		t.Run(perm, func(t *testing.T) {
			f := newClinicalPermsFixture(t)
			router := f.router(f.tenant, []string{perm})
			path := fmt.Sprintf("/clinical/records/%d/vital-signs", f.record.ID)
			if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusOK {
				t.Fatalf("PUT vital signs with %s alone = %d, want 200: %s", perm, w.Code, w.Body.String())
			}
		})
	}
}

func TestVitalSignsRoutes_TriageReadsWhatItRecorded(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"RECORD_VITAL_SIGNS"})
	path := fmt.Sprintf("/clinical/records/%d/vital-signs", f.record.ID)
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
		t.Fatalf("GET vital signs with RECORD_VITAL_SIGNS = %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestVitalSignsRoutes_NeitherPermissionIsForbidden(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"READ_PATIENT", "READ_APPOINTMENT"})
	path := fmt.Sprintf("/clinical/records/%d/vital-signs", f.record.ID)
	if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusForbidden {
		t.Fatalf("PUT vital signs without either permission = %d, want 403", w.Code)
	}
}

// RECORD_VITAL_SIGNS opens the route, not another clinic's records.
func TestVitalSignsRoutes_AnotherTenantsRecordIsNotFound(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.other, []string{"RECORD_VITAL_SIGNS"})
	path := fmt.Sprintf("/clinical/records/%d/vital-signs", f.record.ID)
	if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusNotFound {
		t.Fatalf("PUT another tenant's vital signs = %d, want 404", w.Code)
	}
	var count int64
	f.db.Model(&clinical_models.VitalSigns{}).Where("medical_record_id = ?", f.record.ID).Count(&count)
	if count != 0 {
		t.Fatalf("vital signs written to another tenant's record")
	}
}

// Appointments of another clinic stay out of reach with the permission.
func TestAppointmentRoutes_AnotherTenantsAppointmentIsNotFound(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.other, role_data.RolePermissionMatrix[role_data.RoleRecepcionista])
	path := fmt.Sprintf("/clinical/appointments/%d", f.appointment.ID)
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusNotFound {
		t.Fatalf("GET another tenant's appointment = %d, want 404", w.Code)
	}
}

// ─── Triage: vital signs on the appointment ──────────────────────────────────

// Reception records the vital signs at triage and reads them back.
func TestTriageVitalSignsRoutes_RecordVitalSignsSavesAndReads(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"READ_APPOINTMENT", "RECORD_VITAL_SIGNS"})
	path := fmt.Sprintf("/clinical/appointments/%d/vital-signs", f.appointment.ID)
	for _, weight := range []string{"70", "71.5"} { // create, then update
		if w := serve(router, http.MethodPut, path, `{"weight":`+weight+`,"blood_pressure":"120/80"}`); w.Code != http.StatusOK {
			t.Fatalf("PUT triage vital signs = %d, want 200: %s", w.Code, w.Body.String())
		}
	}
	w := serve(router, http.MethodGet, path, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"weight":71.5`) {
		t.Fatalf("GET triage vital signs = %d %s, want 200 with the last weight", w.Code, w.Body.String())
	}
	var rows int64
	f.db.Model(&clinical_models.VitalSigns{}).Where("appointment_id = ?", f.appointment.ID).Count(&rows)
	if rows != 1 {
		t.Fatalf("triage vital signs rows = %d, want 1", rows)
	}
}

// Managing the agenda doesn't give access to the measurements.
func TestTriageVitalSignsRoutes_AppointmentPermissionsAloneAreForbidden(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"READ_APPOINTMENT", "MANAGE_APPOINTMENT"})
	path := fmt.Sprintf("/clinical/appointments/%d/vital-signs", f.appointment.ID)
	if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusForbidden {
		t.Fatalf("PUT triage vital signs = %d, want 403", w.Code)
	}
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusForbidden {
		t.Fatalf("GET triage vital signs = %d, want 403", w.Code)
	}
}

// The doctor reads them when opening the consultation, but triage is where
// they are recorded.
func TestTriageVitalSignsRoutes_ReadMedicalRecordReadsOnly(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.tenant, []string{"READ_MEDICAL_RECORD"})
	path := fmt.Sprintf("/clinical/appointments/%d/vital-signs", f.appointment.ID)
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
		t.Fatalf("GET triage vital signs with READ_MEDICAL_RECORD = %d, want 200", w.Code)
	}
	if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusForbidden {
		t.Fatalf("PUT triage vital signs with READ_MEDICAL_RECORD = %d, want 403", w.Code)
	}
}

func TestTriageVitalSignsRoutes_AnotherTenantsAppointmentIsNotFound(t *testing.T) {
	f := newClinicalPermsFixture(t)
	router := f.router(f.other, []string{"RECORD_VITAL_SIGNS"})
	path := fmt.Sprintf("/clinical/appointments/%d/vital-signs", f.appointment.ID)
	if w := serve(router, http.MethodPut, path, `{"weight":70}`); w.Code != http.StatusNotFound {
		t.Fatalf("PUT another tenant's triage vital signs = %d, want 404", w.Code)
	}
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusNotFound {
		t.Fatalf("GET another tenant's triage vital signs = %d, want 404", w.Code)
	}
	var rows int64
	f.db.Model(&clinical_models.VitalSigns{}).Where("appointment_id = ?", f.appointment.ID).Count(&rows)
	if rows != 0 {
		t.Fatalf("vital signs written to another tenant's appointment")
	}
}

// The waiting room only learns whether triage is done, not the measurements.
func TestTodayAppointments_ShowTriageDoneWithoutMeasurements(t *testing.T) {
	f := newClinicalPermsFixture(t)
	weight := 70.0
	mustCreate(t, f.db, &clinical_models.VitalSigns{AppointmentID: &f.appointment.ID, Weight: &weight, BloodPressure: "120/80"})
	router := f.router(f.tenant, []string{"READ_APPOINTMENT"})

	w := serve(router, http.MethodGet, "/clinical/appointments/today", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET today = %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []struct {
			ID         uint `json:"ID"`
			VitalSigns *struct {
				ID            uint     `json:"ID"`
				Weight        *float64 `json:"weight"`
				BloodPressure string   `json:"blood_pressure"`
			} `json:"vital_signs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Data) != 1 {
		t.Fatalf("decode today: %v %s", err, w.Body.String())
	}
	vs := body.Data[0].VitalSigns
	if vs == nil || vs.ID == 0 {
		t.Fatalf("today's appointment doesn't show triage as done: %s", w.Body.String())
	}
	if vs.Weight != nil || vs.BloodPressure != "" {
		t.Fatalf("today's list exposes the measurements: %s", w.Body.String())
	}
}
