package y2026

import (
	"encoding/json"
	"fmt"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// With the "age instead of birth date" setting on, the patient forms stored
// the birth date as "the day the form was saved, N years earlier". Mark those
// dates as estimated so they show as an age, never as an exact date.
//
// Only tenants with the setting on can have them. A match is a birth date whose
// day and month (in Ecuador) are those of the day the patient was created or
// last updated. A real birthday on that same day is marked too: it only shows
// the age instead of the date, and entering the date again fixes it.
func init() {
	database.GlobalDBMap["DB20260929_1"] = database.DBExecute{
		ID: "DB20260929_1",
		Execute: func(db *gorm.DB) error {
			db = tenantdb.System(db).Unscoped().Session(&gorm.Session{})
			ecuador, err := time.LoadLocation("America/Guayaquil")
			if err != nil {
				return fmt.Errorf("failed to load Ecuador timezone: %w", err)
			}

			var tenants []tenant_models.Tenant
			if err := db.Select("id", "ui_settings").Find(&tenants).Error; err != nil {
				return fmt.Errorf("failed to list tenants: %w", err)
			}
			var tenantIDs []uint
			for _, t := range tenants {
				var settings tenant_models.UISettings
				if json.Unmarshal([]byte(t.UISettings), &settings) == nil && settings.Clinical.PatientAgeInput {
					tenantIDs = append(tenantIDs, t.ID)
				}
			}
			if len(tenantIDs) == 0 {
				fmt.Println("✅ No tenant uses age input: no birth date marked as estimated.")
				return nil
			}

			var patients []clinical_models.Patient
			if err := db.Select("id", "birth_date", "created_at", "updated_at").
				Where("tenant_id IN ?", tenantIDs).Find(&patients).Error; err != nil {
				return fmt.Errorf("failed to list patients: %w", err)
			}
			var estimated []uint
			for _, p := range patients {
				if p.BirthDate.Year() <= 1 {
					continue // no birth date
				}
				birth := p.BirthDate.In(ecuador)
				if sameDayEarlierYear(birth, p.CreatedAt.In(ecuador)) ||
					sameDayEarlierYear(birth, p.UpdatedAt.In(ecuador)) {
					estimated = append(estimated, p.ID)
				}
			}
			if len(estimated) > 0 {
				if err := db.Model(&clinical_models.Patient{}).Where("id IN ?", estimated).
					UpdateColumn("birth_date_estimated", true).Error; err != nil {
					return fmt.Errorf("failed to mark estimated birth dates: %w", err)
				}
			}
			fmt.Printf("✅ %d of %d patients marked with an estimated birth date.\n", len(estimated), len(patients))
			return nil
		},
	}
}

func sameDayEarlierYear(birth, saved time.Time) bool {
	return birth.Year() < saved.Year() && birth.Month() == saved.Month() && birth.Day() == saved.Day()
}
