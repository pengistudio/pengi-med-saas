package subscription_middleware

import (
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/database"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
)

// runRoleCheck runs RequireRolePermission for a caller whose TenantMiddleware
// already resolved their Environment (a role granting READ_PATIENT only).
func runRoleCheck(t *testing.T, permissionID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{})

	role := user_models.Role{Role: "Recepción", Permissions: []permission_models.Permission{
		{BaseStringID: database.BaseStringID{ID: "READ_PATIENT"}, Name: "Read patient"},
	}}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	if err := db.Create(&env).Error; err != nil {
		t.Fatalf("create environment: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("environment_id", env.ID)
	RequireRolePermission(db, permissionID)(c)
	return c, w
}

func TestRequireRolePermission_AllowsPermissionOfTheCallersRole(t *testing.T) {
	if c, w := runRoleCheck(t, "READ_PATIENT"); c.IsAborted() {
		t.Fatalf("aborted with %d, want allowed", w.Code)
	}
}

func TestRequireRolePermission_RejectsPermissionOutsideTheCallersRole(t *testing.T) {
	if c, w := runRoleCheck(t, "MANAGE_SRI_SETTINGS"); !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}

// runAnyCheck runs RequireAnyPermission for a caller whose role grants
// READ_PATIENT only, on a plan that includes planIDs.
func runAnyCheck(t *testing.T, planIDs []string, permissionIDs ...string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{})

	role := user_models.Role{Role: "Recepción", Permissions: []permission_models.Permission{
		{BaseStringID: database.BaseStringID{ID: "READ_PATIENT"}, Name: "Read patient"},
	}}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	if err := db.Create(&env).Error; err != nil {
		t.Fatalf("create environment: %v", err)
	}
	plan := map[string]bool{}
	for _, id := range planIDs {
		plan[id] = true
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("environment_id", env.ID)
	c.Set(ContextKeyAllowedPermissions, plan)
	RequireAnyPermission(db, permissionIDs...)(c)
	return c, w
}

func TestRequireAnyPermission_AllowsWhenOneIsInPlanAndRole(t *testing.T) {
	c, w := runAnyCheck(t, []string{"READ_PATIENT", "UPDATE_MEDICAL_RECORD"}, "UPDATE_MEDICAL_RECORD", "READ_PATIENT")
	if c.IsAborted() {
		t.Fatalf("aborted with %d, want allowed", w.Code)
	}
}

func TestRequireAnyPermission_RejectsWhenRoleHasNone(t *testing.T) {
	c, w := runAnyCheck(t, []string{"READ_PATIENT", "UPDATE_MEDICAL_RECORD", "RECORD_VITAL_SIGNS"}, "UPDATE_MEDICAL_RECORD", "RECORD_VITAL_SIGNS")
	if !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}

// The role's permission must also be in the plan: one permission in the plan
// and a different one in the role doesn't add up to access.
func TestRequireAnyPermission_RejectsRolePermissionOutsideThePlan(t *testing.T) {
	c, w := runAnyCheck(t, []string{"UPDATE_MEDICAL_RECORD"}, "UPDATE_MEDICAL_RECORD", "READ_PATIENT")
	if !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}

func TestRequireAnyPermission_RejectsWhenPlanHasNone(t *testing.T) {
	c, w := runAnyCheck(t, nil, "READ_PATIENT")
	if !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}
