package mailer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"pengi-med-saas/core/config"
)

const resendURL = "https://api.resend.com/emails"

type Mailer struct {
	apiKey   string
	from     string
	fromName string
}

func NewMailer() *Mailer {
	cfg := config.Get()
	return &Mailer{
		apiKey:   cfg.ResendAPIKey,
		from:     cfg.SMTPFrom,
		fromName: cfg.SMTPFromName,
	}
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64-encoded
}

type resendPayload struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

func (m *Mailer) send(to, subject, html string) error {
	return m.sendWithAttachments(to, subject, html, nil)
}

func (m *Mailer) sendWithAttachments(to, subject, html string, attachments []resendAttachment) error {
	if m.apiKey == "" {
		return fmt.Errorf("Resend not configured: RESEND_API_KEY is empty")
	}

	payload := resendPayload{
		From:        fmt.Sprintf("%s <%s>", m.fromName, m.from),
		To:          []string{to},
		Subject:     subject,
		HTML:        html,
		Attachments: attachments,
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

// SendMedicalDocumentEmail sends a generated PDF (medical report or
// certificate) as an email attachment to the given address.
func (m *Mailer) SendMedicalDocumentEmail(toEmail, subject, documentTitle, pdfFilename string, pdfBytes []byte) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #0d9488;">%s</h2>
  <p>Adjunto encontrarás el documento en formato PDF.</p>
</body>
</html>`, documentTitle)

	attachment := resendAttachment{
		Filename: pdfFilename,
		Content:  base64.StdEncoding.EncodeToString(pdfBytes),
	}

	return m.sendWithAttachments(toEmail, subject, html, []resendAttachment{attachment})
}

// PortalCodeEmail is the already-translated text of a patient portal sign-in
// code email (the caller resolves the i18n keys in the patient's language).
type PortalCodeEmail struct {
	Subject string
	Heading string
	Intro   string // e.g. "Use this code to sign in to the patient portal of Clínica X:"
	Code    string
	Expiry  string // e.g. "The code expires in 10 minutes and can be used only once."
	Ignore  string // e.g. "If you did not request it, ignore this message."
}

// SendPortalCode emails a one-time patient portal sign-in code. Every text is
// HTML-escaped: they carry the clinic's name.
func (m *Mailer) SendPortalCode(toEmail string, e PortalCodeEmail) error {
	esc := html.EscapeString
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #0d9488;">%s</h2>
  <p>%s</p>
  <p style="margin: 24px 0; font-size: 32px; font-weight: bold; letter-spacing: 8px; font-family: monospace;">%s</p>
  <p style="color: #666; font-size: 14px;">%s</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;" />
  <p style="color: #999; font-size: 12px;">%s</p>
</body>
</html>`, esc(e.Heading), esc(e.Intro), esc(e.Code), esc(e.Expiry), esc(e.Ignore))

	return m.send(toEmail, e.Subject, body)
}
