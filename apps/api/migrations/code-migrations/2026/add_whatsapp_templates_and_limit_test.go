package y2026

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/datatypes"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

func TestSeedWhatsAppMessageLimit(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &company_models.Feature{}, &company_models.Plan{})
	if _, ok := database.GlobalDBMap[seedWhatsAppMessageLimitID]; !ok {
		t.Fatalf("migration %s not registered", seedWhatsAppMessageLimitID)
	}
	stamp := time.Now().UnixNano()
	wa := company_models.Feature{Code: "WHATSAPP", Name: "WhatsApp"}
	clinical := company_models.Feature{Code: fmt.Sprintf("CLIN-%d", stamp), Name: "Clinical"}
	for _, f := range []*company_models.Feature{&wa, &clinical} {
		if err := db.Create(f).Error; err != nil {
			t.Fatal(err)
		}
	}
	plan := func(code string, features []company_models.Feature, props datatypes.JSONMap) company_models.Plan {
		p := company_models.Plan{Name: code, Code: fmt.Sprintf("%s-%d", code, stamp), Features: features, Properties: props}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		return p
	}
	withWA := plan("wa", []company_models.Feature{wa, clinical}, datatypes.JSONMap{"max_patients": 100})
	tuned := plan("tuned", []company_models.Feature{wa}, datatypes.JSONMap{"max_whatsapp_messages": 50})
	without := plan("none", []company_models.Feature{clinical}, nil)

	for i := 0; i < 2; i++ { // idempotent
		if err := seedWhatsAppMessageLimit(db); err != nil {
			t.Fatal(err)
		}
	}
	load := func(id uint) datatypes.JSONMap {
		var p company_models.Plan
		db.First(&p, id)
		return p.Properties
	}
	if props := load(withWA.ID); fmt.Sprint(props["max_whatsapp_messages"]) != "300" || fmt.Sprint(props["max_patients"]) != "100" {
		t.Fatalf("plan with WhatsApp = %v", props)
	}
	if props := load(tuned.ID); fmt.Sprint(props["max_whatsapp_messages"]) != "50" {
		t.Fatalf("a configured limit must be kept: %v", props)
	}
	if _, ok := load(without.ID)["max_whatsapp_messages"]; ok {
		t.Fatal("plans without WhatsApp must not get the key")
	}
}

func TestProvisionWhatsAppTemplates_SeedsReminderRow(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppTemplate{})
	if _, ok := database.GlobalDBMap[provisionWhatsAppTemplatesID]; !ok {
		t.Fatalf("migration %s not registered", provisionWhatsAppTemplatesID)
	}
	stamp := time.Now().UnixNano()
	var tenants [2]uint
	for i, status := range []string{"APPROVED", ""} {
		slug := fmt.Sprintf("wa-prov-%d-%d", i, stamp)
		tenant := tenant_models.Tenant{Name: "C", Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		tenants[i] = tenant.ID
		acc := whatsapp_models.WhatsAppAccount{TenantID: tenant.ID, WabaID: "W" + slug, PhoneNumberID: "P" + slug,
			Status: whatsapp_models.AccountStatusConnected, TemplateStatus: status}
		if err := db.Create(&acc).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := provisionWhatsAppTemplates(db, nil); err != nil {
			t.Fatal(err)
		}
	}
	var rows []whatsapp_models.WhatsAppTemplate
	db.Find(&rows)
	if len(rows) != 1 || rows[0].TenantID != tenants[0] || rows[0].Name != whatsapp_models.ReminderTemplate || rows[0].Status != "APPROVED" {
		t.Fatalf("rows = %+v", rows)
	}
}
