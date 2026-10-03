package migrations

import (
	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	integration_models "pengi-med-saas/features/integrations/models"
	kanban_models "pengi-med-saas/features/kanban/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	settings_models "pengi-med-saas/features/settings/models"
	signature_models "pengi-med-saas/features/signatures/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"

	_ "pengi-med-saas/migrations/code-migrations/2026"
)

func RunMigrations(db *gorm.DB) error {
	// Migrations and seeds span every tenant (docs/adr/0002).
	db = tenantdb.System(db)
	err := database.MigrateDB(
		db,
		database.DBExecute{},
		tenant_models.Tenant{},
		permission_models.Permission{},
		company_models.Company{},
		company_models.Plan{},
		company_models.PlanPricing{},
		company_models.Subscription{},
		company_models.SubscriptionPayment{},
		company_models.Feature{},
		user_models.User{},
		user_models.Environment{},
		user_models.Role{},
		user_models.RefreshToken{},
		clinical_models.Patient{},
		clinical_models.MedicalRecord{},
		clinical_models.MedicalRecordDraft{},
		clinical_models.SOAPRecord{},
		clinical_models.Prescription{},
		clinical_models.PrescriptionItem{},
		clinical_models.VitalSigns{},
		clinical_models.Appointment{},
		clinical_models.Cie10Code{},
		clinical_models.MedicalReport{},
		clinical_models.MedicalCertificate{},
		clinical_models.PatientAttachment{},
		signature_models.UserSignature{},
		integration_models.TenantIntegration{},
		backoffice_models.BackofficeUser{},
		billing_models.Invoice{},
		billing_models.InvoiceItem{},
		billing_models.InvoiceCounter{},
		billing_models.CatalogItem{},
		billing_models.CreditNote{},
		billing_models.CreditNoteItem{},
		billing_models.DebitNote{},
		billing_models.DebitNoteMotive{},
		kanban_models.Task{},
		settings_models.SystemSetting{},
		notifications_models.Notification{},
		notifications_models.Announcement{},
	)
	if err != nil {
		return err
	}

	return database.ExecuteAll(db)
}

// RunAllMigrations runs AutoMigrate and the code migrations. Messages are not
// seeded: the message catalog is read from the binary (docs/adr/0003).
func RunAllMigrations(db *gorm.DB) error {
	return RunMigrations(db)
}
