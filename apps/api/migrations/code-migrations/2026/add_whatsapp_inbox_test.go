package y2026

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

func TestAddWhatsAppInboxPermission(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	if _, ok := database.GlobalDBMap[addWhatsAppInboxPermissionID]; !ok {
		t.Fatalf("migration %s not registered", addWhatsAppInboxPermissionID)
	}
	admin := createRoleWith(t, db, "admin")
	doctor := createRoleWith(t, db, "doctor")
	recep := createRoleWith(t, db, "recepcionista")
	contador := createRoleWith(t, db, "contador")

	for i := 0; i < 2; i++ { // idempotent
		if err := addWhatsAppInboxPermission(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []user_models.Role{admin, doctor, recep} {
		if ids := rolePermissionIDs(t, db, r.ID); !has(ids, "USE_WHATSAPP_INBOX") {
			t.Errorf("%s permissions = %v", r.Role, ids)
		}
	}
	if ids := rolePermissionIDs(t, db, contador.ID); has(ids, "USE_WHATSAPP_INBOX") {
		t.Error("contador must not get the inbox")
	}
	var feature company_models.Feature
	if err := db.Preload("Permissions").Where("code = ?", "WHATSAPP").First(&feature).Error; err != nil {
		t.Fatal(err)
	}
	if len(feature.Permissions) != 1 || feature.Permissions[0].ID != "USE_WHATSAPP_INBOX" || feature.Permissions[0].Category != "WHATSAPP" {
		t.Fatalf("WHATSAPP feature permissions = %+v", feature.Permissions)
	}
}

func TestBackfillWhatsAppConversations(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&whatsapp_models.WhatsAppMessage{}, &whatsapp_models.WhatsAppConversation{})
	if _, ok := database.GlobalDBMap[backfillWhatsAppConversationsID]; !ok {
		t.Fatalf("migration %s not registered", backfillWhatsAppConversationsID)
	}
	stamp := time.Now().UnixNano()
	var tenants [2]tenant_models.Tenant
	for i := range tenants {
		slug := fmt.Sprintf("wa-bf-%d-%d", i, stamp)
		tenants[i] = tenant_models.Tenant{Name: fmt.Sprintf("Clínica %d", i), Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenants[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	own, other := tenants[0].ID, tenants[1].ID
	patient := clinical_models.Patient{TenantID: own, FirstName: "Ana", LastName: "Pérez", Phone: "0991234567", Document: fmt.Sprintf("D%d", stamp)}
	if err := db.Create(&patient).Error; err != nil {
		t.Fatal(err)
	}
	appt := clinical_models.Appointment{TenantID: own, PatientID: patient.ID, Title: "Control",
		Date: time.Date(2026, 10, 6, 5, 0, 0, 0, time.UTC), StartTime: "10:30", EndTime: "11:00", Status: "scheduled"}
	if err := db.Create(&appt).Error; err != nil {
		t.Fatal(err)
	}
	sentAt := time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC)
	msgs := []whatsapp_models.WhatsAppMessage{
		{TenantID: own, Kind: whatsapp_models.KindReminder, AppointmentID: &appt.ID, PatientID: &patient.ID, OffsetHours: 24,
			ToPhone: "593991234567", Status: whatsapp_models.MessageStatusRead, SentAt: &sentAt},
		{TenantID: own, Kind: whatsapp_models.KindTest, ToPhone: "593991234567", Status: whatsapp_models.MessageStatusSent},
		{TenantID: own, Kind: whatsapp_models.KindTest, ToPhone: "593980000000", Status: whatsapp_models.MessageStatusFailed},
		{TenantID: own, Kind: whatsapp_models.KindReminder, OffsetHours: 2, Status: whatsapp_models.MessageStatusSkipped}, // never had a phone
		{TenantID: other, Kind: whatsapp_models.KindTest, ToPhone: "593991234567", Status: whatsapp_models.MessageStatusSent},
	}
	for i := range msgs {
		if err := db.Create(&msgs[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ { // a second run finds nothing left to do
		if err := backfillWhatsAppConversations(db); err != nil {
			t.Fatal(err)
		}
	}

	var convs []whatsapp_models.WhatsAppConversation
	db.Order("tenant_id, phone").Find(&convs)
	if len(convs) != 3 {
		t.Fatalf("conversations = %+v", convs)
	}
	byKey := map[string]whatsapp_models.WhatsAppConversation{}
	for _, c := range convs {
		byKey[fmt.Sprintf("%d:%s", c.TenantID, c.Phone)] = c
	}
	ana := byKey[fmt.Sprintf("%d:593991234567", own)]
	if ana.PatientID == nil || *ana.PatientID != patient.ID || ana.LastMessageAt == nil || ana.UnreadCount != 0 ||
		ana.LastDirection != whatsapp_models.DirectionOutbound || ana.LastInboundAt != nil {
		t.Fatalf("ana conversation = %+v", ana)
	}
	if _, ok := byKey[fmt.Sprintf("%d:593991234567", other)]; !ok {
		t.Fatal("the other tenant's message must get its own conversation")
	}

	var stored []whatsapp_models.WhatsAppMessage
	db.Order("id").Find(&stored)
	reminder, test, failed, skipped, foreign := stored[0], stored[1], stored[2], stored[3], stored[4]
	if reminder.ConversationID == nil || *reminder.ConversationID != ana.ID ||
		reminder.Body != "Hola Ana Pérez, te recordamos tu cita en Clínica 0 el martes 6 de octubre a las 10:30. Por favor confirma tu asistencia o cancela si no podrás asistir." {
		t.Fatalf("reminder = %+v", reminder)
	}
	if test.ConversationID == nil || *test.ConversationID != ana.ID || !strings.HasPrefix(test.Body, "Hola Paciente de prueba, te recordamos tu cita en Clínica 0") {
		t.Fatalf("test = %+v", test)
	}
	if failed.ConversationID == nil || failed.Body != "" {
		t.Fatalf("failed = %+v (in its thread, nothing was said)", failed)
	}
	if skipped.ConversationID != nil {
		t.Fatalf("skipped = %+v (no phone, no conversation)", skipped)
	}
	if foreign.ConversationID == nil || *foreign.ConversationID == ana.ID {
		t.Fatalf("foreign = %+v", foreign)
	}
}
