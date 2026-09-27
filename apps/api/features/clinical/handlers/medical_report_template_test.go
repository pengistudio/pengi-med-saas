package clinical_handlers

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	clinical_models "pengi-med-saas/features/clinical/models"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

func renderMedicalReportTemplate(t *testing.T, data medicalReportTemplateData) string {
	t.Helper()
	tmpl, err := template.ParseFS(clinical_templates.FS, "medical_report_template.html")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestMedicalReportTemplate_RendersFullConsultation(t *testing.T) {
	weight := 70.5
	heartRate := uint(72)
	html := renderMedicalReportTemplate(t, medicalReportTemplateData{
		Consultations: []medicalReportConsultationView{{
			Date:        "12/08/2026",
			Motive:      "Dolor lumbar",
			VisitType:   medicalReportVisitType("first"),
			Observation: "obs-text",
			Subjective:  "subj-text",
			Objective:   "obj-text",
			Assessment:  "assess-text",
			Plan:        "plan-text",
			Allergies:   "penicilina",
			Diagnoses:   []clinical_models.DiagnosisItem{{Code: "M54.5", Title: "Lumbago"}},
			VitalSigns: medicalReportVitalSigns(&clinical_models.MedicalReportVitalSigns{
				Weight: &weight, HeartRate: &heartRate, BloodPressure: "120/80",
			}),
			Prescription: &clinical_models.MedicalReportPrescription{
				Indications: "reposo",
				Items:       []clinical_models.MedicalReportPrescriptionItem{{Medication: "Ibuprofeno 400mg", Dose: "1 tableta"}},
			},
		}},
	})

	for _, want := range []string{
		"Primera vez", "obs-text", "subj-text", "obj-text", "assess-text", "plan-text",
		"penicilina", "M54.5", "Lumbago", "70.5 kg", "72 lpm", "120/80 mmHg",
		"Ibuprofeno 400mg", "reposo",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered report missing %q", want)
		}
	}
	if strings.Contains(html, "Temperatura") {
		t.Errorf("empty vital sign should be hidden")
	}
}

func TestMedicalReportTemplate_LegacySummaryOnly(t *testing.T) {
	html := renderMedicalReportTemplate(t, medicalReportTemplateData{
		Consultations: []medicalReportConsultationView{{Date: "01/01/2026", Motive: "Control", Summary: "legacy-summary"}},
	})
	if !strings.Contains(html, "legacy-summary") {
		t.Errorf("legacy summary not rendered")
	}
	for _, hidden := range []string{"Signos vitales", "Subjetivo", "Receta", "Antecedentes"} {
		if strings.Contains(html, hidden) {
			t.Errorf("empty section %q should be hidden", hidden)
		}
	}
}
