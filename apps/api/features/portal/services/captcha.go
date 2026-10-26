package portal_services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"pengi-med-saas/core/config"
)

// TurnstileVerifyURL is Cloudflare Turnstile's server-side validation endpoint.
const TurnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileTimeout bounds the siteverify call; past it the sign-in fails closed.
const turnstileTimeout = 5 * time.Second

// maxTurnstileToken is the longest token Turnstile issues (2048 characters).
const maxTurnstileToken = 2048

// CaptchaVerifier checks the captcha token a browser sent with a sign-in
// request. Any error (Cloudflare unreachable, bad answer) counts as a failure.
type CaptchaVerifier interface {
	Verify(ctx context.Context, token, remoteIP string) (bool, error)
}

// Turnstile verifies Cloudflare Turnstile tokens.
type Turnstile struct {
	secret string
	url    string
	client *http.Client
}

// NewTurnstile verifies against verifyURL (TurnstileVerifyURL in production)
// with client; a nil client gets a 5-second timeout.
func NewTurnstile(secret, verifyURL string, client *http.Client) *Turnstile {
	if client == nil {
		client = &http.Client{Timeout: turnstileTimeout}
	}
	return &Turnstile{secret: secret, url: verifyURL, client: client}
}

// TurnstileFromConfig is the verifier for TURNSTILE_SECRET_KEY, or nil when it
// is unset (no captcha).
func TurnstileFromConfig() CaptchaVerifier {
	secret := strings.TrimSpace(config.Get().TurnstileSecretKey)
	if secret == "" {
		return nil
	}
	return NewTurnstile(secret, TurnstileVerifyURL, nil)
}

func (t *Turnstile) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxTurnstileToken {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, turnstileTimeout)
	defer cancel()
	form := url.Values{"secret": {t.secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := t.client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	var body struct {
		Success bool `json:"success"`
	}
	if res.StatusCode != http.StatusOK {
		return false, nil
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return false, err
	}
	return body.Success, nil
}
