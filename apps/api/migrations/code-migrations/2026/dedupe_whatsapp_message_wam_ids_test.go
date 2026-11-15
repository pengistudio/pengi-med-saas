package y2026

import (
	"testing"

	"pengi-med-saas/core/database"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm/clause"
)

func TestDedupeWhatsAppMessageWamIDs(t *testing.T) {
	db := testutils.SetupTestDB(t, &whatsapp_models.WhatsAppMessage{})
	seed := func(tenantID uint, wamID, body string) uint {
		m := whatsapp_models.WhatsAppMessage{TenantID: tenantID, WamID: wamID, Body: body, Status: whatsapp_models.MessageStatusReceived}
		if err := db.Create(&m).Error; err != nil {
			t.Fatal(err)
		}
		return m.ID
	}
	first := seed(1, "wamid.A", "first")
	seed(1, "wamid.A", "dup")
	seed(1, "wamid.A", "dup again")
	otherTenant := seed(2, "wamid.A", "same id, other clinic")
	single := seed(1, "wamid.B", "only one")
	noID1 := seed(1, "", "queued")
	noID2 := seed(1, "", "queued too")

	for run := 1; run <= 2; run++ { // idempotent
		if err := database.GlobalDBMap[dedupeWhatsAppMessageWamIDsID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}

	var ids []uint
	db.Model(&whatsapp_models.WhatsAppMessage{}).Order("id").Pluck("id", &ids)
	want := []uint{first, otherTenant, single, noID1, noID2}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}
	if !db.Migrator().HasIndex(&whatsapp_models.WhatsAppMessage{}, whatsapp_models.WamIDUniqueIndex) {
		t.Fatal("unique index not created")
	}

	// The index now rejects a duplicate; ON CONFLICT DO NOTHING skips it.
	if err := db.Create(&whatsapp_models.WhatsAppMessage{TenantID: 1, WamID: "wamid.B", Status: "x"}).Error; err == nil {
		t.Fatal("duplicate wam_id accepted")
	}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&whatsapp_models.WhatsAppMessage{TenantID: 1, WamID: "wamid.B", Status: "x"})
	if res.Error != nil || res.RowsAffected != 0 {
		t.Fatalf("do nothing: err=%v rows=%d", res.Error, res.RowsAffected)
	}
	// Messages without a Meta id (queued ones) are not constrained.
	seed(1, "", "queued three")
}
