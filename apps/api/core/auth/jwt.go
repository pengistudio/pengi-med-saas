package auth

import (
	"errors"
	"net/http"
	"pengi-med-saas/core/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token is expired")
)

func GenerateToken(username string, userId int64) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	exp, err := config.GetNumberEnv("AUTH_EXP")
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":     "access_token",
		"username": username,
		"userId":   userId,
		"exp":      time.Now().Add(time.Duration(exp) * time.Minute).Unix(),
	})
	return token.SignedString([]byte(secretKey))

}

func GenerateExchangeToken(username string, userId int64) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":     "exchange_token",
		"username": username,
		"userId":   userId,
		// Short expiration, e.g., 5 minutes
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

const refreshTokenCookieName = "refresh_token"

// refreshTokenMaxAgeSeconds computes the sliding-window cookie lifetime from
// AUTH_REFRESH_EXP (days), defaulting to 30 days if unset/invalid.
func refreshTokenMaxAgeSeconds() int {
	days, err := config.GetNumberEnv("AUTH_REFRESH_EXP")
	if err != nil || days <= 0 {
		days = 30
	}
	return int(days) * 24 * 60 * 60
}

// RefreshTokenTTL exposes the sliding-window duration (AUTH_REFRESH_EXP,
// default 30 days) so callers issuing DB-tracked rotation rows can compute
// the same ExpiresAt the JWT itself carries.
func RefreshTokenTTL() time.Duration {
	return time.Duration(refreshTokenMaxAgeSeconds()) * time.Second
}

// setRefreshTokenCookieAttrs applies the shared cookie attributes so that
// setting and clearing the cookie always agree on Path/Domain/SameSite —
// browsers only delete a cookie if every defining attribute matches.
func setRefreshTokenCookieAttrs(c *gin.Context, value string, maxAge int) {
	httpsEnabled, _ := config.GetBoolEnv("HTTPS_ENABLED")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshTokenCookieName,
		value,
		maxAge,
		"/",
		"",
		httpsEnabled,
		true, // HttpOnly
	)
}

// SetRefreshTokenCookie sets the refresh_token HttpOnly cookie with a
// sliding expiry driven by AUTH_REFRESH_EXP (days).
func SetRefreshTokenCookie(refreshToken string, c *gin.Context) {
	setRefreshTokenCookieAttrs(c, refreshToken, refreshTokenMaxAgeSeconds())
}

// ClearRefreshTokenCookie deletes the refresh_token cookie using the same
// attributes it was set with, on logout or revocation.
func ClearRefreshTokenCookie(c *gin.Context) {
	setRefreshTokenCookieAttrs(c, "", -1)
}

// GenerateRefreshToken creates a refresh-token JWT carrying a familyId
// (stable session identity, survives rotation) and a jti (unique per
// issued token, used to look up its DB row). Expiry is the sliding window
// from AUTH_REFRESH_EXP, reset on every successful rotation.
func GenerateRefreshToken(username string, userId int64, familyID string) (string, string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	jti := uuid.NewString()
	claims := jwt.MapClaims{
		"username": username,
		"userId":   userId,
		"familyId": familyID,
		"jti":      jti,
		"exp":      time.Now().Add(time.Duration(refreshTokenMaxAgeSeconds()) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secretKey))
	return signed, jti, err
}

// RefreshTokenClaims is what ValidateRefreshToken extracts from a presented
// refresh-token JWT. FamilyID/JTI are empty for legacy (pre-rotation)
// tokens issued before this claim existed.
type RefreshTokenClaims struct {
	UserID   uint
	Username string
	FamilyID string
	JTI      string
}

func ValidateRefreshToken(refreshToken string) (RefreshTokenClaims, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil || !token.Valid {
		return RefreshTokenClaims{}, errors.New("invalid refresh token")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return RefreshTokenClaims{}, errors.New("refresh token expired")
		}
	} else {
		return RefreshTokenClaims{}, errors.New("invalid expiration claim")
	}

	userId, ok := claims["userId"].(float64)
	if !ok {
		return RefreshTokenClaims{}, errors.New("invalid user id")
	}
	username, ok := claims["username"].(string)
	if !ok {
		return RefreshTokenClaims{}, errors.New("invalid username")
	}

	// familyId/jti are absent on legacy tokens issued before rotation
	// existed; caller treats that as "upgrade to a new family".
	familyID, _ := claims["familyId"].(string)
	jti, _ := claims["jti"].(string)

	return RefreshTokenClaims{
		UserID:   uint(userId),
		Username: username,
		FamilyID: familyID,
		JTI:      jti,
	}, nil
}

func ValidateCredentials(c *gin.Context) (bool, int64, error) {
	authToken := c.Request.Header.Get("Authorization")
	if authToken == "" {
		return false, -1, ErrInvalidToken
	}
	claims, err := ParseToken(authToken)
	if err != nil {
		return false, -1, err
	}
	userId, ok := claims["userId"].(float64)
	if !ok {
		return false, -1, ErrInvalidToken
	}
	return true, int64(userId), nil
}

func DecryptToken(c *gin.Context) (jwt.MapClaims, error) {
	authToken := c.Request.Header.Get("Authorization")
	if authToken == "" {
		return nil, ErrInvalidToken
	}
	claims, err := ParseToken(authToken)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func ParseToken(token string) (jwt.MapClaims, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secretKey), nil
	})
	if err != nil || parsedToken == nil || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Check if the token is expired
	exp, ok := claims["exp"].(float64)
	if !ok || time.Now().Unix() > int64(exp) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

func ParseExchangeToken(token string) (jwt.MapClaims, error) {
	claims, err := ParseToken(token)
	if err != nil {
		return nil, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "exchange_token" {
		return nil, errors.New("invalid token type, expected exchange_token")
	}

	return claims, nil
}

// GeneratePasswordResetToken creates a JWT for password reset. It expires in
// 24 hours and encodes the user_id of the tenant user that needs to reset.
func GeneratePasswordResetToken(userID uint) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":    "password_reset",
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

// ParsePasswordResetToken validates the token and returns the user_id.
func ParsePasswordResetToken(tokenStr string) (uint, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "password_reset" {
		return 0, errors.New("invalid token type, expected password_reset")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid user_id in token")
	}

	return uint(userIDFloat), nil
}

// GenerateEmailVerificationToken creates a JWT for email verification. Expires in 24h.
func GenerateEmailVerificationToken(userID uint) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":    "email_verification",
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

// ParseEmailVerificationToken validates the token and returns the user_id.
func ParseEmailVerificationToken(tokenStr string) (uint, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "email_verification" {
		return 0, errors.New("invalid token type, expected email_verification")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid user_id in token")
	}

	return uint(userIDFloat), nil
}

// GenerateCompanySignupToken creates a JWT that encodes a company_id and an
// optional role_id. It expires in 72 hours and is used to let new users
// self-register under a specific company with a pre-assigned role.
func GenerateCompanySignupToken(companyID uint, roleID uint) (string, error) {
	secretKey := config.GetEnvWithDefault("AUTH_KEY", "test-secret-key-for-jwt-signing-in-tests-only")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":       "company_signup",
		"company_id": companyID,
		"role_id":    roleID,
		"exp":        time.Now().Add(72 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

// ParseCompanySignupToken validates a company signup token and returns the
// embedded company_id and role_id.
func ParseCompanySignupToken(tokenStr string) (uint, uint, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, 0, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "company_signup" {
		return 0, 0, errors.New("invalid token type, expected company_signup")
	}

	companyIDFloat, ok := claims["company_id"].(float64)
	if !ok {
		return 0, 0, errors.New("invalid company_id in token")
	}

	roleIDFloat, _ := claims["role_id"].(float64)

	return uint(companyIDFloat), uint(roleIDFloat), nil
}
