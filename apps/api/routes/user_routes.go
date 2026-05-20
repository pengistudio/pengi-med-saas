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
	userHandler := user_handlers.NewUserHandler(db, logger.Log)
	environmentHandler := user_handlers.NewEnvironmentHandler(db, logger.Log)
	profileHandler := user_handlers.NewProfileHandler(db, logger.Log)

	// 15 requests/min, burst 15 — for sensitive auth endpoints
	authLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/15), 15)

	userRoutes := router.Group("/users")
	{
		userRoutes.GET("", envelope.Handle(userHandler.GetUsers))
		userRoutes.GET("/environments", auth_middleware.ExchangeAuthMiddleware(), envelope.Handle(environmentHandler.GetEnvironmentsFromUser))
		userRoutes.GET("/profile", auth_middleware.AuthMiddleware(), envelope.Handle(profileHandler.GetProfile))
		userRoutes.PUT("/profile", auth_middleware.AuthMiddleware(), envelope.Handle(profileHandler.UpdateProfile))
	}

	authRoutes := router.Group("/auth", authLimiter.Middleware())
	{
		authRoutes.POST("/register", envelope.Handle(userHandler.Register))
		authRoutes.GET("/verify-email", envelope.Handle(userHandler.VerifyEmail))
		authRoutes.POST("/signup", envelope.Handle(userHandler.SignUp))
		authRoutes.POST("/signup/company", envelope.Handle(userHandler.SignUpWithCompanyToken))
		authRoutes.POST("/login", envelope.Handle(userHandler.Login))
		authRoutes.POST("/refresh", envelope.Handle(userHandler.RefreshAuthToken))
		authRoutes.POST("/extend", auth_middleware.AuthMiddleware(), envelope.Handle(userHandler.ExtendSession))
		authRoutes.POST("/validate", envelope.Handle(userHandler.ValidateBearerToken))
		authRoutes.POST("/reset-password", envelope.Handle(userHandler.ResetPassword))
	}

}
