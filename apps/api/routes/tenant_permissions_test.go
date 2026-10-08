package routes

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantfiles"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_handlers "pengi-med-saas/features/tenants/handlers"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// tenantSettingsRouter mounts the tenant settings routes for a caller whose
// role holds exactly permissionIDs, on a plan with every canonical permission.
func tenantSettingsRouter(t *testing.T, permissionIDs []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &company_models.Company{})
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("tn-perm-%d", now), DisplayToken: fmt.Sprintf("tok-tn-perm-%d", now)}
	mustCreate(t, db, &tenant)

	role := user_models.Role{Role: fmt.Sprintf("role-%d", now)}
	for _, id := range permissionIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(t, db, &role)
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	mustCreate(t, db, &env)
	plan := map[string]bool{}
	for _, ids := range role_data.RolePermissionMatrix {
		for _, id := range ids {
			plan[id] = true
		}
	}

	router := gin.New()
	group := router.Group("/tenants", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
		c.Set("user_id", int64(env.UserID))
		c.Set("username", "tester")
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	})
	registerTenantSettingsRoutes(group, db, tenant_handlers.NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir())))
	return router
}

var sriWrites = []struct{ method, path string }{
	{http.MethodPut, "/tenants/sri/signature"},
	{http.MethodPut, "/tenants/sri/info"},
	{http.MethodPut, "/tenants/logo"},
}

// Only who manages SRI settings may replace the clinic's certificate, issuer
// data or logo: a receptionist or a doctor of the clinic must not.
func TestTenantRoutes_SriWritesNeedManageSriSettings(t *testing.T) {
	for _, roleName := range []string{role_data.RoleRecepcionista, role_data.RoleDoctor} {
		router := tenantSettingsRouter(t, role_data.RolePermissionMatrix[roleName])
		for _, tc := range sriWrites {
			if w := serve(router, tc.method, tc.path, ""); w.Code != http.StatusForbidden {
				t.Errorf("%s %s %s = %d, want 403: %s", roleName, tc.method, tc.path, w.Code, w.Body.String())
			}
		}
	}
}

func TestTenantRoutes_ContadorReachesSriWrites(t *testing.T) {
	router := tenantSettingsRouter(t, role_data.RolePermissionMatrix[role_data.RoleContador])
	for _, tc := range sriWrites {
		// The empty body fails validation; what matters is the permission let it through.
		if w := serve(router, tc.method, tc.path, ""); w.Code == http.StatusForbidden {
			t.Errorf("contador %s %s = 403, want past the permission check: %s", tc.method, tc.path, w.Body.String())
		}
	}
}
