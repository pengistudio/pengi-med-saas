package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	company_models "pengi-med-saas/features/companies/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	provisionWhatsAppTemplatesID = "DB20261005_5"
	seedWhatsAppMessageLimitID   = "DB20261005_6"
	// defaultWhatsAppMessageLimit is the monthly cap given to existing plans
	// with WhatsApp, so none is left unlimited by accident; backoffice tunes it.
	defaultWhatsAppMessageLimit = 300
)

func init() {
	// Creates the whatsapp_templates rows of every connected account: the
	// reminder from the status already stored on the account, and the rest of
	// the catalog by creating / looking them up on the WABA. A Graph failure
	// is not fatal: "Sincronizar" in settings (POST /whatsapp/template/sync)
	// provisions what is missing.
	database.GlobalDBMap[provisionWhatsAppTemplatesID] = database.DBExecute{
		ID: provisionWhatsAppTemplatesID,
		Execute: func(db *gorm.DB) error {
			return provisionWhatsAppTemplates(db, whatsapp.NewFromEnv())
		},
	}

	// Sets max_whatsapp_messages (monthly) on the plans that include the
	// WHATSAPP feature and don't have it yet.
	database.GlobalDBMap[seedWhatsAppMessageLimitID] = database.DBExecute{
		ID:      seedWhatsAppMessageLimitID,
		Execute: seedWhatsAppMessageLimit,
	}
}

// provisionWhatsAppTemplates fills the template rows; client nil only seeds
// the reminder's row (tests).
func provisionWhatsAppTemplates(db *gorm.DB, client *whatsapp.Client) error {
	var accounts []whatsapp_models.WhatsAppAccount
	if err := tenantdb.System(db).Find(&accounts).Error; err != nil {
		return fmt.Errorf("failed to list whatsapp accounts: %w", err)
	}
	for _, acc := range accounts {
		tdb := tenantdb.ForTenant(db, acc.TenantID)
		if acc.TemplateStatus != whatsapp_models.TemplateStatusNone {
			if err := whatsapp_services.SetTemplateStatus(tdb, acc.TenantID, whatsapp_models.ReminderTemplate, acc.TemplateStatus, acc.TemplateReason); err != nil {
				return fmt.Errorf("failed to seed reminder template of tenant %d: %w", acc.TenantID, err)
			}
		}
		if client == nil || acc.Status != whatsapp_models.AccountStatusConnected {
			continue
		}
		token, err := acc.OpenAccessToken()
		if err != nil {
			fmt.Printf("⚠️  WhatsApp account of tenant %d: token unusable, templates not provisioned.\n", acc.TenantID)
			continue
		}
		states, err := whatsapp_services.EnsureTemplates(client, token, acc.WabaID)
		if err != nil {
			fmt.Printf("⚠️  WhatsApp templates of tenant %d partly provisioned: %v\n", acc.TenantID, err)
		}
		if err := whatsapp_services.SaveTemplateStates(tdb, acc.TenantID, states); err != nil {
			return fmt.Errorf("failed to save templates of tenant %d: %w", acc.TenantID, err)
		}
		if status := whatsapp_services.StateOf(states, whatsapp_models.ReminderTemplate); status != whatsapp_models.TemplateStatusNone {
			if err := tdb.Model(&whatsapp_models.WhatsAppAccount{}).Where("id = ?", acc.ID).Update("template_status", status).Error; err != nil {
				return fmt.Errorf("failed to mirror reminder status of tenant %d: %w", acc.TenantID, err)
			}
		}
	}
	fmt.Printf("✅ WhatsApp templates provisioned for %d accounts.\n", len(accounts))
	return nil
}

func seedWhatsAppMessageLimit(db *gorm.DB) error {
	var plans []company_models.Plan
	err := db.Joins("JOIN plan_features pf ON pf.plan_id = plans.id").
		Joins("JOIN features f ON f.id = pf.feature_id AND f.deleted_at IS NULL").
		Where("f.code = ?", "WHATSAPP").
		Find(&plans).Error
	if err != nil {
		return fmt.Errorf("failed to list plans with WhatsApp: %w", err)
	}
	updated := 0
	for _, plan := range plans {
		props := plan.Properties
		if props == nil {
			props = datatypes.JSONMap{}
		}
		if _, ok := props[whatsapp_services.PlanLimitKey]; ok {
			continue
		}
		props[whatsapp_services.PlanLimitKey] = defaultWhatsAppMessageLimit
		if err := db.Model(&company_models.Plan{}).Where("id = ?", plan.ID).Update("properties", props).Error; err != nil {
			return fmt.Errorf("failed to set WhatsApp limit on plan %s: %w", plan.Code, err)
		}
		updated++
	}
	fmt.Printf("✅ %s = %d set on %d plans.\n", whatsapp_services.PlanLimitKey, defaultWhatsAppMessageLimit, updated)
	return nil
}
