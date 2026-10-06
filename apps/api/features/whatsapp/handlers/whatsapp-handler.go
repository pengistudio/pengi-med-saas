package whatsapp_handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
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
}

func NewWhatsAppHandler(db *gorm.DB, logger *zap.Logger, client *whatsapp.Client) *WhatsAppHandler {
	return &WhatsAppHandler{db: db, logger: logger, client: client, sender: whatsapp_services.NewSender(db, logger, client)}
}

// GetConfig returns the public values the frontend needs for Embedded Signup.
func (h *WhatsAppHandler) GetConfig(c *gin.Context) envelope.Response {
	appID, configID := os.Getenv("META_APP_ID"), os.Getenv("META_ES_CONFIG_ID")
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
	return envelope.SuccessResponse(whatsapp_dto.NewAccountResponse(account), "whatsapp.account.get.success")
}

// ConnectEmbedded finishes Embedded Signup: exchanges the code for the
// business token, subscribes our app to the WABA, registers the number with
// a random pin and makes sure the reminder template exists.
func (h *WhatsAppHandler) ConnectEmbedded(c *gin.Context) envelope.Response {
	var req whatsapp_dto.ConnectEmbeddedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	if h.client.AppID == "" || h.client.AppSecret == "" || os.Getenv("META_ES_CONFIG_ID") == "" {
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "whatsapp.error.embedded_unavailable", core_errors.ErrWhatsAppEmbeddedUnavailable)
	}
	if resp, taken := h.phoneTaken(c, req.PhoneNumberID); taken {
		return resp
	}
	box, err := whatsapp_models.Box()
	if err != nil {
		return h.encryptionUnavailable(err)
	}

	token, err := h.client.ExchangeCode(req.Code)
	if err != nil {
		h.logger.Warn("whatsapp code exchange failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_credentials", core_errors.ErrWhatsAppInvalidCredentials)
	}
	phone, err := h.client.GetPhoneNumber(token, req.PhoneNumberID)
	if err != nil {
		return h.graphFailure("read phone number", err)
	}
	if err := h.client.SubscribeApp(token, req.WabaID); err != nil {
		return h.graphFailure("subscribe app", err)
	}
	pin, err := randomPin()
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if err := h.client.RegisterPhone(token, req.PhoneNumberID, pin); err != nil {
		return h.graphFailure("register phone", err)
	}

	account := whatsapp_models.WhatsAppAccount{WabaID: req.WabaID, PhoneNumberID: req.PhoneNumberID, Mode: whatsapp_models.ModeEmbedded}
	if err := account.SealPin(box, pin); err != nil {
		return h.encryptionUnavailable(err)
	}
	return h.saveConnection(c, box, account, token, phone)
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
	return h.saveConnection(c, box, account, req.AccessToken, phone)
}

// saveConnection ensures the catalog templates, seals the token and upserts
// the tenant's account, keeping its reminder settings on a reconnect.
func (h *WhatsAppHandler) saveConnection(c *gin.Context, box interface{ Seal(string) (string, error) }, conn whatsapp_models.WhatsAppAccount, token string, phone whatsapp.PhoneNumber) envelope.Response {
	states, err := whatsapp_services.EnsureTemplates(h.client, token, conn.WabaID)
	if err != nil {
		// Not fatal: the number is connected; POST /whatsapp/template/sync retries.
		h.logger.Warn("whatsapp templates could not be ensured", zap.Error(err))
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
		account = existing
		account.WabaID, account.PhoneNumberID, account.Mode = conn.WabaID, conn.PhoneNumberID, conn.Mode
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

	if err := tenantdb.For(c, h.db).Save(&account).Error; err != nil {
		h.logger.Error("failed to save whatsapp account", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if err := whatsapp_services.SaveTemplateStates(tenantdb.For(c, h.db), account.TenantID, states); err != nil {
		h.logger.Error("failed to save whatsapp template statuses", zap.Error(err))
	}
	return envelope.SuccessResponse(whatsapp_dto.NewAccountResponse(account), "whatsapp.account.connect.success")
}

// SyncTemplate re-reads (or creates) every catalog template and stores their
// statuses; the reminder's is also mirrored on the account. A Graph error is
// only returned when no template could be read.
func (h *WhatsAppHandler) SyncTemplate(c *gin.Context) envelope.Response {
	account, token, resp, ok := h.connected(c)
	if !ok {
		return resp
	}
	states, err := whatsapp_services.EnsureTemplates(h.client, token, account.WabaID)
	if err != nil {
		if len(states) == 0 {
			return h.graphFailure("ensure templates", err)
		}
		h.logger.Warn("some whatsapp templates could not be ensured", zap.Error(err))
	}
	db := tenantdb.For(c, h.db)
	if err := whatsapp_services.SaveTemplateStates(db, account.TenantID, states); err != nil {
		h.logger.Error("failed to save whatsapp template statuses", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	status := whatsapp_services.StateOf(states, whatsapp_models.ReminderTemplate)
	if status == whatsapp_models.TemplateStatusNone {
		status = account.TemplateStatus // the reminder failed this time: keep what we knew
	}
	if err := db.Model(&account).Updates(map[string]any{"template_status": status, "template_reason": ""}).Error; err != nil {
		h.logger.Error("failed to save whatsapp template status", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	account.TemplateStatus, account.TemplateReason = status, ""
	return envelope.SuccessResponse(whatsapp_dto.NewAccountResponse(account), "whatsapp.template.sync.success")
}

// UpdateSettings switches reminders on/off and sets their offsets.
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
	if err := tenantdb.For(c, h.db).Model(&account).Select("reminders_enabled", "reminder_offsets").Updates(&account).Error; err != nil {
		h.logger.Error("failed to save whatsapp settings", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	return envelope.SuccessResponse(whatsapp_dto.NewAccountResponse(account), "whatsapp.settings.update.success")
}

// Disconnect deletes the tenant's account (hard delete, so the number can be
// connected again). The message log stays.
func (h *WhatsAppHandler) Disconnect(c *gin.Context) envelope.Response {
	res := tenantdb.For(c, h.db).Unscoped().Where("tenant_id = ?", tenantdb.TenantID(c)).Delete(&whatsapp_models.WhatsAppAccount{})
	if res.Error != nil {
		h.logger.Error("failed to delete whatsapp account", zap.Error(res.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
	}
	if res.RowsAffected == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected)
	}
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
	if account.TemplateStatus != whatsapp_models.TemplateStatusApproved {
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.template_not_approved", core_errors.ErrWhatsAppTemplateNotApproved)
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
		return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.send_failed", core_errors.ErrWhatsAppSendFailed)
	}
	return envelope.SuccessResponse(whatsapp_dto.NewMessageResponse(msg), "whatsapp.test.send.success")
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

// randomPin is a 6-digit two-step verification pin.
func randomPin() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
