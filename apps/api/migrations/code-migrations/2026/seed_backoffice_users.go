package y2026

import (
	"fmt"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260304_BACKOFFICE_USERS"] = database.DBExecute{
		ID: "DB20260304_BACKOFFICE_USERS",
		Execute: func(db *gorm.DB) error {
			password, err := auth.HashPassword("password")
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			// --- Backoffice User: admin ---
			admin := backoffice_models.BackofficeUser{
				Name:     "Admin",
				UserName: "admin",
				Password: password,
			}
			var existingAdmin backoffice_models.BackofficeUser
			if err := db.Where("user_name = ?", admin.UserName).First(&existingAdmin).Error; err == nil {
				fmt.Printf("✅ Backoffice user '%s' already exists.\n", admin.UserName)
			} else if err == gorm.ErrRecordNotFound {
				if err := db.Create(&admin).Error; err != nil {
					return fmt.Errorf("failed to create backoffice user '%s': %w", admin.UserName, err)
				}
				fmt.Printf("✅ Backoffice user '%s' created.\n", admin.UserName)
			} else {
				return fmt.Errorf("failed to query backoffice user '%s': %w", admin.UserName, err)
			}

			// --- Backoffice User: admin2 ---
			admin2 := backoffice_models.BackofficeUser{
				Name:     "Admin 2",
				UserName: "admin2",
				Password: password,
			}
			var existingAdmin2 backoffice_models.BackofficeUser
			if err := db.Where("user_name = ?", admin2.UserName).First(&existingAdmin2).Error; err == nil {
				fmt.Printf("✅ Backoffice user '%s' already exists.\n", admin2.UserName)
			} else if err == gorm.ErrRecordNotFound {
				if err := db.Create(&admin2).Error; err != nil {
					return fmt.Errorf("failed to create backoffice user '%s': %w", admin2.UserName, err)
				}
				fmt.Printf("✅ Backoffice user '%s' created.\n", admin2.UserName)
			} else {
				return fmt.Errorf("failed to query backoffice user '%s': %w", admin2.UserName, err)
			}

			return nil
		},
	}
}
