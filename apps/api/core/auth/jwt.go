package auth

import (
	"errors"
	"pengi-med-saas/core/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token is expired")
)

func GenerateToken(username string, userId int64) (string, error) {
	secretKey := config.GetEnv("AUTH_KEY")
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
	secretKey := config.GetEnv("AUTH_KEY")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":     "exchange_token",
		"username": username,
		"userId":   userId,
		// Short expiration, e.g., 5 minutes
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

func SetRefreshTokenCookie(refreshToken string, c *gin.Context) {
	https_enabled, err := config.GetBoolEnv("HTTPS_ENABLED")
	if err != nil {
		return
	}
	c.SetCookie(
		"refresh_token", // Nombre de la cookie
		refreshToken,    // Valor de la cookie
		604800,          // Tiempo de vida en segundos (1 hora)
		"/",             // Path
		"",              // Dominio
		https_enabled,   // Habilitar Secure (solo HTTPS)
		true,            // Habilitar HttpOnly
	)
}

func GenerateRefreshToken(username string, userId int64) (string, error) {
	secretKey := config.GetEnv("AUTH_KEY")
	claims := jwt.MapClaims{
		"username": username,
		"userId":   userId,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(), // Expira en 7 días.
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func ValidateRefreshToken(refreshToken string) (uint, string, error) {
	secretKey := config.GetEnv("AUTH_KEY")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil || !token.Valid {
		return 0, "", errors.New("invalid refresh token")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return 0, "", errors.New("refresh token expired")
		}
	} else {
		return 0, "", errors.New("invalid expiration claim")
	}

	userId, ok := claims["userId"].(float64)
	if !ok {
		return 0, "", errors.New("invalid user id")
	}
	username, ok := claims["username"].(string)
	if !ok {
		return 0, "", errors.New("invalid username")
	}

	return uint(userId), username, nil
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
	secretKey := config.GetEnv("AUTH_KEY")
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

// GenerateCompanySignupToken creates a JWT that encodes a company_id.
// It expires in 72 hours and is used to let new users self-register
// under a specific company.
func GenerateCompanySignupToken(companyID uint) (string, error) {
	secretKey := config.GetEnv("AUTH_KEY")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":       "company_signup",
		"company_id": companyID,
		"exp":        time.Now().Add(72 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

// ParseCompanySignupToken validates a company signup token and returns the
// embedded company_id.
func ParseCompanySignupToken(tokenStr string) (uint, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "company_signup" {
		return 0, errors.New("invalid token type, expected company_signup")
	}

	companyIDFloat, ok := claims["company_id"].(float64)
	if !ok {
		return 0, errors.New("invalid company_id in token")
	}

	return uint(companyIDFloat), nil
}
