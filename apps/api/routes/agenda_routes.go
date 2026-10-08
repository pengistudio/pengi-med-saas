package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	agenda_handlers "pengi-med-saas/features/agenda/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterAgendaRoutes mounts doctor schedules and blocks under /doctors and
// clinic blocks, appointment types, availability and the agenda range under
// /agenda. Gating: plan + role (CLINICAL), reusing the doctor permissions:
//   - MANAGE_DOCTORS manages every doctor's schedule and blocks, clinic
//     blocks and appointment types.
//   - The user linked to a doctor manages their own schedule and blocks
//     through /doctors/me/... (no permission, like PUT /doctors/me).
//   - Reading needs what reading doctors needs: READ_APPOINTMENT,
//     READ_PATIENT or MANAGE_DOCTORS.
func RegisterAgendaRoutes(router *gin.RouterGroup, db *gorm.DB) {
	mw := []gin.HandlerFunc{
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	}
	registerAgendaTenantRoutes(router.Group("/doctors", mw...), router.Group("/agenda", mw...), db,
		agenda_handlers.NewAgendaHandler(db, logger.Log))
}

// registerAgendaTenantRoutes is split out so tests can mount the routes behind
// their own tenant/subscription context.
func registerAgendaTenantRoutes(doctors, agenda *gin.RouterGroup, db *gorm.DB, h *agenda_handlers.AgendaHandler) {
	manage := subscription_middleware.RequirePermission(db, "MANAGE_DOCTORS")
	read := subscription_middleware.RequireAnyPermission(db, "READ_APPOINTMENT", "READ_PATIENT", "MANAGE_DOCTORS")

	// Own doctor profile
	doctors.GET("/me/schedule", envelope.Handle(h.GetMySchedule))
	doctors.PUT("/me/schedule", envelope.Handle(h.ReplaceMySchedule))
	doctors.GET("/me/blocks", envelope.Handle(h.ListMyBlocks))
	doctors.POST("/me/blocks", envelope.Handle(h.CreateMyBlock))
	doctors.PUT("/me/blocks/:blockId", envelope.Handle(h.UpdateMyBlock))
	doctors.DELETE("/me/blocks/:blockId", envelope.Handle(h.DeleteMyBlock))

	// Any doctor of the clinic
	doctors.GET("/:id/schedule", read, envelope.Handle(h.GetDoctorSchedule))
	doctors.PUT("/:id/schedule", manage, envelope.Handle(h.ReplaceDoctorSchedule))
	doctors.GET("/:id/blocks", read, envelope.Handle(h.ListDoctorBlocks))
	doctors.POST("/:id/blocks", manage, envelope.Handle(h.CreateDoctorBlock))
	doctors.PUT("/:id/blocks/:blockId", manage, envelope.Handle(h.UpdateDoctorBlock))
	doctors.DELETE("/:id/blocks/:blockId", manage, envelope.Handle(h.DeleteDoctorBlock))

	// Clinic-wide blocks
	agenda.GET("/blocks", read, envelope.Handle(h.ListClinicBlocks))
	agenda.POST("/blocks", manage, envelope.Handle(h.CreateClinicBlock))
	agenda.PUT("/blocks/:id", manage, envelope.Handle(h.UpdateClinicBlock))
	agenda.DELETE("/blocks/:id", manage, envelope.Handle(h.DeleteClinicBlock))

	// Appointment types
	agenda.GET("/appointment-types", read, envelope.Handle(h.ListAppointmentTypes))
	agenda.POST("/appointment-types", manage, envelope.Handle(h.CreateAppointmentType))
	agenda.PUT("/appointment-types/:id", manage, envelope.Handle(h.UpdateAppointmentType))
	agenda.DELETE("/appointment-types/:id", manage, envelope.Handle(h.DeleteAppointmentType))

	// Availability and agenda shading
	agenda.GET("/availability", read, envelope.Handle(h.GetAvailability))
	agenda.GET("/range", read, envelope.Handle(h.GetAgendaRange))
}
