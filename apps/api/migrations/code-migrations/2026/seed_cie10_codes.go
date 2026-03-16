package y2026

import (
	"encoding/csv"
	_ "embed"
	"fmt"
	"strings"

	"pengi-med-saas/core/database"
	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/gorm"
)

//go:embed cie10_codes.csv
var cie10CodesCSV []byte

func init() {
	database.GlobalDBMap["DB20260315_CIE10_SEED"] = database.DBExecute{
		ID: "DB20260315_CIE10_SEED",
		Execute: func(db *gorm.DB) error {
			var count int64
			if err := db.Model(&clinical_models.Cie10Code{}).Count(&count).Error; err != nil {
				return fmt.Errorf("failed to count cie10 codes: %w", err)
			}
			if count > 0 {
				fmt.Printf("✅ CIE-10 codes already seeded (%d records).\n", count)
				return nil
			}

			r := csv.NewReader(strings.NewReader(string(cie10CodesCSV)))
			records, err := r.ReadAll()
			if err != nil {
				return fmt.Errorf("failed to parse cie10 CSV: %w", err)
			}

			// Skip header row
			if len(records) < 2 {
				return fmt.Errorf("cie10 CSV is empty or has no data rows")
			}
			rows := records[1:]

			codes := make([]clinical_models.Cie10Code, 0, len(rows))
			for _, row := range rows {
				if len(row) < 2 {
					continue
				}
				codes = append(codes, clinical_models.Cie10Code{
					Code:  strings.TrimSpace(row[0]),
					Title: strings.TrimSpace(row[1]),
				})
			}

			if err := db.CreateInBatches(codes, 200).Error; err != nil {
				return fmt.Errorf("failed to seed cie10 codes: %w", err)
			}

			fmt.Printf("✅ Seeded %d CIE-10 codes.\n", len(codes))
			return nil
		},
	}
}
