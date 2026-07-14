package company_handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type teamTestFixture struct {
	handler   *CompanyHandler
	tenant    tenant_models.Tenant
	company   company_models.Company
	adminRole user_models.Role
	docRole   user_models.Role
	recRole   user_models.Role
	modRole   user_models.Role
}

func newTeamTestFixture(t *testing.T) *teamTestFixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &company_models.Company{}, &company_models.Subscription{},
		&user_models.User{}, &user_models.Environment{}, &user_models.Role{},
		&permission_models.Permission{},
	)

	tenant := tenant_models.Tenant{Name: "Clinic", Slug: "clinic", DisplayToken: "clinic-display-token"}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}
	company := company_models.Company{LegalName: "Clinic SAS", TradeName: "Clinic", PlanCode: "TRIAL", TenantID: tenant.ID}
	if err := db.Create(&company).Error; err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	adminRole := user_models.Role{Role: role_data.RoleAdmin}
	docRole := user_models.Role{Role: role_data.RoleDoctor}
	recRole := user_models.Role{Role: role_data.RoleRecepcionista}
	modRole := user_models.Role{Role: "moderator"}
	for _, r := range []*user_models.Role{&adminRole, &docRole, &recRole, &modRole} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("failed to create role %s: %v", r.Role, err)
		}
	}

	return &teamTestFixture{
		handler:   NewCompanyHandler(db, zap.NewNop()),
		tenant:    tenant,
		company:   company,
		adminRole: adminRole,
		docRole:   docRole,
		recRole:   recRole,
		modRole:   modRole,
	}
}

func (f *teamTestFixture) createEnvironment(t *testing.T, roleID uint, companyID uint) user_models.Environment {
	t.Helper()
	user := user_models.User{UserName: "user", Email: "u@example.com", Password: "hashed"}
	if err := f.handler.db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	env := user_models.Environment{UserID: user.ID, Name: "env", RoleID: roleID, CompanyID: companyID}
	if err := f.handler.db.Create(&env).Error; err != nil {
		t.Fatalf("failed to create environment: %v", err)
	}
	return env
}

func updateRoleContext(tenantID uint, environmentID uint, roleID uint) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(tenantID, 1)
	c.Params = gin.Params{{Key: "environment_id", Value: strconv.FormatUint(uint64(environmentID), 10)}}
	body, _ := json.Marshal(map[string]uint{"role_id": roleID})
	c.Request = httptest.NewRequest("PUT", "/companies/team/x/role", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func TestUpdateTeamMemberRole_Success(t *testing.T) {
	f := newTeamTestFixture(t)
	env := f.createEnvironment(t, f.docRole.ID, f.company.ID)

	c, _ := updateRoleContext(f.tenant.ID, env.ID, f.recRole.ID)
	resp := f.handler.UpdateTeamMemberRole(c)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var updated user_models.Environment
	f.handler.db.First(&updated, env.ID)
	if updated.RoleID != f.recRole.ID {
		t.Errorf("expected role_id %d, got %d", f.recRole.ID, updated.RoleID)
	}
}

func TestUpdateTeamMemberRole_CrossTenantForbidden(t *testing.T) {
	f := newTeamTestFixture(t)

	// A second company/tenant, and an environment that belongs to it.
	otherTenant := tenant_models.Tenant{Name: "Other", Slug: "other", DisplayToken: "other-display-token"}
	if err := f.handler.db.Create(&otherTenant).Error; err != nil {
		t.Fatalf("failed to create other tenant: %v", err)
	}
	otherCompany := company_models.Company{LegalName: "Other SAS", TradeName: "Other", PlanCode: "TRIAL", TenantID: otherTenant.ID}
	if err := f.handler.db.Create(&otherCompany).Error; err != nil {
		t.Fatalf("failed to create other company: %v", err)
	}
	env := f.createEnvironment(t, f.docRole.ID, otherCompany.ID)

	// Caller is authenticated against f.tenant (company A), targeting an
	// environment that belongs to otherCompany (company B).
	c, _ := updateRoleContext(f.tenant.ID, env.ID, f.recRole.ID)
	resp := f.handler.UpdateTeamMemberRole(c)
	if resp.Code != 403 {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestUpdateTeamMemberRole_InvalidRoleRejected(t *testing.T) {
	f := newTeamTestFixture(t)
	env := f.createEnvironment(t, f.docRole.ID, f.company.ID)

	c, _ := updateRoleContext(f.tenant.ID, env.ID, f.modRole.ID)
	resp := f.handler.UpdateTeamMemberRole(c)
	if resp.Code != 400 {
		t.Fatalf("expected 400 for non-canonical role, got %d", resp.Code)
	}
}

func TestUpdateTeamMemberRole_LastAdminBlocked(t *testing.T) {
	f := newTeamTestFixture(t)
	env := f.createEnvironment(t, f.adminRole.ID, f.company.ID)

	c, _ := updateRoleContext(f.tenant.ID, env.ID, f.docRole.ID)
	resp := f.handler.UpdateTeamMemberRole(c)
	if resp.Code != 409 {
		t.Fatalf("expected 409 when demoting the last admin, got %d", resp.Code)
	}
}

func TestUpdateTeamMemberRole_AllowsDemotionWhenAnotherAdminExists(t *testing.T) {
	f := newTeamTestFixture(t)
	env := f.createEnvironment(t, f.adminRole.ID, f.company.ID)
	f.createEnvironment(t, f.adminRole.ID, f.company.ID) // second admin

	c, _ := updateRoleContext(f.tenant.ID, env.ID, f.docRole.ID)
	resp := f.handler.UpdateTeamMemberRole(c)
	if resp.Code != 200 {
		t.Fatalf("expected 200 when another admin remains, got %d", resp.Code)
	}
}

func TestGetTeamRoles_ExcludesLegacyRoles(t *testing.T) {
	f := newTeamTestFixture(t)

	c, _ := testutils.NewGinContext(f.tenant.ID, 1)
	resp := f.handler.GetTeamRoles(c)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	respBytes, _ := json.Marshal(resp.Data)
	var roles []user_models.Role
	json.Unmarshal(respBytes, &roles)

	for _, r := range roles {
		if r.Role == "moderator" || r.Role == "user" {
			t.Errorf("expected legacy role %q to be excluded, got it in response", r.Role)
		}
	}
	if len(roles) != 3 {
		t.Errorf("expected 3 canonical roles (admin, doctor, recepcionista) seeded in fixture, got %d", len(roles))
	}
}
