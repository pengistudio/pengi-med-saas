package user_handlers

import (
	"errors"
	"net/http"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
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
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrInvalidRequest)
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
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrInvalidRequest)
	}

	// 1) Buscar usuario
	var foundUser user_models.User
	if err := h.db.Omit("password").Where("user_name = ?", user.UserName).First(&foundUser).Error; err != nil {
		h.logger.Error("Failed to find user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "auth.invalid_credentials", core_errors.ErrAuthInvalidCredentials)
	}

	foundUser.Password = user.Password

	// 2) Validar credenciales
	if err := foundUser.ValidateCredentials(h.db); err != nil {
		h.logger.Warn("Failed login attempt", zap.String("username", user.UserName), zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "auth.invalid_credentials", core_errors.ErrAuthInvalidCredentials)
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
		return envelope.ErrorResponse(http.StatusUnauthorized, err.Error(), core_errors.ErrInvalidRequest)
	}

	// Extraer información del token
	userID, ok := claims["userId"].(float64)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid user ID in token", core_errors.ErrInvalidRequest)
	}

	username, ok := claims["username"].(string)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid username in token", core_errors.ErrInvalidRequest)
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

// SignUpWithCompanyToken registers a new user using a company signup token.
// It validates the token, creates the user, and links them to the company
// via an Environment record.
func (h *UserHandler) SignUpWithCompanyToken(c *gin.Context) envelope.Response {
	var req user_dto.CompanySignupDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid company signup request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrInvalidRequest)
	}

	// 1) Validate the signup token
	companyID, roleID, err := auth.ParseCompanySignupToken(req.Token)
	if err != nil {
		h.logger.Warn("Invalid company signup token", zap.Error(err))
		return envelope.ErrorResponse(http.StatusUnauthorized, "Invalid or expired signup token", core_errors.ErrAuthInvalidSignupToken)
	}

	// 2) Verify the company exists
	var company company_models.Company
	if err := h.db.First(&company, companyID).Error; err != nil {
		h.logger.Error("Company not found for signup", zap.Uint("company_id", companyID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Company not found", core_errors.ErrCompanyNotFound)
	}

	// 3) Check max_users plan limit
	var envCount int64
	h.db.Model(&user_models.Environment{}).Where("company_id = ?", company.ID).Count(&envCount)
	if subscription_middleware.ExceedsPlanLimit(h.db, company.ID, "max_users", envCount) {
		return envelope.ErrorResponse(http.StatusForbidden, "plan.limit.users", core_errors.ErrPlanLimitUsers)
	}

	// 4) Check if username already exists
	var existingUser user_models.User
	if err := h.db.Where("user_name = ?", req.UserName).First(&existingUser).Error; err == nil {
		return envelope.ErrorResponse(http.StatusConflict, "Username already exists", core_errors.ErrAuthUserCreateError)
	}

	// 5) Resolve the role: use token's role_id if set, otherwise fall back to admin
	var defaultRole user_models.Role
	if roleID != 0 {
		if err := h.db.First(&defaultRole, roleID).Error; err != nil {
			h.logger.Warn("Role from token not found, falling back to admin", zap.Uint("role_id", roleID))
			roleID = 0
		}
	}
	if roleID == 0 {
		if err := h.db.Where("role = ?", "admin").First(&defaultRole).Error; err != nil {
			if err := h.db.First(&defaultRole).Error; err != nil {
				h.logger.Error("No roles available", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "No roles configured", core_errors.ErrInternal)
			}
		}
	}

	// 6) Create user + environment in a transaction
	var newUser user_models.User
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		newUser = user_models.User{
			UserName: req.UserName,
			Email:    req.Email,
			Password: req.Password,
		}
		if err := newUser.Save(tx); err != nil {
			return err
		}

		env := user_models.Environment{
			UserID:    newUser.ID,
			Name:      req.Name,
			RoleID:    defaultRole.ID,
			CompanyID: company.ID,
		}
		if err := tx.Create(&env).Error; err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		h.logger.Error("Failed to create user with company token", zap.Error(txErr))
		return envelope.ErrorResponse(http.StatusInternalServerError, txErr.Error(), core_errors.ErrAuthUserCreateError)
	}

	h.logger.Info("User registered via company signup token",
		zap.String("username", newUser.UserName),
		zap.Uint("company_id", companyID),
	)
	return envelope.SuccessResponse(gin.H{
		"user_id":  newUser.ID,
		"username": newUser.UserName,
		"email":    newUser.Email,
	}, "user.company_signup.success")
}
