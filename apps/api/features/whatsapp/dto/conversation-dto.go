package whatsapp_dto

import (
	"strings"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// MaxReplyLength is the longest free text WhatsApp accepts (in characters).
const MaxReplyLength = 4096

// ConversationResponse is one row of the inbox list.
type ConversationResponse struct {
	ID                   uint       `json:"id"`
	CreatedAt            time.Time  `json:"created_at"`
	Phone                string     `json:"phone"`
	PatientID            *uint      `json:"patient_id"`
	PatientName          string     `json:"patient_name"`
	PatientWhatsAppOptIn bool       `json:"patient_whatsapp_opt_in"`
	LastMessageAt        *time.Time `json:"last_message_at"`
	LastMessagePreview   string     `json:"last_message_preview"`
	LastDirection        string     `json:"last_direction"`
	LastContentType      string     `json:"last_content_type"`
	LastInboundAt        *time.Time `json:"last_inbound_at"`
	UnreadCount          int        `json:"unread_count"`
	WindowOpen           bool       `json:"window_open"`
	WindowExpiresAt      *time.Time `json:"window_expires_at"`
}

// NewConversationResponse maps a conversation (with Patient preloaded when
// linked); window_open is evaluated at now.
func NewConversationResponse(c whatsapp_models.WhatsAppConversation, now time.Time) ConversationResponse {
	r := ConversationResponse{
		ID: c.ID, CreatedAt: c.CreatedAt, Phone: c.Phone, PatientID: c.PatientID,
		LastMessageAt: c.LastMessageAt, LastMessagePreview: c.LastMessagePreview, LastDirection: c.LastDirection,
		LastContentType: c.LastContentType, LastInboundAt: c.LastInboundAt, UnreadCount: c.UnreadCount,
		WindowOpen: c.WindowOpen(now), WindowExpiresAt: c.WindowExpiresAt(),
	}
	if c.Patient != nil && c.Patient.ID != 0 {
		r.PatientName = strings.TrimSpace(c.Patient.FirstName + " " + c.Patient.LastName)
		r.PatientWhatsAppOptIn = c.Patient.WhatsAppOptIn
	}
	return r
}

// PatientSummary is the linked patient in the conversation header.
type PatientSummary struct {
	ID            uint   `json:"id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Document      string `json:"document"`
	Phone         string `json:"phone"`
	WhatsAppOptIn bool   `json:"whatsapp_opt_in"`
}

// AppointmentSummary is the patient's next appointment in the header.
type AppointmentSummary struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Date      time.Time `json:"date"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	Status    string    `json:"status"`
}

// ConversationDetailResponse is the conversation header: the list row plus
// the patient and their next appointment (null when unknown / none).
type ConversationDetailResponse struct {
	ConversationResponse
	Patient         *PatientSummary     `json:"patient"`
	NextAppointment *AppointmentSummary `json:"next_appointment"`
}

// NewConversationDetailResponse maps the header.
func NewConversationDetailResponse(c whatsapp_models.WhatsAppConversation, next *clinical_models.Appointment, now time.Time) ConversationDetailResponse {
	r := ConversationDetailResponse{ConversationResponse: NewConversationResponse(c, now)}
	if p := c.Patient; p != nil && p.ID != 0 {
		r.Patient = &PatientSummary{ID: p.ID, FirstName: p.FirstName, LastName: p.LastName, Document: p.Document, Phone: p.Phone, WhatsAppOptIn: p.WhatsAppOptIn}
	}
	if next != nil {
		r.NextAppointment = &AppointmentSummary{ID: next.ID, Title: next.Title, Date: next.Date, StartTime: next.StartTime, EndTime: next.EndTime, Status: next.Status}
	}
	return r
}

// ThreadResponse is a page of a conversation's messages, oldest first.
// has_more says whether older messages exist (ask again with before=items[0].id).
type ThreadResponse struct {
	Items   []MessageResponse `json:"items"`
	HasMore bool              `json:"has_more"`
}

// UnreadCountResponse is the inbox badge.
type UnreadCountResponse struct {
	UnreadCount   int64 `json:"unread_count"`  // unread messages
	Conversations int64 `json:"conversations"` // conversations with unread messages
}

// ReplyRequest is a free-text reply.
type ReplyRequest struct {
	Body string `json:"body"`
}

// LinkPatientRequest links a conversation to a patient of the tenant.
type LinkPatientRequest struct {
	PatientID uint `json:"patient_id" binding:"required"`
}

// StartConversationRequest sends a catalog template to a patient, creating
// the conversation with their number when there is none.
// appointment_id is required by templates with date / time (reprogramar).
type StartConversationRequest struct {
	PatientID     uint   `json:"patient_id" binding:"required"`
	Template      string `json:"template" binding:"required"`
	AppointmentID *uint  `json:"appointment_id"`
}

// SendTemplateRequest sends a catalog template in an existing conversation
// (also when the reply window is closed).
type SendTemplateRequest struct {
	Template      string `json:"template" binding:"required"`
	AppointmentID *uint  `json:"appointment_id"`
}

// TemplateSendResponse is the conversation (list row, after the send) and
// the template message as it stays in the thread.
type TemplateSendResponse struct {
	Conversation ConversationResponse `json:"conversation"`
	Message      MessageResponse      `json:"message"`
}
