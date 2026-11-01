package ai_services

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/ai"
	"pengi-med-saas/core/config"
	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/datatypes"
)

// Eval set of fictitious Spanish cases (testdata/summary_eval_cases.json),
// run against the real model before switching the production model:
//
//	AI_API_KEY=... go test ./features/ai/services -run TestSummaryEval -v
//
// AI_BASE_URL and AI_DEFAULT_SUMMARY_MODEL pick the provider and model. Each
// case lists terms the summary must mention (facts in the input) and terms
// it must not (identifiers, or facts absent from the input: hallucinations).
// Without AI_API_KEY only the offline checks run.

type evalCase struct {
	Name    string `json:"name"`
	Patient struct {
		FirstName string   `json:"first_name"`
		LastName  string   `json:"last_name"`
		AgeYears  int      `json:"age_years"`
		Gender    string   `json:"gender"`
		APP       string   `json:"app"`
		APF       string   `json:"apf"`
		APQX      string   `json:"apqx"`
		Allergies []string `json:"allergies"`
	} `json:"patient"`
	Critical bool `json:"critical"`
	Records  []struct {
		Date       string                          `json:"date"`
		VisitType  string                          `json:"visit_type"`
		Motive     string                          `json:"motive"`
		Subjective string                          `json:"subjective"`
		Objective  string                          `json:"objective"`
		Assessment string                          `json:"assessment"`
		Plan       string                          `json:"plan"`
		Diagnoses  []clinical_models.DiagnosisItem `json:"diagnoses"`
		RawRx      []struct {
			Medication string `json:"medication"`
			Dose       string `json:"dose"`
			Frequency  string `json:"frequency"`
			Duration   string `json:"duration"`
		} `json:"prescription"`
		Vitals *struct {
			BP string `json:"bp"`
			HR uint   `json:"hr"`
		} `json:"vitals"`
	} `json:"records"`
	ExamOrders []struct {
		Date     string   `json:"date"`
		Items    []string `json:"items"`
		Status   string   `json:"status"`
		Priority string   `json:"priority"`
	} `json:"exam_orders"`
	Certificates []struct {
		Date      string `json:"date"`
		Diagnosis string `json:"diagnosis"`
		RestDays  int    `json:"rest_days"`
	} `json:"certificates"`
	Expect struct {
		MustMention    []string `json:"must_mention"`
		MustNotMention []string `json:"must_not_mention"`
	} `json:"expect"`
}

var evalNow = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func loadEvalCases(t *testing.T) []evalCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/summary_eval_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []evalCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 10 || len(cases) > 15 {
		t.Fatalf("eval set has %d cases, want 10–15", len(cases))
	}
	return cases
}

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// history turns a case into the structures LoadHistory returns.
func (c evalCase) history(t *testing.T) PatientHistory {
	t.Helper()
	allergies, _ := json.Marshal(c.Patient.Allergies)
	h := PatientHistory{Patient: clinical_models.Patient{
		FirstName: c.Patient.FirstName, LastName: c.Patient.LastName,
		Document: "1712345678", Phone: "0991234567", Email: "paciente@example.com",
		BirthDate: evalNow.AddDate(-c.Patient.AgeYears, -1, 0), Gender: c.Patient.Gender,
		APP: c.Patient.APP, APF: c.Patient.APF, APQX: c.Patient.APQX, Allergies: string(allergies),
		Critical: c.Critical,
	}}
	// Newest first, as LoadHistory orders them.
	for i := len(c.Records) - 1; i >= 0; i-- {
		r := c.Records[i]
		diag, _ := json.Marshal(r.Diagnoses)
		rec := clinical_models.MedicalRecord{
			Date: day(t, r.Date), VisitType: r.VisitType, Motive: r.Motive,
			SOAPRecord: clinical_models.SOAPRecord{Subjective: r.Subjective, Objective: r.Objective, Assessment: r.Assessment, Plan: r.Plan},
			Diagnoses:  datatypes.JSON(diag),
		}
		rec.ID = uint(i + 1)
		if len(r.RawRx) > 0 {
			rx := &clinical_models.Prescription{}
			for _, it := range r.RawRx {
				rx.Items = append(rx.Items, clinical_models.PrescriptionItem{Medication: it.Medication, Dose: it.Dose, Frequency: it.Frequency, Duration: it.Duration})
			}
			rec.Prescription = rx
		}
		if r.Vitals != nil {
			hr := r.Vitals.HR
			rec.VitalSigns = &clinical_models.VitalSigns{BloodPressure: r.Vitals.BP, HeartRate: &hr}
		}
		h.Records = append(h.Records, rec)
	}
	h.TotalRecords = len(h.Records)
	for i := len(c.ExamOrders) - 1; i >= 0; i-- {
		o := c.ExamOrders[i]
		order := clinical_models.ExamOrder{Status: o.Status, Priority: o.Priority}
		order.CreatedAt = day(t, o.Date)
		for _, name := range o.Items {
			order.Items = append(order.Items, clinical_models.ExamOrderItem{Name: name})
		}
		h.ExamOrders = append(h.ExamOrders, order)
	}
	for i := len(c.Certificates) - 1; i >= 0; i-- {
		cert := c.Certificates[i]
		rest := cert.RestDays
		mc := clinical_models.MedicalCertificate{Diagnosis: cert.Diagnosis, RestDays: &rest}
		mc.CreatedAt = day(t, cert.Date)
		h.Certificates = append(h.Certificates, mc)
	}
	return h
}

// TestSummaryEval_InputsAreDeidentified runs offline (CI): no case sends the
// patient's names or contact data to the model.
func TestSummaryEval_InputsAreDeidentified(t *testing.T) {
	for _, c := range loadEvalCases(t) {
		in := BuildSummaryInput(c.history(t), evalNow)
		if in.Empty {
			t.Errorf("%s: input is empty", c.Name)
		}
		leaks := append([]string{"1712345678", "0991234567", "paciente@example.com"}, strings.Fields(c.Patient.FirstName+" "+c.Patient.LastName)...)
		for _, id := range c.Expect.MustNotMention {
			// Identifiers in the case's must-not list (not absent clinical facts).
			if strings.ContainsAny(id, "0123456789@") || strings.Contains(c.Patient.FirstName+c.Patient.LastName, id) {
				leaks = append(leaks, id)
			}
		}
		for _, leak := range leaks {
			if strings.Contains(strings.ToLower(in.Prompt), strings.ToLower(leak)) {
				t.Errorf("%s: prompt leaks %q:\n%s", c.Name, leak, in.Prompt)
			}
		}
	}
	// The free-text identifiers case also loses the professional and address.
	for _, c := range loadEvalCases(t) {
		if c.Name != "datos_identificadores_en_texto" {
			continue
		}
		in := BuildSummaryInput(c.history(t), evalNow)
		for _, leak := range []string{"mtorres", "Diciembre", "Salazar", "0998765432"} {
			if strings.Contains(in.Prompt, leak) {
				t.Errorf("free-text identifier %q leaked:\n%s", leak, in.Prompt)
			}
		}
	}
}

// TestSummaryEval runs the eval set against the real model; skipped without
// AI_API_KEY.
func TestSummaryEval(t *testing.T) {
	cfg := config.Get()
	if cfg.AIAPIKey == "" {
		t.Skip("AI_API_KEY not set: eval skipped")
	}
	client := ai.New(cfg.AIBaseURL, cfg.AIAPIKey)
	model := cfg.AIDefaultSummaryModel
	passed, cases := 0, loadEvalCases(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			in := BuildSummaryInput(c.history(t), evalNow)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			start := time.Now()
			res, err := client.Chat(ctx, SummaryRequest(model, in.Prompt))
			if err != nil {
				t.Fatalf("chat: %v", err)
			}
			content, ok := ParseSummary(res, in.Sources)
			if !ok {
				t.Fatalf("unusable answer: %q tool_calls=%+v", res.Content(), res.Choices[0].Message.ToolCalls)
			}
			out, _ := json.Marshal(content)
			text := strings.ToLower(foldForEval(string(out)))
			t.Logf("%s in %s, tokens %d/%d: %s", model, time.Since(start).Round(time.Millisecond), res.Usage.PromptTokens, res.Usage.CompletionTokens, out)
			fail := false
			for _, term := range c.Expect.MustMention {
				if !strings.Contains(text, strings.ToLower(foldForEval(term))) {
					t.Errorf("missing %q", term)
					fail = true
				}
			}
			for _, term := range c.Expect.MustNotMention {
				if strings.Contains(text, strings.ToLower(foldForEval(term))) {
					t.Errorf("mentions %q (identifier or not in the input)", term)
					fail = true
				}
			}
			if !fail {
				passed++
			}
		})
	}
	t.Logf("eval %s: %d/%d cases passed", model, passed, len(cases))
}

// foldForEval drops Spanish accents so "hipertensión" matches "hipertensi".
func foldForEval(s string) string {
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ñ", "n").Replace(s)
}
