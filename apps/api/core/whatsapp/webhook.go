package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignatureHeader carries the HMAC of a webhook body.
const SignatureHeader = "X-Hub-Signature-256"

// VerifySignature checks header ("sha256=" + hex HMAC-SHA256 of the raw body
// keyed with the app secret) in constant time. An empty secret never verifies.
func VerifySignature(appSecret string, body []byte, header string) bool {
	if appSecret == "" {
		return false
	}
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	received, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	return hmac.Equal(received, Sign(appSecret, body))
}

// Sign returns the raw HMAC-SHA256 of body keyed with appSecret.
func Sign(appSecret string, body []byte) []byte {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	return mac.Sum(nil)
}

// SignatureFor is the header value Meta would send for body (used by tests).
func SignatureFor(appSecret string, body []byte) string {
	return "sha256=" + hex.EncodeToString(Sign(appSecret, body))
}

// WebhookPayload is the body Meta posts to the webhook (object
// "whatsapp_business_account").
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry groups the changes of one WABA (ID is the WABA id).
type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

// WebhookChange is one change; Field says which part of Value is filled.
type WebhookChange struct {
	Field string       `json:"field"` // "messages", "message_template_status_update", ...
	Value WebhookValue `json:"value"`
}

// WebhookValue merges the shapes of the fields we subscribe to.
type WebhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         WebhookMetadata  `json:"metadata"`
	Statuses         []WebhookStatus  `json:"statuses"`
	Messages         []WebhookMessage `json:"messages"`

	// message_template_status_update
	Event                   string `json:"event"`
	MessageTemplateID       any    `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	Reason                  string `json:"reason"`
}

// WebhookMetadata identifies the business number that received the event.
type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// WebhookStatus is a delivery status update of a message we sent.
type WebhookStatus struct {
	ID          string         `json:"id"` // wamid
	Status      string         `json:"status"`
	Timestamp   string         `json:"timestamp"`
	RecipientID string         `json:"recipient_id"`
	Errors      []WebhookError `json:"errors"`
}

// WebhookError is an error attached to a failed status.
type WebhookError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

// WebhookMessage is an incoming message from a user.
type WebhookMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"` // "button" for template quick replies, "interactive", "text", ...
	Button    *struct {
		Payload string `json:"payload"`
		Text    string `json:"text"`
	} `json:"button,omitempty"`
	Interactive *struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
	} `json:"interactive,omitempty"`
	Context *struct {
		ID string `json:"id"` // wamid of the message replied to
	} `json:"context,omitempty"`
	Text *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
	// Media are only labelled (not downloaded); the caption, when any, is kept.
	Image    *WebhookMedia `json:"image,omitempty"`
	Audio    *WebhookMedia `json:"audio,omitempty"`
	Video    *WebhookMedia `json:"video,omitempty"`
	Document *WebhookMedia `json:"document,omitempty"`
	Sticker  *WebhookMedia `json:"sticker,omitempty"`
}

// WebhookMedia is the media object of an incoming image, audio, document...
type WebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"`
}

// ButtonText returns the label of the button the user tapped, or "".
func (m WebhookMessage) ButtonText() string {
	if m.Button != nil {
		return m.Button.Text
	}
	if m.Interactive != nil && m.Interactive.ButtonReply != nil {
		return m.Interactive.ButtonReply.Title
	}
	return ""
}

// Caption returns the caption of a media message, or "".
func (m WebhookMessage) Caption() string {
	for _, media := range []*WebhookMedia{m.Image, m.Video, m.Document} {
		if media != nil && media.Caption != "" {
			return media.Caption
		}
	}
	return ""
}

// optOutKeywords are the replies that withdraw consent to WhatsApp messages.
// "NO" and "CANCELAR" are left out on purpose: patients use them about the
// appointment, not the channel.
var optOutKeywords = map[string]bool{
	"STOP": true, "BAJA": true, "PARAR": true, "DETENER": true,
	"UNSUBSCRIBE": true, "DARME DE BAJA": true, "NO ENVIAR": true,
}

// OptOutRequested reports whether m is a text message asking to stop
// receiving messages (e.g. "STOP", "baja.").
func (m WebhookMessage) OptOutRequested() bool {
	if m.Text == nil {
		return false
	}
	body := strings.ToUpper(strings.Trim(strings.TrimSpace(m.Text.Body), ".!¡ "))
	return optOutKeywords[strings.Join(strings.Fields(body), " ")]
}

// ButtonPayload returns the payload of a quick-reply button press, or "".
func (m WebhookMessage) ButtonPayload() string {
	if m.Button != nil {
		return m.Button.Payload
	}
	if m.Interactive != nil && m.Interactive.ButtonReply != nil {
		return m.Interactive.ButtonReply.ID
	}
	return ""
}
