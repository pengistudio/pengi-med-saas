package clinical_handlers

import (
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
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	signature_services "pengi-med-saas/features/signatures/services"
)

type DownloadRecordHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	renderer *pdfrender.Renderer
	signer   *signature_services.Signer
	files    tenantfiles.Store
}

func NewDownloadRecordHandler(db *gorm.DB, logger *zap.Logger, renderer *pdfrender.Renderer, signer *signature_services.Signer, files tenantfiles.Store) *DownloadRecordHandler {
	return &DownloadRecordHandler{db: db, logger: logger, renderer: renderer, signer: signer, files: files}
}

// ─── PRESCRIPTION DOWNLOAD VIA GOTENBERG ──────────────────────────────────────

type PrescriptionTemplateData struct {
	DoctorName          string
	Date                string
	PatientName         string
	PatientDocument     string
	PatientAge          int
	MedicalRecordID     uint
	Diagnosis           string
	PrescriptionContent string
	Indications         string
	Phone               string
	TradeName           string
	Address             string
	Signature           *pdfsign.Stamp
}

// DownloadPrescription generates and downloads a prescription PDF using Gotenberg
func (h *DownloadRecordHandler) DownloadPrescription(c *gin.Context) {
	idParam := c.Param("id")
	recordID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "Invalid medical record ID format", core_errors.ErrClinicalInvalidRequest))
		return
	}

	// 1. Fetch Medical Record
	var record clinical_models.MedicalRecord
	err = tenantdb.For(c, h.db).
		Preload("Prescription").
		First(&record, recordID).Error

	if err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "Medical record not found", core_errors.ErrClinicalRecordNotFound))
		return
	}

	if record.Prescription == nil || (record.Prescription.Content == "" && record.Prescription.Indications == "") {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "This record has no prescription", core_errors.ErrClinicalRecordNotFound))
		return
	}

	// 2. Fetch Patient Data
	var patient clinical_models.Patient
	err = tenantdb.For(c, h.db).First(&patient, record.PatientID).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "Error retrieving patient data", core_errors.ErrClinicalPatientNotFound))
		return
	}

	audit.RecordAccess(h.db, c, "medical_records", record.ID, &record.PatientID)

	// 3. Generate PDF
	pdfBytes, err := storedOrRendered(c, h.files, record.Prescription.DocumentSignature, func() ([]byte, error) {
		return generatePrescriptionPDF(h.db, h.renderer, c, &record, &patient, nil)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalReportGenerateError))
		return
	}

	// 4. Set Headers
	lastName := strings.ReplaceAll(strings.ToLower(patient.LastName), " ", "_")
	firstName := strings.ReplaceAll(strings.ToLower(patient.FirstName), " ", "_")
	dateStr := record.Date.Format("20060102")
	fileName := fmt.Sprintf("receta_%s_%s_%s.pdf", lastName, firstName, dateStr)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func generatePrescriptionPDF(db *gorm.DB, renderer *pdfrender.Renderer, c *gin.Context, record *clinical_models.MedicalRecord, patient *clinical_models.Patient, stamp *pdfsign.Stamp) ([]byte, error) {
	// Attempt to find company information (for header & footer)
	var company company_models.Company
	tenantdb.For(c, db).First(&company)

	tradeName := "Consultorio Médico"
	if company.TradeName != "" {
		tradeName = company.TradeName
	}

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

	diagnosis := patient.Diagnosis
	if diagnosis == "" {
		diagnosis = "___________________________"
	}

	phone := patient.Phone

	// Calculate age
	age := calculateAge(patient.BirthDate)
	if patient.BirthDate.IsZero() {
		age = 0
	}

	data := PrescriptionTemplateData{
		DoctorName:          doctorName,
		Date:                record.Date.Format("02/01/2006"),
		PatientName:         fullName,
		PatientDocument:     patient.Document,
		PatientAge:          age,
		MedicalRecordID:     record.ID,
		Diagnosis:           diagnosis,
		PrescriptionContent: record.Prescription.Content,
		Indications:         record.Prescription.Indications,
		Phone:               phone,
		TradeName:           tradeName,
		Address:             "Ecuador", // Default, as location isn't currently in models
		Signature:           stamp,
	}

	return renderer.Render(tenantdb.TenantID(c), "prescription_template.html", data, utils.A5Landscape)
}

// SignPrescription signs the record's prescription with the current user's certificate.
func (h *DownloadRecordHandler) SignPrescription(c *gin.Context) envelope.Response {
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var record clinical_models.MedicalRecord
	if err := tenantdb.For(c, h.db).Preload("Prescription").First(&record, recordID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if record.Prescription == nil || (record.Prescription.Content == "" && record.Prescription.Indications == "") {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if record.Prescription.IsSigned() {
		return signature_services.AlreadySignedResponse()
	}
	var patient clinical_models.Patient
	if err := tenantdb.For(c, h.db).First(&patient, record.PatientID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}

	sig, err := signDocument(c, h.db, h.signer, h.files, &clinical_models.Prescription{}, "prescription", record.Prescription.ID, "Receta médica",
		func(stamp *pdfsign.Stamp) ([]byte, error) {
			return generatePrescriptionPDF(h.db, h.renderer, c, &record, &patient, stamp)
		})
	if err != nil {
		if errors.Is(err, errAlreadySigned) {
			return signature_services.AlreadySignedResponse()
		}
		h.logger.Error("Failed to sign prescription", zap.Error(err))
		return signature_services.ErrorResponse(err)
	}
	record.Prescription.DocumentSignature = sig
	return envelope.SuccessResponse(record.Prescription, "signature.document.sign.success")
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}
	return age
}
