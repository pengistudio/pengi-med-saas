package testutils

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates a test database and runs AutoMigrate on provided models.
// In CI environments (GitHub Actions), uses SQLite in-memory.
// Locally, uses PostgreSQL if available, falls back to SQLite.
// Each test gets its own isolated database.
// Use this in your test functions to get a clean test database for each test.
func SetupTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	// Set default AUTH_KEY for tests if not set
	if os.Getenv("AUTH_KEY") == "" {
		os.Setenv("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	}

	// Check if we're in CI environment
	isCI := os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""

	var db *gorm.DB
	var err error

	if isCI {
		// Use SQLite in memory for CI (no network dependency)
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open test DB: %v", err)
		}
	} else {
		// Try PostgreSQL for local development
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "db"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPassword := os.Getenv("DB_PASSWORD")
		if dbPassword == "" {
			dbPassword = "postgres"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "pengi_gentoo"
		}

		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			// Fall back to SQLite if PostgreSQL is not available
			t.Logf("PostgreSQL not available, falling back to SQLite: %v", err)
			db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("failed to open test DB: %v", err)
			}
		}
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
