package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/utils"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	message_cache "pengi-med-saas/i18n/cache"
	"strconv"
)

type DownloadRecordHandler struct {
	db *gorm.DB
}

func NewDownloadRecordHandler(db *gorm.DB) *DownloadRecordHandler {
	return &DownloadRecordHandler{db: db}
}

// DownloadPatientReport generates and downloads a comprehensive PDF report for a patient
func (h *DownloadRecordHandler) DownloadPatientReport(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrClinicalInvalidRequest))
		return
	}

	// Fetch patient with all related data (scoped by tenant)
	var patient clinical_models.Patient
	err = h.db.Scopes(tenant_middleware.TenantScope(c)).
		Preload("MedicalRecords", func(db *gorm.DB) *gorm.DB {
			return db.Order("date DESC")
		}).
		Preload("MedicalRecords.SOAPRecord").
		Preload("MedicalRecords.Prescription").
		Preload("MedicalRecords.Appointment").
		First(&patient, id).Error

	if err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "Error obteniendo el paciente", core_errors.ErrClinicalPatientNotFound))
		return
	}

	// Generate PDF
	lang, _ := c.Get("lang")
	langStr, _ := lang.(string)
	if langStr == "" {
		langStr = "es"
	}
	reportTitle := message_cache.Get(langStr, "clinical.patient.report.title")
	pdfBytes, err := generatePatientReportPDF(&patient, reportTitle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "Error generando el PDF", core_errors.ErrClinicalReportGenerateError))
		return
	}

	// Set headers for download
	// Create filename with format: informe_[apellido]_[nombre]_[cedula].pdf
	// Replace spaces with underscores and convert to lowercase for consistency
	lastName := strings.ReplaceAll(strings.ToLower(patient.LastName), " ", "_")
	firstName := strings.ReplaceAll(strings.ToLower(patient.FirstName), " ", "_")
	fileName := fmt.Sprintf("informe_%s_%s_%s.pdf", lastName, firstName, patient.Document)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func generatePatientReportPDF(patient *clinical_models.Patient, reportTitle string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Set footer function to appear on all pages
	pdf.SetFooterFunc(func() {
		pdf.SetY(-20)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 5, tr("Este documento es confidencial y de uso exclusivo medico"), "0", 0, "C", false, 0, "")
		pdf.Ln(4)
		pdf.CellFormat(0, 5, fmt.Sprintf("Generado el %s", time.Now().Format("02/01/2006 15:04")), "0", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	// === HEADER ===
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(41, 128, 185)
	pdf.Cell(0, 10, tr(reportTitle))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 6, tr(fmt.Sprintf("Fecha de generacion: %s", time.Now().Format("02/01/2006 15:04"))))
	pdf.Ln(12)

	// === PATIENT DATA ===
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(41, 128, 185)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 8, tr("DATOS DEL PACIENTE"), "0", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 10)
	fullName := "N/A"
	if patient.FullName != nil {
		fullName = *patient.FullName
	}
	addField(pdf, tr, "Nombre completo:", fullName)
	addField(pdf, tr, "Cedula:", patient.Document)
	if patient.Phone != "" {
		addField(pdf, tr, "Telefono:", patient.Phone)
	}
	if !patient.BirthDate.IsZero() {
		addField(pdf, tr, "Fecha de nacimiento:", patient.BirthDate.Format("02/01/2006"))
	}
	if patient.Gender != "" {
		addField(pdf, tr, "Genero:", patient.Gender)
	}
	if patient.Insurance != "" {
		addField(pdf, tr, "Seguro:", patient.Insurance)
	}
	if patient.Medic != "" {
		addField(pdf, tr, "Medico tratante:", patient.Medic)
	}
	if patient.Diagnosis != "" {
		addField(pdf, tr, "Diagnostico:", patient.Diagnosis)
	}
	if patient.APP != "" {
		addField(pdf, tr, "APP:", patient.APP)
	}
	if patient.APF != "" {
		addField(pdf, tr, "APF:", patient.APF)
	}
	if patient.APQX != "" {
		addField(pdf, tr, "APQX:", patient.APQX)
	}
	if patient.Institution != "" {
		addField(pdf, tr, "Institucion:", patient.Institution)
	}
	pdf.Ln(5)

	// === SUMMARY ===
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(46, 204, 113)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 8, tr("RESUMEN"), "0", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 10)
	addField(pdf, tr, "Total de consultas:", fmt.Sprintf("%d", len(patient.MedicalRecords)))
	if len(patient.MedicalRecords) > 0 {
		lastRecord := patient.MedicalRecords[0]
		addField(pdf, tr, "Ultima consulta:", lastRecord.Date.Format("02/01/2006"))
		if lastRecord.Appointment != nil {
			addField(pdf, tr, "Proxima cita:", lastRecord.Appointment.Date.Format("02/01/2006"))
		}
	}
	pdf.Ln(5)

	// === MEDICAL RECORDS HISTORY ===
	if len(patient.MedicalRecords) > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetFillColor(230, 126, 34)
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(0, 8, tr("HISTORIAL DE CONSULTAS MEDICAS"), "0", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(4)

		for i, record := range patient.MedicalRecords {
			// Check if we need a new page
			if pdf.GetY() > 250 {
				pdf.AddPage()
			}

			pdf.SetFont("Arial", "B", 12)
			pdf.SetTextColor(41, 128, 185)
			pdf.Cell(0, 6, tr(fmt.Sprintf("Consulta #%d - %s", len(patient.MedicalRecords)-i, record.Date.Format("02/01/2006"))))
			pdf.SetTextColor(0, 0, 0)
			pdf.Ln(7)

			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(35, 6, tr("Motivo:"))
			pdf.SetFont("Arial", "", 10)
			pdf.MultiCell(0, 6, tr(record.Motive), "0", "L", false)
			pdf.Ln(2)

			// SOAP Record
			if record.SOAPRecord.ID != 0 {
				pdf.SetFont("Arial", "B", 11)
				pdf.SetTextColor(52, 152, 219)
				pdf.Cell(0, 6, tr("Registro SOAP:"))
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)

				addSOAPField(pdf, tr, "S (Subjetivo):", record.SOAPRecord.Subjective)
				addSOAPField(pdf, tr, "O (Objetivo):", record.SOAPRecord.Objective)
				addSOAPField(pdf, tr, "A (Analisis):", record.SOAPRecord.Assessment)
				addSOAPField(pdf, tr, "P (Plan):", record.SOAPRecord.Plan)
			}

			// Diagnoses
			var diagnoses []clinical_models.DiagnosisItem
			if json.Unmarshal(record.Diagnoses, &diagnoses) == nil && len(diagnoses) > 0 {
				pdf.SetFont("Arial", "B", 11)
				pdf.SetTextColor(41, 128, 185)
				pdf.Cell(0, 6, tr("Diagnosticos CIE-11:"))
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)
				for _, d := range diagnoses {
					addSOAPField(pdf, tr, d.Code+":", d.Title)
				}
			}

			// Observations
			if record.Observation != "" {
				pdf.SetFont("Arial", "B", 10)
				pdf.Cell(35, 6, tr("Observaciones:"))
				pdf.SetFont("Arial", "", 10)
				pdf.MultiCell(0, 6, tr(record.Observation), "0", "L", false)
				pdf.Ln(2)
			}

			// Prescription
			if record.Prescription != nil && (record.Prescription.Content != "" || record.Prescription.Indications != "") {
				pdf.SetFont("Arial", "B", 11)
				pdf.SetTextColor(52, 152, 219)
				pdf.Cell(0, 6, tr("Receta Medica:"))
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)

				if record.Prescription.Content != "" {
					addSOAPField(pdf, tr, "Contenido:", record.Prescription.Content)
				}
				if record.Prescription.Indications != "" {
					addSOAPField(pdf, tr, "Indicaciones:", record.Prescription.Indications)
				}
			}

			// Linked appointment
			if record.Appointment != nil {
				pdf.SetFont("Arial", "B", 10)
				pdf.SetTextColor(46, 204, 113)
				pdf.Cell(0, 6, tr(fmt.Sprintf("Cita enlazada: %s (%s-%s)", record.Appointment.Date.Format("02/01/2006"), record.Appointment.StartTime, record.Appointment.EndTime)))
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)
			}

			// Separator between consultations
			pdf.SetDrawColor(220, 220, 220)
			pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
			pdf.Ln(8)
		}
	}

	// Convert to bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Helper function to add a labeled field
func addField(pdf *gofpdf.Fpdf, tr func(string) string, label, value string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(50, 6, tr(label))
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 6, tr(value))
	pdf.Ln(6)
}

// Helper function to add SOAP fields with proper formatting
func addSOAPField(pdf *gofpdf.Fpdf, tr func(string) string, label, value string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(30, 5, tr(label))
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(0, 5, tr(value), "0", "L", false)
	pdf.Ln(1)
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
	err = h.db.Scopes(tenant_middleware.TenantScope(c)).
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
	err = h.db.First(&patient, record.PatientID).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "Error retrieving patient data", core_errors.ErrClinicalPatientNotFound))
		return
	}

	// 3. Generate PDF
	pdfBytes, err := generatePrescriptionPDF(h.db, c, &record, &patient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, envelope.ErrorResponse(http.StatusInternalServerError, "Error generating PDF: "+err.Error(), core_errors.ErrClinicalReportGenerateError))
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
