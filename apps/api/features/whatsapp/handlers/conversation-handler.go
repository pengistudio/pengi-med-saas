package whatsapp_handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The inbox (USE_WHATSAPP_INBOX): conversations with patients and free-text
// replies inside Meta's 24 h window. Every query goes through tenantdb.For, so
// another clinic's conversation id answers 404.

const (
	defaultThreadLimit = 50
	maxThreadLimit     = 100
)

// ListConversations lists the tenant's conversations, most recent first.
// Query: search (patient name or phone), unread=true, page, limit (max 100).
func (h *WhatsAppHandler) ListConversations(c *gin.Context) envelope.Response {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	const table = "whatsapp_conversations"
	query := tenantdb.For(c, h.db).Model(&whatsapp_models.WhatsAppConversation{})
	if c.Query("unread") == "true" {
		query = query.Where(table + ".unread_count > 0")
	}
	if search := strings.ToLower(strings.TrimSpace(c.Query("search"))); search != "" {
		like := "%" + search + "%"
		cond := h.db.Where("LOWER(patients.first_name) LIKE ?", like).
			Or("LOWER(patients.last_name) LIKE ?", like).
			Or("LOWER(patients.first_name || ' ' || patients.last_name) LIKE ?", like)
		if digits := phoneDigits(search); digits != "" {
			cond = cond.Or(table+".phone LIKE ?", "%"+digits+"%")
		}
		query = query.Joins("LEFT JOIN patients ON patients.id = " + table + ".patient_id AND patients.deleted_at IS NULL").Where(cond)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("failed to count whatsapp conversations", zap.Error(err))
		return h.internalError()
	}
	var convs []whatsapp_models.WhatsAppConversation
	if err := query.Preload("Patient").
		Order("COALESCE(" + table + ".last_message_at, " + table + ".created_at) DESC").Order(table + ".id DESC").
		Limit(limit).Offset((page - 1) * limit).Find(&convs).Error; err != nil {
		h.logger.Error("failed to list whatsapp conversations", zap.Error(err))
		return h.internalError()
	}
	now := time.Now()
	items := make([]whatsapp_dto.ConversationResponse, 0, len(convs))
	for _, conv := range convs {
		items = append(items, whatsapp_dto.NewConversationResponse(conv, now))
	}
	return envelope.PagedSuccessResponse(items, int(total), page, limit, "whatsapp.conversations.list.success")
}

// UnreadCount is the inbox badge: unread messages across conversations.
func (h *WhatsAppHandler) UnreadCount(c *gin.Context) envelope.Response {
	var out whatsapp_dto.UnreadCountResponse
	err := tenantdb.For(c, h.db).Model(&whatsapp_models.WhatsAppConversation{}).
		Where("unread_count > 0").
		Select("COALESCE(SUM(unread_count), 0) AS unread_count, COUNT(*) AS conversations").
		Scan(&out).Error
	if err != nil {
		h.logger.Error("failed to count unread whatsapp messages", zap.Error(err))
		return h.internalError()
	}
	return envelope.SuccessResponse(out, "whatsapp.conversations.unread.success")
}

// GetConversation is the thread header: conversation, patient and the
// patient's next appointment.
func (h *WhatsAppHandler) GetConversation(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	return h.detail(c, conv, "whatsapp.conversation.get.success")
}

// ListConversationMessages returns a page of the thread, oldest first.
// Query: before (message id, for older pages), limit (default 50, max 100).
func (h *WhatsAppHandler) ListConversationMessages(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultThreadLimit)))
	if limit < 1 || limit > maxThreadLimit {
		limit = defaultThreadLimit
	}
	query := tenantdb.For(c, h.db).Where("conversation_id = ?", conv.ID)
	if before, err := strconv.ParseUint(c.Query("before"), 10, 64); err == nil && before > 0 {
		query = query.Where("id < ?", before)
	}
	var messages []whatsapp_models.WhatsAppMessage
	if err := query.Order("id DESC").Limit(limit + 1).Find(&messages).Error; err != nil {
		h.logger.Error("failed to list whatsapp conversation messages", zap.Error(err))
		return h.internalError()
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	items := make([]whatsapp_dto.MessageResponse, len(messages))
	for i, m := range messages {
		items[len(messages)-1-i] = whatsapp_dto.NewMessageResponse(m)
	}
	return envelope.SuccessResponse(whatsapp_dto.ThreadResponse{Items: items, HasMore: hasMore}, "whatsapp.conversation.messages.success")
}

// SendReply sends free text to the conversation's number, synchronously. The
// patient must have written within the last 24 h (409 otherwise); WhatsApp
// consent does not apply, the patient started the conversation. When Meta
// rejects it the message stays in the thread as failed and the answer is 502.
func (h *WhatsAppHandler) SendReply(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	var req whatsapp_dto.ReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	if utf8.RuneCountInString(body) > whatsapp_dto.MaxReplyLength {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.message_too_long", core_errors.ErrWhatsAppMessageTooLong)
	}
	now := time.Now()
	if !whatsapp_services.WindowOpen(conv, now) {
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.window_closed", core_errors.ErrWhatsAppWindowClosed)
	}
	account, token, resp, ok := h.connected(c)
	if !ok {
		return resp
	}

	db := tenantdb.For(c, h.db)
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    tenantdb.TenantID(c),
		Kind:        whatsapp_models.KindReply,
		PatientID:   conv.PatientID,
		ToPhone:     conv.Phone,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        body,
		ContentType: whatsapp_models.ContentText,
	}
	if uid := c.GetInt64("user_id"); uid > 0 {
		id := uint(uid)
		msg.SentByUserID = &id
	}
	if err := whatsapp_services.RecordOutbound(db, &conv, &msg, now); err != nil {
		h.logger.Error("failed to record whatsapp reply", zap.Error(err))
		return h.internalError()
	}
	if err := h.sender.DeliverText(db, &msg, account, token); err != nil {
		return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.send_failed", core_errors.ErrWhatsAppSendFailed)
	}
	return envelope.SuccessResponse(whatsapp_dto.NewMessageResponse(msg), "whatsapp.conversation.reply.success")
}

// MarkConversationRead clears the conversation's unread count and marks the
// caller's notifications about it as read.
func (h *WhatsAppHandler) MarkConversationRead(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	if err := tenantdb.For(c, h.db).Model(&conv).Update("unread_count", 0).Error; err != nil {
		h.logger.Error("failed to mark whatsapp conversation read", zap.Error(err))
		return h.internalError()
	}
	conv.UnreadCount = 0
	// The caller's new-message notifications of this conversation, too.
	if uid := c.GetInt64("user_id"); uid > 0 {
		if err := whatsapp_services.MarkConversationNotificationsRead(tenantdb.For(c, h.db), uint(uid), conv.ID); err != nil {
			h.logger.Error("failed to mark whatsapp notifications read", zap.Error(err))
		}
	}
	return envelope.SuccessResponse(whatsapp_dto.NewConversationResponse(conv, time.Now()), "whatsapp.conversation.read.success")
}

// LinkPatient links the conversation (typically an unknown number) to a
// patient of the tenant, and its unlinked messages with it.
func (h *WhatsAppHandler) LinkPatient(c *gin.Context) envelope.Response {
	conv, resp, ok := h.loadConversation(c)
	if !ok {
		return resp
	}
	var req whatsapp_dto.LinkPatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "whatsapp.error.invalid_request", core_errors.ErrWhatsAppInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var patient clinical_models.Patient
	if err := db.Where("id = ?", req.PatientID).Limit(1).Find(&patient).Error; err != nil {
		h.logger.Error("failed to load patient", zap.Error(err))
		return h.internalError()
	}
	if patient.ID == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.patient_not_found", core_errors.ErrWhatsAppPatientNotFound)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&conv).Update("patient_id", patient.ID).Error; err != nil {
			return err
		}
		return tx.Model(&whatsapp_models.WhatsAppMessage{}).
			Where("conversation_id = ? AND patient_id IS NULL", conv.ID).
			Update("patient_id", patient.ID).Error
	})
	if err != nil {
		h.logger.Error("failed to link whatsapp conversation to patient", zap.Error(err))
		return h.internalError()
	}
	conv.PatientID, conv.Patient = &patient.ID, &patient
	return h.detail(c, conv, "whatsapp.conversation.patient.success")
}

// loadConversation loads :id in the request's tenant (404 when it is not
// there, including another tenant's id) with its patient.
func (h *WhatsAppHandler) loadConversation(c *gin.Context) (whatsapp_models.WhatsAppConversation, envelope.Response, bool) {
	var conv whatsapp_models.WhatsAppConversation
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return conv, envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.conversation_not_found", core_errors.ErrWhatsAppConversationNotFound), false
	}
	if err := tenantdb.For(c, h.db).Preload("Patient").Where("id = ?", id).Limit(1).Find(&conv).Error; err != nil {
		h.logger.Error("failed to load whatsapp conversation", zap.Error(err))
		return conv, h.internalError(), false
	}
	if conv.ID == 0 {
		return conv, envelope.ErrorResponse(http.StatusNotFound, "whatsapp.error.conversation_not_found", core_errors.ErrWhatsAppConversationNotFound), false
	}
	return conv, envelope.Response{}, true
}

// detail answers the header of conv with the patient's next appointment.
func (h *WhatsAppHandler) detail(c *gin.Context, conv whatsapp_models.WhatsAppConversation, key string) envelope.Response {
	now := time.Now()
	var next *clinical_models.Appointment
	if conv.PatientID != nil {
		appt, err := h.nextAppointment(tenantdb.For(c, h.db), *conv.PatientID, now)
		if err != nil {
			h.logger.Error("failed to load next appointment", zap.Error(err))
			return h.internalError()
		}
		next = appt
	}
	return envelope.SuccessResponse(whatsapp_dto.NewConversationDetailResponse(conv, next, now), key)
}

// nextAppointment is the patient's first scheduled or confirmed appointment
// that has not started yet, or nil.
func (h *WhatsAppHandler) nextAppointment(db *gorm.DB, patientID uint, now time.Time) (*clinical_models.Appointment, error) {
	loc := whatsapp_services.ClinicLocation()
	local := now.In(loc)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	var appts []clinical_models.Appointment
	err := db.Where("patient_id = ? AND status IN ? AND date >= ?", patientID,
		[]string{whatsapp_services.AppointmentScheduled, whatsapp_services.AppointmentConfirmed}, dayStart.Add(-24*time.Hour)).
		Order("date ASC, start_time ASC").Limit(20).Find(&appts).Error
	if err != nil {
		return nil, err
	}
	for i := range appts {
		if start, err := whatsapp_services.AppointmentStart(appts[i], loc); err == nil && start.After(now) {
			return &appts[i], nil
		}
	}
	return nil, nil
}

func (h *WhatsAppHandler) internalError() envelope.Response {
	return envelope.ErrorResponse(http.StatusInternalServerError, "whatsapp.error.internal", core_errors.ErrWhatsAppInternal)
}

// phoneDigits returns the digits of a phone search ("099 123" → "99123"): a
// leading national 0 is dropped, since stored phones are international.
func phoneDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := strings.TrimLeft(b.String(), "0")
	if len(d) < 3 {
		return ""
	}
	return d
}
