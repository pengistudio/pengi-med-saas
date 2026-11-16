package backoffice_handlers

import (
	"net/http"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	backoffice_dto "pengi-med-saas/features/backoffice/dto"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeUserHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	// logins caps failed sign-ins per admin (the route limits per IP).
	logins *auth.LoginLimiter
	// rotated lets a refresh token replaced moments ago still resolve to its
	// successor, so two tabs refreshing at once don't sign the admin out.
	rotated *rotationGrace
	// lockout caps wrong second-factor codes and current-password checks per
	// admin, on the row (holds across instances and restarts).
	lockout adminLockout
	now     func() time.Time
}

func NewBackofficeUserHandler(db *gorm.DB, logger *zap.Logger) *BackofficeUserHandler {
	return &BackofficeUserHandler{
		db:      tenantdb.System(db),
		logger:  logger,
		logins:  auth.NewDefaultLoginLimiter(),
		rotated: newRotationGrace(),
		lockout: newAdminLockout(),
		now:     time.Now,
	}
}

// seedPassword is the password the committed seed gave admin/admin2. The
// migration that replaces it runs only in release, so any other environment
// may still have it: such an account can sign in only in local development.
const seedPassword = "password"

// localDevelopment is GIN_MODE=development (.env.example) or debug (the dev
// compose file).
func localDevelopment() bool {
	mode := config.Get().GinMode
	return mode == "development" || mode == "debug"
}

func passwordTooShort() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "auth.password.too_short", core_errors.ErrAuthPasswordTooShort)
}

func (h *BackofficeUserHandler) GetUsers(c *gin.Context) envelope.Response {
	users := []backoffice_models.BackofficeUser{}
	if err := h.db.Find(&users).Error; err != nil {
		h.logger.Error("Failed to fetch backoffice users", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	var counts []struct {
		BackofficeUserID uint
		N                int64
	}
	if err := h.db.Model(&backoffice_models.BackofficePasskey{}).
		Select("backoffice_user_id, COUNT(*) AS n").Group("backoffice_user_id").Scan(&counts).Error; err != nil {
		h.logger.Error("Failed to count backoffice passkeys", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	byUser := map[uint]int64{}
	for _, row := range counts {
		byUser[row.BackofficeUserID] = row.N
	}
	for i := range users {
		users[i].PasskeyCount = byUser[users[i].ID]
	}

	h.logger.Info("Backoffice users fetched successfully", zap.Int("count", len(users)))
	return envelope.SuccessResponse(users, "backoffice.users.list.success")
}

func (h *BackofficeUserHandler) GetUserByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, id).Error; err != nil {
		h.logger.Error("Backoffice user not found", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
	}
	h.db.Model(&backoffice_models.BackofficePasskey{}).Where("backoffice_user_id = ?", user.ID).Count(&user.PasskeyCount)
	return envelope.SuccessResponse(user, "backoffice.users.found")
}

// CreateBackofficeUserRequest is what an admin may set on a new admin. The
// model can't be bound directly: its Password is json:"-" (it would be dropped)
// and ID/RefreshToken must not come from the client.
type CreateBackofficeUserRequest struct {
	Name     string `json:"name"`
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password" binding:"required"`
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
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
	}

	var req UpdateBackofficeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.UserName != "" {
		updates["user_name"] = req.UserName
	}
	if req.Password != "" {
		userName := user.UserName
		if req.UserName != "" {
			userName = req.UserName
		}
		name := user.Name
		if req.Name != "" {
			name = req.Name
		}
		if errResp := weakAdminPassword(req.Password, userName, name); errResp != nil {
			return *errResp
		}
		hashed, err := auth.HashPassword(req.Password)
		if err != nil {
			h.logger.Error("Failed to hash password", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		updates["password"] = hashed
		// End existing sessions: the stored refresh token no longer matches
		// any cookie, so every refresh issued before the change is rejected.
		updates["refresh_token"] = ""
		updates["refresh_family"] = ""
	}

	if len(updates) > 0 {
		if err := h.db.Model(&user).Updates(updates).Error; err != nil {
			h.logger.Error("Failed to update backoffice user", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
	}

	h.db.First(&user, user.ID)
	return envelope.SuccessResponse(user, "backoffice.users.update.success")
}

func (h *BackofficeUserHandler) DeleteUser(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var user backoffice_models.BackofficeUser
	if err := h.db.First(&user, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrUserNotFound)
	}
	if err := h.db.Delete(&user).Error; err != nil {
		h.logger.Error("Failed to delete backoffice user", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(nil, "backoffice.users.delete.success")
}

// CreateUser creates a backoffice admin. Only reachable by an authenticated
// admin (POST /backoffice/users).
func (h *BackofficeUserHandler) CreateUser(c *gin.Context) envelope.Response {
	var req CreateBackofficeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create backoffice user request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrBackofficeInvalidRequest)
	}
	if errResp := weakAdminPassword(req.Password, req.UserName, req.Name); errResp != nil {
		return *errResp
	}
	user := backoffice_models.BackofficeUser{Name: req.Name, UserName: req.UserName, Password: req.Password}
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

	invalid := envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidCredentials)
	if h.logins.Blocked(user.UserName) {
		h.logger.Warn("Backoffice login blocked: too many failed attempts", zap.String("username", user.UserName))
		return envelope.ErrorResponse(http.StatusTooManyRequests, "error.rate_limit_exceeded", core_errors.ErrRateLimitExceeded)
	}

	var foundUser backoffice_models.BackofficeUser
	if err := h.db.Omit("password").Where("user_name = ?", user.UserName).First(&foundUser).Error; err != nil {
		h.logins.Failed(user.UserName)
		h.logger.Error("Failed to find backoffice user", zap.Error(err))
		return invalid
	}

	foundUser.Password = user.Password

	if err := foundUser.ValidateCredentials(h.db); err != nil {
		h.logins.Failed(user.UserName)
		h.logger.Warn("Failed login attempt", zap.String("username", user.UserName), zap.Error(err))
		return invalid
	}

	// The credentials matched, so the stored hash is the seed's "password".
	if user.Password == seedPassword && !localDevelopment() {
		h.logins.Failed(user.UserName)
		h.logger.Warn("Backoffice login refused: account still has the seed default password; set a new one",
			zap.String("username", foundUser.UserName))
		return invalid
	}
	h.logins.Succeeded(user.UserName)

	if foundUser.LockedUntil != nil && foundUser.LockedUntil.After(h.now()) {
		h.logger.Warn("Backoffice login refused: account locked", zap.Uint("user_id", foundUser.ID))
		return accountLockedResponse()
	}

	// No session from the password alone: the admin either proves a second
	// factor (login/2fa or a passkey) or, without one, enrolls one now.
	return h.secondFactorChallenge(foundUser)
}

// secondFactorChallenge answers a correct password: an mfa token for
// /backoffice/auth/login/2fa when the admin has a second factor, else an
// enrollment token for /backoffice/auth/enroll/*. Neither is a session.
func (h *BackofficeUserHandler) secondFactorChallenge(user backoffice_models.BackofficeUser) envelope.Response {
	var passkeys int64
	if err := h.db.Model(&backoffice_models.BackofficePasskey{}).
		Where("backoffice_user_id = ?", user.ID).Count(&passkeys).Error; err != nil {
		h.logger.Error("Backoffice login: failed to count passkeys", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if user.TwoFactorEnabled || passkeys > 0 {
		token, err := auth.GenerateBackofficeMFAToken(auth.BackofficeMFAPurposeLogin, user.ID, user.Password, h.now())
		if err != nil {
			h.logger.Error("Failed to generate backoffice mfa token", zap.Uint("user_id", user.ID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
		}
		h.logger.Info("Backoffice password accepted, second factor required", zap.Uint("user_id", user.ID))
		return envelope.SuccessResponse(gin.H{
			"two_factor_required": true,
			"mfa_token":           token,
			"totp":                user.TwoFactorEnabled,
			"passkey":             passkeys > 0,
		}, "auth.login.two_factor_required")
	}

	token, err := auth.GenerateBackofficeMFAToken(auth.BackofficeMFAPurposeEnroll, user.ID, user.Password, h.now())
	if err != nil {
		h.logger.Error("Failed to generate backoffice enrollment token", zap.Uint("user_id", user.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}
	h.logger.Info("Backoffice password accepted, second factor enrollment required", zap.Uint("user_id", user.ID))
	return envelope.SuccessResponse(gin.H{
		"enrollment_required": true,
		"enrollment_token":    token,
		"passkeys_available":  passkeysAvailable(),
	}, "backoffice.auth.enrollment_required")
}

// issueSession opens a backoffice session for user: an access token with
// auth_time = now, and a refresh token that starts a new family (stored, and
// set as the refresh cookie). Only called after a second factor was proven.
func (h *BackofficeUserHandler) issueSession(c *gin.Context, user backoffice_models.BackofficeUser) (gin.H, *envelope.Response) {
	fail := func(err error) (gin.H, *envelope.Response) {
		h.logger.Error("Failed to open backoffice session", zap.Uint("user_id", user.ID), zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
		return nil, &resp
	}
	token, err := auth.GenerateBackofficeSessionToken(user.UserName, int64(user.ID), h.now())
	if err != nil {
		return fail(err)
	}
	sid := uuid.NewString()
	refreshToken, err := auth.GenerateBackofficeRefreshToken(user.UserName, int64(user.ID), sid)
	if err != nil {
		return fail(err)
	}
	if err := h.db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", user.ID).
		UpdateColumns(map[string]any{"refresh_token": refreshToken, "refresh_family": sid, "updated_at": h.now()}).Error; err != nil {
		return fail(err)
	}
	auth.SetBackofficeRefreshCookie(c, refreshToken)
	return gin.H{"token": token, "user_id": user.ID}, nil
}

// RefreshAuthToken issues a new access token for the backoffice refresh
// cookie and rotates the refresh token. The refresh token must be a backoffice
// one and still be the one stored for that admin (a newer login replaces it).
// A token rotated out in the last few seconds resolves to its successor
// (concurrent refreshes from two tabs); later it is rejected as stale.
//
// Reuse: a rotated-out token of the current login (same family, sid) presented
// after the grace window means a copy of it is in someone else's hands, so the
// session is revoked (stored token cleared) and the cookie removed, as the
// clinic app revokes the family. A token of an older login is just stale.
// An admin without a second factor (never enrolled, or reset by another
// admin) gets no session from refresh either.
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
	if ok, err := backoffice_models.HasSecondFactor(h.db, user.ID); err != nil || !ok {
		h.logger.Warn("Backoffice refresh refused: admin has no second factor", zap.Uint("user_id", user.ID))
		h.revokeSession(user.ID, "")
		auth.ClearBackofficeRefreshCookie(c)
		return invalid
	}

	var next string
	if user.RefreshToken == refreshToken {
		sid := user.RefreshFamily
		if sid == "" {
			sid = uuid.NewString()
		}
		rotated, err := auth.GenerateBackofficeRefreshToken(user.UserName, int64(user.ID), sid)
		if err != nil {
			h.logger.Error("Failed to generate refresh token during refresh", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
		}
		// Only if it is still the stored token: a concurrent refresh that got
		// here first wins, and this one falls back to the grace lookup below.
		res := h.db.Model(&backoffice_models.BackofficeUser{}).
			Where("id = ? AND refresh_token = ?", user.ID, refreshToken).
			UpdateColumns(map[string]any{"refresh_token": rotated, "refresh_family": sid})
		if res.Error != nil {
			h.logger.Error("Failed to rotate backoffice refresh token", zap.Error(res.Error))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
		}
		if res.RowsAffected == 1 {
			h.rotated.remember(refreshToken, rotated)
			next = rotated
		}
	}
	if next == "" {
		// Replaced moments ago by a concurrent refresh, and that successor is
		// still the current token: hand it out again.
		successor, ok := h.rotated.successor(refreshToken)
		if !ok || !h.isCurrentRefreshToken(user.ID, successor) {
			if claims.SID != "" && claims.SID == user.RefreshFamily {
				h.revokeSession(user.ID, claims.SID)
				auth.ClearBackofficeRefreshCookie(c)
				h.logger.Warn("Backoffice refresh token reuse detected, session revoked",
					zap.Uint("user_id", user.ID), zap.String("ip", c.ClientIP()), zap.String("user_agent", c.Request.UserAgent()))
				return envelope.ErrorResponse(http.StatusUnauthorized, "auth.refresh.reused", core_errors.ErrAuthRefreshTokenReused)
			}
			h.logger.Warn("Backoffice refresh token is not the current one", zap.String("username", user.UserName))
			return invalid
		}
		next = successor
	}

	token, err := auth.GenerateBackofficeToken(user.UserName, int64(user.ID))
	if err != nil {
		h.logger.Error("Failed to generate token during refresh", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrAuthTokenGenerateError)
	}
	auth.SetBackofficeRefreshCookie(c, next)
	return envelope.SuccessResponse(gin.H{"token": token, "user_id": user.ID}, "auth.token.refresh.success")
}

func (h *BackofficeUserHandler) isCurrentRefreshToken(userID uint, token string) bool {
	var n int64
	h.db.Model(&backoffice_models.BackofficeUser{}).
		Where("id = ? AND refresh_token = ?", userID, token).Count(&n)
	return n == 1
}

// revokeSession clears admin userID's stored refresh token, so no refresh
// cookie works any more. With sid set, only if that login is still the
// current one (a newer login is left alone).
func (h *BackofficeUserHandler) revokeSession(userID uint, sid string) {
	q := h.db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", userID)
	if sid != "" {
		q = q.Where("refresh_family = ?", sid)
	}
	if err := q.UpdateColumns(map[string]any{"refresh_token": "", "refresh_family": ""}).Error; err != nil {
		h.logger.Error("Failed to revoke backoffice session", zap.Uint("user_id", userID), zap.Error(err))
	}
}

// Logout ends the admin's session: the stored refresh token is cleared, so the
// refresh cookie stops working, and the cookie is removed from the browser.
func (h *BackofficeUserHandler) Logout(c *gin.Context) envelope.Response {
	if refreshToken, err := c.Cookie(auth.BackofficeRefreshCookie); err == nil && refreshToken != "" {
		if claims, err := auth.ParseBackofficeRefreshToken(refreshToken); err == nil {
			if err := h.db.Model(&backoffice_models.BackofficeUser{}).
				Where("id = ? AND refresh_token = ?", claims.UserID, refreshToken).
				UpdateColumns(map[string]any{"refresh_token": "", "refresh_family": ""}).Error; err != nil {
				h.logger.Error("Failed to revoke backoffice refresh token", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
			}
		}
	}
	auth.ClearBackofficeRefreshCookie(c)
	return envelope.SuccessResponse(nil, "auth.logout.success")
}
