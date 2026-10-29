package whatsapp_handlers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/config"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// seedSlotOffer gives tenantID a connected account on phoneID and a waitlist
// entry offered a slot by a sent slot offer message.
func (f *fixture) seedSlotOffer(tenantID uint, phoneID string) (waitlist_models.WaitlistEntry, whatsapp_models.WhatsAppMessage) {
	f.t.Helper()
	if err := f.db.AutoMigrate(&waitlist_models.WaitlistEntry{}, &doctor_models.Doctor{}, &agenda_models.AppointmentType{}); err != nil {
		f.t.Fatal(err)
	}
	if f.account(tenantID).ID == 0 {
		f.connectedAccount(tenantID, phoneID)
	}
	patient := clinical_models.Patient{TenantID: tenantID, FirstName: "Luis", LastName: "Mora", Phone: "0991234567", WhatsAppOptIn: true,
		Document: fmt.Sprintf("D%d", time.Now().UnixNano())}
	if err := f.db.Create(&patient).Error; err != nil {
		f.t.Fatal(err)
	}
	patientID := patient.ID
	msg := whatsapp_models.WhatsAppMessage{TenantID: tenantID, Kind: whatsapp_models.KindSlotOffer, PatientID: &patientID,
		Template: whatsapp_models.TemplateSlotOffer, Status: whatsapp_models.MessageStatusSent,
		WamID: fmt.Sprintf("wamid.so.%d", time.Now().UnixNano()), ToPhone: "593991234567"}
	if err := f.db.Create(&msg).Error; err != nil {
		f.t.Fatal(err)
	}
	at := time.Now()
	msgID := msg.ID
	entry := waitlist_models.WaitlistEntry{TenantID: tenantID, PatientID: patientID, Status: waitlist_models.StatusOffered,
		ExpiresAt: at.Add(24 * time.Hour), OfferedAt: &at, OfferedDate: "2026-11-04", OfferedStartTime: "10:00",
		OfferChannel: waitlist_models.ChannelWhatsApp, OfferMessageID: &msgID}
	if err := f.db.Create(&entry).Error; err != nil {
		f.t.Fatal(err)
	}
	return entry, msg
}

func (f *fixture) waitlistEntry(id uint) waitlist_models.WaitlistEntry {
	var e waitlist_models.WaitlistEntry
	f.db.First(&e, id)
	return e
}

func TestWebhook_SlotOfferWithoutPortalNotifiesStaff(t *testing.T) {
	f := newFixture(t)
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	f.grant(f.own, []uint{inboxUserID}, "USE_WHATSAPP_INBOX")
	f.grant(f.other, []uint{110}, waitlist_services.ManagePermission)
	entry, msg := f.seedSlotOffer(f.own, "PHONE-1")

	f.signed(replyEvent("PHONE-1", fmt.Sprintf("slot_offer:%d", msg.ID), "593991234567", msg.WamID))

	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusReplied || m.Reply != whatsapp_models.ReplySlotOffer {
		t.Fatalf("message = %+v", m)
	}
	if e := f.waitlistEntry(entry.ID); e.OfferAcceptedAt == nil || e.Status != waitlist_models.StatusOffered {
		t.Fatalf("entry = %+v", e)
	}
	for _, uid := range []uint{100, inboxUserID} {
		ns := f.notifications(uid, waitlist_services.NotificationTypeOfferAccepted)
		if len(ns) != 1 || ns[0].MessageKey != waitlist_services.OfferAcceptedKey || !strings.HasPrefix(ns[0].ActionURL, "/whatsapp?c=") {
			t.Fatalf("notifications of %d = %+v", uid, ns)
		}
	}
	if len(f.notifications(110, waitlist_services.NotificationTypeOfferAccepted)) != 0 {
		t.Fatal("another clinic was told")
	}
	if f.graph.called("POST /v23.0/PHONE-1/messages") {
		t.Fatal("without online booking nothing is sent to the patient")
	}

	// The same tap delivered again: nothing more.
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("slot_offer:%d", msg.ID), "593991234567", msg.WamID))
	if ns := f.notifications(100, waitlist_services.NotificationTypeOfferAccepted); len(ns) != 1 {
		t.Fatalf("notifications after a retried tap = %d", len(ns))
	}
}

func TestWebhook_SlotOfferChecksKindAndSender(t *testing.T) {
	f := newFixture(t)
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	_, msg := f.seedSlotOffer(f.own, "PHONE-1")
	// From another number: ignored.
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("slot_offer:%d", msg.ID), "593987654321", msg.WamID))
	if m := f.message(msg.ID); m.Status != whatsapp_models.MessageStatusSent {
		t.Fatalf("message = %s", m.Status)
	}
	// "slot_offer" on a message of another kind: ignored.
	_, noShow := f.seedNoShow(f.own, "PHONE-1")
	f.signed(replyEvent("PHONE-1", fmt.Sprintf("slot_offer:%d", noShow.ID), "593991234567", noShow.WamID))
	if m := f.message(noShow.ID); m.Status == whatsapp_models.MessageStatusReplied {
		t.Fatal("a no-show message was answered as a slot offer")
	}
	if len(f.notifications(100, waitlist_services.NotificationTypeOfferAccepted)) != 0 {
		t.Fatal("no notification for ignored taps")
	}
}

func TestWebhook_SlotOfferWithPortalRepliesWithLink(t *testing.T) {
	if config.Get().PortalURL == "" {
		t.Skip("PORTAL_URL not set")
	}
	f := newFixture(t)
	f.portalBooking(f.own)
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	_, msg := f.seedSlotOffer(f.own, "PHONE-1")

	f.signed(replyEvent("PHONE-1", fmt.Sprintf("slot_offer:%d", msg.ID), "593991234567", msg.WamID))

	link := strings.TrimRight(config.Get().PortalURL, "/") + "/" + f.slug(f.own)
	sent := f.graph.bodyWith("POST /v23.0/PHONE-1/messages", `"type":"text"`)
	if !strings.Contains(sent, link) {
		t.Fatalf("graph body = %q, want the link %s", sent, link)
	}
	if ns := f.notifications(100, waitlist_services.NotificationTypeOfferAccepted); len(ns) != 0 {
		t.Fatalf("with online booking the staff is not told: %+v", ns)
	}
}

func TestWebhook_CancelFreesSlotForWaitlist(t *testing.T) {
	f := newFixture(t)
	if err := f.db.AutoMigrate(&waitlist_models.WaitlistEntry{}, &doctor_models.Doctor{}); err != nil {
		t.Fatal(err)
	}
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	appt, msg := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	// Somebody else waits for any doctor.
	waiting := clinical_models.Patient{TenantID: f.own, FirstName: "Ana", Document: fmt.Sprintf("W%d", time.Now().UnixNano())}
	f.db.Create(&waiting)
	f.db.Create(&waitlist_models.WaitlistEntry{TenantID: f.own, PatientID: waiting.ID, Status: waitlist_models.StatusWaiting,
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour)})

	f.signed(replyEvent("PHONE-OWN", fmt.Sprintf("cancel:%d", msg.ID), "593991234567", msg.WamID))
	if s := f.appointmentStatus(appt.ID); s != "cancelled" {
		t.Fatalf("status = %s", s)
	}
	if ns := f.notifications(100, waitlist_services.NotificationTypeSlotFreed); len(ns) != 1 {
		t.Fatalf("slot notifications = %d", len(ns))
	}
}
