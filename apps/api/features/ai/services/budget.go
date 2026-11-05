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

// PlanBudgetKey is the plan property (Plan.Properties, edited in backoffice)
// with the AI budget of the month in USD, shared by every function (summary,
// scribe, bot). -1 is unlimited; an absent key means DefaultBudgetUSD, never
// unlimited: every use costs money and there is no overage. Clinics only see
// the percentage used, never dollars or tokens.
const PlanBudgetKey = "max_ai_budget_usd"

// DefaultBudgetUSD is the monthly AI budget of a plan that doesn't set one
// (and of a tenant without an active subscription).
const DefaultBudgetUSD = 10.0

// Budget is a tenant's AI spend this month against its plan. The dollar
// amounts stay server side; the clinic gets Percent. PeriodEnd is exclusive.
type Budget struct {
	UsedUSD     float64   `json:"-"`
	LimitUSD    float64   `json:"-"`       // -1 when unlimited
	Percent     int       `json:"percent"` // 0..100, rounded down; 100 once Exhausted
	Unlimited   bool      `json:"unlimited"`
	Exhausted   bool      `json:"reached"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// Reached reports whether no more uses are allowed this month.
func (b Budget) Reached() bool { return b.Exhausted }

func newBudget(used, limit float64, start, end time.Time) Budget {
	b := Budget{UsedUSD: used, LimitUSD: limit, PeriodStart: start, PeriodEnd: end, Unlimited: limit < 0}
	if b.Unlimited {
		return b
	}
	b.Exhausted = used >= limit
	switch {
	case b.Exhausted:
		b.Percent = 100
	case limit > 0:
		b.Percent = int(used / limit * 100)
	}
	return b
}

// MonthlyBudget returns the tenant's AI spend, all functions together, in the
// calendar month (clinic time zone) containing now. Concurrent uses may
// overshoot the limit by a few; accepted, as with the WhatsApp cap. A use is
// priced when it is logged, so a model without prices set in backoffice costs
// 0 and never exhausts the budget.
func MonthlyBudget(db *gorm.DB, tenantID uint, now time.Time) (Budget, error) {
	start, end := whatsapp_services.Period(now, whatsapp_services.ClinicLocation())
	var used float64
	// Bounds in UTC: SQLite (CI tests) compares timestamps as text.
	err := tenantdb.ForTenant(db, tenantID).Model(&ai_models.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", start.UTC(), end.UTC()).
		Select("COALESCE(SUM(cost_usd), 0)").Scan(&used).Error
	return newBudget(used, PlanBudget(db, tenantID), start, end), err
}

// PlanBudget is the monthly AI budget in USD of the tenant's active plan.
func PlanBudget(db *gorm.DB, tenantID uint) float64 {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Select("id").Limit(1).Find(&company).Error; err != nil || company.ID == 0 {
		return DefaultBudgetUSD
	}
	var sub company_models.Subscription
	if err := tenantdb.System(db).Where("company_id = ? AND status = ? AND expires_at > ?", company.ID, "active", time.Now()).
		Preload("Plan").Limit(1).Find(&sub).Error; err != nil || sub.ID == 0 {
		return DefaultBudgetUSD
	}
	raw, ok := sub.Plan.Properties[PlanBudgetKey]
	if !ok || raw == nil {
		return DefaultBudgetUSD
	}
	switch v := raw.(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	}
	return DefaultBudgetUSD
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
