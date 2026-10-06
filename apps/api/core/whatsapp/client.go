// Package whatsapp is a small client for the WhatsApp Cloud API (Meta Graph
// API): sending template messages, managing message templates, and the calls
// a tenant's number needs when it is connected (code exchange, app
// subscription, phone registration). It knows nothing about tenants; callers
// pass the tenant's access token on every call.
package whatsapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is the Graph API host; tests point BaseURL at httptest.
const DefaultBaseURL = "https://graph.facebook.com"

// DefaultVersion is the Graph API version used when WHATSAPP_GRAPH_VERSION is unset.
const DefaultVersion = "v23.0"

// ErrTransport wraps a failure to reach the Graph API at all (DNS, timeout,
// connection refused, unreadable response). It is always retryable.
var ErrTransport = errors.New("whatsapp graph api unreachable")

// ErrNotConfigured means META_APP_ID / META_APP_SECRET are missing for a call
// that needs them (code exchange).
var ErrNotConfigured = errors.New("whatsapp: meta app credentials not configured")

// Client calls the Graph API. The zero value is not usable; build it with New
// or NewFromEnv.
type Client struct {
	BaseURL   string
	Version   string
	AppID     string
	AppSecret string
	HTTP      *http.Client
}

// New builds a client against baseURL (DefaultBaseURL when empty).
func New(baseURL, version, appID, appSecret string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if version == "" {
		version = DefaultVersion
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Version:   version,
		AppID:     appID,
		AppSecret: appSecret,
		HTTP:      &http.Client{Timeout: 20 * time.Second},
	}
}

// NewFromEnv builds a client from META_APP_ID, META_APP_SECRET and
// WHATSAPP_GRAPH_VERSION.
func NewFromEnv() *Client {
	return New(DefaultBaseURL, os.Getenv("WHATSAPP_GRAPH_VERSION"), os.Getenv("META_APP_ID"), os.Getenv("META_APP_SECRET"))
}

// APIError is an error answered by the Graph API.
type APIError struct {
	HTTPStatus int
	Code       int    `json:"code"`
	Subcode    int    `json:"error_subcode"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	Details    string `json:"-"`
	FBTraceID  string `json:"fbtrace_id"`
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("graph api error %d (http %d): %s", e.Code, e.HTTPStatus, e.Message)
	if e.Details != "" {
		msg += ": " + e.Details
	}
	return msg
}

// retryableCodes are Meta error codes for throttling or temporary outages.
var retryableCodes = map[int]bool{
	1:      true, // unknown, temporary
	2:      true, // service temporarily unavailable
	4:      true, // app rate limit
	80007:  true, // WABA rate limit
	130429: true, // throughput reached
	131000: true, // something went wrong
	131016: true, // service unavailable
	131056: true, // pair rate limit
	133004: true, // server temporarily unavailable
}

// Retryable reports whether the call may succeed if repeated later: 5xx, 429
// or one of Meta's throttling / temporary codes.
func (e *APIError) Retryable() bool {
	return e.HTTPStatus >= 500 || e.HTTPStatus == http.StatusTooManyRequests || retryableCodes[e.Code]
}

// IsAuthError reports whether the token was rejected (expired, revoked, or
// missing permissions).
func (e *APIError) IsAuthError() bool {
	return e.Code == 190 || e.Code == 10 || e.Code == 200 || e.HTTPStatus == http.StatusUnauthorized
}

// IsRetryable reports whether err (from any Client method) is worth retrying.
func IsRetryable(err error) bool {
	if errors.Is(err, ErrTransport) {
		return true
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Retryable()
}

func (c *Client) endpoint(path string, query url.Values) string {
	u := c.BaseURL + "/" + c.Version + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

// do sends one request and decodes a 2xx body into out (when non-nil).
func (c *Client) do(method, path, token string, query url.Values, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.endpoint(path, query), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return transportError(method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return transportError(method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return parseAPIError(resp.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("whatsapp: decode %s %s: %w", method, path, err)
	}
	return nil
}

// transportError wraps a failed request without its URL: *url.Error prints
// the full URL, and the query of a code exchange carries the app secret and
// the OAuth code. Only the method, the path (ids, never secrets) and the
// underlying cause are kept.
func transportError(method, path string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return fmt.Errorf("%w: %s %s: %v", ErrTransport, method, path, err)
}

// redact removes secrets from an error before it can reach a log, keeping
// *APIError (and ErrTransport wrapping) intact for errors.As / errors.Is.
func redact(err error, secrets ...string) error {
	clean := func(s string) string {
		for _, secret := range secrets {
			if secret != "" {
				s = strings.ReplaceAll(s, secret, "[redacted]")
			}
		}
		return s
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		apiErr.Message, apiErr.Details = clean(apiErr.Message), clean(apiErr.Details)
		return err
	}
	if msg := err.Error(); clean(msg) != msg {
		if errors.Is(err, ErrTransport) {
			return fmt.Errorf("%w: %s", ErrTransport, strings.TrimPrefix(clean(msg), ErrTransport.Error()+": "))
		}
		return errors.New(clean(msg))
	}
	return err
}

func parseAPIError(status int, raw []byte) error {
	var envelope struct {
		Error struct {
			APIError
			ErrorData struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"error"`
	}
	apiErr := &APIError{HTTPStatus: status}
	if json.Unmarshal(raw, &envelope) == nil && (envelope.Error.Code != 0 || envelope.Error.Message != "") {
		*apiErr = envelope.Error.APIError
		apiErr.HTTPStatus = status
		apiErr.Details = envelope.Error.ErrorData.Details
	} else {
		apiErr.Message = strings.TrimSpace(string(raw))
		if len(apiErr.Message) > 300 {
			apiErr.Message = apiErr.Message[:300]
		}
	}
	return apiErr
}

// QuickReplyButton is the payload returned by a template's quick-reply button.
type QuickReplyButton struct {
	Payload string
}

// TemplateMessage is a template to send to one recipient.
type TemplateMessage struct {
	To         string // digits only, international format (e.g. 593991234567)
	Name       string
	Language   string
	BodyParams []string
	Buttons    []QuickReplyButton // by index, matching the template's buttons
}

type textParam struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type sendComponent struct {
	Type       string      `json:"type"`
	SubType    string      `json:"sub_type,omitempty"`
	Index      string      `json:"index,omitempty"`
	Parameters []textParam `json:"parameters"`
}

// SendTemplate sends a template message and returns its WhatsApp message id (wamid).
func (c *Client) SendTemplate(token, phoneNumberID string, msg TemplateMessage) (string, error) {
	components := []sendComponent{}
	if len(msg.BodyParams) > 0 {
		body := sendComponent{Type: "body"}
		for _, p := range msg.BodyParams {
			body.Parameters = append(body.Parameters, textParam{Type: "text", Text: p})
		}
		components = append(components, body)
	}
	for i, b := range msg.Buttons {
		components = append(components, sendComponent{
			Type:       "button",
			SubType:    "quick_reply",
			Index:      fmt.Sprint(i),
			Parameters: []textParam{{Type: "payload", Payload: b.Payload}},
		})
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.To,
		"type":              "template",
		"template": map[string]any{
			"name":       msg.Name,
			"language":   map[string]string{"code": msg.Language},
			"components": components,
		},
	}
	return c.sendMessage(phoneNumberID, token, payload)
}

// SendText sends a free-text message (type "text") and returns its wamid.
// Meta only accepts it within 24 h of the user's last message.
func (c *Client) SendText(token, phoneNumberID, to, body string) (string, error) {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text":              map[string]any{"preview_url": false, "body": body},
	}
	return c.sendMessage(phoneNumberID, token, payload)
}

// sendMessage posts payload to the number's messages edge and returns the wamid.
func (c *Client) sendMessage(phoneNumberID, token string, payload map[string]any) (string, error) {
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := c.do(http.MethodPost, phoneNumberID+"/messages", token, nil, payload, &out); err != nil {
		return "", err
	}
	if len(out.Messages) == 0 || out.Messages[0].ID == "" {
		return "", errors.New("whatsapp: send answered without a message id")
	}
	return out.Messages[0].ID, nil
}

// TemplateButton is a quick-reply button of a template definition.
type TemplateButton struct {
	Text string
}

// TemplateDefinition is a template to create on a WABA.
type TemplateDefinition struct {
	Name     string
	Language string
	Category string // UTILITY, MARKETING, AUTHENTICATION
	Body     string
	Example  []string // one sample value per body variable
	Buttons  []TemplateButton
}

// Template is a template as listed by the Graph API.
type Template struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Language string `json:"language"`
	Category string `json:"category"`
}

// CreateTemplate submits a template for review and returns its id and initial status.
func (c *Client) CreateTemplate(token, wabaID string, def TemplateDefinition) (Template, error) {
	body := map[string]any{"type": "BODY", "text": def.Body}
	if len(def.Example) > 0 {
		body["example"] = map[string]any{"body_text": [][]string{def.Example}}
	}
	components := []map[string]any{body}
	if len(def.Buttons) > 0 {
		buttons := make([]map[string]string, 0, len(def.Buttons))
		for _, b := range def.Buttons {
			buttons = append(buttons, map[string]string{"type": "QUICK_REPLY", "text": b.Text})
		}
		components = append(components, map[string]any{"type": "BUTTONS", "buttons": buttons})
	}
	payload := map[string]any{
		"name":       def.Name,
		"language":   def.Language,
		"category":   def.Category,
		"components": components,
	}
	var out Template
	if err := c.do(http.MethodPost, wabaID+"/message_templates", token, nil, payload, &out); err != nil {
		return Template{}, err
	}
	out.Name, out.Language = def.Name, def.Language
	return out, nil
}

// ListTemplates lists the WABA's templates, narrowed to name when non-empty.
func (c *Client) ListTemplates(token, wabaID, name string) ([]Template, error) {
	q := url.Values{"fields": {"id,name,status,language,category"}, "limit": {"100"}}
	if name != "" {
		q.Set("name", name)
	}
	var out struct {
		Data []Template `json:"data"`
	}
	if err := c.do(http.MethodGet, wabaID+"/message_templates", token, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ExchangeCode trades an Embedded Signup code for the business's access token.
func (c *Client) ExchangeCode(code string) (string, error) {
	if c.AppID == "" || c.AppSecret == "" {
		return "", ErrNotConfigured
	}
	q := url.Values{"client_id": {c.AppID}, "client_secret": {c.AppSecret}, "code": {code}}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(http.MethodGet, "oauth/access_token", "", q, nil, &out); err != nil {
		return "", redact(err, c.AppSecret, code)
	}
	if out.AccessToken == "" {
		return "", errors.New("whatsapp: code exchange answered without a token")
	}
	return out.AccessToken, nil
}

// SubscribeApp subscribes our Meta app to the WABA's webhooks.
func (c *Client) SubscribeApp(token, wabaID string) error {
	return c.do(http.MethodPost, wabaID+"/subscribed_apps", token, nil, nil, nil)
}

// RegisterPhone registers the number for Cloud API use with a two-step
// verification pin.
func (c *Client) RegisterPhone(token, phoneNumberID, pin string) error {
	return c.do(http.MethodPost, phoneNumberID+"/register", token, nil,
		map[string]string{"messaging_product": "whatsapp", "pin": pin}, nil)
}

// PhoneNumber is a business phone number as answered by the Graph API.
type PhoneNumber struct {
	ID                 string `json:"id"`
	DisplayPhoneNumber string `json:"display_phone_number"`
	VerifiedName       string `json:"verified_name"`
	QualityRating      string `json:"quality_rating"`
}

// GetPhoneNumber reads a phone number; it doubles as a credentials check.
func (c *Client) GetPhoneNumber(token, phoneNumberID string) (PhoneNumber, error) {
	q := url.Values{"fields": {"id,display_phone_number,verified_name,quality_rating"}}
	var out PhoneNumber
	err := c.do(http.MethodGet, phoneNumberID, token, q, nil, &out)
	return out, err
}
