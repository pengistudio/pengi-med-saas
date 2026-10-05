package tenant_models

import (
	"crypto/rand"
	"encoding/base64"
)

// The display token is the only credential of the public waiting-room TV
// (GET /public/appointments/today?token=…): whoever holds it sees the tenant's
// agenda for today. It is a long random URL-safe secret, carried in the TV link
// and never typed by hand. Older 8-digit pairing codes were brute-forceable and
// were all replaced by migration DB20261004_4.

// displayTokenBytes is the token's entropy: 24 bytes = 192 bits, which encode
// to DisplayTokenLength base64url characters (fits the varchar(36) column).
const displayTokenBytes = 24

// DisplayTokenLength is the length of every token NewDisplayToken returns.
const DisplayTokenLength = 32

// NewDisplayToken returns a fresh random display token from crypto/rand.
func NewDisplayToken() string {
	b := make([]byte, displayTokenBytes)
	// crypto/rand.Read never fails on supported platforms (it panics instead).
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// IsDisplayToken reports whether token has the shape of a display token
// (DisplayTokenLength base64url characters). It says nothing about whether the
// token belongs to a tenant.
func IsDisplayToken(token string) bool {
	if len(token) != DisplayTokenLength {
		return false
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}
