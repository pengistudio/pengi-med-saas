package y2026

import (
	"fmt"
	"time"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"

	"gorm.io/gorm"
)

// The backoffice used to store "expires on the 16th" as 16th 00:00 UTC, which
// is the 15th at 19:00 in Ecuador. An expiry date now means the end of that day
// in Ecuador. Only exact-midnight-UTC expiries came from the backoffice; trial
// and renewed subscriptions keep their time.
func init() {
	database.GlobalDBMap["DB20260926_2"] = database.DBExecute{
		ID: "DB20260926_2",
		Execute: func(db *gorm.DB) error {
			var subs []company_models.Subscription
			if err := db.Unscoped().Find(&subs).Error; err != nil {
				return fmt.Errorf("failed to list subscriptions: %w", err)
			}
			moved := 0
			for _, sub := range subs {
				utc := sub.ExpiresAt.UTC()
				if utc.Hour() != 0 || utc.Minute() != 0 || utc.Second() != 0 || utc.Nanosecond() != 0 {
					continue
				}
				endOfDay, err := company_models.ParseExpiry(utc.Format(time.DateOnly))
				if err != nil {
					return err
				}
				if err := db.Unscoped().Model(&company_models.Subscription{}).Where("id = ?", sub.ID).
					UpdateColumn("expires_at", endOfDay).Error; err != nil {
					return fmt.Errorf("failed to update subscription %d: %w", sub.ID, err)
				}
				moved++
			}
			fmt.Printf("✅ %d subscription expiries moved to the end of their day in Ecuador.\n", moved)
			return nil
		},
	}
}
