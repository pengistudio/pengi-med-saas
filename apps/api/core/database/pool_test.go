package database_test

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/database"
)

// The pool limits from config reach database/sql (sql.Open doesn't connect).
func TestConfigurePool_AppliesLimits(t *testing.T) {
	sqlDB, err := sql.Open("postgres", "host=127.0.0.1 port=1 user=u dbname=n sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	database.ConfigurePool(sqlDB, config.Config{DBMaxOpenConns: 7, DBMaxIdleConns: 3, DBConnMaxLifetime: time.Minute})

	if got := sqlDB.Stats().MaxOpenConnections; got != 7 {
		t.Fatalf("MaxOpenConnections = %d, want 7", got)
	}
}
