package tenant_middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

// testModel is a simple model with tenant_id to test TenantScope
type testModel struct {
	ID       uint
	TenantID uint
	Name     string
}

func TestTenantMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})

	middleware := TenantMiddleware(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	// Intentionally don't set X-Tenant-Slug header

	middleware(c)

	if c.IsAborted() == false {
		t.Errorf("expected context to be aborted, but it wasn't")
	}
	if w.Code != 400 {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTenantMiddleware_UnknownSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})

	middleware := TenantMiddleware(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Tenant-Slug", "nonexistent-slug")

	middleware(c)

	if c.IsAborted() == false {
		t.Errorf("expected context to be aborted for unknown slug")
	}
	if w.Code != 404 {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestTenantScope_FiltersQuery(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &testModel{})

	// Create two tenants with unique display tokens
	tenant1 := tenant_models.Tenant{Slug: "tenant1", Name: "Tenant 1", DisplayToken: "token1"}
	tenant2 := tenant_models.Tenant{Slug: "tenant2", Name: "Tenant 2", DisplayToken: "token2"}
	if err := db.Create(&tenant1).Error; err != nil {
		t.Fatalf("failed to create tenant1: %v", err)
	}
	if err := db.Create(&tenant2).Error; err != nil {
		t.Fatalf("failed to create tenant2: %v", err)
	}

	// Create test records for each tenant
	item1 := testModel{TenantID: tenant1.ID, Name: "Item 1"}
	item2 := testModel{TenantID: tenant1.ID, Name: "Item 2"}
	item3 := testModel{TenantID: tenant2.ID, Name: "Item 3"}
	if err := db.Create(&item1).Error; err != nil {
		t.Fatalf("failed to create item1: %v", err)
	}
	if err := db.Create(&item2).Error; err != nil {
		t.Fatalf("failed to create item2: %v", err)
	}
	if err := db.Create(&item3).Error; err != nil {
		t.Fatalf("failed to create item3: %v", err)
	}

	// Create a test context with tenant1
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant_id", tenant1.ID)
	c.Set("userId", int64(1))

	// Apply TenantScope and query
	var results []testModel
	scope := TenantScope(c)
	if err := scope(db).Find(&results).Error; err != nil {
		t.Fatalf("failed to query with TenantScope: %v", err)
	}

	// Should only get tenant1's items (2 items), not tenant2's item (1 item)
	if len(results) != 2 {
		t.Errorf("expected 2 results for tenant1, got %d", len(results))
	}
	for _, item := range results {
		if item.TenantID != tenant1.ID {
			t.Errorf("expected all results to have tenant_id %d, got %d", tenant1.ID, item.TenantID)
		}
	}
}

// membershipFixture creates two tenants, each with its company, and one user who
// only has a role (Environment) in the first.
type membershipFixture struct {
	db          *gorm.DB
	own, other  tenant_models.Tenant
	ownCompany  company_models.Company
	environment user_models.Environment
	userID      int64
}

func newMembershipFixture(t *testing.T) membershipFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &user_models.Environment{})

	f := membershipFixture{db: db, userID: 7}
	f.own = tenant_models.Tenant{Slug: "own-clinic", Name: "Own", DisplayToken: "tok-own"}
	f.other = tenant_models.Tenant{Slug: "other-clinic", Name: "Other", DisplayToken: "tok-other"}
	for _, tenant := range []*tenant_models.Tenant{&f.own, &f.other} {
		if err := db.Create(tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
		company := company_models.Company{LegalName: tenant.Name, TradeName: tenant.Name, PlanCode: "basic", TenantID: tenant.ID}
		if err := db.Create(&company).Error; err != nil {
			t.Fatalf("create company: %v", err)
		}
		if tenant.ID == f.own.ID {
			f.ownCompany = company
		}
	}
	f.environment = user_models.Environment{UserID: uint(f.userID), Name: "Own", CompanyID: f.ownCompany.ID}
	if err := db.Create(&f.environment).Error; err != nil {
		t.Fatalf("create environment: %v", err)
	}
	return f
}

// request runs TenantMiddleware as the authenticated fixture user (as
// AuthMiddleware would have left the context) against the given tenant slug.
func (f membershipFixture) request(slug string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Tenant-Slug", slug)
	c.Set("user_id", f.userID)
	c.Set("username", "user@clinic.ec")
	TenantMiddleware(f.db)(c)
	return c, w
}

func TestTenantMiddleware_MemberGetsTenantCompanyAndEnvironment(t *testing.T) {
	f := newMembershipFixture(t)

	c, _ := f.request("own-clinic")

	if c.IsAborted() {
		t.Fatalf("a member of the tenant was rejected")
	}
	if got := c.GetUint("tenant_id"); got != f.own.ID {
		t.Fatalf("tenant_id = %d, want %d", got, f.own.ID)
	}
	if got := c.GetUint("company_id"); got != f.ownCompany.ID {
		t.Fatalf("company_id = %d, want %d", got, f.ownCompany.ID)
	}
	if got := c.GetUint("environment_id"); got != f.environment.ID {
		t.Fatalf("environment_id = %d, want %d", got, f.environment.ID)
	}
}

// A user must not reach another company's data by sending its slug.
func TestTenantMiddleware_RejectsUserWithoutRoleInTheTenant(t *testing.T) {
	f := newMembershipFixture(t)

	c, w := f.request("other-clinic")

	if !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
	if _, exists := c.Get("tenant_id"); exists {
		t.Fatalf("tenant_id was set for a non-member")
	}
}

func TestTenantMiddleware_RejectsUserWhoseRoleWasRemoved(t *testing.T) {
	f := newMembershipFixture(t)
	f.db.Delete(&f.environment)

	if c, w := f.request("own-clinic"); !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}

func TestTenantMiddleware_RejectsUnauthenticatedRequest(t *testing.T) {
	f := newMembershipFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Tenant-Slug", "own-clinic")

	TenantMiddleware(f.db)(c)

	if !c.IsAborted() || w.Code != 401 {
		t.Fatalf("aborted=%v code=%d, want aborted with 401", c.IsAborted(), w.Code)
	}
}
