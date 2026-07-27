package company_services

import (
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// CalculateEnabledFeaturesFromFeatures maps a set of Features to the tenant-level
// EnabledFeatures flags via each Feature's Permission categories.
func CalculateEnabledFeaturesFromFeatures(features []company_models.Feature) tenant_models.EnabledFeatures {
	categoriesFound := make(map[string]bool)
	for _, feature := range features {
		for _, perm := range feature.Permissions {
			categoriesFound[perm.Category] = true
		}
	}

	return tenant_models.EnabledFeatures{
		Clinical: categoriesFound["CLINICAL"],
		Billing:  categoriesFound["BILLING"],
		Team:     categoriesFound["TEAM"],
		Kanban:   categoriesFound["KANBAN"],
	}
}

// EnabledFeaturesForPlanCode computes the EnabledFeatures for a plan from its current
// Features association, always reflecting the live plan_features join.
func EnabledFeaturesForPlanCode(db *gorm.DB, planCode string) (tenant_models.EnabledFeatures, error) {
	var plan company_models.Plan
	if err := db.Preload("Features.Permissions").Where("code = ?", planCode).First(&plan).Error; err != nil {
		return tenant_models.EnabledFeatures{}, err
	}
	return CalculateEnabledFeaturesFromFeatures(plan.Features), nil
}

// EnabledFeaturesForCompany resolves the company's most recent Subscription and computes
// EnabledFeatures live from its current Plan. Falls back to defaults if there is no
// subscription on record.
func EnabledFeaturesForCompany(db *gorm.DB, companyID uint) (tenant_models.EnabledFeatures, error) {
	var sub company_models.Subscription
	if err := db.Where("company_id = ?", companyID).Order("expires_at DESC").First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return tenant_models.DefaultEnabledFeatures(), nil
		}
		return tenant_models.EnabledFeatures{}, err
	}
	return EnabledFeaturesForPlanCode(db, sub.PlanCode)
}
