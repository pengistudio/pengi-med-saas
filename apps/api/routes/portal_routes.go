package routes

import (
	"time"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/mailer"
	core_middleware "pengi-med-saas/core/middleware"
	"pengi-med-saas/core/whatsapp"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
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
// token and the section enabled; booking endpoints also need booking on
// (otherwise 404 E-PORTAL-009). /appointment-requests (the clinic's inbox of
// portal requests) is plan + role (MANAGE_APPOINTMENT_REQUESTS).
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

	calendar := clinical_handlers.NewAppointmentHandler(db, logger.Log)
	requests := portal_handlers.NewRequestHandler(db, logger.Log, delivery, calendar)
	requestsGroup := router.Group("/appointment-requests",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	manageRequests := subscription_middleware.RequirePermission(db, "MANAGE_APPOINTMENT_REQUESTS")
	requestsGroup.GET("", manageRequests, envelope.Handle(requests.ListRequests))
	requestsGroup.GET("/pending-count", manageRequests, envelope.Handle(requests.PendingCount))
	requestsGroup.POST("/:id/approve", manageRequests, envelope.Handle(requests.Approve))
	requestsGroup.POST("/:id/reject", manageRequests, envelope.Handle(requests.Reject))

	authHandler := portal_handlers.NewAuthHandler(db, logger.Log, tenantFiles, delivery)
	patientHandler := portal_handlers.NewPatientHandler(db, logger.Log, tenantFiles, attachmentFiles(),
		sri_document.Documents{Files: tenantFiles, Renderer: documentRenderer()})
	bookingHandler := portal_handlers.NewBookingHandler(db, logger.Log, delivery, calendar)
	firstHandler := portal_handlers.NewFirstAppointmentHandler(db, logger.Log, delivery)
	registerPortalRoutes(router.Group("/portal"), db, authHandler, patientHandler, bookingHandler, firstHandler)
}

// registerPortalRoutes mounts the /portal endpoints on group. Per-IP rate
// limits: 5 code requests a minute and 20 a day, 10 verifications a minute.
// The code budgets (per patient, per clinic and global, counted in the DB)
// live in portal_services.IssueCode; the captcha in the Login handler.
// Booking requests: 5 a minute and 30 a day per IP (plus 3 pending per
// patient, checked in the handler).
// New patients (first appointment): contact codes and their verification
// have their own per-IP limits with the login's rates (the DB budgets are
// shared with sign-in codes, see portal_services.IssueContactCode); the
// booking catalog (options, slots) also takes a guest token; requests have
// the booking limits plus the per-contact, per-document and per-clinic caps
// checked in the handler.
func registerPortalRoutes(group *gin.RouterGroup, db *gorm.DB, a *portal_handlers.AuthHandler, p *portal_handlers.PatientHandler, b *portal_handlers.BookingHandler, n *portal_handlers.FirstAppointmentHandler) {
	loginLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/5), 5)
	loginDailyLimiter := core_middleware.NewRateLimiter(rate.Every(24*time.Hour/20), 20)
	verifyLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/10), 10)
	sessionLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/30), 30)
	publicLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/60), 60)
	bookingLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/5), 5)
	bookingDailyLimiter := core_middleware.NewRateLimiter(rate.Every(24*time.Hour/30), 30)
	contactLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/5), 5)
	contactDailyLimiter := core_middleware.NewRateLimiter(rate.Every(24*time.Hour/20), 20)
	contactVerifyLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/10), 10)
	newPatientLimiter := core_middleware.NewRateLimiter(rate.Every(time.Minute/5), 5)
	newPatientDailyLimiter := core_middleware.NewRateLimiter(rate.Every(24*time.Hour/30), 30)

	group.GET("/clinics/:slug", publicLimiter.Middleware(), envelope.Handle(a.GetClinic))
	group.GET("/clinics/:slug/logo", publicLimiter.Middleware(), a.GetLogo)
	group.POST("/clinics/:slug/login", loginLimiter.Middleware(), loginDailyLimiter.Middleware(), envelope.Handle(a.Login))
	group.POST("/clinics/:slug/verify", verifyLimiter.Middleware(), envelope.Handle(a.Verify))
	group.POST("/clinics/:slug/new-patient/code", contactLimiter.Middleware(), contactDailyLimiter.Middleware(), envelope.Handle(n.RequestCode))
	group.POST("/clinics/:slug/new-patient/verify", contactVerifyLimiter.Middleware(), envelope.Handle(n.Verify))
	group.POST("/refresh", sessionLimiter.Middleware(), envelope.Handle(a.Refresh))
	group.POST("/logout", envelope.Handle(a.Logout))

	// The booking catalog: a patient session or a new patient's guest token.
	catalog := group.Group("", portal_middleware.RequirePortalSessionOrGuest(db), portal_middleware.RequireBooking())
	catalog.GET("/booking/options", envelope.Handle(b.GetOptions))
	catalog.GET("/booking/slots", envelope.Handle(b.GetSlots))

	guest := group.Group("/new-patient", portal_middleware.RequirePortalGuest(db), portal_middleware.RequireBooking())
	guest.POST("/requests", newPatientLimiter.Middleware(), newPatientDailyLimiter.Middleware(), envelope.Handle(n.CreateRequest))

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

	booking := authed.Group("", portal_middleware.RequireBooking())
	booking.GET("/booking/requests", envelope.Handle(b.ListRequests))
	booking.POST("/booking/requests", bookingLimiter.Middleware(), bookingDailyLimiter.Middleware(), envelope.Handle(b.CreateRequest))
	booking.POST("/booking/requests/:id/cancel", envelope.Handle(b.CancelRequest))
	booking.POST("/appointments/:id/cancel", envelope.Handle(b.CancelAppointment))
}
