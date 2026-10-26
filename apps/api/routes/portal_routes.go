package routes

import (
	"time"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/mailer"
	core_middleware "pengi-med-saas/core/middleware"
	"pengi-med-saas/core/whatsapp"
	sri_document "pengi-med-saas/features/billing/sri-document"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	portal_handlers "pengi-med-saas/features/portal/handlers"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// RegisterPortalRoutes mounts the patient portal and its clinic-side settings.
//
// Gating: /portal-settings is plan + role (MANAGE_PATIENT_PORTAL, admin).
// /portal has no staff permission: a clinic's portal answers only when its
// plan includes the PATIENT_PORTAL Feature and it enabled the portal
// (otherwise 404 E-PORTAL-001); patient endpoints also need a portal access
// token and the section enabled.
func RegisterPortalRoutes(router *gin.RouterGroup, db *gorm.DB) {
	delivery := portal_services.NewDelivery(db, logger.Log, mailer.NewMailer(), whatsapp.NewFromEnv())

	settings := portal_handlers.NewSettingsHandler(db, logger.Log, delivery.EnsurePortalTemplate)
	settingsGroup := router.Group("/portal-settings",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	manage := subscription_middleware.RequirePermission(db, "MANAGE_PATIENT_PORTAL")
	settingsGroup.GET("", manage, envelope.Handle(settings.GetSettings))
	settingsGroup.PUT("", manage, envelope.Handle(settings.UpdateSettings))

	authHandler := portal_handlers.NewAuthHandler(db, logger.Log, tenantFiles, delivery)
	patientHandler := portal_handlers.NewPatientHandler(db, logger.Log, tenantFiles, attachmentFiles(),
		sri_document.Documents{Files: tenantFiles, Renderer: documentRenderer()})
	registerPortalRoutes(router.Group("/portal"), db, authHandler, patientHandler)
}

// registerPortalRoutes mounts the /portal endpoints on group. Per-IP rate
// limits: 5 code requests a minute and 20 a day, 10 verifications a minute.
// The code budgets (per patient, per clinic and global, counted in the DB)
// live in portal_services.IssueCode; the captcha in the Login handler.
func registerPortalRoutes(group *gin.RouterGroup, db *gorm.DB, a *portal_handlers.AuthHandler, p *portal_handlers.PatientHandler) {
	loginLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/5), 5)
	loginDailyLimiter := core_middleware.NewRateLimiter(rate.Every(24*time.Hour/20), 20)
	verifyLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/10), 10)
	sessionLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/30), 30)
	publicLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/60), 60)

	group.GET("/clinics/:slug", publicLimiter.Middleware(), envelope.Handle(a.GetClinic))
	group.GET("/clinics/:slug/logo", publicLimiter.Middleware(), a.GetLogo)
	group.POST("/clinics/:slug/login", loginLimiter.Middleware(), loginDailyLimiter.Middleware(), envelope.Handle(a.Login))
	group.POST("/clinics/:slug/verify", verifyLimiter.Middleware(), envelope.Handle(a.Verify))
	group.POST("/refresh", sessionLimiter.Middleware(), envelope.Handle(a.Refresh))
	group.POST("/logout", envelope.Handle(a.Logout))

	authed := group.Group("", portal_middleware.RequirePortalSession(db))
	section := portal_middleware.RequireSection
	authed.GET("/me", envelope.Handle(p.Me))
	authed.GET("/appointments", section(portal_middleware.SectionAppointments), envelope.Handle(p.ListAppointments))
	authed.GET("/results", section(portal_middleware.SectionResults), envelope.Handle(p.ListResults))
	authed.GET("/results/files/:id", section(portal_middleware.SectionResults), p.DownloadResultFile)
	authed.GET("/documents", section(portal_middleware.SectionDocuments), envelope.Handle(p.ListDocuments))
	authed.GET("/documents/:type/:id/file", section(portal_middleware.SectionDocuments), p.DownloadDocument)
	authed.GET("/invoices", section(portal_middleware.SectionInvoices), envelope.Handle(p.ListInvoices))
	authed.GET("/invoices/:id/ride", section(portal_middleware.SectionInvoices), p.DownloadInvoiceRide)
	authed.GET("/invoices/:id/xml", section(portal_middleware.SectionInvoices), p.DownloadInvoiceXML)
}
