package auth

import (
	"errors"
	"net/http"
	"time"

	"pengi-med-saas/core/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Backoffice tokens are signed with the same key as the clinic app's, so they
// carry an audience that sets them apart: a backoffice token never opens the
// clinic app and a clinic token never opens the backoffice.
const BackofficeAudience = "backoffice"

// BackofficeRefreshCookie is the backoffice's own refresh cookie, so a
// backoffice session and a clinic session in the same browser don't overwrite
// each other's refresh_token.
const BackofficeRefreshCookie = "backoffice_refresh_token"

const (
	accessTokenType  = "access_token"
	refreshTokenType = "refresh_token"
)

var ErrNotBackofficeToken = errors.New("not a backoffice token")

func signedWithAuthKey(claims jwt.MapClaims) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretKey))
}

// GenerateBackofficeToken issues a backoffice access token (AUTH_EXP minutes).
func GenerateBackofficeToken(username string, userID int64) (string, error) {
	exp, err := config.GetNumberEnv("AUTH_EXP")
	if err != nil {
		return "", err
	}
	return signedWithAuthKey(jwt.MapClaims{
		"type":     accessTokenType,
		"aud":      BackofficeAudience,
		"username": username,
		"userId":   userID,
		"exp":      time.Now().Add(time.Duration(exp) * time.Minute).Unix(),
	})
}

// GenerateBackofficeRefreshToken issues a backoffice refresh token
// (AUTH_REFRESH_EXP days).
func GenerateBackofficeRefreshToken(username string, userID int64) (string, error) {
	return signedWithAuthKey(jwt.MapClaims{
		"type":     refreshTokenType,
		"aud":      BackofficeAudience,
		"username": username,
		"userId":   userID,
		"exp":      time.Now().Add(RefreshTokenTTL()).Unix(),
	})
}

// IsBackofficeToken reports whether parsed claims belong to a backoffice token.
func IsBackofficeToken(claims jwt.MapClaims) bool {
	aud, err := claims.GetAudience()
	if err != nil {
		return false
	}
	for _, a := range aud {
		if a == BackofficeAudience {
			return true
		}
	}
	return false
}

// BackofficeClaims identifies the backoffice user a token was issued to.
type BackofficeClaims struct {
	UserID   int64
	Username string
}

func parseBackoffice(token, wantType string) (BackofficeClaims, error) {
	claims, err := ParseToken(token)
	if err != nil {
		return BackofficeClaims{}, err
	}
	if !IsBackofficeToken(claims) || claims["type"] != wantType {
		return BackofficeClaims{}, ErrNotBackofficeToken
	}
	userID, ok := claims["userId"].(float64)
	if !ok {
		return BackofficeClaims{}, ErrInvalidToken
	}
	username, ok := claims["username"].(string)
	if !ok {
		return BackofficeClaims{}, ErrInvalidToken
	}
	return BackofficeClaims{UserID: int64(userID), Username: username}, nil
}

// ParseBackofficeAccessToken accepts only unexpired backoffice access tokens.
func ParseBackofficeAccessToken(token string) (BackofficeClaims, error) {
	return parseBackoffice(token, accessTokenType)
}

// ParseBackofficeRefreshToken accepts only unexpired backoffice refresh tokens.
func ParseBackofficeRefreshToken(token string) (BackofficeClaims, error) {
	return parseBackoffice(token, refreshTokenType)
}

// SetBackofficeRefreshCookie sets the backoffice refresh cookie with the same
// attributes as the clinic app's refresh cookie.
func SetBackofficeRefreshCookie(c *gin.Context, refreshToken string) {
	httpsEnabled, _ := config.GetBoolEnv("HTTPS_ENABLED")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(BackofficeRefreshCookie, refreshToken, int(RefreshTokenTTL().Seconds()), "/", "", httpsEnabled, true)
}

// ClearBackofficeRefreshCookie removes the backoffice refresh cookie, with the
// same attributes it was set with so the browser deletes it.
func ClearBackofficeRefreshCookie(c *gin.Context) {
	httpsEnabled, _ := config.GetBoolEnv("HTTPS_ENABLED")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(BackofficeRefreshCookie, "", -1, "/", "", httpsEnabled, true)
}
