package whatsapp_handlers

import (
	"strings"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/clause"
)

// warnAccountEvents are the account_update events that mean Meta acted against
// the account (or that it is gone).
var warnAccountEvents = map[string]bool{
	"ACCOUNT_VIOLATION": true, "DISABLED_UPDATE": true, "ACCOUNT_RESTRICTION": true,
	"ACCOUNT_OFFBOARDED": true, "ACCOUNT_DELETED": true,
}

// recordAccountEvent logs and stores one notice from Meta about an account. It
// looks no account up (there may be none yet) and changes none; a failure to
// store it is logged, with the payload, and nothing else.
func (h *WebhookHandler) recordAccountEvent(entry whatsapp.WebhookEntry, change whatsapp.WebhookChange) {
	n := change.AccountNotice()
	event := whatsapp_models.WhatsAppAccountEvent{
		WabaID: n.WabaID, PhoneNumberID: n.PhoneNumberID, PhoneNumber: n.PhoneNumber,
		Field: change.Field, Event: n.Event, Detail: n.Detail(),
		Payload: strings.ToValidUTF8(string(change.Raw), "�"),
	}
	if event.WabaID == "" {
		event.WabaID = entry.ID
	}
	if !entry.Time.IsZero() {
		at := entry.Time.Time
		event.OccurredAt = &at
	}
	event.SetFingerprint()

	level := zapcore.InfoLevel
	if warnAccountEvents[n.Event] ||
		// Meta's enforcement, or the clinic deleted its WhatsApp account.
		(n.Event == "PARTNER_REMOVED" && n.DisconnectionInfo.Reason == "ACCOUNT_DISCONNECTED") ||
		(change.Field == whatsapp.FieldAccountReviewUpdate && !strings.EqualFold(n.Decision, "APPROVED")) {
		level = zapcore.WarnLevel
	}
	fields := []zap.Field{
		zap.String("field", event.Field), zap.String("waba_id", event.WabaID),
		zap.String("event", event.Event), zap.String("detail", event.Detail),
	}
	h.logger.Log(level, "whatsapp webhook: account event", fields...)

	err := tenantdb.System(h.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fingerprint"}}, DoNothing: true,
	}).Create(&event).Error
	if err != nil {
		h.logger.Error("whatsapp webhook: failed to store account event",
			append(fields, zap.String("payload", event.Payload), zap.Error(err))...)
	}
}
