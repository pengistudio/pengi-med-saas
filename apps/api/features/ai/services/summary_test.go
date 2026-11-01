package ai_services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pengi-med-saas/core/ai"
	"pengi-med-saas/core/tenantdb"
	ai_models "pengi-med-saas/features/ai/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	settings_models "pengi-med-saas/features/settings/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// fakeCompleter answers every chat with a tool call citing C1 and records
// what it was sent.
type fakeCompleter struct {
	configured bool
	calls      atomic.Int32
	lastPrompt string
	err        error
}

func (f *fakeCompleter) Configured() bool { return f.configured }

func (f *fakeCompleter) Chat(_ context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	f.lastPrompt = req.Messages[len(req.Messages)-1].Content
	args := `{"overview":"Paciente con HTA en control.","active_problems":[{"text":"Hipertensión arterial","sources":["C1","C9"]}],"current_medications":[{"text":"Losartán 50 mg","sources":["c1"]}],"alerts":[]}`
	return &ai.ChatResponse{
		Choices: []ai.Choice{{Message: ai.Message{Role: "assistant", ToolCalls: []ai.ToolCall{{ID: "1", Type: "function", Function: ai.FunctionCall{Name: summaryToolName, Arguments: args}}}}}},
		Usage:   ai.Usage{PromptTokens: 1000, CompletionTokens: 200},
	}, nil
}

// fakeDial returns a Dialer that always hands back c (ignoring the provider).
func fakeDial(c Completer) Dialer { return func(Provider) Completer { return c } }

type fixture struct {
	t       *testing.T
	db      *gorm.DB
	tenant  tenant_models.Tenant
	patient clinical_models.Patient
}

func newFixture(t *testing.T, optIn bool) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{},
		&clinical_models.Patient{}, &clinical_models.MedicalRecord{}, &clinical_models.SOAPRecord{},
		&clinical_models.Prescription{}, &clinical_models.PrescriptionItem{}, &clinical_models.VitalSigns{},
		&clinical_models.ExamOrder{}, &clinical_models.ExamOrderItem{}, &clinical_models.PatientAttachment{},
		&clinical_models.MedicalCertificate{}, &clinical_models.MedicalReport{},
		&company_models.Company{}, &company_models.Plan{}, &company_models.Feature{}, &company_models.Subscription{},
		&settings_models.SystemSetting{}, &ai_models.AIUsageLog{}, &ai_models.AISummary{})
	// An env provider, so fakes dialed with it count as configured; no
	// backoffice encryption key unless a test sets one.
	t.Setenv("AI_API_KEY", "env-test-key")
	t.Setenv("AI_BASE_URL", "http://127.0.0.1:1/v1")
	t.Setenv(EncryptionKeyEnv, "")
	now := time.Now().UnixNano()
	settings := `{}`
	if optIn {
		settings = `{"ai":{"enabled":true}}`
	}
	f := &fixture{t: t, db: db}
	f.tenant = tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("ai-%d", now), DisplayToken: fmt.Sprintf("ai-%d", now), UISettings: settings}
	f.must(db.Create(&f.tenant).Error)
	f.patient = clinical_models.Patient{
		TenantID: f.tenant.ID, FirstName: "Rosa Elena", LastName: "Guamán Paredes", Document: "1712345678",
		Phone: "0991234567", Email: "rosa@example.com", BirthDate: time.Now().AddDate(-58, -1, 0), Gender: "F",
		APP: "Hipertensión arterial desde 2019", Allergies: `["Penicilina"]`,
	}
	f.must(f.tdb().Create(&f.patient).Error)
	return f
}

func (f *fixture) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) tdb() *gorm.DB { return tenantdb.ForTenant(f.db, f.tenant.ID) }

func (f *fixture) addRecord(motive string) clinical_models.MedicalRecord {
	rec := clinical_models.MedicalRecord{
		TenantID: f.tenant.ID, PatientID: f.patient.ID, Date: time.Now(), Motive: motive,
		SOAPRecord: clinical_models.SOAPRecord{Subjective: "Rosa refiere cefalea. Vive en calle Olmedo 123", Assessment: "HTA controlada"},
		Diagnoses:  datatypes.JSON(`[{"code":"I10","title":"Hipertensión esencial"}]`),
	}
	f.must(f.tdb().Create(&rec).Error)
	return rec
}

func (f *fixture) planWithQuota(limit any) {
	code := fmt.Sprintf("P%d", time.Now().UnixNano())
	plan := company_models.Plan{Name: code, Code: code, Price: 10}
	if limit != nil {
		plan.Properties = datatypes.JSONMap{"max_ai_summaries": limit}
	}
	f.must(f.db.Create(&plan).Error)
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: code, TenantID: f.tenant.ID}
	f.must(f.db.Create(&company).Error)
	f.must(f.db.Create(&company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().Add(720 * time.Hour), CompanyID: company.ID}).Error)
}

func (f *fixture) generate(s *Summarizer) (*Summary, error) {
	return s.Generate(context.Background(), f.db, f.tdb(), f.tenant.ID, 7, f.patient.ID)
}

func TestGenerate_NotEnabledNotConfiguredNoHistory(t *testing.T) {
	off := newFixture(t, false)
	off.addRecord("Control")
	if _, err := off.generate(NewSummarizer(fakeDial(&fakeCompleter{configured: true}))); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("opt-out: err = %v", err)
	}

	f := newFixture(t, true)
	f.addRecord("Control")
	if _, err := f.generate(NewSummarizer(fakeDial(&fakeCompleter{configured: false}))); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no key: err = %v", err)
	}
	// No key in env nor backoffice: the real dialer is never reached.
	t.Setenv("AI_API_KEY", "")
	if _, err := f.generate(NewSummarizer(fakeDial(&fakeCompleter{configured: true}))); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no key anywhere: err = %v", err)
	}

	empty := newFixture(t, true)
	empty.must(empty.tdb().Model(&clinical_models.Patient{}).Where("id = ?", empty.patient.ID).
		Updates(map[string]any{"app": "", "allergies": ""}).Error)
	if _, err := empty.generate(NewSummarizer(fakeDial(&fakeCompleter{configured: true}))); !errors.Is(err, ErrNoHistory) {
		t.Fatalf("empty history: err = %v", err)
	}
}

func TestGenerate_SendsNoIdentifiersAndMapsSources(t *testing.T) {
	f := newFixture(t, true)
	rec := f.addRecord("Control de presión")
	fake := &fakeCompleter{configured: true}
	sum, err := f.generate(NewSummarizer(fakeDial(fake)))
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"Rosa", "Guamán", "Guaman", "1712345678", "0991234567", "rosa@example.com", "Olmedo"} {
		if strings.Contains(fake.lastPrompt, leaked) {
			t.Errorf("prompt leaks %q:\n%s", leaked, fake.lastPrompt)
		}
	}
	for _, want := range []string{"Edad: 58 años", "Sexo: femenino", "Penicilina", "I10", "[C1]"} {
		if !strings.Contains(fake.lastPrompt, want) {
			t.Errorf("prompt lacks %q:\n%s", want, fake.lastPrompt)
		}
	}
	if !sum.AIDraft || sum.Cached || len(sum.Content.ActiveProblems) != 1 {
		t.Fatalf("summary = %+v", sum)
	}
	// C1 maps to the record; the unknown C9 is dropped; lowercase works.
	if got := sum.Content.ActiveProblems[0].Sources; len(got) != 1 || got[0] != rec.ID {
		t.Errorf("sources = %v, want [%d]", got, rec.ID)
	}
	if got := sum.Content.CurrentMedications[0].Sources; len(got) != 1 || got[0] != rec.ID {
		t.Errorf("medication sources = %v", got)
	}
	if len(sum.Sources) != 1 || sum.Sources[0].MedicalRecordID != rec.ID {
		t.Errorf("source list = %+v", sum.Sources)
	}
}

func TestGenerate_CachedUntilHistoryChanges(t *testing.T) {
	f := newFixture(t, true)
	f.addRecord("Primera consulta")
	fake := &fakeCompleter{configured: true}
	s := NewSummarizer(fakeDial(fake))

	if _, err := f.generate(s); err != nil {
		t.Fatal(err)
	}
	again, err := f.generate(s)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Cached || fake.calls.Load() != 1 {
		t.Fatalf("second request: cached=%v calls=%d, want cached and 1 call", again.Cached, fake.calls.Load())
	}
	if c, err := s.Cached(f.tdb(), f.patient.ID); err != nil || c == nil {
		t.Fatalf("Cached = %v, %v", c, err)
	}

	// A new consultation invalidates it.
	f.addRecord("Control")
	if c, err := s.Cached(f.tdb(), f.patient.ID); err != nil || c != nil {
		t.Fatalf("after new record Cached = %v, %v; want nil", c, err)
	}
	if _, err := f.generate(s); err != nil {
		t.Fatal(err)
	}
	// So does a new exam order and a new certificate.
	f.must(f.tdb().Create(&clinical_models.ExamOrder{TenantID: f.tenant.ID, Number: 1, PatientID: f.patient.ID, OrderedByID: 7,
		Items: []clinical_models.ExamOrderItem{{TenantID: f.tenant.ID, Name: "Hemograma", Category: "laboratory"}}}).Error)
	if c, _ := s.Cached(f.tdb(), f.patient.ID); c != nil {
		t.Fatal("a new exam order must invalidate the summary")
	}
	if _, err := f.generate(s); err != nil {
		t.Fatal(err)
	}
	f.must(f.tdb().Create(&clinical_models.MedicalCertificate{TenantID: f.tenant.ID, PatientID: f.patient.ID, Diagnosis: "Lumbalgia"}).Error)
	if c, _ := s.Cached(f.tdb(), f.patient.ID); c != nil {
		t.Fatal("a new certificate must invalidate the summary")
	}
	if fake.calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", fake.calls.Load())
	}
	var n int64
	f.must(f.tdb().Model(&ai_models.AIUsageLog{}).Count(&n).Error)
	if n != 3 {
		t.Fatalf("usage rows = %d, want 3 (cache hits are free)", n)
	}
}

// The record handlers update SOAP, prescriptions, their items and vital signs
// without touching the record row: each edit alone must invalidate the cache.
func TestGenerate_ChildEditsAndDeletesInvalidate(t *testing.T) {
	f := newFixture(t, true)
	rec := clinical_models.MedicalRecord{
		TenantID: f.tenant.ID, PatientID: f.patient.ID, Date: time.Now(), Motive: "Control",
		SOAPRecord: clinical_models.SOAPRecord{Subjective: "Cefalea"},
		Prescription: &clinical_models.Prescription{Indications: "Tomar con agua",
			Items: []clinical_models.PrescriptionItem{{Medication: "Losartán 50 mg"}}},
		VitalSigns: &clinical_models.VitalSigns{BloodPressure: "130/85"},
	}
	f.must(f.tdb().Create(&rec).Error)
	order := clinical_models.ExamOrder{TenantID: f.tenant.ID, Number: 1, PatientID: f.patient.ID, OrderedByID: 7,
		Items: []clinical_models.ExamOrderItem{{TenantID: f.tenant.ID, Name: "Hemograma", Category: "laboratory"}}}
	f.must(f.tdb().Create(&order).Error)
	if rec.Prescription == nil || rec.Prescription.ID == 0 || rec.VitalSigns == nil || rec.VitalSigns.ID == 0 {
		t.Fatal("fixture children not created")
	}
	if rec.PrescriptionID == nil {
		// The record points at its prescription (what Preload follows).
		f.must(f.tdb().Model(&clinical_models.MedicalRecord{}).Where("id = ?", rec.ID).
			UpdateColumn("prescription_id", rec.Prescription.ID).Error)
	}

	s := NewSummarizer(fakeDial(&fakeCompleter{configured: true}))
	edits := []struct {
		name string
		edit func()
	}{
		{"SOAP only", func() {
			f.must(f.db.Model(&clinical_models.SOAPRecord{}).Where("id = ?", rec.SOAPRecordID).Update("subjective", "Sin cefalea").Error)
		}},
		{"prescription only", func() {
			f.must(f.db.Model(&clinical_models.Prescription{}).Where("id = ?", rec.Prescription.ID).Update("indications", "Con comida").Error)
		}},
		{"prescription item", func() {
			f.must(f.db.Model(&clinical_models.PrescriptionItem{}).Where("id = ?", rec.Prescription.Items[0].ID).Update("medication", "Losartán 100 mg").Error)
		}},
		{"new prescription item", func() {
			f.must(f.db.Create(&clinical_models.PrescriptionItem{PrescriptionID: rec.Prescription.ID, Medication: "Amlodipino 5 mg"}).Error)
		}},
		{"vital signs", func() {
			f.must(f.db.Model(&clinical_models.VitalSigns{}).Where("id = ?", rec.VitalSigns.ID).Update("blood_pressure", "120/80").Error)
		}},
		{"exam order item", func() {
			f.must(f.tdb().Model(&clinical_models.ExamOrderItem{}).Where("id = ?", order.Items[0].ID).Update("indications", "En ayunas").Error)
		}},
		{"record deleted", func() {
			f.must(f.tdb().Delete(&clinical_models.MedicalRecord{}, rec.ID).Error)
		}},
	}
	for _, e := range edits {
		if _, err := f.generate(s); err != nil {
			t.Fatalf("before %s: %v", e.name, err)
		}
		if c, err := s.Cached(f.tdb(), f.patient.ID); err != nil || c == nil {
			t.Fatalf("before %s: cache not warm (%v, %v)", e.name, c, err)
		}
		time.Sleep(2 * time.Millisecond) // updated_at must move past the cached state
		e.edit()
		if c, err := s.Cached(f.tdb(), f.patient.ID); err != nil || c != nil {
			t.Errorf("%s: cache still valid (%v, %v), want a miss", e.name, c, err)
		}
	}
}

func TestGenerate_QuotaFromPlanAndDefault(t *testing.T) {
	f := newFixture(t, true)
	f.addRecord("Control")
	f.planWithQuota(1)
	fake := &fakeCompleter{configured: true}
	s := NewSummarizer(fakeDial(fake))
	if _, err := f.generate(s); err != nil {
		t.Fatal(err)
	}
	f.addRecord("Otro control") // invalidates the cache
	if _, err := f.generate(s); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("err = %v, want ErrQuotaExceeded", err)
	}
	if fake.calls.Load() != 1 {
		t.Fatalf("calls = %d: the model must not be called over quota", fake.calls.Load())
	}

	// Usage from last month doesn't count.
	f.must(f.tdb().Model(&ai_models.AIUsageLog{}).Where("1 = 1").Update("created_at", time.Now().AddDate(0, -2, 0)).Error)
	if _, err := f.generate(s); err != nil {
		t.Fatalf("new month: %v", err)
	}

	noPlan := newFixture(t, true)
	if got := PlanQuota(noPlan.db, noPlan.tenant.ID, ai_models.FunctionSummary); got != 300 {
		t.Fatalf("default quota = %d, want 300", got)
	}
	unlimited := newFixture(t, true)
	unlimited.planWithQuota(-1)
	if got := PlanQuota(unlimited.db, unlimited.tenant.ID, ai_models.FunctionSummary); got != -1 {
		t.Fatalf("unlimited quota = %d", got)
	}
	absent := newFixture(t, true)
	absent.planWithQuota(nil)
	if got := PlanQuota(absent.db, absent.tenant.ID, ai_models.FunctionSummary); got != 300 {
		t.Fatalf("absent key quota = %d, want default 300", got)
	}
	if got := PlanQuota(absent.db, absent.tenant.ID, ai_models.FunctionBot); got != 500 {
		t.Fatalf("bot default = %d, want 500", got)
	}
}

func TestGenerate_ProviderErrorIsNotCharged(t *testing.T) {
	f := newFixture(t, true)
	f.addRecord("Control")
	_, err := f.generate(NewSummarizer(fakeDial(&fakeCompleter{configured: true, err: &ai.APIError{Status: 500}})))
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("err = %v", err)
	}
	var n int64
	f.must(f.tdb().Model(&ai_models.AIUsageLog{}).Count(&n).Error)
	if n != 0 {
		t.Fatalf("usage rows = %d, want 0", n)
	}
}

func TestUsageLogCostAndModelSettings(t *testing.T) {
	f := newFixture(t, true)
	sys := tenantdb.System(f.db)
	if got := ModelFor(sys, ai_models.FunctionSummary); got != DefaultModel(ai_models.FunctionSummary) {
		t.Fatalf("model = %q, want default", got)
	}
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary, Model: "llama3.3-70b-instruct", PriceInput: 0.65, PriceOutput: 0.65}))
	if got := ModelFor(sys, ai_models.FunctionSummary); got != "llama3.3-70b-instruct" {
		t.Fatalf("model = %q", got)
	}
	f.must(LogUsage(f.db, UsageEntry{TenantID: f.tenant.ID, UserID: 7, Function: ai_models.FunctionSummary, Model: "m", Host: "h", PromptTokens: 1_000_000, CompletionTokens: 500_000}, time.Now()))
	var row ai_models.AIUsageLog
	f.must(f.tdb().First(&row).Error)
	if row.CostUSD < 0.9749 || row.CostUSD > 0.9751 {
		t.Fatalf("cost = %v, want 0.975", row.CostUSD)
	}
	if err := SaveSettings(sys, SettingsUpdate{Function: "nope"}); err == nil {
		t.Fatal("unknown function must fail")
	}
	// Resetting the model falls back to the default.
	f.must(SaveSettings(sys, SettingsUpdate{Function: ai_models.FunctionSummary}))
	if got := ModelFor(sys, ai_models.FunctionSummary); got != DefaultModel(ai_models.FunctionSummary) {
		t.Fatalf("model after reset = %q", got)
	}
}

func TestParseSummary_FallbacksToText(t *testing.T) {
	src := []SummarySource{{Label: "C1", MedicalRecordID: 5}}
	jsonText := &ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{Content: "```json\n{\"overview\":\"Resumen\",\"alerts\":[{\"text\":\"Alergia\",\"sources\":[\"C1\"]}]}\n```"}}}}
	got, ok := ParseSummary(jsonText, src)
	if !ok || got.Overview != "Resumen" || len(got.Alerts) != 1 || got.Alerts[0].Sources[0] != 5 {
		t.Fatalf("json text: %+v %v", got, ok)
	}
	plain := &ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{Content: "Paciente estable."}}}}
	if got, ok := ParseSummary(plain, src); !ok || got.Overview != "Paciente estable." {
		t.Fatalf("plain: %+v %v", got, ok)
	}
	if _, ok := ParseSummary(&ai.ChatResponse{Choices: []ai.Choice{{}}}, src); ok {
		t.Fatal("empty answer must not be usable")
	}
}

// Small models send "" (or null, or a bare string) instead of an empty list;
// that must not discard the whole answer.
func TestParseSummary_ToleratesScalarLists(t *testing.T) {
	src := []SummarySource{{Label: "C1", MedicalRecordID: 5}}
	args := `{"overview":"Resumen","active_problems":[{"text":"HTA","sources":["C1"]}],"relevant_exams":"","alerts":null,"current_medications":"Enalapril"}`
	res := &ai.ChatResponse{Choices: []ai.Choice{{Message: ai.Message{ToolCalls: []ai.ToolCall{{
		Function: ai.FunctionCall{Name: summaryToolName, Arguments: args},
	}}}}}}
	got, ok := ParseSummary(res, src)
	if !ok || got.Overview != "Resumen" || len(got.ActiveProblems) != 1 {
		t.Fatalf("scalar lists: %+v %v", got, ok)
	}
	if len(got.RelevantExams) != 0 || len(got.Alerts) != 0 {
		t.Fatalf("empty string/null must be an empty list: %+v", got)
	}
	if len(got.CurrentMedications) != 1 || got.CurrentMedications[0].Text != "Enalapril" {
		t.Fatalf("bare string must be one item: %+v", got.CurrentMedications)
	}
}

func TestAgeText(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		birth time.Time
		want  string
	}{
		{time.Date(1970, 11, 1, 0, 0, 0, 0, time.UTC), "55 años"},
		{time.Date(1970, 10, 1, 0, 0, 0, 0, time.UTC), "56 años"},
		{time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC), "10 meses"},
		{time.Time{}, "no registrada"},
	} {
		if got := ageText(tc.birth, now); got != tc.want {
			t.Errorf("ageText(%v) = %q, want %q", tc.birth, got, tc.want)
		}
	}
}
