package clinical_handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"pengi-med-saas/core/tenantdb"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/utils"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
)

type DownloadRecordHandler struct {
	db *gorm.DB
}

func NewDownloadRecordHandler(db *gorm.DB) *DownloadRecordHandler {
	return &DownloadRecordHandler{db: db}
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
	pdfBytes, err := generatePrescriptionPDF(h.db, c, &record, &patient)
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

func generatePrescriptionPDF(db *gorm.DB, c *gin.Context, record *clinical_models.MedicalRecord, patient *clinical_models.Patient) ([]byte, error) {
	// Attempt to find company information (for header & footer)
	tenantID, _ := c.Get("tenant_id")
	var company company_models.Company
	db.Where("tenant_id = ?", tenantID).First(&company)

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
	}

	// Use custom template if tenant has one, otherwise fall back to default
	customPath := filepath.Join("storage", "tenants", fmt.Sprint(tenantID), "prescription_template.html")
	tmplPath := "features/clinical/templates/prescription_template.html"
	if _, err := os.Stat(customPath); err == nil {
		tmplPath = customPath
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error loading prescription template: %w", err)
	}

	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, data)
	if err != nil {
		return nil, fmt.Errorf("error rendering prescription template: %w", err)
	}

	// Use gotenberg locally on dev network
	gotenbergURL := os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		gotenbergURL = "http://gotenberg:3000"
	}

	client := utils.NewGotenbergClient(gotenbergURL)
	return client.GeneratePDFFromHTML(htmlBuffer.String())
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}
