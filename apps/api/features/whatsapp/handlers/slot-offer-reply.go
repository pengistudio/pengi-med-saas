package whatsapp_handlers

import (
	"strings"

	"pengi-med-saas/core/whatsapp"
	portal_services "pengi-med-saas/features/portal/services"
	waitlist_services "pengi-med-saas/features/waitlist/services"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The "Lo quiero" button of a waitlist slot offer ("slot_offer:<message
// id>"). Like "Reagendar": the tap is already in the inbox; the entry
// records that the patient wants it; when the clinic's patients can book
// online, Pengi answers with the portal link (KindSystem, free); otherwise,
// or when that text could not be sent, the users with MANAGE_APPOINTMENT or
// USE_WHATSAPP_INBOX are told. Nothing is booked or held: the booking
// follows the normal portal or reception flow.

// isSlotOffer reports whether payload is a "Lo quiero" tap.
func isSlotOffer(payload string) bool {
	action, _, ok := strings.Cut(payload, ":")
	return ok && action == whatsapp_models.ReplySlotOffer
}

// slotsFreed reacts to slots freed by the webhook (a reminder's "Cancelar").
func (h *WebhookHandler) slotsFreed() *waitlist_services.Slots {
	return waitlist_services.NewSlots(h.db, h.logger).WithClock(h.now, whatsapp_services.ClinicLocation())
}

// applySlotOffer handles a "Lo quiero" tap in conv (nil when the inbox could
// not record it).
func (h *WebhookHandler) applySlotOffer(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation, m whatsapp.WebhookMessage) {
	tenantID := account.TenantID
	_, rawID, _ := strings.Cut(m.ButtonPayload(), ":")
	msg, ok := h.answer(db, tenantID, m, rawID, whatsapp_models.ReplySlotOffer, whatsapp_models.KindSlotOffer)
	if !ok {
		return
	}
	entry, err := waitlist_services.OfferAccepted(db, msg.ID, h.now())
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to record the accepted slot offer", zap.Uint("message_id", msg.ID), zap.Error(err))
	}
	if conv == nil {
		c, err := whatsapp_services.EnsureConversation(db, tenantID, msg.ToPhone, msg.PatientID)
		if err != nil {
			h.logger.Error("whatsapp webhook: failed to ensure conversation for slot offer", zap.Uint("tenant_id", tenantID), zap.Error(err))
			return
		}
		conv = &c
	}

	link, err := portal_services.BookingLink(h.db, tenantID)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to resolve booking link", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
	if link != "" && h.sender != nil {
		body := strings.ReplaceAll(whatsapp_services.PatientText(whatsapp_services.SlotOfferBookingLinkKey), "{url}", link)
		if h.sendSystem(db, account, conv, body) {
			return
		}
	}
	waitlist_services.NotifyOfferAccepted(h.db, h.logger, tenantID, *conv, entry)
}
