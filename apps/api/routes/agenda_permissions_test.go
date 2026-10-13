package routes

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	agenda_handlers "pengi-med-saas/features/agenda/handlers"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	doctor_models "pengi-med-saas/features/doctors/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// agendaRouter mounts the agenda routes for user 7 whose role holds exactly
// permissionIDs, on a plan with every clinical permission and MANAGE_DOCTORS.
// "mine" is linked to user 7, "colleague" is not.
func agendaRouter(t *testing.T, permissionIDs []string) (router *gin.Engine, mine, colleague doctor_models.Doctor) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &doctor_models.Doctor{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&agenda_models.DoctorSchedule{}, &agenda_models.ScheduleBlock{}, &agenda_models.AppointmentType{})
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("ag-perm-%d", now), DisplayToken: fmt.Sprintf("tok-ag-perm-%d", now)}
	mustCreate(t, db, &tenant)
	userID := uint(7)
	mine = doctor_models.Doctor{TenantID: tenant.ID, UserID: &userID, FullName: "Dr. Mío", Specialty: "pediatrics", Color: "#2563EB", Active: true}
	mustCreate(t, db, &mine)
	colleague = doctor_models.Doctor{TenantID: tenant.ID, FullName: "Dra. Colega", Specialty: "pediatrics", Color: "#DC2626", Active: true}
	mustCreate(t, db, &colleague)

	role := user_models.Role{Role: fmt.Sprintf("role-%d", now)}
	for _, id := range permissionIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(t, db, &role)
	env := user_models.Environment{UserID: userID, CompanyID: 1, RoleID: role.ID}
	mustCreate(t, db, &env)
	plan := map[string]bool{"MANAGE_DOCTORS": true}
	for _, ids := range role_data.RolePermissionMatrix {
		for _, id := range ids {
			plan[id] = true
		}
	}

	router = gin.New()
	ctx := func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
		c.Set("user_id", int64(env.UserID))
		c.Set("username", "tester")
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	}
	registerAgendaTenantRoutes(router.Group("/doctors", ctx), router.Group("/agenda", ctx), db,
		agenda_handlers.NewAgendaHandler(db, zap.NewNop()))
	return router, mine, colleague
}

const scheduleBody = `{"slots":[{"weekday":1,"start_time":"08:00","end_time":"12:00"}]}`
const blockBody = `{"start_date":"2026-10-19","end_date":"2026-10-19"}`

func TestAgendaRoutes_DoctorManagesOnlyOwnSchedule(t *testing.T) {
	router, mine, colleague := agendaRouter(t, role_data.RolePermissionMatrix[role_data.RoleDoctor])
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, "/doctors/me/schedule", scheduleBody},
		{http.MethodGet, "/doctors/me/schedule", ""},
		{http.MethodPost, "/doctors/me/blocks", blockBody},
		{http.MethodGet, "/doctors/me/blocks", ""},
		{http.MethodGet, fmt.Sprintf("/doctors/%d/schedule", colleague.ID), ""},
		{http.MethodGet, "/agenda/appointment-types", ""},
		{http.MethodGet, fmt.Sprintf("/agenda/availability?doctor_id=%d&date=2026-10-19&start_time=09:00&end_time=09:30", mine.ID), ""},
		{http.MethodGet, "/agenda/range?from=2026-10-19&to=2026-10-25", ""},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusOK {
			t.Errorf("doctor %s %s = %d, want 200: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, fmt.Sprintf("/doctors/%d/schedule", colleague.ID), scheduleBody},
		{http.MethodPut, fmt.Sprintf("/doctors/%d/schedule", mine.ID), scheduleBody}, // own goes through /me
		{http.MethodPost, fmt.Sprintf("/doctors/%d/blocks", colleague.ID), blockBody},
		{http.MethodPost, "/agenda/blocks", blockBody},
		{http.MethodPost, "/agenda/appointment-types", `{"name":"Control","duration_minutes":30}`},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("doctor %s %s = %d, want 403", tc.method, tc.path, w.Code)
		}
	}
}

func TestAgendaRoutes_ReceptionReadsOnly(t *testing.T) {
	router, mine, _ := agendaRouter(t, role_data.RolePermissionMatrix[role_data.RoleRecepcionista])
	for _, path := range []string{
		fmt.Sprintf("/doctors/%d/schedule", mine.ID), fmt.Sprintf("/doctors/%d/blocks", mine.ID),
		"/agenda/blocks", "/agenda/appointment-types", "/agenda/range?from=2026-10-19&to=2026-10-19",
	} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("recepcionista GET %s = %d, want 200: %s", path, w.Code, w.Body.String())
		}
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, fmt.Sprintf("/doctors/%d/schedule", mine.ID), scheduleBody},
		{http.MethodDelete, fmt.Sprintf("/doctors/%d/blocks/1", mine.ID), ""},
		{http.MethodPut, "/agenda/blocks/1", blockBody},
		{http.MethodDelete, "/agenda/appointment-types/1", ""},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("recepcionista %s %s = %d, want 403", tc.method, tc.path, w.Code)
		}
	}
}

func TestAgendaRoutes_BillingOnlyRoleCannotRead(t *testing.T) {
	router, mine, _ := agendaRouter(t, []string{"READ_BILLING", "CREATE_BILLING"})
	for _, path := range []string{fmt.Sprintf("/doctors/%d/schedule", mine.ID), "/agenda/appointment-types", "/agenda/blocks"} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusForbidden {
			t.Errorf("billing-only GET %s = %d, want 403", path, w.Code)
		}
	}
}

func TestAgendaRoutes_ManagerManagesEveryone(t *testing.T) {
	router, _, colleague := agendaRouter(t, []string{"MANAGE_DOCTORS"})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, fmt.Sprintf("/doctors/%d/schedule", colleague.ID), scheduleBody},
		{http.MethodPost, fmt.Sprintf("/doctors/%d/blocks", colleague.ID), blockBody},
		{http.MethodPost, "/agenda/blocks", blockBody},
		{http.MethodPost, "/agenda/appointment-types", `{"name":"Control","duration_minutes":30}`},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusOK {
			t.Errorf("manager %s %s = %d, want 200: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}
