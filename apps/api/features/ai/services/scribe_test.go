package ai_services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pengi-med-saas/core/ai"
	ai_models "pengi-med-saas/features/ai/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// scribeFake answers with the given tool arguments and records the prompt.
type scribeFake struct {
	args       string
	err        error
	calls      atomic.Int32
	lastPrompt string
}

func (f *scribeFake) Configured() bool { return true }

func (f *scribeFake) Chat(_ context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	f.lastPrompt = req.Messages[len(req.Messages)-1].Content
	return &ai.ChatResponse{
		Choices: []ai.Choice{{Message: ai.Message{ToolCalls: []ai.ToolCall{{Type: "function", Function: ai.FunctionCall{Name: scribeToolName, Arguments: f.args}}}}}},
		Usage:   ai.Usage{PromptTokens: 300, CompletionTokens: 120},
	}, nil
}

const scribeArgs = `{"subjective":"Odinofagia de 2 días, fiebre referida.","objective":"Amígdalas eritematosas con exudado. T 38.2 °C.","assessment":"Faringitis aguda.","plan":"Paracetamol según necesidad. Control en 3 días.","diagnosis_terms":["faringitis aguda","enfermedad inventada xyz"]}`

// fakeResolver knows a fixed term → code table.
type fakeResolver map[string]DiagnosisSuggestion

func (f fakeResolver) Resolve(_ context.Context, term string) (*DiagnosisSuggestion, error) {
	if s, ok := f[strings.ToLower(term)]; ok {
		return &s, nil
	}
	return nil, nil
}

func fakeResolvers(r DiagnosisResolver) ResolverFactory {
	return func(*gorm.DB, string) DiagnosisResolver { return r }
}

var knownDiagnoses = fakeResolver{"faringitis aguda": {Code: "J02.9", Title: "Faringitis aguda, no especificada"}}

const scribeMarker = "MARCADOR-NOTAS-7781"

func scribeReq(f *fixture, notes string) ScribeRequest {
	return ScribeRequest{PatientID: f.patient.ID, Notes: notes, Motive: "Dolor de garganta", Vitals: &ScribeVitals{BloodPressure: "120/80"}}
}

func (f *fixture) organize(s *Scribe, req ScribeRequest) (*ScribeDraft, error) {
	return s.Organize(context.Background(), f.db, f.tdb(), f.tenant.ID, 7, req)
}

func usageRows(f *fixture, fn string) int64 {
	var n int64
	f.must(f.tdb().Model(&ai_models.AIUsageLog{}).Where("function = ?", fn).Count(&n).Error)
	return n
}

func TestScribe_DraftRedactsAndResolvesDiagnoses(t *testing.T) {
	f := newFixture(t, true)
	fake := &scribeFake{args: scribeArgs}
	s := NewScribe(fakeDial(fake), fakeResolvers(knownDiagnoses))
	notes := "Rosa Elena Guamán refiere odinofagia 2 días. Tel 0991234567, rosa@example.com. " + scribeMarker + ". Exudado amigdalino. Dx faringitis aguda."
	draft, err := f.organize(s, scribeReq(f, notes))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Rosa", "Guamán", "0991234567", "rosa@example.com", "1712345678"} {
		if strings.Contains(fake.lastPrompt, leak) {
			t.Errorf("prompt leaks %q:\n%s", leak, fake.lastPrompt)
		}
	}
	for _, want := range []string{"sexo femenino", "58 años", "Dolor de garganta", "PA 120/80", "odinofagia"} {
		if !strings.Contains(fake.lastPrompt, want) {
			t.Errorf("prompt lacks %q:\n%s", want, fake.lastPrompt)
		}
	}
	if draft.Assessment != "Faringitis aguda." || draft.Plan == "" || !draft.AIDraft || draft.DiagnosisSystem != DiagnosisSystemCIE11 {
		t.Fatalf("draft = %+v", draft)
	}
	if len(draft.SuggestedDiagnoses) != 1 || draft.SuggestedDiagnoses[0].Code != "J02.9" {
		t.Fatalf("suggestions = %+v (unknown term must be dropped)", draft.SuggestedDiagnoses)
	}
	if usageRows(f, ai_models.FunctionScribe) != 1 || usageRows(f, ai_models.FunctionSummary) != 0 {
		t.Fatal("one scribe use must be logged")
	}

	// The notes are not persisted anywhere.
	var logs []ai_models.AIUsageLog
	var summaries []ai_models.AISummary
	f.must(f.tdb().Find(&logs).Error)
	f.must(f.tdb().Find(&summaries).Error)
	raw, _ := json.Marshal([]any{logs, summaries})
	if strings.Contains(string(raw), scribeMarker) || strings.Contains(string(raw), "odinofagia") {
		t.Fatalf("notes persisted: %s", raw)
	}
	var records int64
	f.must(f.tdb().Model(&clinical_models.MedicalRecord{}).Count(&records).Error)
	if records != 0 {
		t.Fatal("the scribe must not create records")
	}
}

func TestScribe_GatesAndFailures(t *testing.T) {
	off := newFixture(t, false)
	fake := &scribeFake{args: scribeArgs}
	if _, err := off.organize(NewScribe(fakeDial(fake), fakeResolvers(knownDiagnoses)), scribeReq(off, "notas")); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("opt-out: %v", err)
	}

	f := newFixture(t, true)
	s := NewScribe(fakeDial(fake), fakeResolvers(knownDiagnoses))
	for _, notes := range []string{"   ", strings.Repeat("a", MaxScribeNotesRunes+1)} {
		if _, err := f.organize(s, scribeReq(f, notes)); !errors.Is(err, ErrInvalidNotes) {
			t.Errorf("notes len %d: %v", len(notes), err)
		}
	}
	other := scribeReq(f, "notas")
	other.PatientID = f.patient.ID + 999
	if _, err := f.organize(s, other); !errors.Is(err, ErrPatientNotFound) {
		t.Fatalf("unknown patient: %v", err)
	}
	if fake.calls.Load() != 0 {
		t.Fatalf("calls = %d: nothing must reach the model", fake.calls.Load())
	}

	// Provider failure: not counted.
	failing := &scribeFake{err: &ai.APIError{Status: 502}}
	if _, err := f.organize(NewScribe(fakeDial(failing), fakeResolvers(knownDiagnoses)), scribeReq(f, "notas")); !errors.Is(err, ErrProvider) {
		t.Fatalf("provider: %v", err)
	}
	// An unusable answer (all sections empty) is counted: the tokens were billed.
	empty := &scribeFake{args: `{"subjective":"","objective":null,"assessment":"No consta","plan":"","diagnosis_terms":""}`}
	if _, err := f.organize(NewScribe(fakeDial(empty), fakeResolvers(knownDiagnoses)), scribeReq(f, "notas")); !errors.Is(err, ErrProvider) {
		t.Fatalf("empty answer: %v", err)
	}
	if n := usageRows(f, ai_models.FunctionScribe); n != 1 {
		t.Fatalf("usage rows = %d, want 1 (failed call free, unusable answer charged)", n)
	}

	// No key anywhere.
	t.Setenv("AI_API_KEY", "")
	if _, err := f.organize(s, scribeReq(f, "notas")); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no key: %v", err)
	}
}

func TestScribe_BudgetIsSharedAcrossFunctions(t *testing.T) {
	f := newFixture(t, true)
	f.planWithBudget(2)
	// A summary already spent $1.5 of the $2 budget: the scribe still has room
	// for one more use, and then none.
	f.must(LogUsage(f.db, UsageEntry{TenantID: f.tenant.ID, UserID: 7, Function: ai_models.FunctionSummary, Model: "m"}, time.Now()))
	f.must(f.tdb().Model(&ai_models.AIUsageLog{}).Where("1 = 1").Update("cost_usd", 1.5).Error)
	f.price(ai_models.FunctionScribe, 2000) // scribeFake bills 300 prompt tokens: $0.60 a use
	fake := &scribeFake{args: scribeArgs}
	s := NewScribe(fakeDial(fake), fakeResolvers(knownDiagnoses))
	if _, err := f.organize(s, scribeReq(f, "notas")); err != nil {
		t.Fatalf("first scribe (budget left): %v", err)
	}
	if _, err := f.organize(s, scribeReq(f, "notas")); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("second scribe: %v", err)
	}
	if fake.calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", fake.calls.Load())
	}
}

func TestScribe_DiagnosisSystemFromClinicSettings(t *testing.T) {
	f := newFixture(t, true)
	f.must(f.db.Model(&tenant_models.Tenant{}).Where("id = ?", f.tenant.ID).
		Update("ui_settings", `{"ai":{"enabled":true},"clinical":{"diagnosis_system":"cie10"}}`).Error)
	var gotSystem string
	resolvers := func(_ *gorm.DB, system string) DiagnosisResolver { gotSystem = system; return knownDiagnoses }
	draft, err := f.organize(NewScribe(fakeDial(&scribeFake{args: scribeArgs}), resolvers), scribeReq(f, "notas"))
	if err != nil {
		t.Fatal(err)
	}
	if gotSystem != DiagnosisSystemCIE10 || draft.DiagnosisSystem != DiagnosisSystemCIE10 {
		t.Fatalf("system = %q / %q", gotSystem, draft.DiagnosisSystem)
	}
}

func TestParseScribe_Tolerant(t *testing.T) {
	tool := func(args string) *ai.ChatResponse {
		return &ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{ToolCalls: []ai.ToolCall{{Function: ai.FunctionCall{Name: scribeToolName, Arguments: args}}}}}}}
	}
	cases := []struct {
		name  string
		res   *ai.ChatResponse
		check func(ScribeContent) bool
	}{
		{"null and placeholders", tool(`{"subjective":"Tos","objective":null,"assessment":"No consta.","plan":"N/A","diagnosis_terms":null}`),
			func(c ScribeContent) bool {
				return c.Subjective == "Tos" && c.Objective == "" && c.Assessment == "" && c.Plan == "" && len(c.DiagnosisTerms) == 0
			}},
		{"bare string terms", tool(`{"subjective":"Tos","diagnosis_terms":"bronquitis aguda; asma"}`),
			func(c ScribeContent) bool { return len(c.DiagnosisTerms) == 2 && c.DiagnosisTerms[1] == "asma" }},
		{"object terms and list sections", tool(`{"subjective":["Tos","Fiebre"],"plan":42,"diagnosis_terms":[{"term":"neumonía"},{"text":"asma"},7]}`),
			func(c ScribeContent) bool {
				return c.Subjective == "Tos\nFiebre" && c.Plan == "42" && len(c.DiagnosisTerms) == 2
			}},
		{"odd field does not discard the rest", tool(`{"subjective":{"x":1},"assessment":"Asma","diagnosis_terms":{"a":1}}`),
			func(c ScribeContent) bool { return c.Subjective == "" && c.Assessment == "Asma" }},
		{"spanish keys", tool(`{"Subjetivo":"Tos","Plan":"Reposo"}`),
			func(c ScribeContent) bool { return c.Subjective == "Tos" && c.Plan == "Reposo" }},
		{"json in text", &ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{Content: "```json\n{\"objective\":\"Sibilancias\"}\n```"}}}},
			func(c ScribeContent) bool { return c.Objective == "Sibilancias" }},
	}
	for _, tc := range cases {
		got, ok := ParseScribe(tc.res)
		if !ok || !tc.check(got) {
			t.Errorf("%s: %+v ok=%v", tc.name, got, ok)
		}
	}
	if _, ok := ParseScribe(tool(`{"subjective":"","objective":"no consta"}`)); ok {
		t.Error("all-empty answer must be unusable")
	}
	if _, ok := ParseScribe(&ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{Content: "Lo siento"}}}}); ok {
		t.Error("prose without JSON must be unusable")
	}
}

func TestCIE10Resolver(t *testing.T) {
	f := newFixture(t, true)
	f.must(f.db.AutoMigrate(&clinical_models.Cie10Code{}))
	for _, c := range []clinical_models.Cie10Code{
		{Code: "I10", Title: "hipertension esencial (primaria)"},
		{Code: "J02.0", Title: "faringitis estreptococica"},
		{Code: "J02.9", Title: "faringitis aguda, no especificada"},
		{Code: "J03.9", Title: "amigdalitis aguda, no especificada"},
	} {
		f.must(f.db.Create(&c).Error)
	}
	r := CIE10Resolver{DB: f.db}
	got := ResolveDiagnoses(context.Background(), r, []string{"Hipertension esencial", "faringitis aguda no especificada", "amigdalitis aguda bacteriana", "dolor inventado", "faringitis aguda"})
	codes := []string{}
	for _, s := range got {
		codes = append(codes, s.Code)
	}
	if strings.Join(codes, ",") != "I10,J02.9" {
		t.Fatalf("codes = %v (all words must match; duplicates and unknown dropped)", codes)
	}
}

func TestCIE11Resolver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "faringitis aguda" && strings.Contains(r.URL.Path, "/icd/release/11/2026-01/mms/search") {
			_, _ = w.Write([]byte(`{"destinationEntities":[{"theCode":"","title":"x"},{"theCode":"CA02.Z","title":"Faringitis aguda, sin especificación"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"destinationEntities":[]}`))
	}))
	defer srv.Close()
	r := CIE11Resolver{BaseURL: srv.URL, Release: "2026-01", Language: "es"}
	got := ResolveDiagnoses(context.Background(), r, []string{"faringitis aguda", "no existe"})
	if len(got) != 1 || got[0].Code != "CA02.Z" || got[0].Term != "faringitis aguda" {
		t.Fatalf("got %+v", got)
	}
}
