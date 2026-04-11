package testutils

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database and runs AutoMigrate on provided models.
// Use this in your test functions to get a clean test database for each test.
func SetupTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("failed to migrate test DB: %v", err)
	}

	return db
}

// NewGinContext creates a test Gin context with tenant_id and userId set.
// Returns the context and the ResponseRecorder so you can inspect the response.
func NewGinContext(tenantID uint, userID int64) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", tenantID)
	c.Set("userId", userID)
	return c, w
}
