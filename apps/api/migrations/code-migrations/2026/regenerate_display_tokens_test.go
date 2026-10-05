package y2026

import (
	"fmt"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func TestRegenerateDisplayTokens(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	if _, ok := database.GlobalDBMap[regenerateDisplayTokensID]; !ok {
		t.Fatalf("migration %s not registered", regenerateDisplayTokensID)
	}
	now := time.Now().UnixNano()
	old := map[uint]string{}
	for i, token := range []string{fmt.Sprintf("%08d", now%100_000_000), fmt.Sprintf("%032x", now), "deleted-tenant-token"} {
		tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("regen-%d-%d", now, i), DisplayToken: token}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
		if i == 2 {
			db.Delete(&tenant)
		}
		old[tenant.ID] = token
	}

	if err := regenerateDisplayTokens(db); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for id, prev := range old {
		var tenant tenant_models.Tenant
		if err := db.Unscoped().First(&tenant, id).Error; err != nil {
			t.Fatalf("reload tenant %d: %v", id, err)
		}
		if !tenant_models.IsDisplayToken(tenant.DisplayToken) || tenant.DisplayToken == prev || seen[tenant.DisplayToken] {
			t.Fatalf("tenant %d token = %q (was %q), want a new unique 32-char token", id, tenant.DisplayToken, prev)
		}
		seen[tenant.DisplayToken] = true
	}
}
