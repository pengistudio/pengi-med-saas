package clinical_handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
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
		c.JSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "Invalid patient ID format", core_errors.ErrAuthInvalidRequest))
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
		First(&patient, id).Error

	if err != nil {
		c.JSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "Error obteniendo el paciente", core_errors.ErrClinicalPatientNotFound))
		return
	}

	// Generate PDF
	pdfBytes, err := generatePatientReportPDF(&patient)
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

func generatePatientReportPDF(patient *clinical_models.Patient) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Set footer function to appear on all pages
	pdf.SetFooterFunc(func() {
		pdf.SetY(-20)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 5, "Este documento es confidencial y de uso exclusivo medico", "0", 0, "C", false, 0, "")
		pdf.Ln(4)
		pdf.CellFormat(0, 5, fmt.Sprintf("Generado el %s", time.Now().Format("02/01/2006 15:04")), "0", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	// === HEADER ===
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(41, 128, 185)
	pdf.Cell(0, 10, "INFORME MEDICO DEL PACIENTE")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 6, fmt.Sprintf("Fecha de generacion: %s", time.Now().Format("02/01/2006 15:04")))
	pdf.Ln(12)

	// === PATIENT DATA ===
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(41, 128, 185)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 8, "DATOS DEL PACIENTE", "0", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 10)
	fullName := "N/A"
	if patient.FullName != nil {
		fullName = *patient.FullName
	}
	addField(pdf, "Nombre completo:", fullName)
	addField(pdf, "Cedula:", patient.Document)
	if patient.Phone != "" {
		addField(pdf, "Telefono:", patient.Phone)
	}
	if !patient.BirthDate.IsZero() {
		addField(pdf, "Fecha de nacimiento:", patient.BirthDate.Format("02/01/2006"))
	}
	if patient.Gender != "" {
		addField(pdf, "Genero:", patient.Gender)
	}
	if patient.Insurance != "" {
		addField(pdf, "Seguro:", patient.Insurance)
	}
	if patient.Medic != "" {
		addField(pdf, "Medico tratante:", patient.Medic)
	}
	if patient.Diagnosis != "" {
		addField(pdf, "Diagnostico:", patient.Diagnosis)
	}
	if patient.APP != "" {
		addField(pdf, "APP:", patient.APP)
	}
	if patient.APF != "" {
		addField(pdf, "APF:", patient.APF)
	}
	if patient.APQX != "" {
		addField(pdf, "APQX:", patient.APQX)
	}
	if patient.Institution != "" {
		addField(pdf, "Institucion:", patient.Institution)
	}
	pdf.Ln(5)

	// === SUMMARY ===
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(46, 204, 113)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 8, "RESUMEN", "0", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 10)
	addField(pdf, "Total de consultas:", fmt.Sprintf("%d", len(patient.MedicalRecords)))
	if len(patient.MedicalRecords) > 0 {
		lastRecord := patient.MedicalRecords[0]
		addField(pdf, "Ultima consulta:", lastRecord.Date.Format("02/01/2006"))
		if lastRecord.NextAppointmentDate != nil {
			addField(pdf, "Proxima cita:", lastRecord.NextAppointmentDate.Format("02/01/2006"))
		}
	}
	pdf.Ln(5)

	// === MEDICAL RECORDS HISTORY ===
	if len(patient.MedicalRecords) > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetFillColor(230, 126, 34)
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(0, 8, "HISTORIAL DE CONSULTAS MEDICAS", "0", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(4)

		for i, record := range patient.MedicalRecords {
			// Check if we need a new page
			if pdf.GetY() > 250 {
				pdf.AddPage()
			}

			pdf.SetFont("Arial", "B", 12)
			pdf.SetTextColor(41, 128, 185)
			pdf.Cell(0, 6, fmt.Sprintf("Consulta #%d - %s", len(patient.MedicalRecords)-i, record.Date.Format("02/01/2006")))
			pdf.SetTextColor(0, 0, 0)
			pdf.Ln(7)

			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(35, 6, "Motivo:")
			pdf.SetFont("Arial", "", 10)
			pdf.MultiCell(0, 6, record.Motive, "0", "L", false)
			pdf.Ln(2)

			// SOAP Record
			if record.SOAPRecord.ID != 0 {
				pdf.SetFont("Arial", "B", 11)
				pdf.SetTextColor(52, 152, 219)
				pdf.Cell(0, 6, "Registro SOAP:")
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)

				addSOAPField(pdf, "S (Subjetivo):", record.SOAPRecord.Subjective)
				addSOAPField(pdf, "O (Objetivo):", record.SOAPRecord.Objective)
				addSOAPField(pdf, "A (Analisis):", record.SOAPRecord.Assessment)
				addSOAPField(pdf, "P (Plan):", record.SOAPRecord.Plan)
			}

			// Observations
			if record.Observation != "" {
				pdf.SetFont("Arial", "B", 10)
				pdf.Cell(35, 6, "Observaciones:")
				pdf.SetFont("Arial", "", 10)
				pdf.MultiCell(0, 6, record.Observation, "0", "L", false)
				pdf.Ln(2)
			}

			// Prescription
			if record.Prescription != nil && (record.Prescription.Content != "" || record.Prescription.Indications != "") {
				pdf.SetFont("Arial", "B", 11)
				pdf.SetTextColor(52, 152, 219)
				pdf.Cell(0, 6, "Receta Medica:")
				pdf.SetTextColor(0, 0, 0)
				pdf.Ln(6)

				if record.Prescription.Content != "" {
					addSOAPField(pdf, "Contenido:", record.Prescription.Content)
				}
				if record.Prescription.Indications != "" {
					addSOAPField(pdf, "Indicaciones:", record.Prescription.Indications)
				}
			}

			// Next appointment
			if record.NextAppointmentDate != nil {
				pdf.SetFont("Arial", "B", 10)
				pdf.SetTextColor(46, 204, 113)
				pdf.Cell(0, 6, fmt.Sprintf("Proxima cita: %s", record.NextAppointmentDate.Format("02/01/2006")))
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
func addField(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(50, 6, label)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 6, value)
	pdf.Ln(6)
}

// Helper function to add SOAP fields with proper formatting
func addSOAPField(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(30, 5, label)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(0, 5, value, "0", "L", false)
	pdf.Ln(1)
}
