package routes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	ai_handlers "pengi-med-saas/features/ai/handlers"
	ai_models "pengi-med-saas/features/ai/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	settings_models "pengi-med-saas/features/settings/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// aiRouter mounts the AI routes for a caller whose role holds exactly
// rolePerms, on a plan that includes planPerms, in a clinic that opted in or
// not. No provider key anywhere (AI_API_KEY empty, nothing in backoffice).
func aiRouter(t *testing.T, rolePerms, planPerms []string, optIn bool) (*gin.Engine, clinical_models.Patient, clinical_models.Patient) {
	t.Helper()
	t.Setenv("AI_API_KEY", "")
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.MedicalRecord{}, &clinical_models.ExamOrder{},
		&clinical_models.MedicalCertificate{}, &clinical_models.MedicalReport{}, &clinical_models.SOAPRecord{},
		&clinical_models.Prescription{}, &clinical_models.PrescriptionItem{}, &clinical_models.VitalSigns{},
		&clinical_models.ExamOrderItem{}, &clinical_models.PatientAttachment{}, &company_models.Company{},
		&company_models.Plan{}, &company_models.Subscription{}, &settings_models.SystemSetting{},
		&ai_models.AIUsageLog{}, &ai_models.AISummary{})
	now := time.Now().UnixNano()
	settings := `{}`
	if optIn {
		settings = `{"ai":{"enabled":true}}`
	}
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("ai-perm-%d", now), DisplayToken: fmt.Sprintf("tok-ai-perm-%d", now), UISettings: settings}
	mustCreate(t, db, &tenant)
	patient := clinical_models.Patient{TenantID: tenant.ID, FirstName: "Ana", APP: "Asma"}
	mustCreate(t, db, &patient)
	other := tenant_models.Tenant{Name: "Other", Slug: fmt.Sprintf("ai-perm-o-%d", now), DisplayToken: fmt.Sprintf("tok-ai-perm-o-%d", now), UISettings: settings}
	mustCreate(t, db, &other)
	otherPatient := clinical_models.Patient{TenantID: other.ID, FirstName: "Luis", APP: "Diabetes"}
	mustCreate(t, db, &otherPatient)

	role := user_models.Role{Role: fmt.Sprintf("role-%d", now)}
	for _, id := range rolePerms {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(t, db, &role)
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	mustCreate(t, db, &env)
	plan := map[string]bool{}
	for _, id := range planPerms {
		plan[id] = true
	}

	router := gin.New()
	group := router.Group("/ai", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", int64(7))
		c.Set("environment_id", env.ID)
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	})
	registerAITenantRoutes(group, db, ai_handlers.NewAIHandler(db, zap.NewNop(), nil))
	return router, patient, otherPatient
}

var aiPlan = []string{"USE_AI_SUMMARY", "USE_AI_SCRIBE", "READ_MEDICAL_RECORD", "CREATE_MEDICAL_RECORD"}

func TestAIRoutes_SummaryNeedsAIAndRecordPermissions(t *testing.T) {
	cases := []struct {
		name       string
		role, plan []string
	}{
		{"plan without AI", []string{"USE_AI_SUMMARY", "READ_MEDICAL_RECORD"}, []string{"READ_MEDICAL_RECORD"}},
		{"role without USE_AI_SUMMARY", []string{"READ_MEDICAL_RECORD"}, aiPlan},
		{"role without READ_MEDICAL_RECORD", []string{"USE_AI_SUMMARY"}, aiPlan},
		{"recepcionista", role_data.RolePermissionMatrix[role_data.RoleRecepcionista], aiPlan},
	}
	for _, tc := range cases {
		router, patient, _ := aiRouter(t, tc.role, tc.plan, true)
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			path := fmt.Sprintf("/ai/patients/%d/summary", patient.ID)
			if w := serve(router, method, path, ""); w.Code != http.StatusForbidden {
				t.Errorf("%s: %s %s = %d, want 403", tc.name, method, path, w.Code)
			}
		}
	}
}

func TestAIRoutes_DoctorGetsThroughAndNotConfiguredIsExplicit(t *testing.T) {
	router, patient, _ := aiRouter(t, role_data.RolePermissionMatrix[role_data.RoleDoctor], aiPlan, true)
	path := fmt.Sprintf("/ai/patients/%d/summary", patient.ID)
	if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
		t.Fatalf("GET summary = %d %s, want 200 (no cached summary)", w.Code, w.Body.String())
	}
	if w := serve(router, http.MethodGet, "/ai/status", ""); w.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w.Code)
	}
	w := serve(router, http.MethodPost, path, "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST summary without AI_API_KEY = %d %s, want 503", w.Code, w.Body.String())
	}
}

func TestAIRoutes_ClinicMustOptIn(t *testing.T) {
	router, patient, _ := aiRouter(t, role_data.RolePermissionMatrix[role_data.RoleDoctor], aiPlan, false)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if w := serve(router, method, fmt.Sprintf("/ai/patients/%d/summary", patient.ID), ""); w.Code != http.StatusForbidden {
			t.Errorf("%s summary without opt-in = %d, want 403", method, w.Code)
		}
	}
}

func TestAIRoutes_OtherTenantsPatientIsNotFound(t *testing.T) {
	router, _, otherPatient := aiRouter(t, role_data.RolePermissionMatrix[role_data.RoleDoctor], aiPlan, true)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if w := serve(router, method, fmt.Sprintf("/ai/patients/%d/summary", otherPatient.ID), ""); w.Code != http.StatusNotFound {
			t.Errorf("%s other tenant's patient = %d, want 404", method, w.Code)
		}
	}
}

func TestAIRoutes_ScribeGating(t *testing.T) {
	body := func(patientID uint) string {
		return fmt.Sprintf(`{"patient_id":%d,"notes":"Tos de 3 días"}`, patientID)
	}
	for _, tc := range []struct {
		name       string
		role, plan []string
	}{
		{"plan without AI", []string{"USE_AI_SCRIBE", "CREATE_MEDICAL_RECORD"}, []string{"CREATE_MEDICAL_RECORD"}},
		{"role without USE_AI_SCRIBE", []string{"CREATE_MEDICAL_RECORD"}, aiPlan},
		{"role without CREATE_MEDICAL_RECORD", []string{"USE_AI_SCRIBE"}, aiPlan},
		{"recepcionista", role_data.RolePermissionMatrix[role_data.RoleRecepcionista], aiPlan},
	} {
		router, patient, _ := aiRouter(t, tc.role, tc.plan, true)
		if w := serve(router, http.MethodPost, "/ai/scribe", body(patient.ID)); w.Code != http.StatusForbidden {
			t.Errorf("%s: POST /ai/scribe = %d, want 403", tc.name, w.Code)
		}
	}

	doctor := role_data.RolePermissionMatrix[role_data.RoleDoctor]
	router, patient, otherPatient := aiRouter(t, doctor, aiPlan, true)
	if w := serve(router, http.MethodPost, "/ai/scribe", body(patient.ID)); w.Code != http.StatusServiceUnavailable {
		t.Errorf("doctor without AI key = %d %s, want 503", w.Code, w.Body.String())
	}
	if w := serve(router, http.MethodPost, "/ai/scribe", `{"patient_id":1,"notes":""}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty notes = %d, want 400", w.Code)
	}
	if w := serve(router, http.MethodPost, "/ai/scribe", body(otherPatient.ID)); w.Code != http.StatusNotFound {
		t.Errorf("other tenant's patient = %d, want 404", w.Code)
	}
	if w := serve(router, http.MethodGet, "/ai/status", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"scribe":false`) {
		t.Errorf("status = %d %s", w.Code, w.Body.String())
	}

	optOut, patient2, _ := aiRouter(t, doctor, aiPlan, false)
	if w := serve(optOut, http.MethodPost, "/ai/scribe", body(patient2.ID)); w.Code != http.StatusForbidden {
		t.Errorf("opt-out = %d, want 403", w.Code)
	}
}
