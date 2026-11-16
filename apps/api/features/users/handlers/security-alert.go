package user_handlers

import (
	"time"

	"pengi-med-saas/core/mailer"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// securityEvent is a sensitive account event the owner is emailed about.
type securityEvent string

const (
	eventPasswordChanged          securityEvent = "password_changed"
	eventTwoFactorEnabled         securityEvent = "two_factor_enabled"
	eventTwoFactorDisabled        securityEvent = "two_factor_disabled"
	eventRecoveryCodesRegenerated securityEvent = "recovery_codes_regenerated"
	eventPasskeyAdded             securityEvent = "passkey_added"
	eventPasskeyDeleted           securityEvent = "passkey_deleted"
)

// securityAlertText is the title and sentence of each event's email.
var securityAlertText = map[securityEvent][2]string{
	eventPasswordChanged:          {"Tu contraseña fue cambiada", "la contraseña de tu cuenta de Gentoo fue cambiada."},
	eventTwoFactorEnabled:         {"Activaste la verificación en dos pasos", "se activó la verificación en dos pasos en tu cuenta de Gentoo."},
	eventTwoFactorDisabled:        {"Desactivaste la verificación en dos pasos", "se desactivó la verificación en dos pasos en tu cuenta de Gentoo."},
	eventRecoveryCodesRegenerated: {"Se generaron nuevos códigos de recuperación", "se generaron nuevos códigos de recuperación para tu cuenta de Gentoo; los anteriores dejaron de funcionar."},
	eventPasskeyAdded:             {"Se agregó una llave de acceso", "se agregó una llave de acceso (passkey) a tu cuenta de Gentoo."},
	eventPasskeyDeleted:           {"Se eliminó una llave de acceso", "se eliminó una llave de acceso (passkey) de tu cuenta de Gentoo."},
}

// alertSecurity tells the user about a sensitive event, once it succeeded.
func (h *UserHandler) alertSecurity(user user_models.User, event securityEvent, c *gin.Context) {
	if h.notifySecurity != nil {
		h.notifySecurity(user, event, c)
	}
}

// newSecurityAlertSender returns the default notifySecurity: it emails the
// alert in the background, never blocks or fails the request, and logs send
// errors with logger. The IP and time are read before the goroutine starts
// because gin.Context isn't safe after the handler returns. Users without an
// email are skipped.
func newSecurityAlertSender(logger *zap.Logger) func(user_models.User, securityEvent, *gin.Context) {
	return func(user user_models.User, event securityEvent, c *gin.Context) {
		sendSecurityAlert(logger, user, event, c)
	}
}

func sendSecurityAlert(logger *zap.Logger, user user_models.User, event securityEvent, c *gin.Context) {
	text, ok := securityAlertText[event]
	if !ok || user.Email == "" {
		return
	}
	alert := mailer.SecurityAlert{Title: text[0], Detail: text[1], Time: time.Now(), IP: c.ClientIP()}
	to, name, id := user.Email, user.UserName, user.ID
	go func() {
		if err := mailer.NewMailer().SendSecurityAlert(to, name, alert); err != nil {
			logger.Error("Failed to send security alert email", zap.Uint("user_id", id), zap.String("event", string(event)), zap.Error(err))
		}
	}()
}
