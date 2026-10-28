package clinical_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

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

// Reception marking an appointment no_show queues the WhatsApp no-show
// message; correcting the mark before it is sent withdraws it, and marking
// it again queues the same message again.
func TestUpdateStatus_NoShowQueuesWhatsAppMessage(t *testing.T) {
	s := newIsolation(t)
	if err := s.db.AutoMigrate(&whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppMessage{}, &whatsapp_models.WhatsAppTemplate{}); err != nil {
		t.Fatal(err)
	}
	for _, tenantID := range []uint{s.own.ID, s.other.ID} {
		acc := whatsapp_models.WhatsAppAccount{TenantID: tenantID, WabaID: fmt.Sprintf("W-%d", tenantID), PhoneNumberID: fmt.Sprintf("P-%d", tenantID),
			Status: whatsapp_models.AccountStatusConnected, NoShowMessageEnabled: true}
		if err := s.db.Create(&acc).Error; err != nil {
			t.Fatal(err)
		}
		if err := whatsapp_services.SetTemplateStatus(s.db, tenantID, whatsapp_models.TemplateNoShow, whatsapp_models.TemplateStatusApproved, ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []clinical_models.Patient{s.ownPatient, s.otherPatient} {
		s.db.Model(&p).Updates(map[string]any{"phone": "0991234567", "whatsapp_opt_in": true})
	}
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 11, 0, 0, 0, loc) // inside the send hours
	pub := &fakePublisher{}
	h := NewAppointmentHandler(s.db, zap.NewNop()).
		WithNoShowMessages(whatsapp_services.NewNoShowMessages(s.db, zap.NewNop(), pub).WithClock(func() time.Time { return now }, loc))
	appt := s.appointmentOn(s.own.ID, s.ownPatient.ID, now, "scheduled")

	set := func(id uint, status string) int {
		return h.UpdateStatus(s.ctx(id, map[string]string{"status": status})).Code
	}
	messages := func() []whatsapp_models.WhatsAppMessage {
		var msgs []whatsapp_models.WhatsAppMessage
		s.db.Where("kind = ?", whatsapp_models.KindNoShow).Order("id").Find(&msgs)
		return msgs
	}

	if code := set(appt.ID, "no_show"); code != http.StatusOK {
		t.Fatalf("no_show = %d", code)
	}
	msgs := messages()
	if len(msgs) != 1 || *msgs[0].AppointmentID != appt.ID || msgs[0].Status != whatsapp_models.MessageStatusQueued || len(pub.ids) != 1 {
		t.Fatalf("after mark: %+v, published %v", msgs, pub.ids)
	}

	set(appt.ID, "arrived")
	if msgs := messages(); len(msgs) != 1 || msgs[0].Status != whatsapp_models.MessageStatusSkipped || msgs[0].ErrorCode != whatsapp_models.ErrorCodeNoShowCorrected {
		t.Fatalf("after correction: %+v", msgs)
	}

	set(appt.ID, "no_show")
	if msgs := messages(); len(msgs) != 1 || msgs[0].Status != whatsapp_models.MessageStatusQueued || len(pub.ids) != 2 {
		t.Fatalf("after re-mark: %+v, published %v", msgs, pub.ids)
	}

	// Another clinic's appointment can't be marked from here: nothing queued for it.
	other := s.otherAppointment()
	if code := set(other.ID, "no_show"); code != http.StatusNotFound {
		t.Fatalf("other clinic = %d, want 404", code)
	}
	if msgs := messages(); len(msgs) != 1 {
		t.Fatalf("messages = %+v", msgs)
	}
}
