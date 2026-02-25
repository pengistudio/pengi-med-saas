package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterClinicalRoutes(router *gin.RouterGroup, db *gorm.DB) {
	patientHandler := clinical_handlers.NewPatientHandler(db, logger.Log)
	recordHandler := clinical_handlers.NewMedicalRecordHandler(db, logger.Log)

	downloadHandler := clinical_handlers.NewDownloadRecordHandler(db)

	clinicalGroup := router.Group("/clinical", tenant_middleware.TenantMiddleware(db))
	{

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
		}
	}
}
