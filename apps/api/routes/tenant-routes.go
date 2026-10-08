package routes

import (
	"time"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	core_middleware "pengi-med-saas/core/middleware"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_handlers "pengi-med-saas/features/tenants/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func RegisterTenantRoutes(router *gin.RouterGroup, db *gorm.DB) {
	tenantHandler := tenant_handlers.NewTenantHandler(db, logger.Log, tenantFiles)

	tenantGroup := router.Group("/tenants", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))
	registerTenantSettingsRoutes(tenantGroup, db, tenantHandler)

	// The display token is the credential of the public waiting-room TV, so
	// viewing or rotating it is account administration: the role must manage
	// team members (admin). Role-only, like the team routes: not a plan feature.
	manageDisplay := subscription_middleware.RequireRolePermission(db, "MANAGE_TEAM_MEMBERS")
	tenantGroup.GET("/display-token", manageDisplay, envelope.Handle(tenantHandler.GetDisplayToken))
	tenantGroup.POST("/display-token", manageDisplay, envelope.Handle(tenantHandler.GenerateDisplayToken))
	// PNG QR of the TV link (FRONTEND_URL base), streamed: errors via envelope.Write.
	tenantGroup.GET("/display-token/qr", manageDisplay, tenantHandler.GetDisplayTokenQR)

	// Public — no auth required, validated via display token. 60 requests/min
	// per IP (burst 60): a TV polls every 15 s, a token guesser gets nowhere.
	displayLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/60), 60)
	publicGroup := router.Group("/public")
	publicGroup.GET("/appointments/today", displayLimiter.Middleware(), envelope.Handle(tenantHandler.GetTodayAppointmentsPublic))
}

// registerTenantSettingsRoutes mounts the clinic's SRI, logo and UI settings
// on a group that already authenticated the user and bound the tenant.
func registerTenantSettingsRoutes(tenantGroup *gin.RouterGroup, db *gorm.DB, tenantHandler *tenant_handlers.TenantHandler) {
	// The SRI certificate, the issuer data and the logo printed on the RIDE
	// are the clinic's billing identity: only who manages SRI settings
	// (admin, contador) may replace them. Same check as the SRI settings page.
	manageSri := subscription_middleware.RequirePermission(db, "MANAGE_SRI_SETTINGS")
	tenantGroup.PUT("/sri/signature", manageSri, envelope.Handle(tenantHandler.UploadSignature))
	tenantGroup.GET("/sri/status", envelope.Handle(tenantHandler.GetSriStatus))
	tenantGroup.PUT("/logo", manageSri, envelope.Handle(tenantHandler.UploadLogo))
	tenantGroup.GET("/logo", tenantHandler.DownloadLogo)
	tenantGroup.PUT("/sri/info", manageSri, envelope.Handle(tenantHandler.UpdateSriInfo))
	tenantGroup.GET("/settings", envelope.Handle(tenantHandler.GetUISettings))
	tenantGroup.PUT("/settings", envelope.Handle(tenantHandler.UpdateUISettings))
}
