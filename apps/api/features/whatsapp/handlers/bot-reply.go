package whatsapp_handlers

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/tenantdb"
	ai_services "pengi-med-saas/features/ai/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The WhatsApp assistant on inbound messages. After the webhook stored a
// patient's message (and ran the consent flow), runBot decides, off the
// webhook's request, whether the assistant answers it, hands the chat to
// reception or stays silent. Every message is claimed once by its Meta id
// (WhatsAppBotEvent), so a retried webhook is never answered twice, and the
// chats of one conversation are handled one at a time.

// Assistant limits.
const (
	// MaxBotRepliesPerDay: assistant messages per chat and clinic day; the
	// next message hands the chat to reception.
	MaxBotRepliesPerDay = 6
	// BotHumanQuiet: after a person answered from the inbox the assistant
	// stays silent in that chat this long.
	BotHumanQuiet = 12 * time.Hour
	// botTimeout bounds one decision (the model call).
	botTimeout = 45 * time.Second
	// botWorkers bounds the messages handled at once (goroutines).
	botWorkers = 16
	// Audit entity of the assistant's actions.
	auditEntityBot = "whatsapp_conversations"
)

// MaxBotMessagesPerClinicDay caps the assistant's messages per clinic and
// clinic day (a flood can't run unbounded); beyond it chats are flagged for
// reception without a message. A var so tests can lower it.
var MaxBotMessagesPerClinicDay int64 = 300

// MaxBotEmergencyTextsPerClinicDay caps the 911 texts per clinic and clinic
// day (a ceiling for loops across many chats); beyond it emergencies are
// flagged for reception without a message. A var so tests can lower it.
var MaxBotEmergencyTextsPerClinicDay int64 = 100

// botRunner is what the webhook needs to run the assistant.
//
// Single-process assumption: the per-conversation lock (locks) and the
// worker slots live in this process. With several API instances two
// messages of one chat could be decided concurrently (the per-message claim
// in whatsapp_bot_events still prevents answering one message twice).
type botRunner struct {
	bot   *ai_services.Bot
	async func(func())
	locks *keyedLocks
	slots chan struct{}
}

func newBotRunner() botRunner {
	return botRunner{bot: ai_services.NewBot(nil), async: func(f func()) { go f() }, locks: newKeyedLocks(), slots: make(chan struct{}, botWorkers)}
}

// WithBotDialer makes the assistant dial its provider with dial (tests).
func (h *WebhookHandler) WithBotDialer(dial ai_services.Dialer) *WebhookHandler {
	h.botRun.bot = ai_services.NewBot(dial)
	return h
}

// queueBot hands msg (stored in conv) to the assistant, asynchronously, on
// one of botWorkers slots. When all are busy the message is not dropped: an
// emergency is still answered (inline), anything else flags the chat for
// reception.
func (h *WebhookHandler) queueBot(account whatsapp_models.WhatsAppAccount, conv whatsapp_models.WhatsAppConversation, msg whatsapp_models.WhatsAppMessage) {
	if h.sender == nil || h.botRun.bot == nil || msg.WamID == "" || msg.ID == 0 {
		return
	}
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("whatsapp bot: panic", zap.Uint("tenant_id", account.TenantID), zap.Any("panic", r))
			}
		}()
		h.runBot(account, conv.ID, msg)
	}
	select {
	case h.botRun.slots <- struct{}{}:
		h.botRun.async(func() {
			defer func() { <-h.botRun.slots }()
			run()
		})
	default:
		if ai_services.BotIsEmergency(msg.Body) {
			run()
			return
		}
		h.logger.Warn("whatsapp bot: overloaded, chat handed to reception", zap.Uint("tenant_id", account.TenantID))
		h.flagForHuman(tenantdb.ForTenant(h.db, account.TenantID), account.TenantID, conv, whatsapp_models.BotReasonOverloaded, h.now(), true)
	}
}

// botDecision is what the assistant does with one message. Message is the
// text to send ("" sends nothing); Flag hands the chat to reception.
type botDecision struct {
	Action      string
	Reason      string
	Message     string
	Flag        bool
	ModelCalled bool
	Outcome     ai_services.BotOutcome
	// NoNotify: flag the chat without notifying reception (a throttled
	// emergency repeat).
	NoNotify bool
	// EmergencyClaim: the 911 text was claimed (BotEmergencyTextAt); a
	// failed send releases it.
	EmergencyClaim bool
	Sent           bool
}

func silent(reason string) botDecision {
	return botDecision{Action: whatsapp_models.BotActionSilent, Reason: reason}
}

func (h *WebhookHandler) runBot(account whatsapp_models.WhatsAppAccount, convID uint, msg whatsapp_models.WhatsAppMessage) {
	tenantID := account.TenantID
	unlock := h.botRun.locks.lock(convID)
	defer unlock()

	now := h.now()
	bc, err := ai_services.LoadBotContext(h.db, tenantID, now)
	if err != nil {
		h.logger.Error("whatsapp bot: failed to load settings", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if !bc.Settings.Enabled {
		return
	}
	readiness, err := ai_services.CheckBotReadiness(h.db, tenantID, bc, bc.Settings.FAQ)
	if err != nil {
		h.logger.Error("whatsapp bot: failed to check readiness", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if !readiness.Active() {
		return // plan, opt-in or material gone since it was enabled: off
	}
	// Emergencies are answered even during office hours.
	emergency := ai_services.BotIsEmergency(msg.Body)
	if !emergency && !bc.Settings.DuringOfficeHours && bc.Hours.Open(now, whatsapp_services.ClinicLocation()) {
		return // reception is attending
	}

	db := tenantdb.ForTenant(h.db, tenantID)
	var conv whatsapp_models.WhatsAppConversation
	if err := db.Where("id = ?", convID).Limit(1).Find(&conv).Error; err != nil || conv.ID == 0 {
		return
	}
	event := whatsapp_models.WhatsAppBotEvent{TenantID: tenantID, WamID: msg.WamID, ConversationID: conv.ID, CreatedAt: now}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&event)
	if res.Error != nil {
		h.logger.Error("whatsapp bot: failed to claim message", zap.Uint("tenant_id", tenantID), zap.Error(res.Error))
		return
	}
	if res.RowsAffected == 0 || event.ID == 0 {
		return // already handled (Meta retried the webhook)
	}

	d := h.decideBot(db, account, bc, &conv, msg, emergency, now)
	if d.Message != "" {
		sent := h.sendBot(db, account, &conv, d.Message)
		d.Sent = sent
		if !sent && d.EmergencyClaim {
			if err := db.Model(&whatsapp_models.WhatsAppConversation{}).Where("id = ?", conv.ID).Update("bot_emergency_text_at", nil).Error; err != nil {
				h.logger.Error("whatsapp bot: failed to release emergency claim", zap.Uint("conversation_id", conv.ID), zap.Error(err))
			}
		}
		if !sent && d.Action == whatsapp_models.BotActionAnswer {
			d = botDecision{Action: whatsapp_models.BotActionHandoff, Reason: whatsapp_models.BotReasonSendFailed, Flag: true, ModelCalled: d.ModelCalled}
		}
	}
	if d.Flag {
		h.flagForHuman(db, tenantID, conv, d.Reason, now, !d.NoNotify)
	}
	h.recordBotEvent(db, tenantID, event.ID, conv, d)
}

// decideBot applies the chat rules, then the assistant's pipeline. An
// emergency goes first: whenever the window is open and the number did not
// opt out, it always gets the fixed 911 text and re-flags the chat —
// whatever a person replied, an earlier handoff or the daily limits.
func (h *WebhookHandler) decideBot(db *gorm.DB, account whatsapp_models.WhatsAppAccount, bc ai_services.BotContext, conv *whatsapp_models.WhatsAppConversation, msg whatsapp_models.WhatsAppMessage, emergency bool, now time.Time) botDecision {
	tenantID := account.TenantID
	if !conv.WindowOpen(now) {
		return silent(whatsapp_models.BotReasonOutsideWindow)
	}
	patients, err := whatsapp_services.PatientsByPhone(db, conv.Phone)
	if err != nil {
		h.logger.Error("whatsapp bot: failed to load patients", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return silent(whatsapp_models.BotReasonInternal)
	}
	for _, p := range patients {
		if !p.WhatsAppOptIn && p.WhatsAppOptInSource == clinical_models.WhatsAppOptInSourceStop {
			return silent(whatsapp_models.BotReasonOptedOut)
		}
	}
	counts, err := h.botMessagesSince(db, conv.ID, now)
	if err != nil {
		h.logger.Error("whatsapp bot: failed to count messages", zap.Uint("tenant_id", tenantID), zap.Error(err))
		counts = botCounts{} // an emergency still gets its text
		if !emergency {
			return silent(whatsapp_models.BotReasonInternal)
		}
	}
	disclose := counts.chatLastDay == 0
	if emergency {
		return h.emergencyDecision(db, bc, conv, msg, disclose, now)
	}

	var humanReplies int64
	if err := db.Model(&whatsapp_models.WhatsAppMessage{}).
		Where("conversation_id = ? AND direction = ? AND (sent_by_user_id IS NOT NULL OR kind = ?) AND created_at >= ?",
			conv.ID, whatsapp_models.DirectionOutbound, whatsapp_models.KindApp, now.Add(-BotHumanQuiet).UTC()).
		Count(&humanReplies).Error; err != nil {
		return silent(whatsapp_models.BotReasonInternal)
	}
	if humanReplies > 0 {
		return silent(whatsapp_models.BotReasonHumanActive)
	}
	if conv.NeedsHumanAt != nil && now.Sub(*conv.NeedsHumanAt) < whatsapp_models.BotSilence {
		return silent(whatsapp_models.BotReasonAwaitingHuman)
	}
	if counts.clinicToday >= MaxBotMessagesPerClinicDay {
		// Flood: no more automatic messages today; reception sees the chat.
		return botDecision{Action: whatsapp_models.BotActionHandoff, Reason: whatsapp_models.BotReasonClinicLimit, Flag: true}
	}
	if counts.chatToday >= MaxBotRepliesPerDay {
		return h.handoffMessage(bc, whatsapp_models.BotReasonDailyLimit, msg.Body, disclose)
	}
	if msg.ContentType != whatsapp_models.ContentText {
		return h.handoffMessage(bc, whatsapp_models.BotReasonNonText, msg.Body, disclose)
	}

	ctx, cancel := context.WithTimeout(context.Background(), botTimeout)
	defer cancel()
	outcome := h.botRun.bot.Decide(ctx, h.db, ai_services.BotRequest{
		TenantID:  tenantID,
		Text:      msg.Body,
		Knowledge: bc.Knowledge,
		Session:   ai_services.BotSession{UsageID: conv.BotSessionUsageID, LastAt: conv.BotSessionAt},
		Names:     h.patientNames(db, conv.PatientID),
	})
	if outcome.UsageID != nil {
		// BotSessionAt slides by design: a conversation (one quota use) is
		// the stretch between 24 h gaps without assistant model calls.
		if err := db.Model(&whatsapp_models.WhatsAppConversation{}).Where("id = ?", conv.ID).
			Updates(map[string]any{"bot_session_at": now, "bot_session_usage_id": *outcome.UsageID}).Error; err != nil {
			h.logger.Error("whatsapp bot: failed to store session", zap.Uint("conversation_id", conv.ID), zap.Error(err))
		}
	}
	d := botDecision{
		Action: outcome.Action, Reason: outcome.Reason, ModelCalled: outcome.ModelCalled, Outcome: outcome,
		Message: ai_services.ComposeBotMessage(outcome, bc.Knowledge, msg.Body, disclose),
		Flag:    outcome.Action != whatsapp_models.BotActionAnswer,
	}
	if d.Flag {
		d.Action = whatsapp_models.BotActionHandoff
	}
	return d
}

// emergencyDecision always re-flags the chat; the 911 text and the
// reception notification go out at most once per BotEmergencyThrottle per
// chat (so a patient repeating it, or an auto-responder echoing the text
// back, can't loop), and the text at most MaxBotEmergencyTextsPerClinicDay
// times per clinic day. Each throttle is claimed with a conditional UPDATE
// on the conversation (persisted: survives restarts; atomic: safe beside
// the per-chat lock and the message claim).
func (h *WebhookHandler) emergencyDecision(db *gorm.DB, bc ai_services.BotContext, conv *whatsapp_models.WhatsAppConversation, msg whatsapp_models.WhatsAppMessage, disclose bool, now time.Time) botDecision {
	d := botDecision{Action: whatsapp_models.BotActionHandoff, Reason: whatsapp_models.BotReasonEmergency, Flag: true}
	cutoff := now.Add(-whatsapp_models.BotEmergencyThrottle)
	claim := func(column string) bool {
		res := db.Model(&whatsapp_models.WhatsAppConversation{}).
			Where("id = ? AND ("+column+" IS NULL OR "+column+" <= ?)", conv.ID, cutoff).
			Update(column, now)
		if res.Error != nil {
			h.logger.Error("whatsapp bot: failed to claim emergency throttle", zap.Uint("conversation_id", conv.ID), zap.Error(res.Error))
			return true // on a storage error, err toward telling
		}
		return res.RowsAffected == 1
	}
	d.NoNotify = !claim("bot_emergency_notified_at")

	local := now.In(whatsapp_services.ClinicLocation())
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).UTC()
	var sentToday int64
	if err := db.Model(&whatsapp_models.WhatsAppBotEvent{}).
		Where("reason = ? AND message_sent = ? AND created_at >= ?", whatsapp_models.BotReasonEmergency, true, dayStart).
		Count(&sentToday).Error; err != nil {
		h.logger.Error("whatsapp bot: failed to count emergency texts", zap.Uint("conversation_id", conv.ID), zap.Error(err))
	}
	if sentToday >= MaxBotEmergencyTextsPerClinicDay {
		h.logger.Warn("whatsapp bot: clinic emergency text ceiling reached, flag only", zap.Uint("conversation_id", conv.ID))
		return d
	}
	if claim("bot_emergency_text_at") {
		d.EmergencyClaim = true
		d.Message = h.handoffMessage(bc, whatsapp_models.BotReasonEmergency, msg.Body, disclose).Message
	}
	return d
}

func (h *WebhookHandler) handoffMessage(bc ai_services.BotContext, reason, text string, disclose bool) botDecision {
	outcome := ai_services.BotOutcome{Action: whatsapp_models.BotActionHandoff, Reason: reason}
	return botDecision{
		Action: whatsapp_models.BotActionHandoff, Reason: reason, Flag: true,
		Message: ai_services.ComposeBotMessage(outcome, bc.Knowledge, text, disclose),
	}
}

// botCounts are the assistant's messages: in the chat today (clinic time)
// and in the last BotSilence (none: a new assistant conversation, whose
// first message discloses), and in the whole clinic today.
type botCounts struct {
	chatToday, chatLastDay, clinicToday int64
}

func (h *WebhookHandler) botMessagesSince(db *gorm.DB, convID uint, now time.Time) (botCounts, error) {
	var c botCounts
	local := now.In(whatsapp_services.ClinicLocation())
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).UTC()
	bot := func() *gorm.DB {
		return db.Model(&whatsapp_models.WhatsAppMessage{}).Where("kind = ?", whatsapp_models.KindBot)
	}
	if err := bot().Where("conversation_id = ? AND created_at >= ?", convID, dayStart).Count(&c.chatToday).Error; err != nil {
		return c, err
	}
	if err := bot().Where("conversation_id = ? AND created_at >= ?", convID, now.Add(-whatsapp_models.BotSilence).UTC()).Count(&c.chatLastDay).Error; err != nil {
		return c, err
	}
	err := bot().Where("created_at >= ?", dayStart).Count(&c.clinicToday).Error
	return c, err
}

// patientNames are the linked patient's names, only to redact them.
func (h *WebhookHandler) patientNames(db *gorm.DB, patientID *uint) []string {
	if patientID == nil {
		return nil
	}
	var p clinical_models.Patient
	if err := db.Select("id", "first_name", "last_name", "full_name").Where("id = ?", *patientID).Limit(1).Find(&p).Error; err != nil || p.ID == 0 {
		return nil
	}
	names := []string{p.FirstName, p.LastName}
	if p.FullName != nil {
		names = append(names, *p.FullName)
	}
	return names
}

// sendBot records body in conv's thread as a KindBot message and sends it.
func (h *WebhookHandler) sendBot(db *gorm.DB, account whatsapp_models.WhatsAppAccount, conv *whatsapp_models.WhatsAppConversation, body string) bool {
	token, err := account.OpenAccessToken()
	if err != nil {
		h.logger.Error("whatsapp bot: token unusable", zap.Uint("tenant_id", account.TenantID), zap.Error(err))
		return false
	}
	msg := whatsapp_models.WhatsAppMessage{
		TenantID: account.TenantID, Kind: whatsapp_models.KindBot, PatientID: conv.PatientID, ToPhone: conv.Phone,
		Status: whatsapp_models.MessageStatusSending, Body: body, ContentType: whatsapp_models.ContentText,
	}
	if err := whatsapp_services.RecordOutbound(db, conv, &msg, h.now()); err != nil {
		h.logger.Error("whatsapp bot: failed to record message", zap.Uint("conversation_id", conv.ID), zap.Error(err))
		return false
	}
	if err := h.sender.DeliverText(db, &msg, account, token); err != nil {
		h.logger.Warn("whatsapp bot: message not sent", zap.Uint("message_id", msg.ID), zap.Error(err))
		return false
	}
	return true
}

// flagForHuman marks the chat as needing a person and, with notify, tells
// reception.
func (h *WebhookHandler) flagForHuman(db *gorm.DB, tenantID uint, conv whatsapp_models.WhatsAppConversation, reason string, now time.Time, notify bool) {
	if err := db.Model(&whatsapp_models.WhatsAppConversation{}).Where("id = ?", conv.ID).
		Updates(map[string]any{"needs_human_at": now, "needs_human_reason": reason}).Error; err != nil {
		h.logger.Error("whatsapp bot: failed to flag conversation", zap.Uint("conversation_id", conv.ID), zap.Error(err))
	}
	if notify {
		whatsapp_services.NotifyBotHandoff(h.db, h.logger, tenantID, conv, reason == whatsapp_models.BotReasonEmergency)
	}
}

// recordBotEvent stores the action on the claimed event and audits it:
// action and reason codes only, never text.
func (h *WebhookHandler) recordBotEvent(db *gorm.DB, tenantID, eventID uint, conv whatsapp_models.WhatsAppConversation, d botDecision) {
	if err := db.Model(&whatsapp_models.WhatsAppBotEvent{}).Where("id = ?", eventID).
		Updates(map[string]any{"action": d.Action, "reason": d.Reason, "model_called": d.ModelCalled, "message_sent": d.Sent}).Error; err != nil {
		h.logger.Error("whatsapp bot: failed to record event", zap.Uint("event_id", eventID), zap.Error(err))
	}
	values, _ := json.Marshal(map[string]any{"action": d.Action, "reason": d.Reason, "model_called": d.ModelCalled})
	audit.RecordBotAction(h.db, tenantID, "BOT_"+strings.ToUpper(d.Action), auditEntityBot, conv.ID, conv.PatientID, values)
	h.logger.Info("whatsapp bot: message handled", zap.Uint("tenant_id", tenantID), zap.Uint("conversation_id", conv.ID),
		zap.String("action", d.Action), zap.String("reason", d.Reason), zap.Bool("model_called", d.ModelCalled))
}

// keyedLocks serializes work per key (a conversation) inside this process.
type keyedLocks struct {
	mu sync.Mutex
	m  map[uint]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func newKeyedLocks() *keyedLocks { return &keyedLocks{m: map[uint]*keyedLock{}} }

func (k *keyedLocks) lock(key uint) func() {
	k.mu.Lock()
	l := k.m[key]
	if l == nil {
		l = &keyedLock{}
		k.m[key] = l
	}
	l.refs++
	k.mu.Unlock()
	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		if l.refs--; l.refs == 0 {
			delete(k.m, key)
		}
		k.mu.Unlock()
	}
}
