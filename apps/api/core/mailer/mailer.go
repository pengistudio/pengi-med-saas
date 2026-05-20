package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const resendURL = "https://api.resend.com/emails"

type Mailer struct {
	apiKey   string
	from     string
	fromName string
}

func NewMailer() *Mailer {
	fromName := os.Getenv("SMTP_FROM_NAME")
	if fromName == "" {
		fromName = "Gentoo"
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "noreply@pengistudio.com"
	}
	return &Mailer{
		apiKey:   os.Getenv("RESEND_API_KEY"),
		from:     from,
		fromName: fromName,
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (m *Mailer) send(to, subject, html string) error {
	if m.apiKey == "" {
		return fmt.Errorf("Resend not configured: RESEND_API_KEY is empty")
	}

	payload := resendPayload{
		From:    fmt.Sprintf("%s <%s>", m.fromName, m.from),
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, resendURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API error: status %d", resp.StatusCode)
	}
	return nil
}

func (m *Mailer) SendEmailVerification(toEmail string, verificationURL string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #0d9488;">Verifica tu email</h2>
  <p>Gracias por registrarte en Gentoo. Haz clic en el siguiente enlace para verificar tu email y activar tu cuenta:</p>
  <p style="margin: 24px 0;">
    <a href="%s" style="display: inline-block; padding: 12px 24px; background: #0d9488; color: white; text-decoration: none; border-radius: 6px; font-weight: bold;">
      Verificar email
    </a>
  </p>
  <p style="color: #666; font-size: 14px;">Si el botón no funciona, copia y pega este enlace en tu navegador:</p>
  <p style="color: #666; font-size: 13px; word-break: break-all;">%s</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;" />
  <p style="color: #999; font-size: 12px;">Si no creaste esta cuenta, puedes ignorar este mensaje. Este enlace expira en 24 horas.</p>
</body>
</html>`, verificationURL, verificationURL)

	return m.send(toEmail, "Verifica tu email - Gentoo", html)
}

func (m *Mailer) SendContactMessage(toEmail, name, fromEmail, message string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #0d9488;">Nuevo mensaje de contacto — Gentoo</h2>
  <table style="width:100%%; border-collapse: collapse; margin-top: 16px;">
    <tr><td style="padding: 8px; font-weight: bold; width: 80px;">Nombre:</td><td style="padding: 8px;">%s</td></tr>
    <tr style="background:#f9f9f9;"><td style="padding: 8px; font-weight: bold;">Email:</td><td style="padding: 8px;"><a href="mailto:%s">%s</a></td></tr>
    <tr><td style="padding: 8px; font-weight: bold; vertical-align: top;">Mensaje:</td><td style="padding: 8px; white-space: pre-wrap;">%s</td></tr>
  </table>
</body>
</html>`, name, fromEmail, fromEmail, message)

	subject := fmt.Sprintf("Nuevo contacto de %s — Gentoo", name)
	return m.send(toEmail, subject, html)
}
