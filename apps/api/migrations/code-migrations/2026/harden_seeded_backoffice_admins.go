package y2026

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"gorm.io/gorm"
)

// seededAdminPassword is the password DB20260304_BACKOFFICE_USERS gives the
// seeded admins. That migration is immutable, so in a fresh environment it
// still creates them; this one runs right after it.
const seededAdminPassword = "password"

// In production, backoffice admins still on the seeded password get
// BACKOFFICE_ADMIN_PASSWORD, or a random password nobody knows if it is unset,
// and their sessions end. Development keeps the default for convenience.
func init() {
	database.GlobalDBMap["DB20260926_1"] = database.DBExecute{
		ID: "DB20260926_1",
		Execute: func(db *gorm.DB) error {
			if os.Getenv("GIN_MODE") != "release" {
				return nil
			}

			var admins []backoffice_models.BackofficeUser
			if err := db.Find(&admins).Error; err != nil {
				return fmt.Errorf("failed to list backoffice users: %w", err)
			}
			for _, admin := range admins {
				if !auth.CompareHashAndPassword(admin.Password, seededAdminPassword) {
					continue
				}

				password := os.Getenv("BACKOFFICE_ADMIN_PASSWORD")
				if password == "" {
					random := make([]byte, 32)
					if _, err := rand.Read(random); err != nil {
						return fmt.Errorf("failed to generate a password: %w", err)
					}
					password = base64.RawURLEncoding.EncodeToString(random)
					fmt.Printf("⚠️  Backoffice user '%s' had the seeded password; set a random one. Set BACKOFFICE_ADMIN_PASSWORD or reset it in the database to log in.\n", admin.UserName)
				}
				hash, err := auth.HashPassword(password)
				if err != nil {
					return fmt.Errorf("failed to hash password: %w", err)
				}
				if err := db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", admin.ID).
					Updates(map[string]any{"password": hash, "refresh_token": ""}).Error; err != nil {
					return fmt.Errorf("failed to update backoffice user '%s': %w", admin.UserName, err)
				}
				fmt.Printf("✅ Backoffice user '%s' no longer has the seeded password.\n", admin.UserName)
			}
			return nil
		},
	}
}
