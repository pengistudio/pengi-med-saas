package database_test

import (
	"bytes"
	"strings"
	"testing"

	"gorm.io/gorm"

	"pengi-med-saas/core/database"
	"pengi-med-saas/testutils"
)

type loggedRow struct {
	ID    uint
	Token string
}

// The SQL log must never carry bound values (tokens, patient data, hashes).
func TestNewSQLLogger_KeepsValuesOutOfTheLog(t *testing.T) {
	var out bytes.Buffer
	db := testutils.SetupTestDB(t, &loggedRow{}).Session(&gorm.Session{Logger: database.NewSQLLogger(&out)})
	const secret = "SECRET-display-token-0123456789"

	// A missing row is not logged at all.
	var row loggedRow
	if err := db.Where("token = ?", secret).First(&row).Error; err != gorm.ErrRecordNotFound {
		t.Fatalf("got %v, want ErrRecordNotFound", err)
	}
	if out.Len() != 0 {
		t.Fatalf("record not found was logged: %q", out.String())
	}

	// A failing query is logged with placeholders, not the value.
	if err := db.Raw("SELECT * FROM missing_table WHERE token = ?", secret).Scan(&row).Error; err == nil {
		t.Fatal("want an error from a missing table")
	}
	logged := out.String()
	if !strings.Contains(logged, "missing_table") {
		t.Fatalf("query error not logged: %q", logged)
	}
	if strings.Contains(logged, secret) {
		t.Fatalf("SQL log leaks the bound value: %q", logged)
	}
}
