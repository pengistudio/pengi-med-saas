package migrations

import (
	"fmt"
	"pengi-med-saas/core/database"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	billing_models "pengi-med-saas/features/billing/models"
	integration_models "pengi-med-saas/features/integrations/models"
	kanban_models "pengi-med-saas/features/kanban/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	i18n_messages "pengi-med-saas/i18n/messages"
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
		clinical_models.PrescriptionItem{},
		clinical_models.VitalSigns{},
		clinical_models.Appointment{},
		clinical_models.Cie10Code{},
		integration_models.TenantIntegration{},
		backoffice_models.BackofficeUser{},
		billing_models.Invoice{},
		billing_models.InvoiceItem{},
		billing_models.InvoiceCounter{},
		billing_models.CatalogItem{},
		kanban_models.Task{},
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

	filename := fmt.Sprintf("messages_%s.json", lang)
	return message_models.LoadMessagesFromFS(db, i18n_messages.FS, filename, lang)
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
