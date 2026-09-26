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
