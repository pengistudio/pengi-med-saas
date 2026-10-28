package clinical_workers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

type fakePublisher struct{ ids []uint }

func (p *fakePublisher) Publish(_ string, body []byte) error {
	var task whatsapp_services.SendTask
	if err := json.Unmarshal(body, &task); err != nil {
		return err
	}
	p.ids = append(p.ids, task.MessageID)
	return nil
}

// An appointment the scheduler marks no_show gets the WhatsApp no-show
// message queued when the clinic turned it on.
func TestNoShowScheduler_QueuesWhatsAppMessage(t *testing.T) {
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatal(err)
	}
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.MedicalRecord{}, &clinical_models.VitalSigns{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{},
		&whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppMessage{}, &whatsapp_models.WhatsAppTemplate{})
	f := &noShowFixture{t: t, db: db, loc: loc}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, loc)
	since := time.Date(2026, 10, 27, 0, 0, 0, 0, loc)

	withMessage := f.tenant(agendaOn(since, 30))
	withoutMessage := f.tenant(agendaOn(since, 30)) // switch off
	for _, tenant := range []tenant_models.Tenant{withMessage, withoutMessage} {
		acc := whatsapp_models.WhatsAppAccount{TenantID: tenant.ID, WabaID: fmt.Sprintf("W-%d", tenant.ID), PhoneNumberID: fmt.Sprintf("P-%d", tenant.ID),
			Status: whatsapp_models.AccountStatusConnected, NoShowMessageEnabled: tenant.ID == withMessage.ID}
		if err := db.Create(&acc).Error; err != nil {
			t.Fatal(err)
		}
		if err := whatsapp_services.SetTemplateStatus(db, tenant.ID, whatsapp_models.TemplateNoShow, whatsapp_models.TemplateStatusApproved, ""); err != nil {
			t.Fatal(err)
		}
	}
	reachable := func(tenantID uint) uint {
		p := f.patient(tenantID)
		db.Model(&p).Updates(map[string]any{"phone": "0991234567", "whatsapp_opt_in": true})
		return p.ID
	}
	marked := f.appt(withMessage.ID, reachable(withMessage.ID), nil, now, "09:00", "09:30", "scheduled", "")
	otherClinic := f.appt(withoutMessage.ID, reachable(withoutMessage.ID), nil, now, "09:00", "09:30", "scheduled", "")

	pub := &fakePublisher{}
	clock := func() time.Time { return now }
	s := NewNoShowScheduler(db, zap.NewNop(), nil).
		WithMessages(whatsapp_services.NewNoShowMessages(db, zap.NewNop(), pub)).
		WithClock(clock, loc)
	if n := s.Tick(); n != 2 {
		t.Fatalf("marked %d, want 2", n)
	}
	s.Tick()

	var msgs []whatsapp_models.WhatsAppMessage
	db.Where("kind = ?", whatsapp_models.KindNoShow).Find(&msgs)
	if len(msgs) != 1 || *msgs[0].AppointmentID != marked.ID || msgs[0].TenantID != withMessage.ID || msgs[0].Status != whatsapp_models.MessageStatusQueued {
		t.Fatalf("messages = %+v (other clinic's appointment %d)", msgs, otherClinic.ID)
	}
	if len(pub.ids) != 1 || pub.ids[0] != msgs[0].ID {
		t.Fatalf("published = %v", pub.ids)
	}
}
