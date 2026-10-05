package clinical_templates

import (
	"time"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/utils"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// The clinical printable documents. File names are also the names tenant
// templates are stored under: never rename them.
var (
	Prescription = pdfrender.Document{
		ID:       "prescription",
		File:     "prescription_template.html",
		Defaults: FS,
		Paper:    utils.A5Landscape,
		Feature:  "clinical",
		Sample:   samplePrescription(sampleSignature),
		Variants: []any{samplePrescription(nil)},
		Required: signatureRequired,
	}
	Report = pdfrender.Document{
		ID:       "medical_report",
		File:     "medical_report_template.html",
		Defaults: FS,
		Paper:    utils.A4Portrait,
		Feature:  "clinical",
		Sample:   sampleReport(sampleSignature),
		Variants: []any{sampleReport(nil), ReportData{Consultations: []ReportConsultation{{Date: "03/02/2025", Motive: "Control", Summary: "Paciente estable."}}}},
		Required: signatureRequired,
	}
	Certificate = pdfrender.Document{
		ID:       "medical_certificate",
		File:     "medical_certificate_template.html",
		Defaults: FS,
		Paper:    utils.A4Portrait,
		Feature:  "clinical",
		Sample:   sampleCertificate(sampleSignature),
		Variants: []any{sampleCertificate(nil), CertificateData{}},
		Required: signatureRequired,
	}
	ExamOrder = pdfrender.Document{
		ID:       "exam_order",
		File:     "exam_order_template.html",
		Defaults: FS,
		Paper:    utils.A4Portrait,
		Feature:  "clinical",
		Sample:   sampleExamOrder(sampleSignature),
		Variants: []any{sampleExamOrder(nil), ExamOrderData{}},
		Required: signatureRequired,
	}
)

// sampleSignature is a stamp like the one a signed document carries.
var sampleSignature = mustPreviewStamp("MARÍA FERNANDA ANDRADE LÓPEZ")

// A signed document must show its electronic signature: the QR and the signer.
var signatureRequired = []pdfrender.RequiredField{
	{Field: ".Signature.QR", Value: string(sampleSignature.QR)},
	{Field: ".Signature.Name", Value: sampleSignature.Name},
}

func mustPreviewStamp(name string) *pdfsign.Stamp {
	stamp, err := pdfsign.PreviewStamp(name, pdfsign.Options{
		Reason:   "Documento médico",
		Location: "Quito, Ecuador",
		SignedAt: time.Date(2026, 9, 15, 10, 30, 0, 0, time.FixedZone("ECT", -5*3600)),
	})
	if err != nil {
		panic(err)
	}
	return stamp
}

func samplePrescription(signature *pdfsign.Stamp) PrescriptionData {
	return PrescriptionData{
		DoctorName:      "María Fernanda Andrade López",
		Date:            "15/09/2026",
		PatientName:     "Juan Carlos Pérez Mora",
		PatientDocument: "1712345678",
		PatientAge:      42,
		MedicalRecordID: 1284,
		Diagnosis:       "J02.9 Faringitis aguda, no especificada",
		PrescriptionContent: "Amoxicilina 500 mg — 1 cápsula cada 8 horas por 7 días\n" +
			"Paracetamol 500 mg — 1 tableta cada 6 horas si hay fiebre o dolor\n" +
			"Ibuprofeno 400 mg — 1 tableta cada 8 horas por 3 días, después de las comidas",
		Indications: "Abundantes líquidos, reposo relativo por 48 horas. Volver a control en 7 días o antes si persiste la fiebre.",
		Phone:       "0991234567",
		TradeName:   "Consultorio Médico Andrade",
		Address:     "Av. Amazonas N34-120 y Av. República, Quito",
		Signature:   signature,
	}
}

func sampleReport(signature *pdfsign.Stamp) ReportData {
	return ReportData{
		TradeName:       "Consultorio Médico Andrade",
		DoctorName:      "María Fernanda Andrade López",
		Date:            "15/09/2026 10:30",
		PatientName:     "Juan Carlos Pérez Mora",
		PatientDocument: "1712345678",
		PatientAge:      42,
		PatientPhone:    "0991234567",
		Plan:            "Completar antibioticoterapia y control en una semana. Solicitar biometría hemática si persiste la fiebre.",
		Signature:       signature,
		Consultations: []ReportConsultation{
			{
				Date:        "08/09/2026",
				Motive:      "Dolor de garganta y fiebre de 2 días de evolución",
				VisitType:   "Primera vez",
				Observation: "Paciente refiere odinofagia intensa y malestar general.",
				Subjective:  "Dolor al tragar, fiebre no cuantificada, escalofríos.",
				Objective:   "Faringe hiperémica con exudado amigdalino bilateral. Adenopatías cervicales dolorosas.",
				Assessment:  "Faringoamigdalitis aguda de probable origen bacteriano.",
				Plan:        "Antibioticoterapia, analgésicos y control en 7 días.",
				APP:         "Hipertensión arterial controlada.",
				APF:         "Padre con diabetes mellitus tipo 2.",
				APQX:        "Apendicectomía (2010).",
				Allergies:   "Penicilina: no. Sulfas: sí.",
				Diagnoses:   []clinical_models.DiagnosisItem{{Code: "J03.9", Title: "Amigdalitis aguda, no especificada"}},
				VitalSigns: []ReportVitalSign{
					{Label: "Peso", Value: "78.5 kg"},
					{Label: "Talla", Value: "172 cm"},
					{Label: "Presión arterial", Value: "130/85 mmHg"},
					{Label: "Temperatura", Value: "38.4 °C"},
					{Label: "Frecuencia cardíaca", Value: "92 lpm"},
				},
				Prescription: &clinical_models.MedicalReportPrescription{
					Indications: "Abundantes líquidos y reposo relativo.",
					Items: []clinical_models.MedicalReportPrescriptionItem{
						{Medication: "Amoxicilina 500 mg", Dose: "1 cápsula", Frequency: "Cada 8 horas", Duration: "7 días"},
						{Medication: "Paracetamol 500 mg", Dose: "1 tableta", Frequency: "Cada 6 horas", Duration: "Si hay fiebre", Notes: "No exceder 4 g al día"},
					},
				},
			},
			{
				Date:       "15/09/2026",
				Motive:     "Control de faringoamigdalitis",
				VisitType:  "Subsecuente",
				Subjective: "Mejoría del dolor, persiste febrícula vespertina.",
				Objective:  "Faringe levemente hiperémica, sin exudado.",
				Assessment: "Faringoamigdalitis en resolución.",
				Plan:       "Completar tratamiento.",
				Diagnoses:  []clinical_models.DiagnosisItem{{Code: "J03.9", Title: "Amigdalitis aguda, no especificada"}},
				VitalSigns: []ReportVitalSign{
					{Label: "Peso", Value: "78 kg"},
					{Label: "Temperatura", Value: "37.6 °C"},
					{Label: "Frecuencia cardíaca", Value: "80 lpm"},
					{Label: "Saturación O2", Value: "98 %"},
				},
			},
		},
	}
}

func sampleCertificate(signature *pdfsign.Stamp) CertificateData {
	return CertificateData{
		TradeName:       "Consultorio Médico Andrade",
		DoctorName:      "María Fernanda Andrade López",
		Date:            "15/09/2026",
		PatientName:     "Juan Carlos Pérez Mora",
		PatientDocument: "1712345678",
		PatientAge:      42,
		PatientPhone:    "0991234567",
		Diagnosis:       "J03.9 Amigdalitis aguda, no especificada",
		Observations:    "El paciente requiere reposo domiciliario y no debe realizar actividad laboral durante el periodo indicado.",
		RestText:        "Del 15/09/2026 al 17/09/2026 (3 día(s))",
		Signature:       signature,
	}
}

func sampleExamOrder(signature *pdfsign.Stamp) ExamOrderData {
	return ExamOrderData{
		TradeName:       "Consultorio Médico Andrade",
		DoctorName:      "María Fernanda Andrade López",
		Date:            "15/09/2026",
		Code:            "ORD-000123",
		PatientName:     "Juan Carlos Pérez Mora",
		PatientDocument: "1712345678",
		PatientAge:      42,
		PatientPhone:    "0991234567",
		Urgent:          true,
		Priority:        "Urgente",
		Diagnoses:       []clinical_models.DiagnosisItem{{Code: "E11.9", Title: "Diabetes mellitus tipo 2, sin complicaciones"}},
		DestinationLab:  "Laboratorio Clínico Pasteur",
		Notes:           "Paciente con control trimestral. Traer resultados a la próxima consulta.",
		Groups: []ExamOrderGroup{
			{Category: "Laboratorio", Subgroups: []ExamOrderSubgroup{
				{Name: "Hematología", Exams: []ExamOrderExam{{Name: "Biometría hemática completa"}}},
				{Name: "Química sanguínea", Exams: []ExamOrderExam{
					{Name: "Glucosa en ayunas", Indications: "Ayuno de 8 horas."},
					{Name: "Hemoglobina glicosilada (HbA1c)"},
					{Name: "Colesterol total", Indications: "Ayuno de 12 horas."},
				}},
				{Name: "Orina", Exams: []ExamOrderExam{{Name: "Microalbuminuria", Indications: "Primera orina de la mañana."}}},
			}},
			{Category: "Imagen", Subgroups: []ExamOrderSubgroup{
				{Name: "Ecografía", Exams: []ExamOrderExam{{Name: "Ecografía abdominal", Indications: "Ayuno de 8 horas."}}},
			}},
			{Category: "Otros", Subgroups: []ExamOrderSubgroup{
				{Exams: []ExamOrderExam{{Name: "Electrocardiograma"}}},
			}},
		},
		Signature: signature,
	}
}
