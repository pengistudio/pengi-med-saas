package mailer

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type Mailer struct {
	host     string
	port     int
	user     string
	password string
	from     string
	fromName string
}

func NewMailer() *Mailer {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	fromName := os.Getenv("SMTP_FROM_NAME")
	if fromName == "" {
		fromName = "Pengi"
	}
	return &Mailer{
		host:     os.Getenv("SMTP_HOST"),
		port:     port,
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
		fromName: fromName,
	}
}

func (m *Mailer) SendEmailVerification(toEmail string, verificationURL string) error {
	if m.host == "" {
		return fmt.Errorf("SMTP not configured: SMTP_HOST is empty")
	}

	subject := "Verifica tu email - Pengi"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #0d9488;">Verifica tu email</h2>
  <p>Gracias por registrarte en Pengi. Haz clic en el siguiente enlace para verificar tu email y activar tu cuenta:</p>
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

	fromHeader := fmt.Sprintf("%s <%s>", m.fromName, m.from)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	smtpAuth := smtp.PlainAuth("", m.user, m.password, m.host)
	return smtp.SendMail(addr, smtpAuth, m.from, []string{toEmail}, []byte(msg))
}

func (m *Mailer) SendContactMessage(toEmail, name, fromEmail, message string) error {
	if m.host == "" {
		return fmt.Errorf("SMTP not configured: SMTP_HOST is empty")
	}

	subject := fmt.Sprintf("Nuevo contacto de %s — Gentoo", name)
	body := fmt.Sprintf(`<!DOCTYPE html>
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

	fromHeader := fmt.Sprintf("%s <%s>", m.fromName, m.from)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	smtpAuth := smtp.PlainAuth("", m.user, m.password, m.host)
	return smtp.SendMail(addr, smtpAuth, m.from, []string{toEmail}, []byte(msg))
}
