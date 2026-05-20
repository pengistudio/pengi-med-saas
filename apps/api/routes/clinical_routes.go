package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
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
	icd11Handler := clinical_handlers.NewICD11Handler(logger.Log)
	icd10Handler := clinical_handlers.NewICD10Handler(db, logger.Log)

	downloadHandler := clinical_handlers.NewDownloadRecordHandler(db)
	prescriptionTemplateHandler := clinical_handlers.NewPrescriptionTemplateHandler(db, logger.Log)

	clinicalGroup := router.Group("/clinical", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db), subscription_middleware.SubscriptionMiddleware(db))
	{

		// ICD search routes
		clinicalGroup.GET("/icd11/search", envelope.Handle(icd11Handler.Search))
		clinicalGroup.GET("/icd10/search", envelope.Handle(icd10Handler.Search))

		rp := subscription_middleware.RequirePermission

		// Patient routes
		patientGroup := clinicalGroup.Group("/patients")
		{
			patientGroup.POST("", rp(db, "CREATE_PATIENT"), envelope.Handle(patientHandler.CreatePatient))
			patientGroup.PUT("/:id", rp(db, "UPDATE_PATIENT"), envelope.Handle(patientHandler.UpdatePatient))
			patientGroup.PUT("/:id/critical", rp(db, "UPDATE_PATIENT"), envelope.Handle(patientHandler.UpdatePatientCritical))
			patientGroup.PUT("/:id/critical-revert", rp(db, "UPDATE_PATIENT"), envelope.Handle(patientHandler.UpdatePatientCriticalRevert))
			patientGroup.GET("", rp(db, "READ_PATIENT"), envelope.Handle(patientHandler.GetAllPatients))
			patientGroup.GET("/follow-up", rp(db, "READ_PATIENT"), envelope.Handle(patientHandler.GetAllPatientsWithLastFollowUp))
			patientGroup.GET("/:id", rp(db, "READ_PATIENT"), envelope.Handle(patientHandler.GetPatientByID))
			patientGroup.GET("/:id/report", rp(db, "DOWNLOAD_PATIENT_REPORT"), downloadHandler.DownloadPatientReport)
			patientGroup.POST("/delete-multiple", rp(db, "DELETE_PATIENT"), envelope.Handle(patientHandler.DeleteMultiplePatients))
			patientGroup.DELETE("/delete-multiple/:id", rp(db, "DELETE_PATIENT"), envelope.Handle(patientHandler.DeleteOnePatient))
		}

		// Medical Record routes
		recordGroup := clinicalGroup.Group("/records")
		{
			recordGroup.POST("", rp(db, "CREATE_MEDICAL_RECORD"), envelope.Handle(recordHandler.CreateMedicalRecord))
			recordGroup.PUT("/:id", rp(db, "UPDATE_MEDICAL_RECORD"), envelope.Handle(recordHandler.UpdateMedicalRecord))
			recordGroup.GET("/patient/:id", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(recordHandler.GetMedicalRecords))
			recordGroup.GET("/:id", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(recordHandler.GetMedicalRecord))
			recordGroup.PUT("/:id/prescription", rp(db, "UPDATE_PRESCRIPTION"), envelope.Handle(recordHandler.UpdatePrescription))
			recordGroup.GET("/:id/prescription/download", rp(db, "UPDATE_PRESCRIPTION"), downloadHandler.DownloadPrescription)
			recordGroup.PUT("/:id/vital-signs", rp(db, "UPDATE_MEDICAL_RECORD"), envelope.Handle(vitalSignsHandler.UpsertVitalSigns))
			recordGroup.GET("/:id/vital-signs", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(vitalSignsHandler.GetVitalSigns))
		}

		// Prescription template routes
		clinicalGroup.GET("/prescription-template/status", envelope.Handle(prescriptionTemplateHandler.GetPrescriptionTemplateStatus))
		clinicalGroup.POST("/prescription-template", envelope.Handle(prescriptionTemplateHandler.UploadPrescriptionTemplate))
		clinicalGroup.DELETE("/prescription-template", envelope.Handle(prescriptionTemplateHandler.DeletePrescriptionTemplate))

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
