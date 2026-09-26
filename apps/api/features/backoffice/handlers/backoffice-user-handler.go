package backoffice_handlers

import (
	"errors"
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	backoffice_dto "pengi-med-saas/features/backoffice/dto"
	backoffice_models "pengi-med-saas/features/backoffice/models"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeUserHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeUserHandler(db *gorm.DB, logger *zap.Logger) *BackofficeUserHandler {
	return &BackofficeUserHandler{
		db:     tenantdb.System(db),
		logger: logger,
	}
}

func (h *BackofficeUserHandler) GetUsers(c *gin.Context) envelope.Response {
	users := []backoffice_models.BackofficeUser{}
	if err := h.db.Find(&users).Error; err != nil {
		h.logger.Error("Failed to fetch backoffice users", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining backoffice users", core_errors.ErrUserNotFound)
	}

	h.logger.Info("Backoffice users fetched successfully", zap.Int("count", len(users)))
	return envelope.SuccessResponse(users, "backoffice.users.list.success")
}

func (h *BackofficeUserHandler) GetUserByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, id).Error; err != nil {
		h.logger.Error("Backoffice user not found", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "User not found", core_errors.ErrUserNotFound)
	}
	return envelope.SuccessResponse(user, "backoffice.users.found")
}

type UpdateBackofficeUserRequest struct {
	Name     string `json:"name"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

func (h *BackofficeUserHandler) UpdateUser(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "User not found", core_errors.ErrUserNotFound)
	}

	var req UpdateBackofficeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.UserName != "" {
		updates["user_name"] = req.UserName
	}
	if req.Password != "" {
		hashed, err := auth.HashPassword(req.Password)
		if err != nil {
			h.logger.Error("Failed to hash password", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating user", core_errors.ErrInternal)
		}
		updates["password"] = hashed
	}

	if len(updates) > 0 {
		if err := h.db.Model(&user).Updates(updates).Error; err != nil {
			h.logger.Error("Failed to update backoffice user", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating user", core_errors.ErrInternal)
		}
	}

	h.db.First(&user, user.ID)
	return envelope.SuccessResponse(user, "backoffice.users.update.success")
}

func (h *BackofficeUserHandler) DeleteUser(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "User not found", core_errors.ErrUserNotFound)
	}
	if err := h.db.Delete(&user).Error; err != nil {
		h.logger.Error("Failed to delete backoffice user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting user", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(nil, "backoffice.users.delete.success")
}

// CreateUser creates a backoffice admin. Only reachable by an authenticated
// admin (POST /backoffice/users).
func (h *BackofficeUserHandler) CreateUser(c *gin.Context) envelope.Response {
	var user backoffice_models.BackofficeUser
	if err := c.ShouldBind(&user); err != nil {
		h.logger.Error("Invalid create backoffice user request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}
	if err := user.Save(h.db); err != nil {
		h.logger.Error("Failed to create backoffice user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthUserCreateError)
	}
	return envelope.SuccessResponse(user, "backoffice_user.create.success")
}

func (h *BackofficeUserHandler) Login(c *gin.Context) envelope.Response {
	var user backoffice_dto.LoginDTO
	if err := c.ShouldBind(&user); err != nil {
		h.logger.Error("Invalid login request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}

	var foundUser backoffice_models.BackofficeUser
	if err := h.db.Omit("password").Where("user_name = ?", user.UserName).First(&foundUser).Error; err != nil {
		h.logger.Error("Failed to find backoffice user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidCredentials)
	}

	foundUser.Password = user.Password

	if err := foundUser.ValidateCredentials(h.db); err != nil {
		h.logger.Warn("Failed login attempt", zap.String("username", user.UserName), zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidCredentials)
	}

	token, err := auth.GenerateBackofficeToken(foundUser.UserName, int64(foundUser.ID))
	if err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}

	refreshToken, err := auth.GenerateBackofficeRefreshToken(foundUser.UserName, int64(foundUser.ID))
	if err != nil {
		h.logger.Error("Failed to generate refresh token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}

	if err := foundUser.UpdateRefreshToken(h.db, refreshToken); err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}

	auth.SetBackofficeRefreshCookie(c, refreshToken)

	h.logger.Info("Backoffice user logged in successfully", zap.String("username", foundUser.UserName))
	return envelope.SuccessResponse(gin.H{
		"token":   token,
		"user_id": foundUser.ID,
	}, "login.successful")
}

// RefreshAuthToken issues a new access token for the backoffice refresh
// cookie. The refresh token must be a backoffice one and still be the one
// stored for that admin (a newer login replaces it).
func (h *BackofficeUserHandler) RefreshAuthToken(c *gin.Context) envelope.Response {
	invalid := envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidRefreshToken)
	refreshToken, err := c.Cookie(auth.BackofficeRefreshCookie)
	if err != nil || refreshToken == "" {
		return invalid
	}
	claims, err := auth.ParseBackofficeRefreshToken(refreshToken)
	if err != nil {
		return invalid
	}
	var user backoffice_models.BackofficeUser
	if err := h.db.Where("id = ? AND user_name = ?", claims.UserID, claims.Username).First(&user).Error; err != nil {
		return invalid
	}
	if user.RefreshToken != refreshToken {
		h.logger.Warn("Backoffice refresh token is not the current one", zap.String("username", user.UserName))
		return invalid
	}
	token, err := auth.GenerateBackofficeToken(user.UserName, int64(user.ID))
	if err != nil {
		h.logger.Error("Failed to generate token during refresh", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}
	return envelope.SuccessResponse(gin.H{"token": token, "user_id": user.ID}, "auth.token.refresh.success")
}

func (h *BackofficeUserHandler) ExtendSession(c *gin.Context) envelope.Response {
	userId := c.GetInt64("user_id")
	var user backoffice_models.BackofficeUser
	if err := h.db.Model(&backoffice_models.BackofficeUser{}).First(&user, userId).Error; err != nil {
		h.logger.Error("Failed to find backoffice user for session extension", zap.Int64("userId", userId), zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrAuthUserInvalidID)
	}
	token, err := auth.GenerateBackofficeToken(user.UserName, int64(user.ID))
	if err != nil {
		h.logger.Error("Failed to generate token for session extension", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}

	h.logger.Info("Session extended successfully", zap.String("username", user.UserName))
	return envelope.SuccessResponse(gin.H{"token": token, "user_id": user.ID}, "auth.session.extend.success")
}

func (h *BackofficeUserHandler) ValidateBearerToken(c *gin.Context) envelope.Response {
	claims, token, err := ExtractAndValidateBearerToken(c)
	if err != nil {
		h.logger.Warn("Bearer token validation failed", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrBackofficeInvalidRequest)
	}

	userID, ok := claims["userId"].(float64)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid user ID in token", core_errors.ErrBackofficeInvalidRequest)
	}

	username, ok := claims["username"].(string)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid username in token", core_errors.ErrBackofficeInvalidRequest)
	}

	return envelope.SuccessResponse(gin.H{
		"valid":    true,
		"user_id":  int64(userID),
		"username": username,
		"token":    token,
		"message":  "Token is valid",
	}, "auth.token.valid")
}

func ExtractAndValidateBearerToken(c *gin.Context) (map[string]interface{}, string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, "", errors.New("authorization header missing")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, "", errors.New("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, "", errors.New("token is empty")
	}

	if _, err := auth.ParseBackofficeAccessToken(token); err != nil {
		return nil, "", err
	}
	claims, err := auth.ParseToken(token)
	if err != nil {
		return nil, "", err
	}

	return claims, token, nil
}
