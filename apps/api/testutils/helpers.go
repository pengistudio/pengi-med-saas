package testutils

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupTestDB creates a PostgreSQL database and runs AutoMigrate on provided models.
// Each test gets its own isolated database.
// Use this in your test functions to get a clean test database for each test.
func SetupTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	// Get DB connection from environment or use defaults
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
