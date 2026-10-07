package whatsapp_services

import (
	"errors"
	"strings"
)

// ErrInvalidPhone means a phone number cannot be turned into an international
// WhatsApp number.
var ErrInvalidPhone = errors.New("invalid phone number")

// ecuadorCode is the country code assumed for national numbers (the clinics
// are in Ecuador), as the wa.me links in apps/web/src/lib/utils.ts do.
const ecuadorCode = "593"

// NormalizePhone turns a free-text patient phone into the digits-only
// international (E.164 without "+") form WhatsApp expects. Port of
// generateWhatsAppLink (apps/web/src/lib/utils.ts): non-digits are dropped and
// a leading national "0" becomes 593. Additionally "00" is read as the
// international prefix, a bare 9-digit Ecuadorian mobile (9XXXXXXXX) gets 593,
// and the result must be 10–15 digits long.
func NormalizePhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()

	switch {
	case strings.HasPrefix(raw, "+"):
		// already international
	case strings.HasPrefix(digits, "00"):
		digits = digits[2:]
	case strings.HasPrefix(digits, "0"):
		digits = ecuadorCode + digits[1:]
	case len(digits) == 9 && digits[0] == '9':
		digits = ecuadorCode + digits
	}

	if len(digits) < 10 || len(digits) > 15 || digits[0] == '0' {
		return "", ErrInvalidPhone
	}
	// Ecuadorian mobiles are 593 + 9 digits starting with 9.
	if strings.HasPrefix(digits, ecuadorCode) && len(digits) != 12 && len(digits) != 11 {
		return "", ErrInvalidPhone
	}
	return digits, nil
}
