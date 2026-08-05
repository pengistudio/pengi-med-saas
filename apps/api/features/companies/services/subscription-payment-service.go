package company_services

import (
	"time"

	company_models "pengi-med-saas/features/companies/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ApplyPaidSubscription applies a confirmed SubscriptionPayment to its Subscription.
// sub.Plan must be preloaded (its Price is used to decide whether a plan change should be
// deferred to period end or applied immediately).
func ApplyPaidSubscription(db *gorm.DB, logger *zap.Logger, sub *company_models.Subscription, company *company_models.Company, payment *company_models.SubscriptionPayment) {
	isUpgrade := payment.Months == 0 && payment.TargetPlanCode != "" && payment.TargetPlanCode != sub.PlanCode

	if isUpgrade {
		// Upgrade proration paid — apply plan immediately, no period extension
		sub.PlanCode = payment.TargetPlanCode
		sub.Status = "active"
		sub.NextPlanCode = ""
		sub.PlanChangeAt = nil
		db.Model(company).Update("plan_code", payment.TargetPlanCode)
		db.Save(sub)
		logger.Info("Upgrade confirmed, plan applied immediately",
			zap.Uint("subscription_id", sub.ID),
			zap.String("plan_code", payment.TargetPlanCode))
		return
	}

	months := payment.Months
	if months <= 0 {
		months = 1
	}

	// Only defer the plan change to period end when the CURRENT plan is also paid
	// (e.g. downgrading a paid plan). Coming from a free/trial plan, the change
	// always applies immediately, regardless of remaining trial time.
	isActivePlanChange := payment.TargetPlanCode != "" &&
		payment.TargetPlanCode != sub.PlanCode &&
		sub.ExpiresAt.After(time.Now()) &&
		sub.Plan.Price > 0

	if isActivePlanChange {
		// Active paid subscription with plan change — defer features to period end
		planChangeAt := sub.ExpiresAt
		sub.ExpiresAt = planChangeAt.AddDate(0, months, 0)
		sub.NextPlanCode = payment.TargetPlanCode
		sub.PlanChangeAt = &planChangeAt
	} else {
		sub.ExpiresAt = sub.ExpiresAt.AddDate(0, months, 0)
		if payment.TargetPlanCode != "" && payment.TargetPlanCode != sub.PlanCode {
			sub.PlanCode = payment.TargetPlanCode
			sub.NextPlanCode = ""
			sub.PlanChangeAt = nil
			db.Model(company).Update("plan_code", payment.TargetPlanCode)
		}
	}

	sub.Status = "active"
	db.Save(sub)
	logger.Info("Subscription renewed",
		zap.Uint("subscription_id", sub.ID),
		zap.Int("months", months),
		zap.Time("new_expires_at", sub.ExpiresAt),
		zap.Bool("plan_change_deferred", isActivePlanChange))
}
