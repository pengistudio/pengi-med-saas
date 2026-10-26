package auth

import (
	"errors"
	"net/http"
	"time"

	"pengi-med-saas/core/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Patient portal tokens are signed with the same key as the clinic app's, so
// they carry their own audience and token types: a portal token never opens a
// staff or backoffice route, and a staff or backoffice token never opens the
// portal. They identify a patient of one tenant, never a user.
const PortalAudience = "portal"

// Portal token types.
const (
	PortalAccessTokenType  = "portal_access"
	PortalRefreshTokenType = "portal_refresh"
)

// Portal token lifetimes.
const (
	PortalAccessTTL  = 15 * time.Minute
	PortalRefreshTTL = 7 * 24 * time.Hour
)

// PortalRefreshCookie holds the portal refresh token, scoped to the portal API
// so it never travels with staff requests.
const (
	PortalRefreshCookie     = "portal_refresh"
	PortalRefreshCookiePath = "/api/v1/portal"
)

var ErrNotPortalToken = errors.New("not a portal token")

// PortalClaims identifies the patient (of one tenant) a portal token was issued to.
type PortalClaims struct {
	TenantID  uint
	PatientID uint
}

func generatePortal(tokenType string, ttl time.Duration, tenantID, patientID uint) (string, error) {
	return signedWithAuthKey(jwt.MapClaims{
		"type":       tokenType,
		"aud":        PortalAudience,
		"tenant_id":  tenantID,
		"patient_id": patientID,
		"exp":        time.Now().Add(ttl).Unix(),
	})
}

// GeneratePortalAccessToken issues a portal access token (15 minutes).
func GeneratePortalAccessToken(tenantID, patientID uint) (string, error) {
	return generatePortal(PortalAccessTokenType, PortalAccessTTL, tenantID, patientID)
}

// GeneratePortalRefreshToken issues a portal refresh token (7 days).
func GeneratePortalRefreshToken(tenantID, patientID uint) (string, error) {
	return generatePortal(PortalRefreshTokenType, PortalRefreshTTL, tenantID, patientID)
}

// IsPortalToken reports whether parsed claims belong to a patient portal token.
func IsPortalToken(claims jwt.MapClaims) bool {
	if t, _ := claims["type"].(string); t == PortalAccessTokenType || t == PortalRefreshTokenType {
		return true
	}
	aud, err := claims.GetAudience()
	if err != nil {
		return false
	}
	for _, a := range aud {
		if a == PortalAudience {
			return true
		}
	}
	return false
}

func parsePortal(token, wantType string) (PortalClaims, error) {
	claims, err := ParseToken(token)
	if err != nil {
		return PortalClaims{}, err
	}
	aud, err := claims.GetAudience()
	if err != nil || len(aud) != 1 || aud[0] != PortalAudience || claims["type"] != wantType {
		return PortalClaims{}, ErrNotPortalToken
	}
	tenantID, ok1 := claims["tenant_id"].(float64)
	patientID, ok2 := claims["patient_id"].(float64)
	if !ok1 || !ok2 || tenantID <= 0 || patientID <= 0 {
		return PortalClaims{}, ErrInvalidToken
	}
	return PortalClaims{TenantID: uint(tenantID), PatientID: uint(patientID)}, nil
}

// ParsePortalAccessToken accepts only unexpired portal access tokens.
func ParsePortalAccessToken(token string) (PortalClaims, error) {
	return parsePortal(token, PortalAccessTokenType)
}

// ParsePortalRefreshToken accepts only unexpired portal refresh tokens.
func ParsePortalRefreshToken(token string) (PortalClaims, error) {
	return parsePortal(token, PortalRefreshTokenType)
}

// portalCookieSecure: Secure in release (the portal is served over HTTPS
// behind Caddy) or whenever HTTPS_ENABLED is set.
func portalCookieSecure() bool {
	cfg := config.Get()
	return cfg.IsRelease() || cfg.HTTPSEnabled
}

// SetPortalRefreshCookie sets the httpOnly portal refresh cookie.
func SetPortalRefreshCookie(c *gin.Context, refreshToken string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(PortalRefreshCookie, refreshToken, int(PortalRefreshTTL.Seconds()), PortalRefreshCookiePath, "", portalCookieSecure(), true)
}

// ClearPortalRefreshCookie removes the portal refresh cookie, with the same
// attributes it was set with so the browser deletes it.
func ClearPortalRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(PortalRefreshCookie, "", -1, PortalRefreshCookiePath, "", portalCookieSecure(), true)
}
