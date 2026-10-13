package routes

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	company_models "pengi-med-saas/features/companies/models"
	doctor_handlers "pengi-med-saas/features/doctors/handlers"
	doctor_models "pengi-med-saas/features/doctors/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// doctorRouter mounts the doctor routes for a caller whose role holds exactly
// permissionIDs, on a plan with every clinical permission and MANAGE_DOCTORS.
func doctorRouter(t *testing.T, permissionIDs []string) (*gin.Engine, doctor_models.Doctor) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &company_models.Company{}, &doctor_models.Doctor{})
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("dr-perm-%d", now), DisplayToken: fmt.Sprintf("tok-dr-perm-%d", now)}
	mustCreate(t, db, &tenant)
	doctor := doctor_models.Doctor{TenantID: tenant.ID, FullName: "Dra. Ana", Specialty: "pediatrics", Color: "#2563EB", Active: true}
	mustCreate(t, db, &doctor)

	role := user_models.Role{Role: fmt.Sprintf("role-%d", now)}
	for _, id := range permissionIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(t, db, &role)
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	mustCreate(t, db, &env)
	plan := map[string]bool{"MANAGE_DOCTORS": true}
	for _, ids := range role_data.RolePermissionMatrix {
		for _, id := range ids {
			plan[id] = true
		}
	}

	router := gin.New()
	group := router.Group("/doctors", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
		c.Set("user_id", int64(env.UserID))
		c.Set("username", "tester")
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	})
	registerDoctorTenantRoutes(group, db, doctor_handlers.NewDoctorHandler(db, zap.NewNop()))
	return router, doctor
}

func TestDoctorRoutes_ReadersSeeListButCannotManage(t *testing.T) {
	router, doctor := doctorRouter(t, role_data.RolePermissionMatrix[role_data.RoleRecepcionista])
	for _, path := range []string{"/doctors", "/doctors?active=true", fmt.Sprintf("/doctors/%d", doctor.ID), "/doctors/specialties", "/doctors/status"} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("recepcionista GET %s = %d, want 200: %s", path, w.Code, w.Body.String())
		}
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/doctors", `{"full_name":"X","specialty":"pediatrics"}`},
		{http.MethodPut, fmt.Sprintf("/doctors/%d", doctor.ID), `{"full_name":"X"}`},
		{http.MethodPut, fmt.Sprintf("/doctors/%d/user", doctor.ID), `{"user_id":null}`},
		{http.MethodPut, fmt.Sprintf("/doctors/%d/deactivate", doctor.ID), ""},
		{http.MethodPut, fmt.Sprintf("/doctors/%d/activate", doctor.ID), ""},
		{http.MethodPut, "/doctors/review", ""},
		{http.MethodDelete, fmt.Sprintf("/doctors/%d", doctor.ID), ""},
		// Creating one's own profile is for those who attend patients.
		{http.MethodPost, "/doctors/me", `{"full_name":"X","specialty":"pediatrics"}`},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("recepcionista %s %s = %d, want 403", tc.method, tc.path, w.Code)
		}
	}
}

func TestDoctorRoutes_DoctorRoleCreatesOwnProfileOnly(t *testing.T) {
	router, doctor := doctorRouter(t, role_data.RolePermissionMatrix[role_data.RoleDoctor])
	if w := serve(router, http.MethodPut, fmt.Sprintf("/doctors/%d/deactivate", doctor.ID), ""); w.Code != http.StatusForbidden {
		t.Errorf("doctor role deactivate = %d, want 403", w.Code)
	}
	// Reaches the handler (no permission 403); the caller isn't a tenant member here, so it's a 400.
	if w := serve(router, http.MethodPost, "/doctors/me", `{"full_name":"Yo","specialty":"pediatrics"}`); w.Code == http.StatusForbidden {
		t.Errorf("doctor role POST /doctors/me = 403, want it to reach the handler")
	}
}

func TestDoctorRoutes_ManagerCanManage(t *testing.T) {
	router, doctor := doctorRouter(t, []string{"MANAGE_DOCTORS"})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/doctors", ""},
		{http.MethodPut, fmt.Sprintf("/doctors/%d", doctor.ID), `{"full_name":"Dra. Ana María"}`},
		{http.MethodPut, fmt.Sprintf("/doctors/%d/deactivate", doctor.ID), ""},
		{http.MethodPut, "/doctors/review", ""},
		{http.MethodDelete, fmt.Sprintf("/doctors/%d", doctor.ID), ""},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusOK {
			t.Errorf("manager %s %s = %d, want 200: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}
