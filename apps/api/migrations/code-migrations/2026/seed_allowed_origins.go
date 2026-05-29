package y2026

import (
	"fmt"
	"os"

	"pengi-med-saas/core/database"
	settings_models "pengi-med-saas/features/settings/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260529_1"] = database.DBExecute{
		ID: "DB20260529_1",
		Execute: func(db *gorm.DB) error {
			var existing settings_models.SystemSetting
			if err := db.Where("key = ?", "allowed_origins").First(&existing).Error; err == nil {
				fmt.Println("✅ allowed_origins already seeded, skipping.")
				return nil
			}

			value := os.Getenv("ALLOWED_ORIGINS")
			if value == "" {
				value = "https://app.gentoo.pengistudio.com,https://admin.gentoo.pengistudio.com,https://pengistudio.com,https://gentoo.pengistudio.com,https://api.gentoo.pengistudio.com"
			}

			setting := settings_models.SystemSetting{
				Key:   "allowed_origins",
				Value: value,
			}
			if err := db.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to seed allowed_origins: %w", err)
			}
			fmt.Printf("✅ Seeded allowed_origins: %s\n", value)
			return nil
		},
	}
}
