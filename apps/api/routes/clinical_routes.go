package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterClinicalRoutes(router *gin.RouterGroup, db *gorm.DB) {
	patientHandler := clinical_handlers.NewPatientHandler(db, logger.Log)
	recordHandler := clinical_handlers.NewMedicalRecordHandler(db, logger.Log)
	vitalSignsHandler := clinical_handlers.NewVitalSignsHandler(db, logger.Log)
	appointmentHandler := clinical_handlers.NewAppointmentHandler(db, logger.Log)
	dashboardHandler := clinical_handlers.NewDashboardHandler(db, logger.Log)
	icd11Handler := clinical_handlers.NewICD11Handler(logger.Log)
	icd10Handler := clinical_handlers.NewICD10Handler(db, logger.Log)

	downloadHandler := clinical_handlers.NewDownloadRecordHandler(db)

	clinicalGroup := router.Group("/clinical", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))
	{

		// ICD search routes
		clinicalGroup.GET("/icd11/search", envelope.Handle(icd11Handler.Search))
		clinicalGroup.GET("/icd10/search", envelope.Handle(icd10Handler.Search))

		// Dashboard route
		clinicalGroup.GET("/dashboard/stats", envelope.Handle(dashboardHandler.GetDashboardStats))

		// Patient routes
		patientGroup := clinicalGroup.Group("/patients")
		{
			patientGroup.POST("", envelope.Handle(patientHandler.CreatePatient))
			patientGroup.PUT("/:id", envelope.Handle(patientHandler.UpdatePatient))
			patientGroup.PUT("/:id/critical", envelope.Handle(patientHandler.UpdatePatientCritical))
			patientGroup.PUT("/:id/critical-revert", envelope.Handle(patientHandler.UpdatePatientCriticalRevert))
			patientGroup.GET("", envelope.Handle(patientHandler.GetAllPatients))
			patientGroup.GET("/follow-up", envelope.Handle(patientHandler.GetAllPatientsWithLastFollowUp))
			patientGroup.GET("/:id", envelope.Handle(patientHandler.GetPatientByID))
			patientGroup.GET("/:id/report", downloadHandler.DownloadPatientReport)
			patientGroup.POST("/delete-multiple", envelope.Handle(patientHandler.DeleteMultiplePatients))
			patientGroup.DELETE("/delete-multiple/:id", envelope.Handle(patientHandler.DeleteOnePatient))
		}

		// Medical Record routes
		recordGroup := clinicalGroup.Group("/records")
		{
			recordGroup.POST("", envelope.Handle(recordHandler.CreateMedicalRecord))
			recordGroup.PUT("/:id", envelope.Handle(recordHandler.UpdateMedicalRecord))
			recordGroup.GET("/patient/:id", envelope.Handle(recordHandler.GetMedicalRecords))
			recordGroup.GET("/:id", envelope.Handle(recordHandler.GetMedicalRecord))
			recordGroup.PUT("/:id/prescription", envelope.Handle(recordHandler.UpdatePrescription))
			recordGroup.GET("/:id/prescription/download", downloadHandler.DownloadPrescription)
			recordGroup.PUT("/:id/vital-signs", envelope.Handle(vitalSignsHandler.UpsertVitalSigns))
			recordGroup.GET("/:id/vital-signs", envelope.Handle(vitalSignsHandler.GetVitalSigns))
		}

		// Appointment routes
		appointmentGroup := clinicalGroup.Group("/appointments")
		{
			appointmentGroup.GET("", envelope.Handle(appointmentHandler.GetAppointments))
			appointmentGroup.GET("/today", envelope.Handle(appointmentHandler.GetTodayAppointments))
			appointmentGroup.GET("/:id", envelope.Handle(appointmentHandler.GetAppointment))
			appointmentGroup.GET("/patient/:id", envelope.Handle(appointmentHandler.GetPatientAppointments))
			appointmentGroup.POST("", envelope.Handle(appointmentHandler.CreateAppointment))
			appointmentGroup.PUT("/:id", envelope.Handle(appointmentHandler.UpdateAppointment))
			appointmentGroup.PUT("/:id/status", envelope.Handle(appointmentHandler.UpdateStatus))
			appointmentGroup.DELETE("/:id", envelope.Handle(appointmentHandler.DeleteAppointment))
		}
	}
}
