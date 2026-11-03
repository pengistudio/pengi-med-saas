package ai_services

import (
	"encoding/json"
	"time"

	"pengi-med-saas/core/tenantdb"
	ai_models "pengi-med-saas/features/ai/models"
	company_models "pengi-med-saas/features/companies/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

// Plan properties (Plan.Properties, edited in backoffice) with the monthly
// number of uses of each function. -1 is unlimited; an absent key means the
// default below, never unlimited: every use costs money and there is no
// overage.
var planQuotaKeys = map[string]string{
	ai_models.FunctionSummary: "max_ai_summaries",
	ai_models.FunctionScribe:  "max_ai_scribes",
	ai_models.FunctionBot:     "max_ai_bot_conversations",
}

// DefaultQuotas are the monthly uses of a plan that doesn't set them (and of
// a tenant without an active subscription).
var DefaultQuotas = map[string]int64{
	ai_models.FunctionSummary: 300,
	ai_models.FunctionScribe:  300,
	ai_models.FunctionBot:     500,
}

// PlanQuotaKey is the plan property holding fn's monthly quota.
func PlanQuotaKey(fn string) string { return planQuotaKeys[fn] }

// Quota is a tenant's uses of one function this month against its plan.
// Limit is -1 when unlimited. PeriodEnd is exclusive.
type Quota struct {
	Function    string    `json:"function"`
	Used        int64     `json:"used"`
	Limit       int64     `json:"limit"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// Reached reports whether no more uses are allowed this month.
func (q Quota) Reached() bool { return q.Limit >= 0 && q.Used >= q.Limit }

// MonthlyQuota returns the tenant's quota of fn in the calendar month (clinic
// time zone) containing now. Concurrent uses may overshoot the limit by a
// few; accepted, as with the WhatsApp cap.
func MonthlyQuota(db *gorm.DB, tenantID uint, fn string, now time.Time) (Quota, error) {
	start, end := whatsapp_services.Period(now, whatsapp_services.ClinicLocation())
	q := Quota{Function: fn, Limit: PlanQuota(db, tenantID, fn), PeriodStart: start, PeriodEnd: end}
	var used int64
	// Bounds in UTC: SQLite (CI tests) compares timestamps as text.
	err := tenantdb.ForTenant(db, tenantID).Model(&ai_models.AIUsageLog{}).
		Where("function = ? AND created_at >= ? AND created_at < ?", fn, start.UTC(), end.UTC()).
		Count(&used).Error
	q.Used = used
	return q, err
}

// PlanQuota is the monthly limit of fn in the tenant's active plan.
func PlanQuota(db *gorm.DB, tenantID uint, fn string) int64 {
	def := DefaultQuotas[fn]
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Select("id").Limit(1).Find(&company).Error; err != nil || company.ID == 0 {
		return def
	}
	var sub company_models.Subscription
	if err := tenantdb.System(db).Where("company_id = ? AND status = ? AND expires_at > ?", company.ID, "active", time.Now()).
		Preload("Plan").Limit(1).Find(&sub).Error; err != nil || sub.ID == 0 {
		return def
	}
	raw, ok := sub.Plan.Properties[PlanQuotaKey(fn)]
	if !ok || raw == nil {
		return def
	}
	switch v := raw.(type) {
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return def
}

// UsageEntry is one charged use to log. Host is the provider's host, never
// the URL's credentials nor the key.
type UsageEntry struct {
	TenantID         uint
	UserID           uint
	Function         string
	Model            string
	Host             string
	PromptTokens     int
	CompletionTokens int
}

// LogUsage records one charged use (tokens and cost, never the prompt).
func LogUsage(db *gorm.DB, e UsageEntry, now time.Time) error {
	_, err := logUsage(db, e, now)
	return err
}

// logUsage is LogUsage returning the row's id.
func logUsage(db *gorm.DB, e UsageEntry, now time.Time) (uint, error) {
	row := ai_models.AIUsageLog{
		TenantID:         e.TenantID,
		UserID:           e.UserID,
		Function:         e.Function,
		Model:            e.Model,
		ProviderHost:     e.Host,
		PromptTokens:     e.PromptTokens,
		CompletionTokens: e.CompletionTokens,
		CostUSD:          Cost(tenantdb.System(db), e.Function, e.PromptTokens, e.CompletionTokens),
		CreatedAt:        now.UTC(),
	}
	err := tenantdb.ForTenant(db, e.TenantID).Create(&row).Error
	return row.ID, err
}

// addUsage adds the tokens and cost of one more model call to the use
// usageID of the tenant (an assistant conversation already counted). False
// when the row is gone.
func addUsage(db *gorm.DB, tenantID, usageID uint, e UsageEntry) (bool, error) {
	cost := Cost(tenantdb.System(db), e.Function, e.PromptTokens, e.CompletionTokens)
	res := tenantdb.ForTenant(db, tenantID).Model(&ai_models.AIUsageLog{}).
		Where("id = ? AND function = ?", usageID, e.Function).
		Updates(map[string]any{
			"prompt_tokens":     gorm.Expr("prompt_tokens + ?", e.PromptTokens),
			"completion_tokens": gorm.Expr("completion_tokens + ?", e.CompletionTokens),
			"cost_usd":          gorm.Expr("cost_usd + ?", cost),
		})
	return res.RowsAffected == 1, res.Error
}
