package whatsapp_services

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The functions here take a handle already bound to the conversation's tenant
// (tenantdb.For / tenantdb.ForTenant); tenantID is only stamped on new rows.

// EnsureConversation finds or creates the tenant's conversation with phone
// (digits-only E.164). A new conversation is linked to patientHint when given
// (the reminder's patient), else to the only patient of the tenant whose phone
// normalizes to phone; with none or several it stays unlinked. An existing
// unlinked conversation is linked to patientHint.
func EnsureConversation(db *gorm.DB, tenantID uint, phone string, patientHint *uint) (whatsapp_models.WhatsAppConversation, error) {
	var conv whatsapp_models.WhatsAppConversation
	if phone == "" {
		return conv, errors.New("whatsapp conversation without phone")
	}
	if err := db.Where("phone = ?", phone).Limit(1).Find(&conv).Error; err != nil {
		return conv, err
	}
	if conv.ID != 0 {
		if conv.PatientID == nil && patientHint != nil {
			if err := db.Model(&conv).Update("patient_id", *patientHint).Error; err != nil {
				return conv, err
			}
			conv.PatientID = patientHint
		}
		return conv, nil
	}

	patientID := patientHint
	if patientID == nil {
		id, err := MatchPatientByPhone(db, phone)
		if err != nil {
			return conv, err
		}
		patientID = id
	}
	conv = whatsapp_models.WhatsAppConversation{TenantID: tenantID, Phone: phone, PatientID: patientID}
	// A concurrent webhook may have created it first: keep that one.
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&conv).Error; err != nil {
		return conv, err
	}
	if conv.ID == 0 {
		if err := db.Where("phone = ?", phone).Limit(1).Find(&conv).Error; err != nil {
			return conv, err
		}
	}
	return conv, nil
}

// MatchPatientByPhone returns the id of the only patient of the tenant whose
// phone normalizes to phone, or nil when there are none or several. Phones are
// free text, so the match is done in Go.
func MatchPatientByPhone(db *gorm.DB, phone string) (*uint, error) {
	var patients []clinical_models.Patient
	if err := db.Select("id", "phone").Where("phone <> ''").Find(&patients).Error; err != nil {
		return nil, err
	}
	var match *uint
	for _, p := range patients {
		if n, err := NormalizePhone(p.Phone); err == nil && n == phone {
			if match != nil {
				return nil, nil
			}
			id := p.ID
			match = &id
		}
	}
	return match, nil
}

// RecordOutbound puts msg in conv's thread: it creates msg (ID 0) or attaches
// an existing one, and moves the conversation's last message, in one
// transaction. A reply (sent by a user) also clears the unread count: whoever
// answered has read the thread.
func RecordOutbound(db *gorm.DB, conv *whatsapp_models.WhatsAppConversation, msg *whatsapp_models.WhatsAppMessage, at time.Time) error {
	msg.ConversationID = &conv.ID
	msg.Direction = whatsapp_models.DirectionOutbound
	if msg.ContentType == "" {
		msg.ContentType = whatsapp_models.ContentText
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if msg.ID == 0 {
			if err := tx.Create(msg).Error; err != nil {
				return err
			}
		} else if err := tx.Model(msg).Updates(map[string]any{
			"conversation_id": conv.ID, "direction": msg.Direction, "body": msg.Body, "content_type": msg.ContentType,
		}).Error; err != nil {
			return err
		}
		updates := lastMessageUpdates(conv, msg, at)
		if msg.Kind == whatsapp_models.KindReply {
			updates["unread_count"] = 0
		}
		return applyConversationUpdates(tx, conv, updates)
	})
}

// RecordInbound stores the patient's message msg in conv's thread, moves the
// conversation's last message and reply window and counts it as unread, in
// one transaction. Meta retries webhooks, so a message whose WamID is already
// stored is skipped: recorded is false and nothing changes.
func RecordInbound(db *gorm.DB, conv *whatsapp_models.WhatsAppConversation, msg *whatsapp_models.WhatsAppMessage, at time.Time) (recorded bool, err error) {
	if msg.WamID != "" {
		var count int64
		if err := db.Model(&whatsapp_models.WhatsAppMessage{}).
			Where("wam_id = ? AND direction = ?", msg.WamID, whatsapp_models.DirectionInbound).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}
	msg.ConversationID = &conv.ID
	msg.Direction = whatsapp_models.DirectionInbound
	msg.Kind = whatsapp_models.KindInbound
	msg.Status = whatsapp_models.MessageStatusReceived
	msg.ToPhone = conv.Phone
	if msg.PatientID == nil {
		msg.PatientID = conv.PatientID
	}
	if msg.ContentType == "" {
		msg.ContentType = whatsapp_models.ContentText
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		updates := lastMessageUpdates(conv, msg, at)
		updates["unread_count"] = gorm.Expr("unread_count + 1")
		if conv.LastInboundAt == nil || at.After(*conv.LastInboundAt) {
			updates["last_inbound_at"] = at
		}
		return applyConversationUpdates(tx, conv, updates)
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// lastMessageUpdates moves the conversation's last message to msg, unless a
// later message is already there (a late webhook).
func lastMessageUpdates(conv *whatsapp_models.WhatsAppConversation, msg *whatsapp_models.WhatsAppMessage, at time.Time) map[string]any {
	updates := map[string]any{}
	if conv.LastMessageAt == nil || !at.Before(*conv.LastMessageAt) {
		updates["last_message_at"] = at
		updates["last_message_preview"] = Preview(msg.Body)
		updates["last_direction"] = msg.Direction
		updates["last_content_type"] = msg.ContentType
	}
	return updates
}

// applyConversationUpdates writes updates and reloads conv.
func applyConversationUpdates(tx *gorm.DB, conv *whatsapp_models.WhatsAppConversation, updates map[string]any) error {
	if len(updates) > 0 {
		if err := tx.Model(&whatsapp_models.WhatsAppConversation{}).Where("id = ?", conv.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return tx.Where("id = ?", conv.ID).Limit(1).Find(conv).Error
}

// Preview shortens body to PreviewLength runes on one line.
func Preview(body string) string {
	s := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(s) <= whatsapp_models.PreviewLength {
		return s
	}
	r := []rune(s)
	return string(r[:whatsapp_models.PreviewLength-1]) + "…"
}

// WindowOpen reports whether free text can be sent to conv at now: the
// patient wrote within the last 24 h.
func WindowOpen(conv whatsapp_models.WhatsAppConversation, now time.Time) bool {
	return conv.WindowOpen(now)
}
