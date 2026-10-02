package clinical_handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantdb"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/tenantfiles"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	clinical_templates "pengi-med-saas/features/clinical/templates"
	company_models "pengi-med-saas/features/companies/models"
	signature_services "pengi-med-saas/features/signatures/services"
	auth_middleware "pengi-med-saas/features/users/middleware"
)

type MedicalDocumentHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	mailer   *mailer.Mailer
	renderer *pdfrender.Renderer
	signer   *signature_services.Signer
	files    tenantfiles.Store
}

func NewMedicalDocumentHandler(db *gorm.DB, logger *zap.Logger, mailer *mailer.Mailer, renderer *pdfrender.Renderer, signer *signature_services.Signer, files tenantfiles.Store) *MedicalDocumentHandler {
	return &MedicalDocumentHandler{db: db, logger: logger, mailer: mailer, renderer: renderer, signer: signer, files: files}
}

func parseFlexibleDate(s string) (time.Time, error) {
	for _, format := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}

// ─── MEDICAL REPORT ────────────────────────────────────────────────────────

func (h *MedicalDocumentHandler) CreateMedicalReport(c *gin.Context) envelope.Response {
	patientIDParam := c.Param("id")
	patientID, err := strconv.ParseUint(patientIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.CreateMedicalReportDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Invalid create medical report request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := tenantdb.For(c, h.db).First(&patient, patientID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}

	entries := make([]clinical_models.MedicalReportConsultationEntry, 0, len(dto.Consultations))
	for _, entry := range dto.Consultations {
		date, err := parseFlexibleDate(entry.Date)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		}
		entries = append(entries, clinical_models.MedicalReportConsultationEntry{
			MedicalRecordID: entry.MedicalRecordID,
			Date:            date,
			Motive:          entry.Motive,
			VisitType:       entry.VisitType,
			Observation:     entry.Observation,
			Subjective:      entry.Subjective,
			Objective:       entry.Objective,
			Assessment:      entry.Assessment,
			Plan:            entry.Plan,
			APP:             entry.APP,
			APF:             entry.APF,
			APQX:            entry.APQX,
			Allergies:       entry.Allergies,
			Diagnoses:       entry.Diagnoses,
			VitalSigns:      entry.VitalSigns,
			Prescription:    entry.Prescription,
		})
	}

	consultationsJSON, err := json.Marshal(entries)
	if err != nil {
		h.logger.Error("Failed to marshal report consultations", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalReportError)
	}

	report := &clinical_models.MedicalReport{
		TenantID:      tenantdb.TenantID(c),
		PatientID:     uint(patientID),
		Consultations: consultationsJSON,
		Plan:          dto.Plan,
	}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		report.GeneratedByID = uint(uid)
	}

	if err := tenantdb.For(c, h.db).Create(report).Error; err != nil {
		h.logger.Error("Failed to create medical report", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalReportError)
	}

	return envelope.SuccessResponse(report, "clinical.medical_report.create.success")
}

func (h *MedicalDocumentHandler) ListMedicalReports(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var reports []clinical_models.MedicalReport
	if err := tenantdb.For(c, h.db).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Find(&reports).Error; err != nil {
		h.logger.Error("Failed to list medical reports", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalReportError)
	}

	return envelope.SuccessResponse(reports, "clinical.medical_report.list.success")
}

func (h *MedicalDocumentHandler) DownloadMedicalReport(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest))
		return
	}

	var report clinical_models.MedicalReport
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&report, id).Error; err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound))
		return
	}

	audit.RecordAccess(h.db, c, "patients", report.PatientID, &report.PatientID)

	pdfBytes, err := storedOrRendered(c, h.files, report.DocumentSignature, func() ([]byte, error) {
		return h.generateMedicalReportPDF(c, &report, nil)
	})
	if err != nil {
		h.logger.Error("Failed to generate medical report PDF", zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalReportError))
		return
	}

	fileName := medicalDocumentFileName("informe", report.Patient)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (h *MedicalDocumentHandler) EmailMedicalReport(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.EmailMedicalDocumentDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var report clinical_models.MedicalReport
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&report, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}

	pdfBytes, err := storedOrRendered(c, h.files, report.DocumentSignature, func() ([]byte, error) {
		return h.generateMedicalReportPDF(c, &report, nil)
	})
	if err != nil {
		h.logger.Error("Failed to generate medical report PDF for email", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalReportError)
	}

	fileName := medicalDocumentFileName("informe", report.Patient)
	if err := h.mailer.SendMedicalDocumentEmail(dto.Email, "Informe Médico", "Informe Médico", fileName, pdfBytes); err != nil {
		h.logger.Error("Failed to email medical report", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalDocumentEmailError)
	}

	return envelope.SuccessResponse(nil, "clinical.medical_report.email.success")
}

func medicalReportVisitType(visitType string) string {
	switch visitType {
	case "first":
		return "Primera vez"
	case "followup":
		return "Subsecuente"
	}
	return ""
}

func medicalReportVitalSigns(v *clinical_models.MedicalReportVitalSigns) []clinical_templates.ReportVitalSign {
	if v == nil {
		return nil
	}
	var out []clinical_templates.ReportVitalSign
	if v.Weight != nil {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Peso", Value: fmt.Sprintf("%g kg", *v.Weight)})
	}
	if v.Height != nil {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Talla", Value: fmt.Sprintf("%g cm", *v.Height)})
	}
	if v.BloodPressure != "" {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Presión arterial", Value: v.BloodPressure + " mmHg"})
	}
	if v.Temperature != nil {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Temperatura", Value: fmt.Sprintf("%g °C", *v.Temperature)})
	}
	if v.HeartRate != nil {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Frecuencia cardíaca", Value: fmt.Sprintf("%d lpm", *v.HeartRate)})
	}
	if v.O2Saturation != nil {
		out = append(out, clinical_templates.ReportVitalSign{Label: "Saturación O2", Value: fmt.Sprintf("%d %%", *v.O2Saturation)})
	}
	return out
}

func (h *MedicalDocumentHandler) generateMedicalReportPDF(c *gin.Context, report *clinical_models.MedicalReport, stamp *pdfsign.Stamp) ([]byte, error) {
	var company company_models.Company
	tenantdb.For(c, h.db).First(&company)

	tradeName := "Consultorio Médico"
	if company.TradeName != "" {
		tradeName = company.TradeName
	}

	patient := report.Patient
	doctorName := patient.Medic
	if doctorName == "" {
		doctorName = "Médico Tratante"
	}

	fullName := "No especificado"
	if patient.FullName != nil && *patient.FullName != "" {
		fullName = *patient.FullName
	} else {
		fullName = strings.TrimSpace(patient.FirstName + " " + patient.LastName)
	}

	var entries []clinical_models.MedicalReportConsultationEntry
	_ = json.Unmarshal(report.Consultations, &entries)

	consultations := make([]clinical_templates.ReportConsultation, 0, len(entries))
	for _, entry := range entries {
		prescription := entry.Prescription
		if prescription != nil && prescription.Indications == "" && len(prescription.Items) == 0 {
			prescription = nil
		}
		consultations = append(consultations, clinical_templates.ReportConsultation{
			Date:         entry.Date.Format("02/01/2006"),
			Motive:       entry.Motive,
			VisitType:    medicalReportVisitType(entry.VisitType),
			Observation:  entry.Observation,
			Subjective:   entry.Subjective,
			Objective:    entry.Objective,
			Assessment:   entry.Assessment,
			Plan:         entry.Plan,
			APP:          entry.APP,
			APF:          entry.APF,
			APQX:         entry.APQX,
			Allergies:    entry.Allergies,
			Diagnoses:    entry.Diagnoses,
			VitalSigns:   medicalReportVitalSigns(entry.VitalSigns),
			Prescription: prescription,
			Summary:      entry.Summary,
		})
	}

	data := clinical_templates.ReportData{
		TradeName:       tradeName,
		DoctorName:      doctorName,
		Date:            report.CreatedAt.Format("02/01/2006 15:04"),
		PatientName:     fullName,
		PatientDocument: patient.Document,
		PatientAge:      patientAge(patient),
		PatientPhone:    patient.Phone,
		Consultations:   consultations,
		Plan:            report.Plan,
		Signature:       stamp,
	}

	return h.renderer.Render(tenantdb.TenantID(c), clinical_templates.Report, data)
}

// ─── MEDICAL CERTIFICATE ───────────────────────────────────────────────────

func (h *MedicalDocumentHandler) CreateMedicalCertificate(c *gin.Context) envelope.Response {
	patientIDParam := c.Param("id")
	patientID, err := strconv.ParseUint(patientIDParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.CreateMedicalCertificateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Invalid create medical certificate request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var patient clinical_models.Patient
	if err := tenantdb.For(c, h.db).First(&patient, patientID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}

	certificate := &clinical_models.MedicalCertificate{
		TenantID:     tenantdb.TenantID(c),
		PatientID:    uint(patientID),
		Diagnosis:    dto.Diagnosis,
		Observations: dto.Observations,
		RestDays:     dto.RestDays,
		RestFrom:     dto.RestFrom,
		RestTo:       dto.RestTo,
	}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		certificate.GeneratedByID = uint(uid)
	}

	if err := tenantdb.For(c, h.db).Create(certificate).Error; err != nil {
		h.logger.Error("Failed to create medical certificate", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalCertificateError)
	}

	return envelope.SuccessResponse(certificate, "clinical.medical_certificate.create.success")
}

func (h *MedicalDocumentHandler) ListMedicalCertificates(c *gin.Context) envelope.Response {
	patientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var certificates []clinical_models.MedicalCertificate
	if err := tenantdb.For(c, h.db).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Find(&certificates).Error; err != nil {
		h.logger.Error("Failed to list medical certificates", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalCertificateError)
	}

	return envelope.SuccessResponse(certificates, "clinical.medical_certificate.list.success")
}

func (h *MedicalDocumentHandler) DownloadMedicalCertificate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest))
		return
	}

	var certificate clinical_models.MedicalCertificate
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&certificate, id).Error; err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound))
		return
	}

	audit.RecordAccess(h.db, c, "patients", certificate.PatientID, &certificate.PatientID)

	pdfBytes, err := storedOrRendered(c, h.files, certificate.DocumentSignature, func() ([]byte, error) {
		return h.generateMedicalCertificatePDF(c, &certificate, nil)
	})
	if err != nil {
		h.logger.Error("Failed to generate medical certificate PDF", zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalCertificateError))
		return
	}

	fileName := medicalDocumentFileName("certificado", certificate.Patient)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (h *MedicalDocumentHandler) EmailMedicalCertificate(c *gin.Context) envelope.Response {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var dto clinical_dto.EmailMedicalDocumentDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	var certificate clinical_models.MedicalCertificate
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&certificate, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}

	pdfBytes, err := storedOrRendered(c, h.files, certificate.DocumentSignature, func() ([]byte, error) {
		return h.generateMedicalCertificatePDF(c, &certificate, nil)
	})
	if err != nil {
		h.logger.Error("Failed to generate medical certificate PDF for email", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalMedicalCertificateError)
	}

	fileName := medicalDocumentFileName("certificado", certificate.Patient)
	if err := h.mailer.SendMedicalDocumentEmail(dto.Email, "Certificado Médico", "Certificado Médico", fileName, pdfBytes); err != nil {
		h.logger.Error("Failed to email medical certificate", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalDocumentEmailError)
	}

	return envelope.SuccessResponse(nil, "clinical.medical_certificate.email.success")
}

func (h *MedicalDocumentHandler) generateMedicalCertificatePDF(c *gin.Context, certificate *clinical_models.MedicalCertificate, stamp *pdfsign.Stamp) ([]byte, error) {
	var company company_models.Company
	tenantdb.For(c, h.db).First(&company)

	tradeName := "Consultorio Médico"
	if company.TradeName != "" {
		tradeName = company.TradeName
	}

	patient := certificate.Patient
	doctorName := patient.Medic
	if doctorName == "" {
		doctorName = "Médico Tratante"
	}

	fullName := "No especificado"
	if patient.FullName != nil && *patient.FullName != "" {
		fullName = *patient.FullName
	} else {
		fullName = strings.TrimSpace(patient.FirstName + " " + patient.LastName)
	}

	restText := ""
	if certificate.RestFrom != nil && certificate.RestTo != nil {
		restText = fmt.Sprintf("Del %s al %s", certificate.RestFrom.Format("02/01/2006"), certificate.RestTo.Format("02/01/2006"))
	}
	if certificate.RestDays != nil {
		days := fmt.Sprintf("%d día(s)", *certificate.RestDays)
		if restText != "" {
			restText = fmt.Sprintf("%s (%s)", restText, days)
		} else {
			restText = days
		}
	}

	data := clinical_templates.CertificateData{
		TradeName:       tradeName,
		DoctorName:      doctorName,
		Date:            certificate.CreatedAt.Format("02/01/2006"),
		PatientName:     fullName,
		PatientDocument: patient.Document,
		PatientAge:      patientAge(patient),
		PatientPhone:    patient.Phone,
		Diagnosis:       certificate.Diagnosis,
		Observations:    certificate.Observations,
		RestText:        restText,
		Signature:       stamp,
	}

	return h.renderer.Render(tenantdb.TenantID(c), clinical_templates.Certificate, data)
}

// ─── ELECTRONIC SIGNATURE ──────────────────────────────────────────────────

// SignMedicalReport signs the report with the current user's certificate.
func (h *MedicalDocumentHandler) SignMedicalReport(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var report clinical_models.MedicalReport
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&report, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if report.IsSigned() {
		return signature_services.AlreadySignedResponse()
	}

	sig, err := signDocument(c, h.db, h.signer, h.files, &clinical_models.MedicalReport{}, "report", report.ID, "Informe médico",
		func(stamp *pdfsign.Stamp) ([]byte, error) { return h.generateMedicalReportPDF(c, &report, stamp) })
	if err != nil {
		return h.signErrorResponse(err)
	}
	report.DocumentSignature = sig
	return envelope.SuccessResponse(report, "signature.document.sign.success")
}

// SignMedicalCertificate signs the certificate with the current user's certificate.
func (h *MedicalDocumentHandler) SignMedicalCertificate(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var certificate clinical_models.MedicalCertificate
	if err := tenantdb.For(c, h.db).Preload("Patient").First(&certificate, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if certificate.IsSigned() {
		return signature_services.AlreadySignedResponse()
	}

	sig, err := signDocument(c, h.db, h.signer, h.files, &clinical_models.MedicalCertificate{}, "certificate", certificate.ID, "Certificado médico",
		func(stamp *pdfsign.Stamp) ([]byte, error) {
			return h.generateMedicalCertificatePDF(c, &certificate, stamp)
		})
	if err != nil {
		return h.signErrorResponse(err)
	}
	certificate.DocumentSignature = sig
	return envelope.SuccessResponse(certificate, "signature.document.sign.success")
}

func (h *MedicalDocumentHandler) signErrorResponse(err error) envelope.Response {
	if errors.Is(err, errAlreadySigned) {
		return signature_services.AlreadySignedResponse()
	}
	h.logger.Error("Failed to sign medical document", zap.Error(err))
	return signature_services.ErrorResponse(err)
}

// ─── SHARED HELPERS ────────────────────────────────────────────────────────

func patientAge(patient *clinical_models.Patient) int {
	if patient == nil || patient.BirthDate.IsZero() {
		return 0
	}
	return calculateAge(patient.BirthDate)
}

func medicalDocumentFileName(prefix string, patient *clinical_models.Patient) string {
	if patient == nil {
		return fmt.Sprintf("%s.pdf", prefix)
	}
	lastName := strings.ReplaceAll(strings.ToLower(patient.LastName), " ", "_")
	firstName := strings.ReplaceAll(strings.ToLower(patient.FirstName), " ", "_")
	return fmt.Sprintf("%s_%s_%s_%s.pdf", prefix, lastName, firstName, patient.Document)
}
