package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Backoffice second-factor tokens: what a backoffice login gets after the
// password step instead of a session.
//
//   - BackofficeMFAPurposeLogin: the admin has a second factor; the token is
//     only good for POST /backoffice/auth/login/2fa.
//   - BackofficeMFAPurposeEnroll: the admin has none yet; the token is only
//     good for the /backoffice/auth/enroll/* endpoints, which end in a
//     session once a factor is enrolled.
//
// They are signed with their own key derived from AUTH_KEY (not AUTH_KEY, nor
// the clinic mfa key), so every other parser rejects them on the signature:
// they never open a backoffice endpoint, nor the clinic's /auth/login/2fa
// (backoffice and clinic user IDs overlap). Like the clinic's mfa token they
// are bound to the password hash: a password change voids them.
const (
	BackofficeMFATokenType     = "backoffice_mfa_pending"
	BackofficeMFAPurposeLogin  = "backoffice_login_2fa"
	BackofficeMFAPurposeEnroll = "backoffice_enroll"
	BackofficeMFALoginTTL      = 5 * time.Minute
	// Enrolling takes longer: installing an authenticator app, scanning.
	BackofficeMFAEnrollTTL = 15 * time.Minute
)

func backofficeMFAKey() ([]byte, error) {
	key, err := authKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("pengi/backoffice-mfa-token/v1"))
	return mac.Sum(nil), nil
}

func backofficeMFATTL(purpose string) time.Duration {
	if purpose == BackofficeMFAPurposeEnroll {
		return BackofficeMFAEnrollTTL
	}
	return BackofficeMFALoginTTL
}

// GenerateBackofficeMFAToken issues the token of a backoffice login whose
// password step passed, for purpose (login second step or enrollment).
func GenerateBackofficeMFAToken(purpose string, userID uint, passwordHash string, now time.Time) (string, error) {
	if purpose != BackofficeMFAPurposeLogin && purpose != BackofficeMFAPurposeEnroll {
		return "", ErrInvalidMFAToken
	}
	key, err := backofficeMFAKey()
	if err != nil {
		return "", err
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":    BackofficeMFATokenType,
		"purpose": purpose,
		"aud":     BackofficeAudience,
		"uid":     userID,
		"pwd":     passwordFingerprint(passwordHash),
		"iat":     now.Unix(),
		"exp":     now.Add(backofficeMFATTL(purpose)).Unix(),
	}).SignedString(key)
}

// ParseBackofficeMFAToken validates a backoffice mfa token for purpose:
// signature with the backoffice mfa key, type, purpose, audience and expiry.
func ParseBackofficeMFAToken(token, purpose string, now time.Time) (MFAClaims, error) {
	key, err := backofficeMFAKey()
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
	if claims["type"] != BackofficeMFATokenType || claims["purpose"] != purpose || !IsBackofficeToken(claims) {
		return MFAClaims{}, ErrInvalidMFAToken
	}
	uid, ok := claims["uid"].(float64)
	if !ok || uid <= 0 {
		return MFAClaims{}, ErrInvalidMFAToken
	}
	fp, _ := claims["pwd"].(string)
	return MFAClaims{UserID: uint(uid), fingerprint: fp}, nil
}
