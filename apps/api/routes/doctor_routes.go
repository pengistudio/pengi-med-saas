package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	doctor_handlers "pengi-med-saas/features/doctors/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterDoctorRoutes mounts /doctors. Gating: plan + role (CLINICAL).
//   - Managing profiles needs MANAGE_DOCTORS.
//   - Reading them (selectors) needs READ_APPOINTMENT or READ_PATIENT.
//   - /me and /status are the caller's own data: any tenant member; creating
//     one's own profile needs MANAGE_DOCTORS or CREATE_MEDICAL_RECORD (someone
//     who attends patients).
func RegisterDoctorRoutes(router *gin.RouterGroup, db *gorm.DB) {
	group := router.Group("/doctors",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	registerDoctorTenantRoutes(group, db, doctor_handlers.NewDoctorHandler(db, logger.Log))
}

// registerDoctorTenantRoutes is split out so tests can mount the routes behind
// their own tenant/subscription context.
func registerDoctorTenantRoutes(group *gin.RouterGroup, db *gorm.DB, h *doctor_handlers.DoctorHandler) {
	rp := subscription_middleware.RequirePermission
	read := subscription_middleware.RequireAnyPermission(db, "READ_APPOINTMENT", "READ_PATIENT", "MANAGE_DOCTORS")

	// Own profile and onboarding status
	group.GET("/status", envelope.Handle(h.GetStatus))
	group.GET("/me", envelope.Handle(h.GetMyDoctor))
	group.POST("/me", subscription_middleware.RequireAnyPermission(db, "MANAGE_DOCTORS", "CREATE_MEDICAL_RECORD"), envelope.Handle(h.CreateMyDoctor))
	group.PUT("/me", envelope.Handle(h.UpdateMyDoctor))

	// Catalog and reads
	group.GET("/specialties", read, envelope.Handle(h.GetSpecialties))
	group.GET("", read, envelope.Handle(h.GetDoctors))
	group.GET("/:id", read, envelope.Handle(h.GetDoctor))

	// Administration
	group.POST("", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.CreateDoctor))
	group.PUT("/review", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.MarkReviewed))
	group.PUT("/:id", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.UpdateDoctor))
	group.PUT("/:id/user", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.LinkUser))
	group.PUT("/:id/activate", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.ActivateDoctor))
	group.PUT("/:id/deactivate", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.DeactivateDoctor))
	group.DELETE("/:id", rp(db, "MANAGE_DOCTORS"), envelope.Handle(h.DeleteDoctor))
}
