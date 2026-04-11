package tenant_middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	tenant_models "pengi-med-saas/features/tenants/models"
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

func TestTenantMiddleware_SetsContextValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})

	tenant := tenant_models.Tenant{
		Slug: "test-tenant",
		Name: "Test Tenant",
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	middleware := TenantMiddleware(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Tenant-Slug", "test-tenant")

	middleware(c)

	// Check if context was aborted (should not be aborted for valid slug)
	if c.IsAborted() {
		t.Errorf("expected context to NOT be aborted for valid slug")
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		t.Errorf("expected tenant_id to be set in context")
	}

	if tenantID != tenant.ID {
		t.Errorf("expected tenant_id %d, got %v", tenant.ID, tenantID)
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
