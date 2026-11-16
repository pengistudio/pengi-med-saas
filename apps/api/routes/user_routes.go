package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	core_middleware "pengi-med-saas/core/middleware"
	user_handlers "pengi-med-saas/features/users/handlers"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB) {
	// AuthMiddleware (every route group) rejects tokens of deleted users.
	auth_middleware.UseUserDB(db)

	userHandler := user_handlers.NewUserHandler(db, logger.Log)
	environmentHandler := user_handlers.NewEnvironmentHandler(db, logger.Log)
	profileHandler := user_handlers.NewProfileHandler(db, logger.Log)

	// 15 requests/min, burst 15 — for sensitive auth endpoints
	authLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/15), 15)

	userRoutes := router.Group("/users")
	{
		userRoutes.GET("/environments", auth_middleware.ExchangeAuthMiddleware(), envelope.Handle(environmentHandler.GetEnvironmentsFromUser))
		// Same list, for an already-signed-in session: the web app re-reads its
		// environment (permissions, enabled features) from here on bootstrap.
		userRoutes.GET("/me/environments", auth_middleware.AuthMiddleware(), envelope.Handle(environmentHandler.GetEnvironmentsFromUser))
		userRoutes.POST("/change-password", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.ChangePassword))
		userRoutes.GET("/profile", auth_middleware.AuthMiddleware(), envelope.Handle(profileHandler.GetProfile))
		userRoutes.PUT("/profile", auth_middleware.AuthMiddleware(), envelope.Handle(profileHandler.UpdateProfile))

		// Two-factor authentication of the signed-in user. Wrong password or
		// code answers 422 (never 401, which the web client reads as an
		// expired session).
		userRoutes.GET("/me/2fa", auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.GetTwoFactorStatus))
		userRoutes.POST("/me/2fa/setup", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.SetupTwoFactor))
		userRoutes.POST("/me/2fa/enable", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.EnableTwoFactor))
		userRoutes.POST("/me/2fa/disable", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.DisableTwoFactor))
		userRoutes.POST("/me/2fa/recovery-codes", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.RegenerateRecoveryCodes))

		// Passkeys of the signed-in user. Adding one asks for the password (and
		// the 2FA code), since signing in with a passkey skips 2FA.
		userRoutes.GET("/me/passkeys", auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.ListPasskeys))
		userRoutes.POST("/me/passkeys/register/begin", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.BeginPasskeyRegistration))
		userRoutes.POST("/me/passkeys/register/finish", authLimiter.Middleware(), auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.FinishPasskeyRegistration))
		userRoutes.DELETE("/me/passkeys/:id", auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.DeletePasskey))
	}

	authRoutes := router.Group("/auth", authLimiter.Middleware())
	{
		authRoutes.POST("/register", envelope.Handle(userHandler.Register))
		authRoutes.GET("/verify-email", envelope.Handle(userHandler.VerifyEmail))
		authRoutes.POST("/signup/company", envelope.Handle(userHandler.SignUpWithCompanyToken))
		authRoutes.GET("/signup/company/check-email", envelope.Handle(userHandler.CheckCompanySignupEmail))
		authRoutes.POST("/signup/company/join", envelope.Handle(userHandler.JoinCompanyWithExistingAccount))
		authRoutes.POST("/login", envelope.Handle(userHandler.Login))
		authRoutes.POST("/login/2fa", envelope.Handle(userHandler.LoginTwoFactor))
		authRoutes.POST("/passkey/begin", envelope.Handle(userHandler.BeginPasskeyLogin))
		authRoutes.POST("/passkey/login", envelope.Handle(userHandler.FinishPasskeyLogin))
		authRoutes.POST("/refresh", envelope.Handle(userHandler.RefreshAuthToken))
		authRoutes.POST("/logout", envelope.Handle(userHandler.Logout))
		authRoutes.POST("/forgot-password", envelope.Handle(userHandler.ForgotPassword))
		authRoutes.POST("/reset-password", envelope.Handle(userHandler.ResetPassword))
	}

}
