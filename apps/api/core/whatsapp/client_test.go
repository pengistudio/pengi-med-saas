package whatsapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pengi-med-saas/core/whatsapp"
)

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	valid := whatsapp.SignatureFor("app-secret", body)

	cases := []struct {
		name   string
		secret string
		body   []byte
		header string
		want   bool
	}{
		{"valid", "app-secret", body, valid, true},
		{"wrong secret", "other-secret", body, valid, false},
		{"tampered body", "app-secret", []byte(`{"object":"x"}`), valid, false},
		{"missing prefix", "app-secret", body, valid[len("sha256="):], false},
		{"not hex", "app-secret", body, "sha256=zzzz", false},
		{"empty header", "app-secret", body, "", false},
		{"empty secret never verifies", "", body, whatsapp.SignatureFor("", body), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := whatsapp.VerifySignature(tc.secret, tc.body, tc.header); got != tc.want {
				t.Fatalf("VerifySignature = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSendTemplate_RequestShape(t *testing.T) {
	var gotPath, gotAuth string
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.ABC"}]}`))
	}))
	defer srv.Close()

	c := whatsapp.New(srv.URL, "v23.0", "", "")
	id, err := c.SendTemplate("tok", "PHONE1", whatsapp.TemplateMessage{
		To: "593991234567", Name: "pengi_cita_recordatorio", Language: "es",
		BodyParams: []string{"Ana", "Clínica", "lunes 6 de octubre", "10:30"},
		Buttons:    []whatsapp.QuickReplyButton{{Payload: "confirm:7"}, {Payload: "cancel:7"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "wamid.ABC" {
		t.Fatalf("id = %q", id)
	}
	if gotPath != "/v23.0/PHONE1/messages" || gotAuth != "Bearer tok" {
		t.Fatalf("path %q auth %q", gotPath, gotAuth)
	}
	if got["messaging_product"] != "whatsapp" || got["type"] != "template" || got["to"] != "593991234567" {
		t.Fatalf("unexpected payload: %v", got)
	}
	tpl := got["template"].(map[string]any)
	comps := tpl["components"].([]any)
	if len(comps) != 3 {
		t.Fatalf("components = %d, want body + 2 buttons", len(comps))
	}
	btn := comps[2].(map[string]any)
	param := btn["parameters"].([]any)[0].(map[string]any)
	if btn["sub_type"] != "quick_reply" || btn["index"] != "1" || param["payload"] != "cancel:7" {
		t.Fatalf("second button = %v", btn)
	}
}

func TestAPIError_ParsingAndRetryable(t *testing.T) {
	status, body := http.StatusBadRequest, `{"error":{"message":"(#131026) Message undeliverable","type":"OAuthException","code":131026,"error_data":{"details":"no wa"},"fbtrace_id":"x"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	c := whatsapp.New(srv.URL, "", "", "")

	_, err := c.SendTemplate("tok", "P", whatsapp.TemplateMessage{To: "1"})
	var apiErr *whatsapp.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 131026 || apiErr.Details != "no wa" {
		t.Fatalf("err = %#v", err)
	}
	if whatsapp.IsRetryable(err) {
		t.Fatal("131026 must not be retryable")
	}

	status, body = http.StatusServiceUnavailable, `oops`
	if _, err = c.SendTemplate("tok", "P", whatsapp.TemplateMessage{To: "1"}); !whatsapp.IsRetryable(err) {
		t.Fatalf("5xx must be retryable: %v", err)
	}
	status, body = http.StatusTooManyRequests, `{"error":{"code":130429,"message":"rate"}}`
	if _, err = c.SendTemplate("tok", "P", whatsapp.TemplateMessage{To: "1"}); !whatsapp.IsRetryable(err) {
		t.Fatalf("429 must be retryable: %v", err)
	}

	srv.Close()
	if _, err = c.SendTemplate("tok", "P", whatsapp.TemplateMessage{To: "1"}); !errors.Is(err, whatsapp.ErrTransport) || !whatsapp.IsRetryable(err) {
		t.Fatalf("transport error = %v", err)
	}
}

func TestExchangeCode_NeedsAppCredentials(t *testing.T) {
	if _, err := whatsapp.New("http://unused", "", "", "").ExchangeCode("code"); !errors.Is(err, whatsapp.ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v23.0/oauth/access_token" || q.Get("client_id") != "app" || q.Get("client_secret") != "secret" || q.Get("code") != "the-code" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"business-token","token_type":"bearer"}`))
	}))
	defer srv.Close()
	token, err := whatsapp.New(srv.URL, "v23.0", "app", "secret").ExchangeCode("the-code")
	if err != nil || token != "business-token" {
		t.Fatalf("token %q err %v", token, err)
	}
}

func TestExchangeCode_ErrorsNeverCarrySecretOrCode(t *testing.T) {
	const secret, code = "super-app-secret", "oauth-code-123"

	// Transport failure: *url.Error would print the URL with both in its query.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Close()
	_, err := whatsapp.New(base, "v23.0", "app", secret).ExchangeCode(code)
	if err == nil || !errors.Is(err, whatsapp.ErrTransport) {
		t.Fatalf("err = %v, want a transport error", err)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), code) {
		t.Fatalf("transport error leaks secrets: %v", err)
	}

	// Graph error whose body echoes the request.
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":100,"message":"bad code ` + r.URL.Query().Get("code") + ` for ` + r.URL.Query().Get("client_secret") + `"}}`))
	}))
	defer echo.Close()
	_, err = whatsapp.New(echo.URL, "v23.0", "app", secret).ExchangeCode(code)
	var apiErr *whatsapp.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 100 {
		t.Fatalf("err = %#v", err)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), code) {
		t.Fatalf("graph error leaks secrets: %v", err)
	}
}

func TestOptOutRequested(t *testing.T) {
	text := func(body string) whatsapp.WebhookMessage {
		m := whatsapp.WebhookMessage{Type: "text"}
		m.Text = &struct {
			Body string `json:"body"`
		}{Body: body}
		return m
	}
	for _, body := range []string{"STOP", "stop", " Baja. ", "darme  de baja", "PARAR!"} {
		if !text(body).OptOutRequested() {
			t.Errorf("%q should opt out", body)
		}
	}
	for _, body := range []string{"no", "cancelar", "hola", "stop por favor", ""} {
		if text(body).OptOutRequested() {
			t.Errorf("%q should not opt out", body)
		}
	}
	if (whatsapp.WebhookMessage{Type: "button"}).OptOutRequested() {
		t.Error("message without text should not opt out")
	}
}
