package y2026

import (
	"testing"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

func hardenFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutils.SetupTestDB(t, &backoffice_models.BackofficeUser{})
	for _, u := range []struct{ name, password string }{
		{"admin", "password"},           // still on the seed's default
		{"ops", "a-real-strong-secret"}, // changed by a person: must be left alone
	} {
		hash, err := auth.HashPassword(u.password)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&backoffice_models.BackofficeUser{UserName: u.name, Password: hash, RefreshToken: "old-refresh"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func passwordOf(t *testing.T, db *gorm.DB, userName string) backoffice_models.BackofficeUser {
	t.Helper()
	var u backoffice_models.BackofficeUser
	if err := db.Where("user_name = ?", userName).First(&u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func runHarden(t *testing.T, db *gorm.DB) {
	t.Helper()
	m, ok := database.GlobalDBMap["DB20260926_1"]
	if !ok {
		t.Fatal("migration DB20260926_1 not registered")
	}
	if err := m.Execute(db); err != nil {
		t.Fatal(err)
	}
}

func TestHardenSeededAdmins_InProductionReplacesTheDefaultPassword(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv("BACKOFFICE_ADMIN_PASSWORD", "from-the-environment-123")
	db := hardenFixture(t)

	runHarden(t, db)

	admin := passwordOf(t, db, "admin")
	if auth.CompareHashAndPassword(admin.Password, "password") {
		t.Fatal("seeded admin still accepts the default password")
	}
	if !auth.CompareHashAndPassword(admin.Password, "from-the-environment-123") {
		t.Fatal("seeded admin does not accept BACKOFFICE_ADMIN_PASSWORD")
	}
	if admin.RefreshToken != "" {
		t.Fatal("seeded admin's refresh token was not cleared")
	}
	ops := passwordOf(t, db, "ops")
	if !auth.CompareHashAndPassword(ops.Password, "a-real-strong-secret") || ops.RefreshToken != "old-refresh" {
		t.Fatal("an admin with a changed password was modified")
	}
}

func TestHardenSeededAdmins_InProductionWithoutEnvLocksTheDefaultPassword(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv("BACKOFFICE_ADMIN_PASSWORD", "")
	db := hardenFixture(t)

	runHarden(t, db)

	if auth.CompareHashAndPassword(passwordOf(t, db, "admin").Password, "password") {
		t.Fatal("seeded admin still accepts the default password")
	}
}

func TestHardenSeededAdmins_InDevelopmentKeepsTheDefault(t *testing.T) {
	t.Setenv("GIN_MODE", "development")
	db := hardenFixture(t)

	runHarden(t, db)

	if !auth.CompareHashAndPassword(passwordOf(t, db, "admin").Password, "password") {
		t.Fatal("development admin lost the default password")
	}
}
