package whatsapp_handlers

import (
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Asking for WhatsApp consent by WhatsApp. When a number of patient(s)
// without consent writes, Pengi answers once per conversation (free text,
// inside the window the patient just opened) "responde SÍ"; a later SÍ / SI /
// ACEPTO / OK grants consent to the patients of that number. Both texts are
// KindSystem: in the thread, not counted toward the monthly cap.

// handleConsent runs the consent flow for a text the patient sent in conv.
func (h *WebhookHandler) handleConsent(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation, body string) {
	if h.sender == nil {
		return
	}
	// A loose "sí" in a chat is not consent: only after we asked.
	if conv.OptInRequestedAt != nil && whatsapp_services.IsOptInYes(body) {
		h.acceptOptIn(db, account, conv)
		return
	}
	h.askOptIn(db, account, conv)
}

// askOptIn asks for consent unless already asked in conv, reminders can't be
// sent, the number matches no patient, a patient of the number opted out with
// STOP, or they all consented already.
func (h *WebhookHandler) askOptIn(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation) {
	if conv.OptInRequestedAt != nil || !account.CanSendReminders() {
		return
	}
	patients, err := whatsapp_services.PatientsByPhone(db, conv.Phone)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to load patients for consent", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return
	}
	missing := false
	for _, p := range patients {
		if p.WhatsAppOptInSource == clinical_models.WhatsAppOptInSourceStop {
			return
		}
		if !p.WhatsAppOptIn {
			missing = true
		}
	}
	if !missing {
		return
	}
	// Claim the one ask of this conversation (a concurrent webhook loses).
	now := h.now()
	res := db.Model(&whatsapp_models.WhatsAppConversation{}).
		Where("id = ? AND opt_in_requested_at IS NULL", conv.ID).
		Update("opt_in_requested_at", now)
	if res.Error != nil {
		h.logger.Error("whatsapp webhook: failed to mark consent asked", zap.Uint("conversation_id", conv.ID), zap.Error(res.Error))
		return
	}
	if res.RowsAffected == 0 {
		return
	}
	conv.OptInRequestedAt = &now
	h.sendSystem(db, account, conv, whatsapp_services.PatientText(whatsapp_services.OptInRequestKey))
}

// acceptOptIn grants consent to the patients of conv's number that lack it
// and confirms it; nothing happens when they all have it already.
func (h *WebhookHandler) acceptOptIn(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation) {
	patients, err := whatsapp_services.PatientsByPhone(db, conv.Phone)
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to load patients for consent", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return
	}
	var ids []uint
	for _, p := range patients {
		if !p.WhatsAppOptIn {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	if err := db.Model(&clinical_models.Patient{}).Where("id IN ?", ids).Updates(map[string]any{
		"whatsapp_opt_in": true, "whatsapp_opt_in_at": h.now(), "whatsapp_opt_in_source": clinical_models.WhatsAppOptInSourceWhatsApp,
	}).Error; err != nil {
		h.logger.Error("whatsapp webhook: failed to grant consent", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return
	}
	h.logger.Info("whatsapp webhook: patients opted in by whatsapp", zap.Uint("tenant_id", account.TenantID), zap.Int("count", len(ids)))
	h.sendSystem(db, account, conv, whatsapp_services.PatientText(whatsapp_services.OptInConfirmedKey))
}

// sendSystem records body in conv's thread as a KindSystem message and sends
// it as free text. Failures are logged; the message stays failed.
func (h *WebhookHandler) sendSystem(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation, body string) {
	token, err := account.OpenAccessToken()
	if err != nil {
		h.logger.Error("whatsapp webhook: token unusable for system message", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return
	}
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    account.TenantID,
		Kind:        whatsapp_models.KindSystem,
		PatientID:   conv.PatientID,
		ToPhone:     conv.Phone,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        body,
		ContentType: whatsapp_models.ContentText,
	}
	if err := whatsapp_services.RecordOutbound(db, conv, &msg, h.now()); err != nil {
		h.logger.Error("whatsapp webhook: failed to record system message", zap.Uint("conversation_id", conv.ID), zap.Error(err))
		return
	}
	if err := h.sender.DeliverText(db, &msg, account, token); err != nil {
		h.logger.Warn("whatsapp webhook: system message not sent", zap.Uint("message_id", msg.ID), zap.Error(err))
	}
}
