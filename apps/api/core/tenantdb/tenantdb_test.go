package tenantdb_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type note struct {
	gorm.Model
	TenantID uint
	Text     string
	Tags     []tag `gorm:"foreignKey:NoteID"`
}

type tag struct {
	gorm.Model
	TenantID uint
	NoteID   uint
	Label    string
}

// catalogue has no tenant_id: shared across tenants, never filtered.
type catalogue struct {
	gorm.Model
	Code string
}

const ownTenant, otherTenant = uint(1), uint(2)

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutils.SetupTestDB(t, &note{}, &tag{}, &catalogue{}) // registers the plugin
	for _, n := range []note{
		{TenantID: ownTenant, Text: "own-1", Tags: []tag{{TenantID: ownTenant, Label: "own"}}},
		{TenantID: ownTenant, Text: "own-2"},
		{TenantID: otherTenant, Text: "other-1", Tags: []tag{{TenantID: otherTenant, Label: "other"}}},
	} {
		if err := tenantdb.System(db).Create(&n).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	tenantdb.System(db).Create(&catalogue{Code: "shared"})
	return db
}

// member is a request context as TenantMiddleware leaves it for a member of tenant.
func member(tenant uint) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("tenant_id", tenant)
	c.Set("user_id", int64(7))
	c.Set("username", "u")
	return c
}

func otherNoteID(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	var n note
	if err := tenantdb.System(db).Where("text = ?", "other-1").First(&n).Error; err != nil {
		t.Fatalf("find other note: %v", err)
	}
	return n.ID
}

func TestFor_QueriesSeeOnlyTheCallersTenant(t *testing.T) {
	db := setup(t)
	var notes []note
	if err := tenantdb.For(member(ownTenant), db).Find(&notes).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want the tenant's 2", len(notes))
	}
}

func TestFor_AnotherTenantsRowByIDIsNotFound(t *testing.T) {
	db := setup(t)
	var n note
	err := tenantdb.For(member(ownTenant), db).First(&n, otherNoteID(t, db)).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("err = %v, want ErrRecordNotFound", err)
	}
}

func TestFor_UpdatesAndDeletesCannotTouchAnotherTenant(t *testing.T) {
	db := setup(t)
	id := otherNoteID(t, db)
	c := member(ownTenant)

	if res := tenantdb.For(c, db).Model(&note{}).Where("id = ?", id).Update("text", "hijacked"); res.RowsAffected != 0 {
		t.Fatalf("update affected %d rows of another tenant", res.RowsAffected)
	}
	if res := tenantdb.For(c, db).Delete(&note{}, id); res.RowsAffected != 0 {
		t.Fatalf("delete affected %d rows of another tenant", res.RowsAffected)
	}
	var n note
	tenantdb.System(db).First(&n, id)
	if n.Text != "other-1" {
		t.Fatalf("another tenant's note became %q", n.Text)
	}
}

func TestFor_CreateStampsTheCallersTenant(t *testing.T) {
	db := setup(t)
	n := note{Text: "new"}
	if err := tenantdb.For(member(ownTenant), db).Create(&n).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if n.TenantID != ownTenant {
		t.Fatalf("tenant_id = %d, want %d", n.TenantID, ownTenant)
	}
}

func TestFor_CreateForAnotherTenantIsRefused(t *testing.T) {
	db := setup(t)
	err := tenantdb.For(member(ownTenant), db).Create(&note{TenantID: otherTenant, Text: "planted"}).Error
	if !errors.Is(err, tenantdb.ErrTenantMismatch) {
		t.Fatalf("err = %v, want tenantdb.ErrTenantMismatch", err)
	}
}

func TestFor_PreloadsAreScopedToo(t *testing.T) {
	db := setup(t)
	// Plant a tag of the other tenant under one of our notes.
	var own note
	tenantdb.System(db).Where("text = ?", "own-2").First(&own)
	tenantdb.System(db).Create(&tag{TenantID: otherTenant, NoteID: own.ID, Label: "planted"})

	var got note
	if err := tenantdb.For(member(ownTenant), db).Preload("Tags").First(&got, own.ID).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Fatalf("preloaded another tenant's tags: %+v", got.Tags)
	}
}

func TestFor_JoinsAreNotAmbiguous(t *testing.T) {
	db := setup(t)
	var count int64
	err := tenantdb.For(member(ownTenant), db).Model(&note{}).
		Joins("JOIN tags ON tags.note_id = notes.id").Count(&count).Error
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v, want 1 own note with tags", count, err)
	}
}

func TestFor_SharedTablesAreNotFiltered(t *testing.T) {
	db := setup(t)
	var rows []catalogue
	tenantdb.For(member(ownTenant), db).Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("got %d catalogue rows, want the shared 1", len(rows))
	}
}

func TestFor_SetsAuditMetadata(t *testing.T) {
	db := setup(t)
	tx := tenantdb.For(member(ownTenant), db)
	if v, _ := tx.Get("audit_tenant_id"); v != ownTenant {
		t.Fatalf("audit_tenant_id = %v, want %d", v, ownTenant)
	}
	if v, _ := tx.Get("audit_user_id"); v != int64(7) {
		t.Fatalf("audit_user_id = %v, want 7", v)
	}
}

func TestSystem_SeesEveryTenant(t *testing.T) {
	db := setup(t)
	var notes []note
	tenantdb.System(db).Find(&notes)
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want all 3", len(notes))
	}
}

// Stage 1 of ADR 0002: until every call site is migrated, a query without a
// tenant in its context keeps today's behaviour.
func TestUnboundQueriesAreUnchangedWhilePermissive(t *testing.T) {
	db := setup(t)
	var notes []note
	db.Find(&notes)
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want all 3", len(notes))
	}
}

func TestStrict_UnboundQueryOnATenantTableFails(t *testing.T) {
	db := setup(t)
	restore := tenantdb.SetStrict(true)
	defer restore()

	var notes []note
	if err := db.Find(&notes).Error; !errors.Is(err, tenantdb.ErrNoTenant) {
		t.Fatalf("err = %v, want tenantdb.ErrNoTenant", err)
	}
	if err := tenantdb.System(db).Find(&notes).Error; err != nil {
		t.Fatalf("system query failed in strict mode: %v", err)
	}
	if err := db.Find(&[]catalogue{}).Error; err != nil {
		t.Fatalf("shared table query failed in strict mode: %v", err)
	}
}

func TestTenantID(t *testing.T) {
	if got := tenantdb.TenantID(member(ownTenant)); got != ownTenant {
		t.Fatalf("TenantID = %d, want %d", got, ownTenant)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := tenantdb.TenantID(c); got != 0 {
		t.Fatalf("TenantID without tenant = %d, want 0", got)
	}
}

// Background work for one tenant (e.g. calendar sync after the request ended)
// must stay isolated without depending on the request's context.
func TestForTenant_BindsBackgroundWorkToOneTenant(t *testing.T) {
	db := setup(t)
	var notes []note
	if err := tenantdb.ForTenant(db, otherTenant).Find(&notes).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(notes) != 1 || notes[0].Text != "other-1" {
		t.Fatalf("got %+v, want only other-1", notes)
	}
}

// Handlers keep the bound handle in a variable and run several statements on
// it; conditions from one must not leak into the next.
func TestFor_HandleIsReusableAcrossStatements(t *testing.T) {
	db := setup(t)
	for _, bound := range map[string]*gorm.DB{
		"For":       tenantdb.For(member(ownTenant), db),
		"ForTenant": tenantdb.ForTenant(db, ownTenant),
	} {
		var first note
		if err := bound.Where("text = ?", "own-1").First(&first).Error; err != nil {
			t.Fatalf("first statement: %v", err)
		}
		var all []note
		if err := bound.Find(&all).Error; err != nil || len(all) != 2 {
			t.Fatalf("second statement got %d notes (err %v), want 2: conditions leaked", len(all), err)
		}
	}
}
