package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	backoffice_handlers "pengi-med-saas/features/backoffice/handlers"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterBackofficeRoutes(router *gin.RouterGroup, db *gorm.DB) {
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
		backofficeRoutes.GET("/permissions", auth_middleware.AuthMiddleware(), envelope.Handle(backofficePermissionHandler.GetPermissions))
		backofficeRoutes.GET("/dashboard", auth_middleware.AuthMiddleware(), envelope.Handle(backofficeDashboardHandler.GetDashboardStats))
	}

	backofficeUserRoutes := router.Group("/backoffice/users", auth_middleware.AuthMiddleware())
	{
		backofficeUserRoutes.GET("", envelope.Handle(backofficeUserHandler.GetUsers))
		backofficeUserRoutes.GET("/:id", envelope.Handle(backofficeUserHandler.GetUserByID))
		backofficeUserRoutes.POST("", envelope.Handle(backofficeUserHandler.SignUp))
		backofficeUserRoutes.PUT("/:id", envelope.Handle(backofficeUserHandler.UpdateUser))
		backofficeUserRoutes.DELETE("/:id", envelope.Handle(backofficeUserHandler.DeleteUser))
	}

	backofficeAuthRoutes := router.Group("/backoffice/auth")
	{
		backofficeAuthRoutes.POST("/signup", envelope.Handle(backofficeUserHandler.SignUp))
		backofficeAuthRoutes.POST("/login", envelope.Handle(backofficeUserHandler.Login))
		backofficeAuthRoutes.POST("/refresh", envelope.Handle(backofficeUserHandler.RefreshAuthToken))
		backofficeAuthRoutes.POST("/extend", auth_middleware.AuthMiddleware(), envelope.Handle(backofficeUserHandler.ExtendSession))
		backofficeAuthRoutes.POST("/validate", envelope.Handle(backofficeUserHandler.ValidateBearerToken))
	}

	backofficeCompanyRoutes := router.Group("/backoffice/companies", auth_middleware.AuthMiddleware())
	{
		backofficeCompanyRoutes.GET("", envelope.Handle(backofficeCompanyHandler.GetCompanies))
		backofficeCompanyRoutes.GET("/:id", envelope.Handle(backofficeCompanyHandler.GetCompanyByID))
		backofficeCompanyRoutes.POST("", envelope.Handle(backofficeCompanyHandler.CreateCompany))
		backofficeCompanyRoutes.PUT("/:id", envelope.Handle(backofficeCompanyHandler.UpdateCompany))
		backofficeCompanyRoutes.DELETE("/:id", envelope.Handle(backofficeCompanyHandler.DeleteCompany))
		backofficeCompanyRoutes.GET("/:id/signup-token", envelope.Handle(backofficeCompanyHandler.GenerateCompanySignupToken))
		backofficeCompanyRoutes.GET("/:id/users", envelope.Handle(backofficeCompanyHandler.GetCompanyUsers))
		backofficeCompanyRoutes.PUT("/:id/users/:user_id", envelope.Handle(backofficeCompanyHandler.UpdateCompanyUser))
	}

	backofficeRoleRoutes := router.Group("/backoffice/roles", auth_middleware.AuthMiddleware())
	{
		backofficeRoleRoutes.GET("", envelope.Handle(backofficeRoleHandler.GetRoles))
		backofficeRoleRoutes.GET("/:id", envelope.Handle(backofficeRoleHandler.GetRoleByID))
		backofficeRoleRoutes.POST("", envelope.Handle(backofficeRoleHandler.CreateRole))
		backofficeRoleRoutes.PUT("/:id", envelope.Handle(backofficeRoleHandler.UpdateRole))
		backofficeRoleRoutes.DELETE("/:id", envelope.Handle(backofficeRoleHandler.DeleteRole))
	}

	backofficeFeatureRoutes := router.Group("/backoffice/features", auth_middleware.AuthMiddleware())
	{
		backofficeFeatureRoutes.GET("", envelope.Handle(backofficeFeatureHandler.GetFeatures))
		backofficeFeatureRoutes.GET("/:id", envelope.Handle(backofficeFeatureHandler.GetFeatureByID))
		backofficeFeatureRoutes.POST("", envelope.Handle(backofficeFeatureHandler.CreateFeature))
		backofficeFeatureRoutes.PUT("/:id", envelope.Handle(backofficeFeatureHandler.UpdateFeature))
		backofficeFeatureRoutes.DELETE("/:id", envelope.Handle(backofficeFeatureHandler.DeleteFeature))
	}

	backofficePlanRoutes := router.Group("/backoffice/plans", auth_middleware.AuthMiddleware())
	{
		backofficePlanRoutes.GET("", envelope.Handle(backofficePlanHandler.GetPlans))
		backofficePlanRoutes.GET("/:id", envelope.Handle(backofficePlanHandler.GetPlanByID))
		backofficePlanRoutes.POST("", envelope.Handle(backofficePlanHandler.CreatePlan))
		backofficePlanRoutes.PUT("/:id", envelope.Handle(backofficePlanHandler.UpdatePlan))
		backofficePlanRoutes.DELETE("/:id", envelope.Handle(backofficePlanHandler.DeletePlan))
	}

	backofficeSubscriptionRoutes := router.Group("/backoffice/subscriptions", auth_middleware.AuthMiddleware())
	{
		backofficeSubscriptionRoutes.GET("", envelope.Handle(backofficeSubscriptionHandler.GetSubscriptions))
		backofficeSubscriptionRoutes.GET("/company/:id", envelope.Handle(backofficeSubscriptionHandler.GetSubscriptionsByCompany))
		backofficeSubscriptionRoutes.POST("", envelope.Handle(backofficeSubscriptionHandler.CreateSubscription))
		backofficeSubscriptionRoutes.PUT("/:id", envelope.Handle(backofficeSubscriptionHandler.UpdateSubscription))
		backofficeSubscriptionRoutes.DELETE("/:id", envelope.Handle(backofficeSubscriptionHandler.DeleteSubscription))
	}
}
