package clinical_templates

import (
	"pengi-med-saas/core/pdfsign"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// PrescriptionData is the model passed to prescription_template.html.
type PrescriptionData struct {
	DoctorName          string
	DoctorRegistry      string // professional registry (Senescyt/MSP); printed when set
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
	DoctorRegistry  string // professional registry (Senescyt/MSP); printed when set
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
	DoctorRegistry  string // professional registry (Senescyt/MSP); printed when set
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

// ExamOrderData is the model passed to exam_order_template.html.
type ExamOrderData struct {
	TradeName       string
	DoctorName      string
	DoctorRegistry  string // professional registry (Senescyt/MSP); printed when set
	Date            string
	Code            string // ORD-000123
	PatientName     string
	PatientDocument string
	PatientAge      int
	PatientPhone    string
	Urgent          bool
	Priority        string // "Rutina" | "Urgente"
	Diagnoses       []clinical_models.DiagnosisItem
	DestinationLab  string
	Notes           string
	Groups          []ExamOrderGroup
	Signature       *pdfsign.Stamp // nil until the order is signed
}

// ExamOrderGroup is the requested exams of one category ("Laboratorio").
type ExamOrderGroup struct {
	Category  string
	Subgroups []ExamOrderSubgroup
}

// ExamOrderSubgroup is the requested exams of one subgroup ("Hematología");
// Name is empty for exams without a subgroup.
type ExamOrderSubgroup struct {
	Name  string
	Exams []ExamOrderExam
}

// ExamOrderExam is one requested exam with its indications for the patient.
type ExamOrderExam struct {
	Name        string
	Indications string
}
