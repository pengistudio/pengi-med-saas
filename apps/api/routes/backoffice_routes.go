package routes

import (
	"time"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	core_middleware "pengi-med-saas/core/middleware"
	backoffice_handlers "pengi-med-saas/features/backoffice/handlers"
	backoffice_middleware "pengi-med-saas/features/backoffice/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func RegisterBackofficeRoutes(router *gin.RouterGroup, db *gorm.DB) {
	backofficeAuth := backoffice_middleware.BackofficeAuthMiddleware(db)
	backofficeUserHandler := backoffice_handlers.NewBackofficeUserHandler(db, logger.Log)
	backofficeCompanyHandler := backoffice_handlers.NewBackofficeCompanyHandler(db, logger.Log)
	backofficeFeatureHandler := backoffice_handlers.NewBackofficeFeatureHandler(db, logger.Log)
	backofficePlanHandler := backoffice_handlers.NewBackofficePlanHandler(db, logger.Log)
	backofficeSubscriptionHandler := backoffice_handlers.NewBackofficeSubscriptionHandler(db, logger.Log)
	backofficePermissionHandler := backoffice_handlers.NewBackofficePermissionHandler(db, logger.Log)
	backofficeDashboardHandler := backoffice_handlers.NewBackofficeDashboardHandler(db, logger.Log)
	backofficeRoleHandler := backoffice_handlers.NewBackofficeRoleHandler(db, logger.Log)

	backofficeRoutes := router.Group("/backoffice")
	{
		backofficeRoutes.GET("/permissions", backofficeAuth, envelope.Handle(backofficePermissionHandler.GetPermissions))
		backofficeRoutes.GET("/dashboard", backofficeAuth, envelope.Handle(backofficeDashboardHandler.GetDashboardStats))
	}

	backofficeUserRoutes := router.Group("/backoffice/users", backofficeAuth)
	{
		backofficeUserRoutes.GET("", envelope.Handle(backofficeUserHandler.GetUsers))
		backofficeUserRoutes.GET("/:id", envelope.Handle(backofficeUserHandler.GetUserByID))
		backofficeUserRoutes.POST("", envelope.Handle(backofficeUserHandler.CreateUser))
		backofficeUserRoutes.PUT("/:id", envelope.Handle(backofficeUserHandler.UpdateUser))
		backofficeUserRoutes.DELETE("/:id", envelope.Handle(backofficeUserHandler.DeleteUser))
	}

	authLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/15), 15)
	backofficeAuthRoutes := router.Group("/backoffice/auth", authLimiter.Middleware())
	{
		backofficeAuthRoutes.POST("/login", envelope.Handle(backofficeUserHandler.Login))
		backofficeAuthRoutes.POST("/refresh", envelope.Handle(backofficeUserHandler.RefreshAuthToken))
		backofficeAuthRoutes.POST("/logout", envelope.Handle(backofficeUserHandler.Logout))
	}

	backofficeCompanyRoutes := router.Group("/backoffice/companies", backofficeAuth)
	{
		backofficeCompanyRoutes.GET("", envelope.Handle(backofficeCompanyHandler.GetCompanies))
		backofficeCompanyRoutes.GET("/:id", envelope.Handle(backofficeCompanyHandler.GetCompanyByID))
		backofficeCompanyRoutes.POST("", envelope.Handle(backofficeCompanyHandler.CreateCompany))
		backofficeCompanyRoutes.POST("/register-token", envelope.Handle(backofficeCompanyHandler.GenerateCompanyRegisterToken))
		backofficeCompanyRoutes.PUT("/:id", envelope.Handle(backofficeCompanyHandler.UpdateCompany))
		backofficeCompanyRoutes.DELETE("/:id", envelope.Handle(backofficeCompanyHandler.DeleteCompany))
		backofficeCompanyRoutes.GET("/:id/signup-token", envelope.Handle(backofficeCompanyHandler.GenerateCompanySignupToken))
		backofficeCompanyRoutes.GET("/:id/users", envelope.Handle(backofficeCompanyHandler.GetCompanyUsers))
		backofficeCompanyRoutes.PUT("/:id/users/:user_id", envelope.Handle(backofficeCompanyHandler.UpdateCompanyUser))
		backofficeCompanyRoutes.GET("/:id/users/:user_id/password-reset-link", envelope.Handle(backofficeCompanyHandler.GenerateUserPasswordResetLink))
		backofficeCompanyRoutes.DELETE("/:id/users/:user_id", envelope.Handle(backofficeCompanyHandler.DeleteCompanyUser))
	}

	backofficeRoleRoutes := router.Group("/backoffice/roles", backofficeAuth)
	{
		backofficeRoleRoutes.GET("", envelope.Handle(backofficeRoleHandler.GetRoles))
		backofficeRoleRoutes.GET("/:id", envelope.Handle(backofficeRoleHandler.GetRoleByID))
		backofficeRoleRoutes.POST("", envelope.Handle(backofficeRoleHandler.CreateRole))
		backofficeRoleRoutes.PUT("/:id", envelope.Handle(backofficeRoleHandler.UpdateRole))
		backofficeRoleRoutes.DELETE("/:id", envelope.Handle(backofficeRoleHandler.DeleteRole))
	}

	backofficeFeatureRoutes := router.Group("/backoffice/features", backofficeAuth)
	{
		backofficeFeatureRoutes.GET("", envelope.Handle(backofficeFeatureHandler.GetFeatures))
		backofficeFeatureRoutes.GET("/:id", envelope.Handle(backofficeFeatureHandler.GetFeatureByID))
		backofficeFeatureRoutes.POST("", envelope.Handle(backofficeFeatureHandler.CreateFeature))
		backofficeFeatureRoutes.PUT("/:id", envelope.Handle(backofficeFeatureHandler.UpdateFeature))
		backofficeFeatureRoutes.DELETE("/:id", envelope.Handle(backofficeFeatureHandler.DeleteFeature))
	}

	backofficePlanRoutes := router.Group("/backoffice/plans", backofficeAuth)
	{
		backofficePlanRoutes.GET("", envelope.Handle(backofficePlanHandler.GetPlans))
		backofficePlanRoutes.GET("/:id", envelope.Handle(backofficePlanHandler.GetPlanByID))
		backofficePlanRoutes.POST("", envelope.Handle(backofficePlanHandler.CreatePlan))
		backofficePlanRoutes.PUT("/:id", envelope.Handle(backofficePlanHandler.UpdatePlan))
		backofficePlanRoutes.DELETE("/:id", envelope.Handle(backofficePlanHandler.DeletePlan))
	}

	backofficePaymentHandler := backoffice_handlers.NewBackofficePaymentHandler(db, logger.Log)
	backofficePaymentRoutes := router.Group("/backoffice/payments", backofficeAuth)
	{
		backofficePaymentRoutes.POST("/generate", envelope.Handle(backofficePaymentHandler.GeneratePayments))
		backofficePaymentRoutes.GET("", envelope.Handle(backofficePaymentHandler.GetPayments))
	}

	backofficeSubscriptionRoutes := router.Group("/backoffice/subscriptions", backofficeAuth)
	{
		backofficeSubscriptionRoutes.GET("", envelope.Handle(backofficeSubscriptionHandler.GetSubscriptions))
		backofficeSubscriptionRoutes.GET("/company/:id", envelope.Handle(backofficeSubscriptionHandler.GetSubscriptionsByCompany))
		backofficeSubscriptionRoutes.GET("/:id", envelope.Handle(backofficeSubscriptionHandler.GetSubscriptionByID))
		backofficeSubscriptionRoutes.POST("", envelope.Handle(backofficeSubscriptionHandler.CreateSubscription))
		backofficeSubscriptionRoutes.PUT("/:id", envelope.Handle(backofficeSubscriptionHandler.UpdateSubscription))
		backofficeSubscriptionRoutes.DELETE("/:id", envelope.Handle(backofficeSubscriptionHandler.DeleteSubscription))
	}
}
