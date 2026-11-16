package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TOTP (RFC 6238) as authenticator apps implement it: HMAC-SHA1, 6 digits,
// 30-second steps. Standard library only, so the clock is a plain argument
// and the matched step is returned for replay protection.
const (
	TOTPPeriod = 30
	TOTPDigits = 6
	// TOTPSkew is how many steps before and after the current one are
	// accepted, for clock drift between the server and the phone.
	TOTPSkew = 1
	// TOTPIssuer is the name authenticator apps show next to the account.
	TOTPIssuer = "Gentoo"

	totpSecretBytes = 20 // 160 bits, the RFC 4226 recommendation
)

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a random secret, base32 without padding (the form
// authenticator apps take).
func NewTOTPSecret() (string, error) {
	raw := make([]byte, totpSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return totpEncoding.EncodeToString(raw), nil
}

// TOTPStep is the time step t falls in.
func TOTPStep(t time.Time) int64 { return t.Unix() / TOTPPeriod }

// TOTPCode is the code of secret at step.
func TOTPCode(secret string, step int64) (string, error) {
	key, err := totpEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("totp: invalid secret: %w", err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1_000_000), nil
}

// MatchTOTP checks code against secret at now ± TOTPSkew steps and returns
// the step it matched. Every candidate is compared (constant time), so the
// time taken doesn't depend on which step matched.
func MatchTOTP(secret, code string, now time.Time) (int64, bool) {
	if len(code) != TOTPDigits {
		return 0, false
	}
	current := TOTPStep(now)
	var matched int64
	found := false
	for step := current - TOTPSkew; step <= current+TOTPSkew; step++ {
		want, err := TOTPCode(secret, step)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 && !found {
			matched, found = step, true
		}
	}
	return matched, found
}

// TOTPURI is the otpauth:// URI an authenticator app scans as a QR code.
func TOTPURI(secret, account string) string {
	label := url.PathEscape(TOTPIssuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", TOTPIssuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(TOTPDigits))
	q.Set("period", fmt.Sprint(TOTPPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}
