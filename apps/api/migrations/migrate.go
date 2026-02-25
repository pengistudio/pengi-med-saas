package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"pengi-med-saas/core/database"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	message_models "pengi-med-saas/i18n/models"

	"gorm.io/gorm"

	_ "pengi-med-saas/migrations/code-migrations/2026"
)

func RunMigrations(db *gorm.DB) error {
	err := database.MigrateDB(
		db,
		database.DBExecute{},
		tenant_models.Tenant{},
		permission_models.Permission{},
		message_models.Message{},
		company_models.Company{},
		company_models.Plan{},
		company_models.Subscription{},
		company_models.Feature{},
		user_models.User{},
		user_models.Environment{},
		user_models.Role{},
		clinical_models.Patient{},
		clinical_models.MedicalRecord{},
		clinical_models.SOAPRecord{},
		clinical_models.Prescription{},
		clinical_models.Appointment{},
	)
	if err != nil {
		return err
	}

	return database.ExecuteAll(db)
}

func MigrateMessages(db *gorm.DB, lang string) error {

	if lang == "" {
		lang = "es" // Default language
	}

	// Obtener el directorio actual de trabajo
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	// Construir la ruta absoluta usando path/filepath
	file := filepath.Join(workDir, "i18n", "messages", fmt.Sprintf("messages_%s.json", lang))
	return message_models.LoadMessagesFromFile(db, file, lang)

}

func RunAllMigrations(db *gorm.DB) error {
	err := RunMigrations(db)
	if err != nil {
		return err
	}
	err = MigrateMessages(db, "es") // Migrate messages for Spanish language
	if err != nil {
		return err
	}
	err = MigrateMessages(db, "en") // Migrate messages for English language
	if err != nil {
		return err
	}

	return nil

}
