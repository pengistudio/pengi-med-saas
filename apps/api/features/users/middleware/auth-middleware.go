package auth_middleware

import (
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Verificar que el header Authorization esté presente
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Authorization header missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		// 2) Verificar que tenga el formato "Bearer {token}"
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid authorization header format", core_errors.ErrAuthInvalidRequest))
			return
		}

		// 3) Extraer el token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Token is empty", core_errors.ErrAuthInvalidRequest))
			return
		}

		// 4) Validar el token
		claims, err := auth.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid or expired token", core_errors.ErrAuthInvalidRequest))
			return
		}

		// Extract info
		// In jwt.go, ParseToken returns map[string]interface{}.
		// We expect "userId" (float64) and "username" (string).
		userID, ok := claims["userId"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid token payload: userId missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid token payload: username missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		// 6) Agregar información del usuario al contexto para usar en handlers
		c.Set("user_id", int64(userID))
		c.Set("username", username)
		c.Set("auth_token", token)
		c.Set("authenticated", true)

		// 7) Continuar con el siguiente middleware/handler
		c.Next()
	}
}

func ExchangeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 1) First, try to get from query string
		token = c.Query("exchange_token")

		// 2) If not in query, try to get from JSON body
		if token == "" && c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			var body struct {
				ExchangeToken string `json:"exchange_token"`
			}
			// ShouldBindBodyWith allows reading the body multiple times if needed by other handlers
			if err := c.ShouldBindJSON(&body); err == nil {
				token = body.ExchangeToken
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Exchange token missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		claims, err := auth.ParseExchangeToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid or expired exchange token", core_errors.ErrAuthInvalidRequest))
			return
		}

		userID, ok := claims["userId"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid token payload: userId missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "Invalid token payload: username missing", core_errors.ErrAuthInvalidRequest))
			return
		}

		c.Set("user_id", int64(userID))
		c.Set("username", username)
		c.Set("auth_token", token)
		c.Set("authenticated", true)

		c.Next()
	}
}

// OptionalAuthMiddleware es similar a AuthMiddleware pero no bloquea si no hay token
// Útil para endpoints que pueden funcionar con o sin autenticación
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Si no hay header o no tiene formato correcto, continuar sin autenticación
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("authenticated", false)
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.Set("authenticated", false)
			c.Next()
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			// Token inválido, tratamos como no autenticado
			c.Set("authenticated", false)
			c.Next()
			return
		}

		userID, ok1 := claims["userId"].(float64)
		username, ok2 := claims["username"].(string)

		if ok1 && ok2 {
			c.Set("user_id", int64(userID))
			c.Set("username", username)
			c.Set("auth_token", token)
			c.Set("authenticated", true)
		} else {
			c.Set("authenticated", false)
		}

		c.Next()
	}
}

// GetUserFromContext es una función helper para obtener la info del usuario desde el contexto
func GetUserFromContext(c *gin.Context) (userID int64, username string, exists bool) {
	userIDVal, userIDExists := c.Get("user_id")
	usernameVal, usernameExists := c.Get("username")

	if !userIDExists || !usernameExists {
		return 0, "", false
	}

	userID, ok1 := userIDVal.(int64)
	username, ok2 := usernameVal.(string)

	if !ok1 || !ok2 {
		return 0, "", false
	}

	return userID, username, true
}

// GetAuthTokenFromContext obtiene el token de autenticación desde el contexto
func GetAuthTokenFromContext(c *gin.Context) (string, bool) {
	tokenVal, exists := c.Get("auth_token")
	if !exists {
		return "", false
	}

	token, ok := tokenVal.(string)
	return token, ok
}

// IsAuthenticatedFromContext verifica si el usuario está autenticado desde el contexto
func IsAuthenticatedFromContext(c *gin.Context) bool {
	authVal, exists := c.Get("authenticated")
	if !exists {
		// Si no existe la clave "authenticated", verificar si hay user_id
		_, _, userExists := GetUserFromContext(c)
		return userExists
	}

	authenticated, ok := authVal.(bool)
	return ok && authenticated
}
