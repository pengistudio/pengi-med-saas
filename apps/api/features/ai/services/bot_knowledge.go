package ai_services

import (
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	agenda_services "pengi-med-saas/features/agenda/services"
	ai_models "pengi-med-saas/features/ai/models"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

// BotKnowledge is everything the WhatsApp assistant may answer from: the
// clinic's FAQ plus its name, address (Tenant.Address), office hours (the
// doctors' weekly schedules, agenda) and phone (the connected WhatsApp
// number). Nothing about the patient is ever part of it. BookingLink is
// appended by the server, never given to the model.
type BotKnowledge struct {
	ClinicName  string
	Address     string
	Phone       string
	Hours       string
	FAQ         []whatsapp_models.BotFAQEntry
	BookingLink string
}

// MaterialText is the material as the model reads it and the guards check
// it (without the booking link).
func (k BotKnowledge) MaterialText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Nombre de la clínica: %s\n", oneLine(k.ClinicName))
	fmt.Fprintf(&b, "Dirección: %s\n", oneLine(k.Address))
	if k.Phone != "" {
		fmt.Fprintf(&b, "Teléfono / WhatsApp de la clínica: %s\n", oneLine(k.Phone))
	}
	fmt.Fprintf(&b, "Horario de atención:\n%s\n", k.Hours)
	if len(k.FAQ) > 0 {
		b.WriteString("Preguntas frecuentes:\n")
		for _, e := range k.FAQ {
			fmt.Fprintf(&b, "P: %s\nR: %s\n", oneLine(e.Question), oneLine(e.Answer))
		}
	}
	return b.String()
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// BotContext is a tenant's assistant: its settings, material and hours.
type BotContext struct {
	Settings  whatsapp_models.WhatsAppBotSettings
	Knowledge BotKnowledge
	Hours     agenda_services.ClinicHours
	Connected bool // a connected WhatsApp account
}

// LoadBotContext reads the tenant's assistant settings and material (db
// unbound; everything is read on tenantID's handle). The hours include the
// clinic-wide holidays of the week starting at now.
func LoadBotContext(db *gorm.DB, tenantID uint, now time.Time) (BotContext, error) {
	tdb := tenantdb.ForTenant(db, tenantID)
	var bc BotContext
	if err := tdb.Limit(1).Find(&bc.Settings).Error; err != nil {
		return bc, err
	}
	var tenant tenant_models.Tenant
	if err := tenantdb.System(db).Select("id", "name", "trade_name", "address").Where("id = ?", tenantID).Limit(1).Find(&tenant).Error; err != nil {
		return bc, err
	}
	var account whatsapp_models.WhatsAppAccount
	if err := tdb.Select("id", "display_phone", "status").Limit(1).Find(&account).Error; err != nil {
		return bc, err
	}
	bc.Connected = account.ID != 0 && account.Status == whatsapp_models.AccountStatusConnected
	loc := whatsapp_services.ClinicLocation()
	from := now.In(loc).Format(agenda_services.DateLayout)
	to := now.In(loc).AddDate(0, 0, 7).Format(agenda_services.DateLayout)
	hours, err := agenda_services.LoadClinicHours(tdb, from, to)
	if err != nil {
		return bc, err
	}
	bc.Hours = hours
	link, err := portal_services.BookingLink(db, tenantID)
	if err != nil {
		return bc, err
	}
	bc.Knowledge = BotKnowledge{
		ClinicName:  tenant.DisplayName(),
		Address:     strings.TrimSpace(tenant.Address),
		Phone:       strings.TrimSpace(account.DisplayPhone),
		FAQ:         bc.Settings.FAQ,
		BookingLink: link,
	}
	if !hours.Empty() {
		bc.Knowledge.Hours = hours.Text()
	}
	return bc, nil
}

// BotReadiness is what enabling the assistant needs. Each flag is one item
// of the checklist the settings page shows.
type BotReadiness struct {
	PlanWhatsApp       bool `json:"plan_whatsapp"`
	PlanAI             bool `json:"plan_ai"`
	AIOptIn            bool `json:"ai_opt_in"`
	ProviderConfigured bool `json:"provider_configured"`
	WhatsAppConnected  bool `json:"whatsapp_connected"`
	OfficeHours        bool `json:"office_hours"`
	Address            bool `json:"address"`
	FAQ                bool `json:"faq"`
}

// Ready reports whether every requirement is met.
func (r BotReadiness) Ready() bool { return r.MissingKey() == "" }

// Active reports whether the assistant may run on a message: everything but
// the provider key, whose absence means handing every chat to reception.
func (r BotReadiness) Active() bool {
	r.ProviderConfigured = true
	return r.Ready()
}

// MissingKey is the i18n key of the first requirement not met ("" when
// ready): the error enabling the assistant answers with.
func (r BotReadiness) MissingKey() string {
	switch {
	case !r.PlanWhatsApp || !r.PlanAI:
		return "whatsapp.bot.error.plan"
	case !r.WhatsAppConnected:
		return "whatsapp.bot.error.not_connected"
	case !r.AIOptIn:
		return "whatsapp.bot.error.ai_opt_in"
	case !r.ProviderConfigured:
		return "whatsapp.bot.error.provider"
	case !r.OfficeHours:
		return "whatsapp.bot.error.office_hours"
	case !r.Address:
		return "whatsapp.bot.error.address"
	case !r.FAQ:
		return "whatsapp.bot.error.faq"
	}
	return ""
}

// CheckBotReadiness evaluates bc's requirements for tenantID; faq is the
// FAQ to judge (the one being saved, or the stored one).
func CheckBotReadiness(db *gorm.DB, tenantID uint, bc BotContext, faq []whatsapp_models.BotFAQEntry) (BotReadiness, error) {
	r := BotReadiness{
		WhatsAppConnected: bc.Connected,
		OfficeHours:       !bc.Hours.Empty(),
		Address:           strings.TrimSpace(bc.Knowledge.Address) != "",
		FAQ:               len(faq) > 0,
	}
	features, err := tenantFeatures(db, tenantID)
	if err != nil {
		return r, err
	}
	r.PlanWhatsApp, r.PlanAI = features.WhatsApp, features.AI
	if r.AIOptIn, err = Enabled(db, tenantID); err != nil {
		return r, err
	}
	provider, err := ResolveProvider(tenantdb.System(db), ai_models.FunctionBot)
	r.ProviderConfigured = err == nil && provider.Configured()
	return r, nil
}

// tenantFeatures are the plan features of the tenant's company (defaults
// without one: no WhatsApp, no AI).
func tenantFeatures(db *gorm.DB, tenantID uint) (tenant_models.EnabledFeatures, error) {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Select("id").Limit(1).Find(&company).Error; err != nil {
		return tenant_models.EnabledFeatures{}, err
	}
	if company.ID == 0 {
		return tenant_models.DefaultEnabledFeatures(), nil
	}
	return company_services.EnabledFeaturesForCompany(tenantdb.System(db), company.ID)
}
