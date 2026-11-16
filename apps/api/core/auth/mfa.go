package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// The mfa token proves the password step of a login for a user with
// two-factor authentication, and is only good for POST /auth/login/2fa.
//
// It is signed with a key derived from AUTH_KEY, not AUTH_KEY itself, so every
// other parser (ParseToken and everything built on it: AuthMiddleware,
// ExchangeAuthMiddleware, OptionalAuthMiddleware, the refresh endpoint,
// backoffice and portal) rejects it on the signature. It also has its own
// type and purpose and no userId/username claims, so it fails those parsers'
// claim checks too.
const (
	MFATokenType    = "mfa_pending"
	MFATokenPurpose = "login_2fa"
	MFATokenTTL     = 5 * time.Minute
)

var ErrInvalidMFAToken = errors.New("invalid mfa token")

func mfaKey() ([]byte, error) {
	key, err := authKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("pengi/mfa-token/v1"))
	return mac.Sum(nil), nil
}

// GenerateMFAToken issues the token of a login whose password step passed.
// It is bound to the current password hash: a password change voids it.
func GenerateMFAToken(userID uint, passwordHash string, now time.Time) (string, error) {
	key, err := mfaKey()
	if err != nil {
		return "", err
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":    MFATokenType,
		"purpose": MFATokenPurpose,
		"uid":     userID,
		"pwd":     passwordFingerprint(passwordHash),
		"iat":     now.Unix(),
		"exp":     now.Add(MFATokenTTL).Unix(),
	}).SignedString(key)
}

// MFAClaims is what ParseMFAToken extracts.
type MFAClaims struct {
	UserID      uint
	fingerprint string
}

// MatchesPassword reports whether the token was issued for passwordHash.
func (c MFAClaims) MatchesPassword(passwordHash string) bool {
	return c.fingerprint != "" && hmac.Equal([]byte(c.fingerprint), []byte(passwordFingerprint(passwordHash)))
}

// ParseMFAToken validates an mfa token: signature with the mfa key, type,
// purpose and expiry. Access, exchange, refresh and any other token fail.
func ParseMFAToken(token string, now time.Time) (MFAClaims, error) {
	key, err := mfaKey()
	if err != nil {
		return MFAClaims{}, err
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) {
		return key, nil
	}, hs256Only, jwt.WithTimeFunc(func() time.Time { return now }), jwt.WithExpirationRequired())
	if err != nil || parsed == nil || !parsed.Valid {
		return MFAClaims{}, ErrInvalidMFAToken
	}
	if claims["type"] != MFATokenType || claims["purpose"] != MFATokenPurpose {
		return MFAClaims{}, ErrInvalidMFAToken
	}
	uid, ok := claims["uid"].(float64)
	if !ok || uid <= 0 {
		return MFAClaims{}, ErrInvalidMFAToken
	}
	fp, _ := claims["pwd"].(string)
	return MFAClaims{UserID: uint(uid), fingerprint: fp}, nil
}
