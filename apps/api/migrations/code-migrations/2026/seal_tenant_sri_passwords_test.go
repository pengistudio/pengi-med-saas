package y2026

import (
	"fmt"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/secretbox"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

const sealSriPasswordsID = "DB20260927_5"

func runSealSriPasswords(t *testing.T, db *gorm.DB) error {
	t.Helper()
	m, ok := database.GlobalDBMap[sealSriPasswordsID]
	if !ok {
		t.Fatalf("migration %s not registered", sealSriPasswordsID)
	}
	return m.Execute(db)
}

func createSriTenant(t *testing.T, db *gorm.DB, plain, sealed string) tenant_models.Tenant {
	t.Helper()
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{
		Name:                 "Clinic",
		Slug:                 fmt.Sprintf("seal-%d", now),
		DisplayToken:         fmt.Sprintf("tok-seal-%d", now),
		SriPassword:          plain,
		SriPasswordEncrypted: sealed,
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant
}

func reloadTenant(t *testing.T, db *gorm.DB, id uint) tenant_models.Tenant {
	t.Helper()
	var tenant tenant_models.Tenant
	if err := db.Unscoped().First(&tenant, id).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	return tenant
}

func openSealed(t *testing.T, sealed string) string {
	t.Helper()
	box, err := secretbox.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	plain, err := box.Open(sealed)
	if err != nil {
		t.Fatalf("open %q: %v", sealed, err)
	}
	return plain
}

func TestSealTenantSriPasswords_SealsEveryTenantAndIsIdempotent(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	a := createSriTenant(t, db, "pass-a", "")
	b := createSriTenant(t, db, "pass-b", "")
	none := createSriTenant(t, db, "", "")
	deleted := createSriTenant(t, db, "pass-deleted", "")
	db.Delete(&deleted)

	if err := runSealSriPasswords(t, db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   uint
		want string
	}{{a.ID, "pass-a"}, {b.ID, "pass-b"}, {deleted.ID, "pass-deleted"}} {
		got := reloadTenant(t, db, tc.id)
		if got.SriPassword != "" {
			t.Fatalf("tenant %d plaintext = %q, want cleared", tc.id, got.SriPassword)
		}
		if plain := openSealed(t, got.SriPasswordEncrypted); plain != tc.want {
			t.Fatalf("tenant %d sealed password opens to %q, want %q", tc.id, plain, tc.want)
		}
	}
	if got := reloadTenant(t, db, none.ID); got.SriPassword != "" || got.SriPasswordEncrypted != "" {
		t.Fatalf("tenant without password got plain=%q sealed=%q", got.SriPassword, got.SriPasswordEncrypted)
	}

	sealedBefore := reloadTenant(t, db, a.ID).SriPasswordEncrypted
	if err := runSealSriPasswords(t, db); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got := reloadTenant(t, db, a.ID).SriPasswordEncrypted; got != sealedBefore {
		t.Fatalf("second run re-sealed an already sealed password")
	}
}

// A plaintext password next to a sealed one was written by an older instance
// after the last seal, so it is the current one.
func TestSealTenantSriPasswords_PlaintextWinsOverAnOlderSealedValue(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	box, _ := secretbox.FromEnv()
	old, _ := box.Seal("old-pass")
	tenant := createSriTenant(t, db, "new-pass", old)

	if err := runSealSriPasswords(t, db); err != nil {
		t.Fatal(err)
	}
	got := reloadTenant(t, db, tenant.ID)
	if got.SriPassword != "" || openSealed(t, got.SriPasswordEncrypted) != "new-pass" {
		t.Fatalf("plain=%q sealed opens to %q, want cleared/new-pass", got.SriPassword, openSealed(t, got.SriPasswordEncrypted))
	}
}

// In release without the key, plaintext passwords cannot be sealed: the
// migration fails (so it is retried on the next start with the key) instead of
// being recorded as done while secrets stay in plaintext.
func TestSealTenantSriPasswords_WithoutKey_FailsAndLeavesRowsUntouched(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	tenant := createSriTenant(t, db, "pass-a", "")
	t.Setenv("GIN_MODE", "release")
	t.Setenv(secretbox.EnvKey, "")

	if err := runSealSriPasswords(t, db); err == nil {
		t.Fatal("migration succeeded without SIGNATURE_ENCRYPTION_KEY")
	}
	if got := reloadTenant(t, db, tenant.ID); got.SriPassword != "pass-a" || got.SriPasswordEncrypted != "" {
		t.Fatalf("plain=%q sealed=%q, want untouched", got.SriPassword, got.SriPasswordEncrypted)
	}
}

// With nothing to seal, a missing key does not block startup.
func TestSealTenantSriPasswords_WithoutKeyAndNothingToSeal_Succeeds(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	createSriTenant(t, db, "", "")
	t.Setenv("GIN_MODE", "release")
	t.Setenv(secretbox.EnvKey, "")

	if err := runSealSriPasswords(t, db); err != nil {
		t.Fatalf("migration failed with nothing to seal: %v", err)
	}
}
