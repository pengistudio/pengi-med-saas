package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantfiles"
	clinical_handlers "pengi-med-saas/features/clinical/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	signature_services "pengi-med-saas/features/signatures/services"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterClinicalRoutes(router *gin.RouterGroup, db *gorm.DB) {
	patientHandler := clinical_handlers.NewPatientHandler(db, logger.Log)
	recordHandler := clinical_handlers.NewMedicalRecordHandler(db, logger.Log)
	vitalSignsHandler := clinical_handlers.NewVitalSignsHandler(db, logger.Log)
	appointmentHandler := clinical_handlers.NewAppointmentHandler(db, logger.Log)
	icd11Handler := clinical_handlers.NewICD11Handler(logger.Log)
	icd10Handler := clinical_handlers.NewICD10Handler(db, logger.Log)

	signer := signature_services.NewSigner(db, tenantFiles)
	downloadHandler := clinical_handlers.NewDownloadRecordHandler(db, logger.Log, documentRenderer(), signer, tenantFiles)
	draftHandler := clinical_handlers.NewMedicalRecordDraftHandler(db, logger.Log)
	medicalDocumentHandler := clinical_handlers.NewMedicalDocumentHandler(db, logger.Log, mailer.NewMailer(), documentRenderer(), signer, tenantFiles)
	attachments := attachmentFiles()
	attachmentHandler := clinical_handlers.NewPatientAttachmentHandler(db, logger.Log, attachments)
	examCatalogHandler := clinical_handlers.NewExamCatalogHandler(db, logger.Log)
	examOrderHandler := clinical_handlers.NewExamOrderHandler(db, logger.Log, mailer.NewMailer(), documentRenderer(), signer, tenantFiles, attachments)

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
			patientGroup.POST("/delete-multiple", rp(db, "DELETE_PATIENT"), envelope.Handle(patientHandler.DeleteMultiplePatients))
			patientGroup.DELETE("/delete-multiple/:id", rp(db, "DELETE_PATIENT"), envelope.Handle(patientHandler.DeleteOnePatient))
			patientGroup.POST("/:id/reports", rp(db, "CREATE_MEDICAL_REPORT"), envelope.Handle(medicalDocumentHandler.CreateMedicalReport))
			patientGroup.GET("/:id/reports", rp(db, "CREATE_MEDICAL_REPORT"), envelope.Handle(medicalDocumentHandler.ListMedicalReports))
			patientGroup.POST("/:id/certificates", rp(db, "CREATE_MEDICAL_CERTIFICATE"), envelope.Handle(medicalDocumentHandler.CreateMedicalCertificate))
			patientGroup.GET("/:id/certificates", rp(db, "CREATE_MEDICAL_CERTIFICATE"), envelope.Handle(medicalDocumentHandler.ListMedicalCertificates))

			// Patient attachments (Adjuntos): upload, list, download
			attachmentHandler.Mount(patientGroup, func(permissionID string) gin.HandlerFunc { return rp(db, permissionID) })
		}

		// Medical report / certificate document routes (generate, download, email)
		clinicalGroup.GET("/reports/:id/download", rp(db, "CREATE_MEDICAL_REPORT"), medicalDocumentHandler.DownloadMedicalReport)
		clinicalGroup.POST("/reports/:id/email", rp(db, "CREATE_MEDICAL_REPORT"), envelope.Handle(medicalDocumentHandler.EmailMedicalReport))
		clinicalGroup.POST("/reports/:id/sign", rp(db, "CREATE_MEDICAL_REPORT"), rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(medicalDocumentHandler.SignMedicalReport))
		clinicalGroup.GET("/certificates/:id/download", rp(db, "CREATE_MEDICAL_CERTIFICATE"), medicalDocumentHandler.DownloadMedicalCertificate)
		clinicalGroup.POST("/certificates/:id/email", rp(db, "CREATE_MEDICAL_CERTIFICATE"), envelope.Handle(medicalDocumentHandler.EmailMedicalCertificate))
		clinicalGroup.POST("/certificates/:id/sign", rp(db, "CREATE_MEDICAL_CERTIFICATE"), rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(medicalDocumentHandler.SignMedicalCertificate))

		// Medical Record routes
		recordGroup := clinicalGroup.Group("/records")
		{
			recordGroup.POST("", rp(db, "CREATE_MEDICAL_RECORD"), envelope.Handle(recordHandler.CreateMedicalRecord))
			recordGroup.PUT("/:id", rp(db, "UPDATE_MEDICAL_RECORD"), envelope.Handle(recordHandler.UpdateMedicalRecord))
			recordGroup.GET("/patient/:id", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(recordHandler.GetMedicalRecords))
			recordGroup.GET("/:id", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(recordHandler.GetMedicalRecord))
			recordGroup.PUT("/:id/prescription", rp(db, "UPDATE_PRESCRIPTION"), envelope.Handle(recordHandler.UpdatePrescription))
			recordGroup.GET("/:id/prescription/download", rp(db, "UPDATE_PRESCRIPTION"), downloadHandler.DownloadPrescription)
			recordGroup.POST("/:id/prescription/sign", rp(db, "UPDATE_PRESCRIPTION"), rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(downloadHandler.SignPrescription))
			registerVitalSignsRoutes(recordGroup, db, vitalSignsHandler)
			recordGroup.GET("/draft/:patient_id", rp(db, "READ_MEDICAL_RECORD"), envelope.Handle(draftHandler.GetDraft))
			recordGroup.PUT("/draft/:patient_id", rp(db, "CREATE_MEDICAL_RECORD"), envelope.Handle(draftHandler.SaveDraft))
			recordGroup.DELETE("/draft/:patient_id", rp(db, "CREATE_MEDICAL_RECORD"), envelope.Handle(draftHandler.DeleteDraft))
		}

		// Exam catalog and profiles
		clinicalGroup.GET("/exam-catalog", rp(db, "READ_EXAM_ORDER"), envelope.Handle(examCatalogHandler.GetExamCatalog))
		clinicalGroup.POST("/exam-catalog", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.CreateExamCatalogItem))
		clinicalGroup.POST("/exam-catalog/restore-defaults", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.RestoreExamCatalogDefaults))
		clinicalGroup.PUT("/exam-catalog/:id", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.UpdateExamCatalogItem))
		clinicalGroup.DELETE("/exam-catalog/:id", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.DeleteExamCatalogItem))
		clinicalGroup.GET("/exam-profiles", rp(db, "READ_EXAM_ORDER"), envelope.Handle(examCatalogHandler.GetExamProfiles))
		clinicalGroup.POST("/exam-profiles", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.CreateExamProfile))
		clinicalGroup.PUT("/exam-profiles/:id", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.UpdateExamProfile))
		clinicalGroup.DELETE("/exam-profiles/:id", rp(db, "MANAGE_EXAM_CATALOG"), envelope.Handle(examCatalogHandler.DeleteExamProfile))

		// Exam orders, results and review
		examOrderGroup := clinicalGroup.Group("/exam-orders")
		{
			examOrderGroup.GET("", rp(db, "READ_EXAM_ORDER"), envelope.Handle(examOrderHandler.GetExamOrders))
			examOrderGroup.GET("/:id", rp(db, "READ_EXAM_ORDER"), envelope.Handle(examOrderHandler.GetExamOrder))
			examOrderGroup.GET("/:id/download", rp(db, "READ_EXAM_ORDER"), examOrderHandler.DownloadExamOrder)
			examOrderGroup.POST("", rp(db, "CREATE_EXAM_ORDER"), envelope.Handle(examOrderHandler.CreateExamOrder))
			examOrderGroup.PUT("/:id", rp(db, "CREATE_EXAM_ORDER"), envelope.Handle(examOrderHandler.UpdateExamOrder))
			examOrderGroup.POST("/:id/void", rp(db, "CREATE_EXAM_ORDER"), envelope.Handle(examOrderHandler.VoidExamOrder))
			examOrderGroup.POST("/:id/close", rp(db, "CREATE_EXAM_ORDER"), envelope.Handle(examOrderHandler.CloseExamOrder))
			examOrderGroup.POST("/:id/email", rp(db, "CREATE_EXAM_ORDER"), envelope.Handle(examOrderHandler.EmailExamOrder))
			examOrderGroup.POST("/:id/sign", rp(db, "CREATE_EXAM_ORDER"), rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(examOrderHandler.SignExamOrder))
			// A result is a patient attachment (Adjunto): uploading one goes through the
			// attachment upload path; deleting one also needs the attachment delete permission.
			examOrderGroup.POST("/:id/results", rp(db, "UPLOAD_EXAM_RESULTS"), envelope.Handle(examOrderHandler.UploadExamResult))
			examOrderGroup.DELETE("/:id/results/:attachmentId", rp(db, "UPLOAD_EXAM_RESULTS"), rp(db, "DELETE_PATIENT_ATTACHMENT"), envelope.Handle(examOrderHandler.DeleteExamResult))
			examOrderGroup.POST("/:id/review", rp(db, "REVIEW_EXAM_RESULTS"), envelope.Handle(examOrderHandler.ReviewExamResults))
		}

		registerAppointmentRoutes(clinicalGroup.Group("/appointments"), db, appointmentHandler)
	}
}

// registerVitalSignsRoutes mounts a medical record's vital signs. Triage staff
// record them with RECORD_VITAL_SIGNS without being able to edit the record;
// the doctor keeps doing it as part of editing the record.
func registerVitalSignsRoutes(recordGroup *gin.RouterGroup, db *gorm.DB, h *clinical_handlers.VitalSignsHandler) {
	anyOf := subscription_middleware.RequireAnyPermission
	recordGroup.PUT("/:id/vital-signs", anyOf(db, "UPDATE_MEDICAL_RECORD", "RECORD_VITAL_SIGNS"), envelope.Handle(h.UpsertVitalSigns))
	recordGroup.GET("/:id/vital-signs", anyOf(db, "READ_MEDICAL_RECORD", "RECORD_VITAL_SIGNS"), envelope.Handle(h.GetVitalSigns))
}

// registerAppointmentRoutes mounts the agenda and waiting room: reading needs
// READ_APPOINTMENT, any change (create, edit, status, delete) MANAGE_APPOINTMENT.
func registerAppointmentRoutes(group *gin.RouterGroup, db *gorm.DB, h *clinical_handlers.AppointmentHandler) {
	rp := subscription_middleware.RequirePermission
	read, manage := rp(db, "READ_APPOINTMENT"), rp(db, "MANAGE_APPOINTMENT")
	group.GET("", read, envelope.Handle(h.GetAppointments))
	group.GET("/today", read, envelope.Handle(h.GetTodayAppointments))
	group.GET("/:id", read, envelope.Handle(h.GetAppointment))
	group.GET("/patient/:id", read, envelope.Handle(h.GetPatientAppointments))
	group.POST("", manage, envelope.Handle(h.CreateAppointment))
	group.PUT("/:id", manage, envelope.Handle(h.UpdateAppointment))
	group.PUT("/:id/status", manage, envelope.Handle(h.UpdateStatus))
	group.DELETE("/:id", manage, envelope.Handle(h.DeleteAppointment))
}

// attachmentFiles is the encrypted store for patient attachments, or nil when
// ATTACHMENT_ENCRYPTION_KEY is missing or invalid: the attachment routes then
// answer 503 and the rest of the API keeps working.
func attachmentFiles() tenantfiles.Store {
	files, err := tenantfiles.EncryptedFromEnv(tenantFiles)
	if err != nil {
		logger.Log.Error("patient attachments disabled: "+tenantfiles.AttachmentKeyEnv+" is missing or invalid", zap.Error(err))
		return nil
	}
	return files
}
