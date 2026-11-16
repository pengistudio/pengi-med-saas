package backoffice_handlers

import (
	"net/http"
	"strings"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
)

// commonAdminPasswords are well-known passwords long enough to pass the
// length rule (shorter ones already fail it). Compared lowercased.
var commonAdminPasswords = map[string]bool{
	"0123456789": true, "1234567890": true, "0987654321": true, "9876543210": true,
	"12345678910": true, "123456789012": true, "1q2w3e4r5t": true, "1qaz2wsx3edc": true,
	"qwertyuiop": true, "qwerty1234": true, "qwerty12345": true, "qwerty123456": true,
	"asdfghjkl1": true, "zxcvbnm123": true, "abcdefghij": true, "abc1234567": true,
	"password12": true, "password123": true, "password1234": true, "password1!": true,
	"passw0rd123": true, "p@ssw0rd123": true, "p@ssword123": true,
	"administrator": true, "admin12345": true, "admin123456": true, "adminadmin": true,
	"admin@1234": true, "letmein123": true, "welcome123": true, "iloveyou12": true,
	"contrasena": true, "contraseña": true, "contrasena123": true, "contraseña123": true,
	"backoffice": true, "backoffice1": true, "backoffice123": true,
	"gentoo1234": true, "gentoo12345": true, "pengi12345": true, "pengistudio": true,
}

// weakAdminPassword reports why password is not acceptable for an admin, as
// the response to return, or nil. Beyond the minimum length it refuses the
// common passwords above, a single repeated character, and the admin's own
// user name or name (case-insensitive).
func weakAdminPassword(password string, identities ...string) *envelope.Response {
	if len(password) < auth.MinPasswordLength {
		resp := passwordTooShort()
		return &resp
	}
	lower := strings.ToLower(strings.TrimSpace(password))
	weak := lower == "" || commonAdminPasswords[lower] || strings.Count(lower, lower[:1]) == len(lower)
	for _, id := range identities {
		if id = strings.ToLower(strings.TrimSpace(id)); id != "" && id == lower {
			weak = true
		}
	}
	if weak {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "auth.password.too_weak", core_errors.ErrAuthPasswordTooWeak)
		return &resp
	}
	return nil
}
