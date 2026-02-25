package user_handlers

import (
	"errors"
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserHandler(db *gorm.DB, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
	}
}

func (h *UserHandler) GetUsers(c *gin.Context) envelope.Response {
	users := []user_models.User{}
	if err := h.db.Find(&users).Error; err != nil {
		h.logger.Error("Failed to fetch users", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining users", core_errors.ErrUserNotFound)
	}

	h.logger.Info("Users fetched successfully", zap.Int("count", len(users)))
	return envelope.SuccessResponse(users, "user.list.success")
}

func (h *UserHandler) SignUp(c *gin.Context) envelope.Response {
	var user user_models.User
	if err := c.ShouldBind(&user); err != nil {
		h.logger.Error("Invalid signup request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}
	if err := user.Save(h.db); err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthUserCreateError)
	}
	return envelope.SuccessResponse(user, "user.create.success")
}

func (h *UserHandler) Login(c *gin.Context) envelope.Response {
	// 0) Bind
	var user user_dto.LoginDTO
	if err := c.ShouldBind(&user); err != nil {
		h.logger.Error("Invalid login request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// 1) Buscar usuario
	var foundUser user_models.User
	if err := h.db.Omit("password").Where("user_name = ?", user.UserName).First(&foundUser).Error; err != nil {
		h.logger.Error("Failed to find user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, err.Error(), core_errors.ErrAuthInvalidCredentials)
	}

	foundUser.Password = user.Password

	// 2) Validar credenciales
	if err := foundUser.ValidateCredentials(h.db); err != nil {
		h.logger.Warn("Failed login attempt", zap.String("username", user.UserName), zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, err.Error(), core_errors.ErrAuthInvalidCredentials)
	}

	// 3) Generar tokens
	token, err := auth.GenerateToken(foundUser.UserName, int64(foundUser.ID))
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}

	refreshToken, err := auth.GenerateRefreshToken(foundUser.UserName, int64(foundUser.ID))
	if err != nil {
		h.logger.Error("Failed to generate refresh token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}

	// 4) Guardar refresh token (chequear error)
	if err := foundUser.UpdateRefreshToken(h.db, refreshToken); err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}

	exchangeToken, err := auth.GenerateExchangeToken(foundUser.UserName, int64(foundUser.ID))
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}

	// 5) Setear cookie y responder 200 una sola vez
	auth.SetRefreshTokenCookie(refreshToken, c)

	h.logger.Info("User logged in successfully", zap.String("username", foundUser.UserName))
	return envelope.SuccessResponse(gin.H{
		"token":          token,
		"exchange_token": exchangeToken,
		"user_id":        foundUser.ID,
	}, "login.successful")
}

func (h *UserHandler) RefreshAuthToken(c *gin.Context) envelope.Response {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRefreshToken)
	}
	userID, username, err := auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthInvalidRefreshToken)
	}
	token, err := auth.GenerateToken(username, int64(userID))
	if err != nil {
		h.logger.Error("Failed to generate token during refresh", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}
	h.logger.Info("Token refreshed successfully", zap.String("username", username))
	return envelope.SuccessResponse(gin.H{"token": token, "user_id": userID}, "auth.token.refresh.success")
}

func (h *UserHandler) ExtendSession(c *gin.Context) envelope.Response {
	userId := c.GetInt64("userId")
	var user user_models.User
	// Assuming logic matches user snippet: finding user by ID
	if err := h.db.Model(&user_models.User{}).First(&user, userId).Error; err != nil {
		h.logger.Error("Failed to find user for session extension", zap.Int64("userId", userId), zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrAuthUserInvalidID)
	}
	token, err := auth.GenerateToken(user.UserName, int64(user.ID))
	if err != nil {
		h.logger.Error("Failed to generate token for session extension", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrAuthTokenGenerateError)
	}

	h.logger.Info("Session extended successfully", zap.String("username", user.UserName))
	return envelope.SuccessResponse(gin.H{"token": token, "user_id": user.ID}, "auth.session.extend.success")
}

func (h *UserHandler) ValidateBearerToken(c *gin.Context) envelope.Response {
	// Usar helper para extraer y validar el token
	claims, token, err := ExtractAndValidateBearerToken(c)
	if err != nil {
		// ExtractAndValidateBearerToken returns error which we map
		h.logger.Warn("Bearer token validation failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, err.Error(), core_errors.ErrAuthInvalidRequest)
	}

	// Extraer información del token
	userID, ok := claims["userId"].(float64)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid user ID in token", core_errors.ErrAuthInvalidRequest)
	}

	username, ok := claims["username"].(string)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid username in token", core_errors.ErrAuthInvalidRequest)
	}

	// Responder con la información del token validado
	return envelope.SuccessResponse(gin.H{
		"valid":    true,
		"user_id":  int64(userID),
		"username": username,
		"token":    token,
		"message":  "Token is valid",
	}, "auth.token.valid")
}

// ExtractAndValidateBearerToken es una función helper que extrae y valida un Bearer token
// Retorna (claims, token, error)
func ExtractAndValidateBearerToken(c *gin.Context) (map[string]interface{}, string, error) {
	// 1) Extraer el token del header Authorization
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, "", errors.New("authorization header missing")
	}

	// 2) Verificar que empiece con "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, "", errors.New("invalid authorization header format")
	}

	// 3) Extraer el token (quitar "Bearer " del inicio)
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, "", errors.New("token is empty")
	}

	// 4) Validar el token usando ParseToken
	claims, err := auth.ParseToken(token)
	if err != nil {
		return nil, "", err
	}

	return claims, token, nil
}
