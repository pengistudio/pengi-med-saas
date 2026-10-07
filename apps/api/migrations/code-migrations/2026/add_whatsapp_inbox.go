package y2026

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

const (
	addWhatsAppInboxPermissionID     = "DB20261005_3"
	backfillWhatsAppConversationsID  = "DB20261005_4"
	whatsAppInboxPermission          = "USE_WHATSAPP_INBOX"
	whatsAppInboxTestPatientName     = "Paciente de prueba" // what SendTest put in test messages
	whatsAppInboxTestAppointmentHour = 10
)

func init() {
	// Creates USE_WHATSAPP_INBOX, adds it to the WHATSAPP Feature (so every plan
	// that already has WhatsApp gets the inbox) and gives it to admin,
	// recepcionista and doctor, as role_data.RolePermissionMatrix does for new
	// installs.
	database.GlobalDBMap[addWhatsAppInboxPermissionID] = database.DBExecute{
		ID:      addWhatsAppInboxPermissionID,
		Execute: addWhatsAppInboxPermission,
	}

	// Groups the existing WhatsApp messages into conversations (tenant +
	// to_phone), links them and fills the body of sent reminders and tests.
	database.GlobalDBMap[backfillWhatsAppConversationsID] = database.DBExecute{
		ID:      backfillWhatsAppConversationsID,
		Execute: backfillWhatsAppConversations,
	}
}

func addWhatsAppInboxPermission(db *gorm.DB) error {
	var perm permission_models.Permission
	for _, p := range permission_data.WhatsAppPermissions {
		if p.ID == whatsAppInboxPermission {
			perm = p
		}
	}
	if perm.ID == "" {
		return fmt.Errorf("%s missing from the WhatsApp permission catalog", whatsAppInboxPermission)
	}
	if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
		return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
	}

	feature := company_models.Feature{Code: "WHATSAPP", Name: "WhatsApp"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to find WHATSAPP feature: %w", err)
	}
	if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
		return fmt.Errorf("failed to associate '%s' with WHATSAPP feature: %w", perm.ID, err)
	}

	for _, roleName := range []string{role_data.RoleAdmin, role_data.RoleRecepcionista, role_data.RoleDoctor} {
		var role user_models.Role
		if err := db.Where(user_models.Role{Role: roleName}).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && roleName != role_data.RoleAdmin {
				fmt.Printf("⚠️  %s role not found, skipping %s.\n", roleName, perm.ID)
				continue
			}
			return fmt.Errorf("failed to find %s role: %w", roleName, err)
		}
		if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("failed to assign '%s' to %s role: %w", perm.ID, roleName, err)
		}
		fmt.Printf("✅ Assigned permission '%s' to %s role.\n", perm.ID, roleName)
	}
	return nil
}

func backfillWhatsAppConversations(db *gorm.DB) error {
	var messages []whatsapp_models.WhatsAppMessage
	err := tenantdb.System(db).
		Where("conversation_id IS NULL AND to_phone <> '' AND direction = ?", whatsapp_models.DirectionOutbound).
		Order("id ASC").Find(&messages).Error
	if err != nil {
		return fmt.Errorf("failed to load whatsapp messages: %w", err)
	}
	loc := whatsapp_services.ClinicLocation()
	clinics := map[uint]string{}
	for i := range messages {
		msg := &messages[i]
		tdb := tenantdb.ForTenant(db, msg.TenantID)
		conv, err := whatsapp_services.EnsureConversation(tdb, msg.TenantID, msg.ToPhone, msg.PatientID)
		if err != nil {
			return fmt.Errorf("failed to ensure conversation for message %d: %w", msg.ID, err)
		}
		if msg.Body == "" && wasSent(msg.Status) {
			if _, ok := clinics[msg.TenantID]; !ok {
				clinics[msg.TenantID] = whatsapp_services.ClinicName(db, msg.TenantID)
			}
			if data, ok := backfillReminderData(tdb, *msg, clinics[msg.TenantID], loc); ok {
				msg.Body = whatsapp_services.RenderReminderBody(data)
			}
		}
		at := msg.CreatedAt
		if msg.SentAt != nil {
			at = *msg.SentAt
		}
		if err := whatsapp_services.RecordOutbound(tdb, &conv, msg, at); err != nil {
			return fmt.Errorf("failed to add message %d to its conversation: %w", msg.ID, err)
		}
	}
	fmt.Printf("✅ %d WhatsApp messages grouped into conversations.\n", len(messages))
	return nil
}

func wasSent(status string) bool {
	switch status {
	case whatsapp_models.MessageStatusSent, whatsapp_models.MessageStatusDelivered,
		whatsapp_models.MessageStatusRead, whatsapp_models.MessageStatusReplied:
		return true
	}
	return false
}

// backfillReminderData rebuilds what a sent message said: a reminder from
// its appointment, a test from SendTest's sample data (tomorrow 10:00 from
// when it was sent).
func backfillReminderData(db *gorm.DB, msg whatsapp_models.WhatsAppMessage, clinic string, loc *time.Location) (whatsapp_services.ReminderData, bool) {
	switch msg.Kind {
	case whatsapp_models.KindTest:
		day := msg.CreatedAt.In(loc).AddDate(0, 0, 1)
		return whatsapp_services.ReminderData{
			PatientName: whatsAppInboxTestPatientName,
			ClinicName:  clinic,
			Start:       time.Date(day.Year(), day.Month(), day.Day(), whatsAppInboxTestAppointmentHour, 0, 0, 0, loc),
		}, true
	case whatsapp_models.KindReminder:
		if msg.AppointmentID == nil {
			return whatsapp_services.ReminderData{}, false
		}
		var appt clinical_models.Appointment
		if err := db.Unscoped().Preload("Patient").Where("id = ?", *msg.AppointmentID).Limit(1).Find(&appt).Error; err != nil || appt.ID == 0 {
			return whatsapp_services.ReminderData{}, false
		}
		start, err := whatsapp_services.AppointmentStart(appt, loc)
		if err != nil {
			return whatsapp_services.ReminderData{}, false
		}
		return whatsapp_services.ReminderData{
			PatientName: strings.TrimSpace(appt.Patient.FirstName + " " + appt.Patient.LastName),
			ClinicName:  clinic,
			Start:       start,
		}, true
	}
	return whatsapp_services.ReminderData{}, false
}
