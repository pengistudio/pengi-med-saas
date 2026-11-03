package whatsapp_handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	ai_models "pengi-med-saas/features/ai/models"
	ai_services "pengi-med-saas/features/ai/services"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// The WhatsApp assistant's settings (MANAGE_WHATSAPP): the switches, the
// FAQ, the readiness checklist and the "Probar" panel, which runs the same
// pipeline on a sample message without sending, logging or counting.

// Test panel rate: BotTestBurst calls, then one every BotTestEvery, per
// tenant.
const (
	BotTestBurst = 3
	BotTestEvery = 20 * time.Second
)

type botSettings struct {
	bot     *ai_services.Bot
	limiter *tenantLimiter
}

func newBotSettings(dial ai_services.Dialer) botSettings {
	return botSettings{bot: ai_services.NewBot(dial), limiter: newTenantLimiter(rate.Every(BotTestEvery), BotTestBurst)}
}

// WithBotDialer makes the test panel dial its provider with dial (tests).
func (h *WhatsAppHandler) WithBotDialer(dial ai_services.Dialer) *WhatsAppHandler {
	h.assistant.bot = ai_services.NewBot(dial)
	return h
}

// BotFAQLimits are the FAQ editor's limits.
type BotFAQLimits struct {
	MaxEntries  int `json:"max_entries"`
	MaxQuestion int `json:"max_question"`
	MaxAnswer   int `json:"max_answer"`
}

// BotSettingsResponse is the assistant's settings page.
type BotSettingsResponse struct {
	Enabled           bool                          `json:"enabled"`
	DuringOfficeHours bool                          `json:"during_office_hours"`
	FAQ               []whatsapp_models.BotFAQEntry `json:"faq"`
	Readiness         ai_services.BotReadiness      `json:"readiness"`
	Ready             bool                          `json:"ready"`
	// MissingKey: the i18n key of the first requirement not met.
	MissingKey string `json:"missing_key"`
	// The fixed material the assistant also answers from.
	OfficeHours   string            `json:"office_hours"`
	Address       string            `json:"address"`
	ClinicPhone   string            `json:"clinic_phone"`
	BookingLink   bool              `json:"booking_link"`
	InOfficeHours bool              `json:"in_office_hours"`
	Quota         ai_services.Quota `json:"quota"`
	Limits        BotFAQLimits      `json:"limits"`
}

// BotSettingsRequest saves the assistant's settings.
type BotSettingsRequest struct {
	Enabled           bool                          `json:"enabled"`
	DuringOfficeHours bool                          `json:"during_office_hours"`
	FAQ               []whatsapp_models.BotFAQEntry `json:"faq"`
}

// BotTestRequest is a sample patient message for the test panel.
type BotTestRequest struct {
	Message string `json:"message"`
}

// GetBotSettings answers GET /whatsapp/bot.
func (h *WhatsAppHandler) GetBotSettings(c *gin.Context) envelope.Response {
	tenantID := tenantdb.TenantID(c)
	bc, readiness, err := h.botState(tenantID, nil)
	if err != nil {
		h.logger.Error("failed to load whatsapp bot settings", zap.Error(err))
		return h.internalError()
	}
	return h.botResponse(tenantID, bc, readiness, "whatsapp.bot.fetch.success")
}

// UpdateBotSettings answers PUT /whatsapp/bot. The FAQ is trimmed (empty
// pairs dropped) and validated; enabling needs every requirement met
// (422 with the first missing one otherwise). Disabling is always allowed.
func (h *WhatsAppHandler) UpdateBotSettings(c *gin.Context) envelope.Response {
	var req BotSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	faq, ok := NormalizeBotFAQ(req.FAQ)
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.bot.error.invalid_faq", core_errors.ErrWhatsAppBotInvalidFAQ)
	}
	tenantID := tenantdb.TenantID(c)
	bc, readiness, err := h.botState(tenantID, faq)
	if err != nil {
		h.logger.Error("failed to load whatsapp bot settings", zap.Error(err))
		return h.internalError()
	}
	if req.Enabled && !readiness.Ready() {
		return envelope.ErrorResponse(http.StatusUnprocessableEntity, readiness.MissingKey(), core_errors.ErrWhatsAppBotNotReady)
	}
	s := bc.Settings
	if s.ID == 0 {
		s.TenantID = tenantID
	}
	if req.Enabled && !s.Enabled {
		now := time.Now()
		s.EnabledAt = &now
	}
	s.Enabled, s.DuringOfficeHours, s.FAQ = req.Enabled, req.DuringOfficeHours, faq
	db := tenantdb.For(c, h.db)
	if err := db.Save(&s).Error; err != nil {
		h.logger.Error("failed to save whatsapp bot settings", zap.Error(err))
		return h.internalError()
	}
	bc.Settings = s
	bc.Knowledge.FAQ = faq
	return h.botResponse(tenantID, bc, readiness, "whatsapp.bot.update.success")
}

// TestBot answers POST /whatsapp/bot/test: what the assistant would send
// for a sample message, as the first message of a conversation. Nothing is
// sent, logged or counted. Needs the plan's WhatsApp + AI and the clinic's
// AI opt-in; rate limited per tenant.
func (h *WhatsAppHandler) TestBot(c *gin.Context) envelope.Response {
	var req BotTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.bot.error.test_message", core_errors.ErrWhatsAppInvalidRequest)
	}
	text := strings.TrimSpace(req.Message)
	if text == "" || utf8.RuneCountInString(text) > ai_services.MaxBotMessageRunes {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.bot.error.test_message", core_errors.ErrWhatsAppInvalidRequest)
	}
	tenantID := tenantdb.TenantID(c)
	bc, readiness, err := h.botState(tenantID, nil)
	if err != nil {
		h.logger.Error("failed to load whatsapp bot settings", zap.Error(err))
		return h.internalError()
	}
	if !readiness.PlanWhatsApp || !readiness.PlanAI || !readiness.AIOptIn {
		r := ai_services.BotReadiness{PlanWhatsApp: readiness.PlanWhatsApp, PlanAI: readiness.PlanAI, AIOptIn: readiness.AIOptIn,
			ProviderConfigured: true, WhatsAppConnected: true, OfficeHours: true, Address: true, FAQ: true}
		return envelope.ErrorResponse(http.StatusUnprocessableEntity, r.MissingKey(), core_errors.ErrWhatsAppBotNotReady)
	}
	if !h.assistant.limiter.allow(tenantID) {
		return envelope.ErrorResponse(http.StatusTooManyRequests, "whatsapp.bot.error.rate_limited", core_errors.ErrWhatsAppBotRateLimited)
	}
	preview := h.assistant.bot.Preview(c.Request.Context(), h.db, tenantID, bc, text)
	return envelope.SuccessResponse(preview, "whatsapp.bot.test.success")
}

// botState loads the assistant and its readiness; faq, when not nil, is
// judged instead of the stored one.
func (h *WhatsAppHandler) botState(tenantID uint, faq []whatsapp_models.BotFAQEntry) (ai_services.BotContext, ai_services.BotReadiness, error) {
	bc, err := ai_services.LoadBotContext(h.db, tenantID, time.Now())
	if err != nil {
		return bc, ai_services.BotReadiness{}, err
	}
	if faq == nil {
		faq = bc.Settings.FAQ
	}
	readiness, err := ai_services.CheckBotReadiness(h.db, tenantID, bc, faq)
	return bc, readiness, err
}

func (h *WhatsAppHandler) botResponse(tenantID uint, bc ai_services.BotContext, readiness ai_services.BotReadiness, key string) envelope.Response {
	quota, err := ai_services.MonthlyQuota(h.db, tenantID, ai_models.FunctionBot, time.Now())
	if err != nil {
		h.logger.Error("failed to compute whatsapp bot quota", zap.Error(err))
		return h.internalError()
	}
	faq := bc.Settings.FAQ
	if faq == nil {
		faq = []whatsapp_models.BotFAQEntry{}
	}
	return envelope.SuccessResponse(BotSettingsResponse{
		Enabled: bc.Settings.Enabled, DuringOfficeHours: bc.Settings.DuringOfficeHours, FAQ: faq,
		Readiness: readiness, Ready: readiness.Ready(), MissingKey: readiness.MissingKey(),
		OfficeHours: bc.Knowledge.Hours, Address: bc.Knowledge.Address, ClinicPhone: bc.Knowledge.Phone,
		BookingLink:   bc.Knowledge.BookingLink != "",
		InOfficeHours: bc.Hours.Open(time.Now(), whatsapp_services.ClinicLocation()),
		Quota:         quota,
		Limits: BotFAQLimits{
			MaxEntries: whatsapp_models.MaxBotFAQEntries, MaxQuestion: whatsapp_models.MaxBotFAQQuestion, MaxAnswer: whatsapp_models.MaxBotFAQAnswer,
		},
	}, key)
}

// NormalizeBotFAQ trims every pair (spaces collapsed), drops the fully empty
// ones and checks the limits: at most MaxBotFAQEntries pairs, each with a
// question (≤ MaxBotFAQQuestion) and an answer (≤ MaxBotFAQAnswer). Never
// nil.
func NormalizeBotFAQ(in []whatsapp_models.BotFAQEntry) ([]whatsapp_models.BotFAQEntry, bool) {
	out := []whatsapp_models.BotFAQEntry{}
	for _, e := range in {
		q := strings.Join(strings.Fields(e.Question), " ")
		a := strings.TrimSpace(e.Answer)
		if q == "" && a == "" {
			continue
		}
		if q == "" || a == "" || utf8.RuneCountInString(q) > whatsapp_models.MaxBotFAQQuestion || utf8.RuneCountInString(a) > whatsapp_models.MaxBotFAQAnswer {
			return nil, false
		}
		out = append(out, whatsapp_models.BotFAQEntry{Question: q, Answer: a})
	}
	if len(out) > whatsapp_models.MaxBotFAQEntries {
		return nil, false
	}
	return out, true
}

// tenantLimiter is a token bucket per tenant (in this process).
type tenantLimiter struct {
	mu    sync.Mutex
	r     rate.Limit
	burst int
	m     map[uint]*rate.Limiter
}

func newTenantLimiter(r rate.Limit, burst int) *tenantLimiter {
	return &tenantLimiter{r: r, burst: burst, m: map[uint]*rate.Limiter{}}
}

func (t *tenantLimiter) allow(tenantID uint) bool {
	t.mu.Lock()
	l := t.m[tenantID]
	if l == nil {
		l = rate.NewLimiter(t.r, t.burst)
		t.m[tenantID] = l
	}
	t.mu.Unlock()
	return l.Allow()
}
