package testutils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"pengi-med-saas/core/tenantdb"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DefaultTestDBName is the Postgres database tests use when TEST_DB_NAME is not
// set. It is deliberately NOT DB_NAME: inside the api container DB_NAME points
// at the dev database, and tests must never write there.
const DefaultTestDBName = "pengi_test"

// SetupTestDB returns an empty, isolated database with the given models
// migrated, and releases it when the test ends.
//
//   - In CI (CI / GITHUB_ACTIONS set) it is a SQLite in-memory database.
//   - Otherwise it tries Postgres (DB_HOST/DB_PORT/DB_USER/DB_PASSWORD,
//     database TEST_DB_NAME, default "pengi_test", created if missing). Each
//     test gets its own schema, dropped in t.Cleanup, so tests never see rows
//     from other tests or earlier runs and leave nothing behind.
//   - If Postgres is unreachable it falls back to SQLite in-memory.
func SetupTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	// Set default AUTH_KEY and AUTH_EXP for tests if not set
	if os.Getenv("AUTH_KEY") == "" {
		os.Setenv("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	}
	if os.Getenv("AUTH_EXP") == "" {
		os.Setenv("AUTH_EXP", "60") // 60 minutes
	}
	if os.Getenv("AUTH_REFRESH_EXP") == "" {
		os.Setenv("AUTH_REFRESH_EXP", "30") // 30 days
	}

	isCI := os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""

	var db *gorm.DB
	if !isCI {
		db = openPostgresSchema(t)
	}
	if db == nil {
		db = openSQLite(t)
	}

	// Same data-layer tenant isolation as production (core/database/connect.go).
	if err := tenantdb.Register(db); err != nil {
		t.Fatalf("failed to register tenant isolation: %v", err)
	}

	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("failed to migrate test DB: %v", err)
	}

	return db
}

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	closeOnCleanup(t, db)
	return db
}

func closeOnCleanup(t *testing.T, db *gorm.DB) {
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

// pgAdmin is one connection pool per test binary to the test database, used to
// create and drop the per-test schemas. It is nil when Postgres is unreachable.
var (
	pgOnce  sync.Once
	pgAdmin *gorm.DB
	pgBase  string
	pgErr   error
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func postgresAdmin() (*gorm.DB, string, error) {
	pgOnce.Do(func() {
		server := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable",
			envOr("DB_HOST", "db"), envOr("DB_PORT", "5432"),
			envOr("DB_USER", "postgres"), envOr("DB_PASSWORD", "postgres"))
		dbName := envOr("TEST_DB_NAME", DefaultTestDBName)
		pgBase = server + " dbname=" + dbName

		cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
		db, err := gorm.Open(postgres.Open(pgBase), cfg)
		if err != nil {
			// The test database may not exist yet: create it from the
			// maintenance database. Concurrent test binaries may race here,
			// so the CREATE error is ignored and the reconnect decides.
			root, rootErr := gorm.Open(postgres.Open(server+" dbname=postgres"), cfg)
			if rootErr != nil {
				pgErr = rootErr
				return
			}
			root.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName))
			if sqlDB, e := root.DB(); e == nil {
				_ = sqlDB.Close()
			}
			if db, err = gorm.Open(postgres.Open(pgBase), cfg); err != nil {
				pgErr = err
				return
			}
		}
		pgAdmin = db
	})
	return pgAdmin, pgBase, pgErr
}

// openPostgresSchema opens a connection whose search_path is a fresh schema
// that is dropped (with everything in it) when the test ends. It returns nil
// when Postgres is unreachable so the caller falls back to SQLite.
func openPostgresSchema(t *testing.T) *gorm.DB {
	t.Helper()
	admin, base, err := postgresAdmin()
	if err != nil {
		t.Logf("PostgreSQL not available, falling back to SQLite: %v", err)
		return nil
	}

	suffix := make([]byte, 8)
	_, _ = rand.Read(suffix)
	schema := "test_" + hex.EncodeToString(suffix)
	if err := admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	t.Cleanup(func() {
		admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema))
	})

	// No FK constraints, matching SQLite (which doesn't enforce them): tests
	// migrate only the models they touch and build partial fixtures (a company
	// for tenant 9101 with no tenants row, etc.). Against the shared dev DB
	// those passed only because its rows and tables happened to exist.
	db, err := gorm.Open(postgres.Open(base+" search_path="+schema), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test schema %s: %v", schema, err)
	}
	// Registered after the DROP so it runs first (Cleanup is LIFO): the pool
	// is closed before its schema is dropped.
	closeOnCleanup(t, db)
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

// Ptr returns a pointer to the given value. Useful for populating optional
// (*T) struct fields in tests without an intermediate variable.
func Ptr[T any](v T) *T {
	return &v
}
