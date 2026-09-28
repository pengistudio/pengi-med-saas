package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	"pengi-med-saas/testutils"
)

// legacyMessage mirrors the removed i18n Message model's table.
type legacyMessage struct {
	ID    uint
	Key   string
	Value string
	Lang  string
}

func (legacyMessage) TableName() string { return "messages" }

func TestDropMessagesTable_DropsItAndIsIdempotent(t *testing.T) {
	db := testutils.SetupTestDB(t, &legacyMessage{})
	if err := db.Create(&legacyMessage{Key: "k", Value: "v", Lang: "es"}).Error; err != nil {
		t.Fatal(err)
	}

	m, ok := database.GlobalDBMap["DB20260927_6"]
	if !ok {
		t.Fatal("migration DB20260927_6 not registered")
	}
	for run := 1; run <= 2; run++ {
		if err := m.Execute(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if db.Migrator().HasTable("messages") {
			t.Fatalf("run %d: messages table still exists", run)
		}
	}
}
