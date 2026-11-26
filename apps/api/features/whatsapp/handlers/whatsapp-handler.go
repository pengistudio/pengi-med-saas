package whatsapp_handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"pengi-med-saas/core/config"
	"strconv"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// defaultOffsets are the reminder offsets of a newly connected account.
var defaultOffsets = []int{24}

// WhatsAppHandler manages the tenant's WhatsApp connection, reminder settings
// and message log (MANAGE_WHATSAPP) and the conversations inbox
// (USE_WHATSAPP_INBOX, conversation-handler.go).
type WhatsAppHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	client *whatsapp.Client
	sender *whatsapp_services.Sender
	// assistant serves the WhatsApp assistant's settings and test panel
	// (bot-settings-handler.go).
	assistant botSettings
}

func NewWhatsAppHandler(db *gorm.DB, logger *zap.Logger, client *whatsapp.Client) *WhatsAppHandler {
	return &WhatsAppHandler{db: db, logger: logger, client: client, sender: whatsapp_services.NewSender(db, logger, client), assistant: newBotSettings(nil)}
}

// GetConfig returns the public values the frontend needs for Embedded Signup.
func (h *WhatsAppHandler) GetConfig(c *gin.Context) envelope.Response {
	cfg := config.Get()
	appID, configID := cfg.MetaAppID, cfg.MetaESConfigID
	return envelope.SuccessResponse(whatsapp_dto.ConfigResponse{
		AppID:                   appID,
		ConfigID:                configID,
		GraphVersion:            h.client.Version,
		EmbeddedSignupAvailable: appID != "" && configID != "" && h.client.AppSecret != "",
	}, "whatsapp.config.get.success")
}

// GetAccount returns the tenant's connection ({connected:false} when none).
func (h *WhatsAppHandler) GetAccount(c *gin.Context) envelope.Response {
	account, found, err := h.loadAccount(c)
	if err != nil {
		h.logger.Error("failed to load whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if !found {
		return envelope.SuccessResponse(whatsapp_dto.NotConnected(), "whatsapp.account.get.success")
	}
	return envelope.SuccessResponse(h.accountResponse(c, account), "whatsapp.account.get.success")
}

// ConnectEmbedded finishes Embedded Signup: exchanges the code for the
// business token, subscribes our app to the WABA, registers the number with
// a random pin and makes sure the reminder template exists. A Coexistence
// number is not registered; once saved, its app data sync is requested
// (app-data-sync.go).
//
// Every attempt (also the failed ones) leaves one "whatsapp connect" Info line
// with the ids, the coexistence flag and how long each Graph step took, to
// diagnose onboardings Meta flags. It never logs the code nor the token.
func (h *WhatsAppHandler) ConnectEmbedded(c *gin.Context) (resp envelope.Response) {
	var req whatsapp_dto.ConnectEmbeddedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	steps := newStepTimer()
	resolved := req.PhoneNumberID == ""
	defer func() {
		h.logger.Info("whatsapp connect", append([]zap.Field{
			zap.Uint("tenant_id", tenantdb.TenantID(c)),
			zap.String("mode", whatsapp_models.ModeEmbedded),
			zap.String("waba_id", req.WabaID),
			zap.String("phone_number_id", req.PhoneNumberID),
			zap.Bool("coexistence", req.Coexistence),
			zap.Bool("phone_id_resolved", resolved),
			zap.Int("status", resp.Code),
			zap.String("result", resp.Message),
		}, steps.fields()...)...)
	}()
	if h.client.AppID == "" || h.client.AppSecret == "" || config.Get().MetaESConfigID == "" {
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "whatsapp.error.embedded_unavailable", core_errors.ErrWhatsAppEmbeddedUnavailable)
	}
	box, err := whatsapp_models.Box()
	if err != nil {
		return h.encryptionUnavailable(err)
	}

	// Check ownership before spending the one-time Meta code.
	if req.PhoneNumberID != "" {
		if resp, taken := h.phoneTaken(c, req.PhoneNumberID); taken {
			return resp
		}
	}

	done := steps.start("exchange_code")
	token, err := h.client.ExchangeCode(req.Code)
	done()
	if err != nil {
		h.logger.Warn("whatsapp code exchange failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_credentials", core_errors.ErrWhatsAppInvalidCredentials)
	}
	// The Business app onboarding (Coexistence) may report only the WABA.
	if req.PhoneNumberID == "" {
		done := steps.start("list_phone_numbers")
		id, resp, ok := h.resolvePhoneNumberID(token, req.WabaID)
		done()
		if !ok {
			return resp
		}
		req.PhoneNumberID = id
		if resp, taken := h.phoneTaken(c, req.PhoneNumberID); taken {
			return resp
		}
	}
	done = steps.start("get_phone_number")
	phone, err := h.client.GetPhoneNumber(token, req.PhoneNumberID)
	done()
	if err != nil {
		return h.graphFailure("read phone number", err)
	}
	done = steps.start("subscribe_app")
	err = h.client.SubscribeApp(token, req.WabaID)
	done()
	if err != nil {
		return h.graphFailure("subscribe app", err)
	}
	account := whatsapp_models.WhatsAppAccount{WabaID: req.WabaID, PhoneNumberID: req.PhoneNumberID, Mode: whatsapp_models.ModeEmbedded, Coexistence: req.Coexistence}
	// A number linked from the WhatsApp Business app (Coexistence) is already
	// registered by Meta: registering it again would fail or break the link.
	if !req.Coexistence {
		pin, err := randomPin()
		if err != nil {
			return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
		}
		done := steps.start("register_phone")
		err = h.client.RegisterPhone(token, req.PhoneNumberID, pin)
		done()
		if err != nil {
			return h.graphFailure("register phone", err)
		}
		if err := account.SealPin(box, pin); err != nil {
			return h.encryptionUnavailable(err)
		}
	}
	resp = h.saveConnection(c, box, account, token, phone, steps)
	if req.Coexistence && resp.Code == http.StatusOK {
		// After the save: the history webhooks it triggers need the account.
		done := steps.start("app_data_sync")
		h.requestAppDataSync(tenantdb.TenantID(c), token, req.PhoneNumberID, appDataSyncWait)
		done()
	}
	return resp
}

// ConnectManual connects with a WABA id, phone number id and a (system user)
// token, used until Embedded Signup is approved by Meta.
func (h *WhatsAppHandler) ConnectManual(c *gin.Context) envelope.Response {
	var req whatsapp_dto.ConnectManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	req.Trim()
	if req.WabaID == "" || req.PhoneNumberID == "" || req.AccessToken == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	if resp, taken := h.phoneTaken(c, req.PhoneNumberID); taken {
		return resp
	}
	box, err := whatsapp_models.Box()
	if err != nil {
		return h.encryptionUnavailable(err)
	}

	phone, err := h.client.GetPhoneNumber(req.AccessToken, req.PhoneNumberID)
	if err != nil {
		var apiErr *whatsapp.APIError
		if errors.As(err, &apiErr) && !apiErr.Retryable() {
			h.logger.Warn("whatsapp manual credentials rejected", zap.Error(err))
			return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_credentials", core_errors.ErrWhatsAppInvalidCredentials)
		}
		return h.graphFailure("read phone number", err)
	}
	if err := h.client.SubscribeApp(req.AccessToken, req.WabaID); err != nil {
		return h.graphFailure("subscribe app", err)
	}

	account := whatsapp_models.WhatsAppAccount{WabaID: req.WabaID, PhoneNumberID: req.PhoneNumberID, Mode: whatsapp_models.ModeManual}
	return h.saveConnection(c, box, account, req.AccessToken, phone, nil)
}

// templateWait is how long a connect waits for the catalog templates before
// answering (a var so tests can shorten it). Each template costs Graph round
// trips and a slow WABA made the connect take 43 s, leaving the UI spinning.
var templateWait = 5 * time.Second

// templatesFunc reads (or reads and creates) the catalog templates of a WABA:
// whatsapp_services.ReadTemplateStates or CreateMissingTemplates.
type templatesFunc func(client *whatsapp.Client, token, wabaID string) ([]whatsapp_services.TemplateState, error)

// connectTemplates is what a connect does with the catalog templates. A
// Coexistence number only gets its statuses read: in production Meta blocked
// such numbers right after Pengi tried to create templates on them (Graph
// error 10). Creating them is then an explicit action
// (POST /whatsapp/template/create-missing).
func connectTemplates(conn whatsapp_models.WhatsAppAccount) templatesFunc {
	if conn.Coexistence {
		return whatsapp_services.ReadTemplateStates
	}
	return whatsapp_services.CreateMissingTemplates
}

// ensureTemplatesWithin runs run and waits up to wait for it.
// finished=false means it is still running: a goroutine then stores the
// template statuses once it ends, and the reminder's on the account once
// saved is closed (the caller closes it after upserting the account, so the
// update neither misses the row nor is overwritten by the upsert).
func (h *WhatsAppHandler) ensureTemplatesWithin(run templatesFunc, tenantID uint, token, wabaID string, wait time.Duration, saved <-chan struct{}) (states []whatsapp_services.TemplateState, finished bool, err error) {
	type result struct {
		states []whatsapp_services.TemplateState
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		s, e := run(h.client, token, wabaID)
		ch <- result{s, e}
	}()
	select {
	case r := <-ch:
		return r.states, true, r.err
	case <-time.After(wait):
		go func() {
			r := <-ch
			if r.err != nil {
				h.logger.Warn("whatsapp templates could not be ensured", zap.Uint("tenant_id", tenantID), zap.Error(r.err))
			}
			db := tenantdb.ForTenant(h.db, tenantID)
			if e := whatsapp_services.SaveTemplateStates(db, tenantID, r.states); e != nil {
				h.logger.Error("failed to save whatsapp template statuses", zap.Error(e))
				return
			}
			status := whatsapp_services.StateOf(r.states, whatsapp_models.ReminderTemplate)
			if status == whatsapp_models.TemplateStatusNone {
				return
			}
			<-saved
			if e := db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", tenantID).
				Updates(map[string]any{"template_status": status, "template_reason": ""}).Error; e != nil {
				h.logger.Error("failed to save whatsapp template status", zap.Error(e))
			}
		}()
		return nil, false, nil
	}
}

// saveConnection reads (Coexistence) or ensures the catalog templates, seals
// the token and upserts the tenant's account, keeping its reminder settings on
// a reconnect. steps (nil-safe) times the template step ("ensure_templates")
// for the caller's diagnostic log.
func (h *WhatsAppHandler) saveConnection(c *gin.Context, box interface{ Seal(string) (string, error) }, conn whatsapp_models.WhatsAppAccount, token string, phone whatsapp.PhoneNumber, steps *stepTimer) envelope.Response {
	tenantID := tenantdb.TenantID(c)
	// Closed on return (after the account upsert, or on an early error):
	// a background template run then updates the account's reminder status.
	saved := make(chan struct{})
	defer close(saved)
	done := steps.start("ensure_templates")
	states, finished, err := h.ensureTemplatesWithin(connectTemplates(conn), tenantID, token, conn.WabaID, templateWait, saved)
	done()
	if !finished {
		// Meta is slow: answer now; the states are stored when it finishes.
		h.logger.Warn("whatsapp templates still running, finishing in background", zap.Duration("waited", templateWait))
	} else if err != nil {
		// Not fatal: the number is connected; settings can sync or create
		// the missing templates later.
		h.logger.Warn("whatsapp templates could not be ensured", zap.Bool("coexistence", conn.Coexistence),
			zap.Bool("permission_denied", whatsapp_services.IsTemplatePermissionError(err)), zap.Error(err))
	}
	templateStatus := whatsapp_services.StateOf(states, whatsapp_models.ReminderTemplate)
	sealed, err := box.Seal(token)
	if err != nil {
		return h.encryptionUnavailable(err)
	}

	existing, found, err := h.loadAccount(c)
	if err != nil {
		h.logger.Error("failed to load whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	account := conn
	if found {
		// A new onboarding over a live connection (no disconnect in between).
		h.logger.Info("whatsapp connect over existing account",
			zap.Uint("tenant_id", existing.TenantID),
			zap.String("previous_waba_id", existing.WabaID),
			zap.String("previous_phone_number_id", existing.PhoneNumberID),
			zap.String("previous_mode", existing.Mode),
			zap.Bool("previous_coexistence", existing.Coexistence),
			zap.Duration("previous_connected_for", connectedFor(existing)),
			zap.String("waba_id", conn.WabaID),
			zap.String("phone_number_id", conn.PhoneNumberID),
			zap.String("mode", conn.Mode),
			zap.Bool("coexistence", conn.Coexistence),
		)
		account = existing
		account.WabaID, account.PhoneNumberID, account.Mode = conn.WabaID, conn.PhoneNumberID, conn.Mode
		account.Coexistence = conn.Coexistence
		account.PinEncrypted = conn.PinEncrypted
	} else {
		account.TenantID = tenantdb.TenantID(c)
		account.RemindersEnabled = true
		account.ReminderOffsets = defaultOffsets
	}
	now := time.Now()
	account.AccessTokenEncrypted = sealed
	account.DisplayPhone, account.VerifiedName = phone.DisplayPhoneNumber, phone.VerifiedName
	account.Status = whatsapp_models.AccountStatusConnected
	account.TemplateStatus, account.TemplateReason = templateStatus, ""
	account.ConnectedAt = &now
	account.ContactsSyncRequestID, account.HistorySyncRequestID = "", ""
	account.AppDataSyncError, account.AppDataSyncAt = "", nil

	if err := tenantdb.For(c, h.db).Save(&account).Error; err != nil {
		h.logger.Error("failed to save whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if err := whatsapp_services.SaveTemplateStates(tenantdb.For(c, h.db), account.TenantID, states); err != nil {
		h.logger.Error("failed to save whatsapp template statuses", zap.Error(err))
	}
	return envelope.SuccessResponse(h.accountResponse(c, account), "whatsapp.account.connect.success")
}

// accountResponse maps account, lists the catalog templates the tenant has
// no status for (absent on the WABA, or never read) and sums up their
// readiness (templates_state).
func (h *WhatsAppHandler) accountResponse(c *gin.Context, account whatsapp_models.WhatsAppAccount) whatsapp_dto.AccountResponse {
	resp := whatsapp_dto.NewAccountResponse(account)
	rows, err := whatsapp_services.TemplateRows(tenantdb.For(c, h.db))
	if err != nil {
		h.logger.Error("failed to load whatsapp templates", zap.Error(err))
		return resp
	}
	for _, spec := range whatsapp_services.Catalog {
		if whatsapp_services.IsOptionalTemplate(spec.Definition.Name) {
			continue
		}
		if rows[spec.Definition.Name].Status == whatsapp_models.TemplateStatusNone {
			resp.MissingTemplates = append(resp.MissingTemplates, spec.Definition.Name)
		}
	}
	state, list := whatsapp_services.TemplatesReadiness(rows)
	resp.TemplatesState = state
	for _, t := range list {
		resp.Templates = append(resp.Templates, whatsapp_dto.TemplateStateResponse{Name: t.Name, Status: t.Status, Reason: t.Reason, Optional: whatsapp_services.IsOptionalTemplate(t.Name)})
	}
	return resp
}

// SyncTemplate re-reads the status of every catalog template (one Graph
// call, never creates any) and stores them; the reminder's is also mirrored
// on the account. Creating the missing ones is CreateMissingTemplates.
func (h *WhatsAppHandler) SyncTemplate(c *gin.Context) envelope.Response {
	account, token, resp, ok := h.connected(c)
	if !ok {
		return resp
	}
	states, err := whatsapp_services.ReadTemplateStates(h.client, token, account.WabaID)
	if err != nil {
		return h.graphFailure("read templates", err)
	}
	if resp, ok := h.storeTemplateStates(c, &account, states); !ok {
		return resp
	}
	return envelope.SuccessResponse(h.accountResponse(c, account), "whatsapp.template.sync.success")
}

// CreateMissingTemplates creates the catalog templates missing on the WABA,
// one by one, stopping at the first Graph error (whatsapp_services
// .CreateMissingTemplates). Whatever was read or created is stored even when
// it fails. A permission refusal (Graph error 10) gets its own error so the
// user knows nothing else was attempted.
func (h *WhatsAppHandler) CreateMissingTemplates(c *gin.Context) envelope.Response {
	account, token, resp, ok := h.connected(c)
	if !ok {
		return resp
	}
	states, createErr := whatsapp_services.CreateMissingTemplates(h.client, token, account.WabaID)
	if len(states) > 0 {
		if resp, ok := h.storeTemplateStates(c, &account, states); !ok {
			return resp
		}
	}
	if createErr != nil {
		if whatsapp_services.IsTemplatePermissionError(createErr) {
			h.logger.Warn("whatsapp app lacks permission to create templates", zap.Uint("tenant_id", account.TenantID), zap.Error(createErr))
			return envelope.ErrorResponse(http.StatusUnprocessableEntity, "whatsapp.error.template_permission", core_errors.ErrWhatsAppTemplatePermission)
		}
		if errors.Is(createErr, whatsapp_services.ErrTemplateListIncomplete) {
			h.logger.Warn("whatsapp template listing incomplete, nothing created", zap.Uint("tenant_id", account.TenantID))
			return envelope.ErrorResponse(http.StatusUnprocessableEntity, "whatsapp.error.template_list_incomplete", core_errors.ErrWhatsAppTemplateListIncomplete)
		}
		return h.graphFailure("create missing templates", createErr)
	}
	return envelope.SuccessResponse(h.accountResponse(c, account), "whatsapp.template.create_missing.success")
}

// storeTemplateStates saves states and mirrors the reminder's status on
// account (kept as it was when the reminder is not among them).
func (h *WhatsAppHandler) storeTemplateStates(c *gin.Context, account *whatsapp_models.WhatsAppAccount, states []whatsapp_services.TemplateState) (envelope.Response, bool) {
	db := tenantdb.For(c, h.db)
	if err := whatsapp_services.SaveTemplateStates(db, account.TenantID, states); err != nil {
		h.logger.Error("failed to save whatsapp template statuses", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal), false
	}
	status := whatsapp_services.StateOf(states, whatsapp_models.ReminderTemplate)
	if status == whatsapp_models.TemplateStatusNone {
		status = account.TemplateStatus // not read this time: keep what we knew
	}
	if err := db.Model(account).Updates(map[string]any{"template_status": status, "template_reason": ""}).Error; err != nil {
		h.logger.Error("failed to save whatsapp template status", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal), false
	}
	account.TemplateStatus, account.TemplateReason = status, ""
	return envelope.Response{}, true
}

// UpdateSettings switches reminders on/off, sets their offsets and switches
// the no-show message on/off.
func (h *WhatsAppHandler) UpdateSettings(c *gin.Context) envelope.Response {
	var req whatsapp_dto.SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	offsets, err := whatsapp_services.NormalizeOffsets(req.ReminderOffsets)
	if err != nil || (*req.RemindersEnabled && len(offsets) == 0) {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_offsets", core_errors.ErrWhatsAppInvalidOffsets)
	}
	account, found, err := h.loadAccount(c)
	if err != nil {
		h.logger.Error("failed to load whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if !found {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected)
	}
	account.RemindersEnabled, account.ReminderOffsets = *req.RemindersEnabled, offsets
	if req.NoShowMessageEnabled != nil {
		account.NoShowMessageEnabled = *req.NoShowMessageEnabled
	}
	if err := tenantdb.For(c, h.db).Model(&account).Select("reminders_enabled", "reminder_offsets", "no_show_message_enabled").Updates(&account).Error; err != nil {
		h.logger.Error("failed to save whatsapp settings", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	return envelope.SuccessResponse(h.accountResponse(c, account), "whatsapp.settings.update.success")
}

// Disconnect deletes the tenant's account (hard delete, so the number can be
// connected again). The message log stays. It leaves a "whatsapp disconnect"
// Info line (ids, mode, how long it was connected) so a quick reconnect of the
// same number shows up in the logs; it doesn't block anything.
func (h *WhatsAppHandler) Disconnect(c *gin.Context) envelope.Response {
	account, found, err := h.loadAccount(c)
	if err != nil {
		h.logger.Error("failed to load whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if !found {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected)
	}
	res := tenantdb.For(c, h.db).Unscoped().Where("tenant_id = ?", tenantdb.TenantID(c)).Delete(&whatsapp_models.WhatsAppAccount{})
	if res.Error != nil {
		h.logger.Error("failed to delete whatsapp account", zap.Error(res.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if res.RowsAffected == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected)
	}
	h.logger.Info("whatsapp disconnect",
		zap.Uint("tenant_id", account.TenantID),
		zap.String("waba_id", account.WabaID),
		zap.String("phone_number_id", account.PhoneNumberID),
		zap.String("mode", account.Mode),
		zap.Bool("coexistence", account.Coexistence),
		zap.Duration("connected_for", connectedFor(account)),
	)
	// Template statuses belong to the WABA, which may differ on reconnect.
	if err := tenantdb.For(c, h.db).Unscoped().Where("tenant_id = ?", tenantdb.TenantID(c)).Delete(&whatsapp_models.WhatsAppTemplate{}).Error; err != nil {
		h.logger.Error("failed to delete whatsapp template statuses", zap.Error(err))
	}
	return envelope.SuccessResponse(whatsapp_dto.NotConnected(), "whatsapp.account.disconnect.success")
}

// SendTest sends the reminder template with sample data to phone, right away.
func (h *WhatsAppHandler) SendTest(c *gin.Context) envelope.Response {
	var req whatsapp_dto.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	to, err := whatsapp_services.NormalizePhone(req.Phone)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_phone", core_errors.ErrWhatsAppInvalidPhone)
	}
	account, token, resp, ok := h.connected(c)
	if !ok {
		return resp
	}
	// The reminder's own row, not the account's mirror of it.
	if !whatsapp_services.TemplateApproved(h.db, tenantdb.TenantID(c), whatsapp_models.ReminderTemplate) {
		return templateNotApprovedResponse()
	}
	allowed, _, err := whatsapp_services.CanSendTemplate(h.db, tenantdb.TenantID(c), time.Now())
	if err != nil {
		h.logger.Error("failed to check whatsapp monthly limit", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if !allowed {
		return monthlyLimitResponse()
	}

	db := tenantdb.For(c, h.db)
	msg := whatsapp_models.WhatsAppMessage{
		TenantID: tenantdb.TenantID(c),
		Kind:     whatsapp_models.KindTest,
		ToPhone:  to,
		Template: whatsapp_models.ReminderTemplate,
		Status:   whatsapp_models.MessageStatusSending,
	}
	if err := db.Create(&msg).Error; err != nil {
		h.logger.Error("failed to create whatsapp test message", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	loc := whatsapp_services.ClinicLocation()
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1)
	data := whatsapp_services.ReminderData{
		PatientName: "Paciente de prueba",
		ClinicName:  whatsapp_services.ClinicName(h.db, msg.TenantID),
		Start:       time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, loc),
	}
	if err := h.sender.Deliver(db, &msg, account, token, data); err != nil {
		if errors.Is(err, whatsapp_services.ErrTemplateNotApproved) {
			return templateNotApprovedResponse()
		}
		return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.send_failed", core_errors.ErrWhatsAppSendFailed)
	}
	return envelope.SuccessResponse(whatsapp_dto.NewMessageResponse(msg), "whatsapp.test.send.success")
}

func templateNotApprovedResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.template_not_approved", core_errors.ErrWhatsAppTemplateNotApproved)
}

// ListMessages is the paginated log of sent messages (reminders, tests and
// inbox replies), newest first. Query: page, limit (max 100), status, kind,
// appointment_id.
func (h *WhatsAppHandler) ListMessages(c *gin.Context) envelope.Response {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	// Sent messages only: what patients wrote lives in the conversations.
	query := tenantdb.For(c, h.db).Model(&whatsapp_models.WhatsAppMessage{}).Where("direction = ?", whatsapp_models.DirectionOutbound)
	if kind := c.Query("kind"); kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if apptID, err := strconv.Atoi(c.Query("appointment_id")); err == nil && apptID > 0 {
		query = query.Where("appointment_id = ?", apptID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("failed to count whatsapp messages", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	var messages []whatsapp_models.WhatsAppMessage
	if err := query.Preload("Patient").Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&messages).Error; err != nil {
		h.logger.Error("failed to list whatsapp messages", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	items := make([]whatsapp_dto.MessageResponse, 0, len(messages))
	for _, m := range messages {
		items = append(items, whatsapp_dto.NewMessageResponse(m))
	}
	return envelope.PagedSuccessResponse(items, int(total), page, limit, "whatsapp.messages.list.success")
}

func (h *WhatsAppHandler) loadAccount(c *gin.Context) (whatsapp_models.WhatsAppAccount, bool, error) {
	var account whatsapp_models.WhatsAppAccount
	err := tenantdb.For(c, h.db).Limit(1).Find(&account).Error
	return account, err == nil && account.ID != 0, err
}

// connected loads the account and opens its token, or answers why it can't.
func (h *WhatsAppHandler) connected(c *gin.Context) (whatsapp_models.WhatsAppAccount, string, envelope.Response, bool) {
	account, found, err := h.loadAccount(c)
	if err != nil {
		h.logger.Error("failed to load whatsapp account", zap.Error(err))
		return account, "", envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal), false
	}
	if !found {
		return account, "", envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected), false
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return account, "", h.encryptionUnavailable(err), false
	}
	return account, token, envelope.Response{}, true
}

// phoneTaken answers 409 when another tenant already connected the number.
func (h *WhatsAppHandler) phoneTaken(c *gin.Context, phoneNumberID string) (envelope.Response, bool) {
	var count int64
	err := tenantdb.System(h.db).Model(&whatsapp_models.WhatsAppAccount{}).
		Where("phone_number_id = ? AND tenant_id <> ?", phoneNumberID, tenantdb.TenantID(c)).Count(&count).Error
	if err != nil {
		h.logger.Error("failed to check whatsapp phone", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal), true
	}
	if count > 0 {
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.phone_in_use", core_errors.ErrWhatsAppPhoneInUse), true
	}
	return envelope.Response{}, false
}

// resolvePhoneNumberID finds the number of a WABA when Embedded Signup didn't
// report it: it must have exactly one, otherwise we can't tell which to use.
func (h *WhatsAppHandler) resolvePhoneNumberID(token, wabaID string) (string, envelope.Response, bool) {
	phones, err := h.client.ListPhoneNumbers(token, wabaID)
	if err != nil {
		return "", h.graphFailure("list phone numbers", err), false
	}
	switch len(phones) {
	case 1:
		return phones[0].ID, envelope.Response{}, true
	case 0:
		return "", envelope.ErrorResponse(http.StatusUnprocessableEntity, "whatsapp.error.no_phone_number", core_errors.ErrWhatsAppPhoneUnresolved), false
	default:
		h.logger.Warn("whatsapp waba has several phone numbers", zap.String("waba_id", wabaID), zap.Int("count", len(phones)))
		return "", envelope.ErrorResponse(http.StatusUnprocessableEntity, "whatsapp.error.several_phone_numbers", core_errors.ErrWhatsAppPhoneUnresolved), false
	}
}

func (h *WhatsAppHandler) graphFailure(step string, err error) envelope.Response {
	h.logger.Warn("whatsapp graph call failed", zap.String("step", step), zap.Error(err))
	var apiErr *whatsapp.APIError
	if errors.As(err, &apiErr) && apiErr.IsAuthError() {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_credentials", core_errors.ErrWhatsAppInvalidCredentials)
	}
	return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.graph", core_errors.ErrWhatsAppGraphError)
}

func (h *WhatsAppHandler) encryptionUnavailable(err error) envelope.Response {
	h.logger.Error("whatsapp encryption unavailable", zap.Error(err))
	return envelope.ErrorResponse(http.StatusServiceUnavailable, "whatsapp.error.encryption_unavailable", core_errors.ErrWhatsAppEncryptionUnavailable)
}

// connectedFor is how long the account has been connected (0 when unknown).
func connectedFor(account whatsapp_models.WhatsAppAccount) time.Duration {
	if account.ConnectedAt == nil {
		return 0
	}
	return time.Since(*account.ConnectedAt)
}

// stepTimer collects how long each Graph step of a connection took, for the
// diagnostic log. A nil *stepTimer records nothing.
type stepTimer struct {
	began time.Time
	steps []zap.Field
}

func newStepTimer() *stepTimer { return &stepTimer{began: time.Now()} }

// start times the step called name until the returned func is called.
func (t *stepTimer) start(name string) func() {
	if t == nil {
		return func() {}
	}
	from := time.Now()
	return func() { t.steps = append(t.steps, zap.Duration(name, time.Since(from))) }
}

// fields are the recorded steps plus the total so far.
func (t *stepTimer) fields() []zap.Field {
	return append(append([]zap.Field{}, t.steps...), zap.Duration("total", time.Since(t.began)))
}

// randomPin is a 6-digit two-step verification pin.
func randomPin() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
