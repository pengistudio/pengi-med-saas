package clinical_templates

import (
	"pengi-med-saas/core/pdfsign"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// PrescriptionData is the model passed to prescription_template.html.
type PrescriptionData struct {
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
	Signature           *pdfsign.Stamp // nil until the prescription is signed
}

// ReportData is the model passed to medical_report_template.html.
type ReportData struct {
	TradeName       string
	DoctorName      string
	Date            string
	PatientName     string
	PatientDocument string
	PatientAge      int
	PatientPhone    string
	Consultations   []ReportConsultation
	Plan            string
	Signature       *pdfsign.Stamp // nil until the report is signed
}

// ReportConsultation is one consultation inside a medical report.
type ReportConsultation struct {
	Date         string
	Motive       string
	VisitType    string
	Observation  string
	Subjective   string
	Objective    string
	Assessment   string
	Plan         string
	APP          string
	APF          string
	APQX         string
	Allergies    string
	Diagnoses    []clinical_models.DiagnosisItem
	VitalSigns   []ReportVitalSign
	Prescription *clinical_models.MedicalReportPrescription
	Summary      string // only on reports created before the full snapshot
}

// ReportVitalSign is one measured vital sign, already formatted ("72 lpm").
type ReportVitalSign struct {
	Label string
	Value string
}

// CertificateData is the model passed to medical_certificate_template.html.
type CertificateData struct {
	TradeName       string
	DoctorName      string
	Date            string
	PatientName     string
	PatientDocument string
	PatientAge      int
	PatientPhone    string
	Diagnosis       string
	Observations    string
	RestText        string
	Signature       *pdfsign.Stamp // nil until the certificate is signed
}
