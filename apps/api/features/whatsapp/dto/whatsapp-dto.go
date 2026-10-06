package whatsapp_dto

import (
	"strings"
	"time"

	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// ConfigResponse is what the frontend needs to start Meta's Embedded Signup.
type ConfigResponse struct {
	AppID                   string `json:"app_id"`
	ConfigID                string `json:"config_id"`
	GraphVersion            string `json:"graph_version"`
	EmbeddedSignupAvailable bool   `json:"embedded_signup_available"`
}

// AccountResponse describes the tenant's connection. The token is never sent.
type AccountResponse struct {
	Connected        bool       `json:"connected"`
	Mode             string     `json:"mode,omitempty"`
	Status           string     `json:"status,omitempty"`
	WabaID           string     `json:"waba_id,omitempty"`
	PhoneNumberID    string     `json:"phone_number_id,omitempty"`
	DisplayPhone     string     `json:"display_phone,omitempty"`
	VerifiedName     string     `json:"verified_name,omitempty"`
	TemplateName     string     `json:"template_name,omitempty"`
	TemplateStatus   string     `json:"template_status"`
	TemplateReason   string     `json:"template_reason,omitempty"`
	RemindersEnabled bool       `json:"reminders_enabled"`
	ReminderOffsets  []int      `json:"reminder_offsets"`
	ConnectedAt      *time.Time `json:"connected_at,omitempty"`
}

// NotConnected is the AccountResponse of a tenant without an account.
func NotConnected() AccountResponse {
	return AccountResponse{Connected: false, ReminderOffsets: []int{}}
}

// NewAccountResponse maps an account.
func NewAccountResponse(a whatsapp_models.WhatsAppAccount) AccountResponse {
	offsets := a.ReminderOffsets
	if offsets == nil {
		offsets = []int{}
	}
	return AccountResponse{
		Connected:        true,
		Mode:             a.Mode,
		Status:           a.Status,
		WabaID:           a.WabaID,
		PhoneNumberID:    a.PhoneNumberID,
		DisplayPhone:     a.DisplayPhone,
		VerifiedName:     a.VerifiedName,
		TemplateName:     whatsapp_models.ReminderTemplate,
		TemplateStatus:   a.TemplateStatus,
		TemplateReason:   a.TemplateReason,
		RemindersEnabled: a.RemindersEnabled,
		ReminderOffsets:  offsets,
		ConnectedAt:      a.ConnectedAt,
	}
}

// ConnectEmbeddedRequest is what the Embedded Signup popup returns.
type ConnectEmbeddedRequest struct {
	Code          string `json:"code" binding:"required"`
	WabaID        string `json:"waba_id" binding:"required"`
	PhoneNumberID string `json:"phone_number_id" binding:"required"`
}

// ConnectManualRequest connects with credentials copied from Meta's dashboard.
type ConnectManualRequest struct {
	WabaID        string `json:"waba_id" binding:"required"`
	PhoneNumberID string `json:"phone_number_id" binding:"required"`
	AccessToken   string `json:"access_token" binding:"required"`
}

// Trim removes surrounding spaces pasted along with the values.
func (r *ConnectManualRequest) Trim() {
	r.WabaID, r.PhoneNumberID, r.AccessToken = strings.TrimSpace(r.WabaID), strings.TrimSpace(r.PhoneNumberID), strings.TrimSpace(r.AccessToken)
}

// SettingsRequest updates the reminder settings.
type SettingsRequest struct {
	RemindersEnabled *bool `json:"reminders_enabled" binding:"required"`
	ReminderOffsets  []int `json:"reminder_offsets"`
}

// TestRequest sends the reminder template with sample data to phone.
type TestRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// MessageResponse is one row of the message log.
type MessageResponse struct {
	ID            uint       `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	Kind          string     `json:"kind"`
	AppointmentID *uint      `json:"appointment_id"`
	PatientID     *uint      `json:"patient_id"`
	PatientName   string     `json:"patient_name"`
	ToPhone       string     `json:"to_phone"`
	Template      string     `json:"template"`
	OffsetHours   int        `json:"offset_hours"`
	Status        string     `json:"status"`
	ErrorCode     string     `json:"error_code"`
	ErrorDetail   string     `json:"error_detail"`
	Reply         string     `json:"reply"`
	SentAt        *time.Time `json:"sent_at"`
	DeliveredAt   *time.Time `json:"delivered_at"`
	ReadAt        *time.Time `json:"read_at"`
	RepliedAt     *time.Time `json:"replied_at"`

	ConversationID *uint  `json:"conversation_id"`
	Direction      string `json:"direction"`
	Body           string `json:"body"`
	ContentType    string `json:"content_type"`
	SentByUserID   *uint  `json:"sent_by_user_id"`
}

// NewMessageResponse maps a message (with Patient preloaded when available).
func NewMessageResponse(m whatsapp_models.WhatsAppMessage) MessageResponse {
	name := ""
	if m.Patient != nil {
		name = strings.TrimSpace(m.Patient.FirstName + " " + m.Patient.LastName)
	}
	return MessageResponse{
		ID: m.ID, CreatedAt: m.CreatedAt, Kind: m.Kind, AppointmentID: m.AppointmentID, PatientID: m.PatientID,
		PatientName: name, ToPhone: m.ToPhone, Template: m.Template, OffsetHours: m.OffsetHours, Status: m.Status,
		ErrorCode: m.ErrorCode, ErrorDetail: m.ErrorDetail, Reply: m.Reply, SentAt: m.SentAt,
		DeliveredAt: m.DeliveredAt, ReadAt: m.ReadAt, RepliedAt: m.RepliedAt,
		ConversationID: m.ConversationID, Direction: m.Direction, Body: m.Body, ContentType: m.ContentType,
		SentByUserID: m.SentByUserID,
	}
}

// TemplateResponse is one template of the catalog with its review status on
// the tenant's WABA.
type TemplateResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"` // Meta's value (APPROVED, PENDING, REJECTED, ...); "" when not created yet
	Reason string `json:"reason"` // rejection reason from Meta, if any
	// Variables names the body placeholders in order: patient, clinic, date, time.
	Variables        []string `json:"variables"`
	NeedsAppointment bool     `json:"needs_appointment"`
	// Inbox: can be sent from the inbox (every template but the reminder).
	Inbox bool `json:"inbox"`
	// Usable: Inbox and APPROVED.
	Usable  bool     `json:"usable"`
	Body    string   `json:"body"`    // with {{1}}… placeholders
	Preview string   `json:"preview"` // body filled with sample values
	Buttons []string `json:"buttons"`
}

// UsageResponse is the tenant's charged messages this calendar month (clinic
// time zone) against the plan's cap; limit -1 means unlimited. period_end is
// exclusive (first instant of next month).
type UsageResponse struct {
	Used        int64     `json:"used"`
	Limit       int64     `json:"limit"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}
