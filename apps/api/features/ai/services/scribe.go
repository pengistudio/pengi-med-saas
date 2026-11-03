package ai_services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/ai"
	ai_models "pengi-med-saas/features/ai/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// MaxScribeNotesRunes bounds the notes of one use.
const MaxScribeNotesRunes = 6000

// ErrInvalidNotes: the notes are empty or too long.
var ErrInvalidNotes = errors.New("ai scribe: notes empty or too long")

// ScribeVitals are the vital signs already typed in the consultation form.
type ScribeVitals struct {
	Weight        *float64 `json:"weight"`
	Height        *float64 `json:"height"`
	BloodPressure string   `json:"blood_pressure"`
	Temperature   *float64 `json:"temperature"`
	HeartRate     *uint    `json:"heart_rate"`
	O2Saturation  *uint    `json:"o2_saturation"`
}

// ScribeRequest is one "Ordenar con IA": the doctor's quick notes and the
// context already in the form. The notes are transient: never stored, never
// logged, never audited; only the S/O/A/P the doctor confirms is saved, by
// the normal record flow.
type ScribeRequest struct {
	PatientID uint          `json:"patient_id" binding:"required"`
	Notes     string        `json:"notes" binding:"required"`
	Motive    string        `json:"motive"`
	Vitals    *ScribeVitals `json:"vitals"`
}

// ScribeDraft is the AI draft of the SOAP fields, for the doctor to review.
type ScribeDraft struct {
	Subjective         string                `json:"subjective"`
	Objective          string                `json:"objective"`
	Assessment         string                `json:"assessment"`
	Plan               string                `json:"plan"`
	SuggestedDiagnoses []DiagnosisSuggestion `json:"suggested_diagnoses"`
	DiagnosisSystem    string                `json:"diagnosis_system"`
	Model              string                `json:"model"`
	AIDraft            bool                  `json:"ai_draft"`
}

// Scribe turns quick consultation notes into a SOAP draft.
type Scribe struct {
	Dial      Dialer
	Resolvers ResolverFactory
	Now       func() time.Time
}

// NewScribe returns a Scribe that dials the resolved "scribe" provider of
// each call (DialProvider when dial is nil) and resolves suggested diagnoses
// with resolvers (DefaultResolvers when nil).
func NewScribe(dial Dialer, resolvers ResolverFactory) *Scribe {
	if dial == nil {
		dial = DialProvider
	}
	if resolvers == nil {
		resolvers = DefaultResolvers
	}
	return &Scribe{Dial: dial, Resolvers: resolvers, Now: time.Now}
}

// Organize returns the SOAP draft of req. It counts one "scribe" use once
// the provider answers. tdb must be bound to tenantID; db is unbound.
func (s *Scribe) Organize(ctx context.Context, db, tdb *gorm.DB, tenantID, userID uint, req ScribeRequest) (*ScribeDraft, error) {
	notes := strings.TrimSpace(req.Notes)
	if notes == "" || utf8.RuneCountInString(notes) > MaxScribeNotesRunes {
		return nil, ErrInvalidNotes
	}
	var tenant tenant_models.Tenant
	if err := db.Select("id", "ui_settings").First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}
	settings := tenant_models.ParseUISettings(tenant.UISettings)
	if !settings.AI.Enabled {
		return nil, ErrNotEnabled
	}
	var patient clinical_models.Patient
	if err := tdb.Select("id", "first_name", "last_name", "full_name", "birth_date", "gender").First(&patient, req.PatientID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	now := s.Now()
	provider, client, err := openFunction(s.Dial, db, tenantID, ai_models.FunctionScribe, now)
	if err != nil {
		return nil, err
	}

	prompt := BuildScribeInput(patient, req, now)
	res, err := callAndLog(ctx, db, client, provider, ScribeChatRequest(prompt),
		UsageEntry{TenantID: tenantID, UserID: userID, Function: ai_models.FunctionScribe}, now)
	if err != nil {
		return nil, err
	}
	raw, ok := ParseScribe(res)
	if !ok {
		return nil, fmt.Errorf("%w: unusable answer", ErrProvider)
	}
	system := settings.Clinical.DiagnosisSystem
	if system != DiagnosisSystemCIE10 {
		system = DiagnosisSystemCIE11
	}
	return &ScribeDraft{
		Subjective:         raw.Subjective,
		Objective:          raw.Objective,
		Assessment:         raw.Assessment,
		Plan:               raw.Plan,
		SuggestedDiagnoses: ResolveDiagnoses(ctx, s.Resolvers(db, system), raw.DiagnosisTerms),
		DiagnosisSystem:    system,
		Model:              provider.Model,
		AIDraft:            true,
	}, nil
}

// BuildScribeInput is the de-identified text sent to the model: age and sex
// (never name or birth date), the motive, the typed vitals and the notes,
// free text through ai.Redactor with the patient's names.
func BuildScribeInput(p clinical_models.Patient, req ScribeRequest, now time.Time) string {
	names := []string{p.FirstName, p.LastName}
	if p.FullName != nil {
		names = append(names, *p.FullName)
	}
	r := ai.NewRedactor(names...)
	var b strings.Builder
	fmt.Fprintf(&b, "Paciente: %s, sexo %s.\n", ageText(p.BirthDate, now), sexText(p.Gender))
	if m := strings.TrimSpace(r.Redact(req.Motive)); m != "" {
		fmt.Fprintf(&b, "Motivo de consulta: %s\n", m)
	}
	if v := scribeVitalsText(req.Vitals); v != "" {
		fmt.Fprintf(&b, "Signos vitales registrados en el formulario: %s\n", v)
	}
	fmt.Fprintf(&b, "\nNotas rápidas del médico:\n%s\n", strings.TrimSpace(r.Redact(req.Notes)))
	return b.String()
}

func scribeVitalsText(v *ScribeVitals) string {
	if v == nil {
		return ""
	}
	parts := []string{}
	if base := vitalsText(&clinical_models.VitalSigns{
		Weight: v.Weight, Height: v.Height, BloodPressure: strings.TrimSpace(v.BloodPressure),
		Temperature: v.Temperature, HeartRate: v.HeartRate,
	}); base != "" {
		parts = append(parts, base)
	}
	if v.O2Saturation != nil {
		parts = append(parts, fmt.Sprintf("SatO2 %d%%", *v.O2Saturation))
	}
	return strings.Join(parts, ", ")
}

// scribeToolName is the function the model calls with the SOAP draft.
const scribeToolName = "ordenar_nota_soap"

const scribeSystemPrompt = `Eres un asistente para médicos. Recibes las notas rápidas que un médico tomó durante una consulta (sin datos de identificación) y las ordenas en formato SOAP.

Reglas:
- Usa SOLO lo que dicen las notas y el contexto dado. No inventes síntomas, hallazgos, signos vitales, valores, dosis, exámenes ni diagnósticos.
- Si una sección no tiene información en las notas, déjala vacía (""). No escribas "No consta" ni rellenes.
- Subjetivo: lo que refiere el paciente (síntomas, tiempo de evolución, antecedentes mencionados).
- Objetivo: examen físico y signos vitales mencionados.
- Análisis: la impresión diagnóstica que el médico escribió en las notas; no agregues diagnósticos propios.
- Plan: las indicaciones, exámenes y controles que el médico escribió. Si menciona medicamentos, cópialos tal cual; no agregues dosis, frecuencias ni duraciones que no estén escritas. No armes recetas.
- diagnosis_terms: los diagnósticos que el médico mencionó, como términos médicos completos en español, sin abreviaturas ni lateralidad (p. ej. "OMA derecha" → "otitis media aguda", "HTA" → "hipertensión arterial", "DM2" → "diabetes mellitus tipo 2"). Sin códigos. Vacío si no mencionó ninguno.
- Español, registro clínico, conciso; conserva la información relevante de las notas.
- Entrega el resultado llamando a la función ordenar_nota_soap.`

var scribeToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "subjective": {"type": "string", "description": "Subjetivo: lo que refiere el paciente. Vacío si no consta."},
    "objective": {"type": "string", "description": "Objetivo: examen físico y signos vitales. Vacío si no consta."},
    "assessment": {"type": "string", "description": "Análisis / impresión diagnóstica escrita por el médico. Vacío si no consta."},
    "plan": {"type": "string", "description": "Plan: indicaciones, exámenes, controles. Vacío si no consta."},
    "diagnosis_terms": {"type": "array", "items": {"type": "string"}, "description": "Diagnósticos mencionados en las notas, como términos (sin códigos)."}
  },
  "required": ["subjective", "objective", "assessment", "plan", "diagnosis_terms"]
}`)

// ScribeChatRequest is the chat request of one scribe use (model and
// reasoning effort are set from the resolved provider).
func ScribeChatRequest(prompt string) ai.ChatRequest {
	temperature := 0.1
	return ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: scribeSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Tools:       []ai.Tool{ai.FunctionTool(scribeToolName, "Registra la nota de la consulta ordenada en formato SOAP.", scribeToolSchema)},
		ToolChoice:  ai.ForceTool(scribeToolName),
		Temperature: &temperature,
		MaxTokens:   3000,
	}
}

// RawScribe is the tool arguments, read tolerantly.
type RawScribe struct {
	Subjective     looseString  `json:"subjective"`
	Objective      looseString  `json:"objective"`
	Assessment     looseString  `json:"assessment"`
	Plan           looseString  `json:"plan"`
	DiagnosisTerms looseStrings `json:"diagnosis_terms"`
}

// ScribeContent is the parsed answer.
type ScribeContent struct {
	Subjective, Objective, Assessment, Plan string
	DiagnosisTerms                          []string
}

// ParseScribe reads the model's answer: the tool call, else JSON in the text.
// Small models send "" / null / bare strings / lists where another shape is
// expected; each field is read on its own so one odd field never discards
// the answer. Placeholders like "No consta" become empty. False when no
// SOAP field has content.
func ParseScribe(res *ai.ChatResponse) (ScribeContent, bool) {
	var raw RawScribe
	parsed := false
	if args, ok := res.ToolArguments(scribeToolName); ok {
		parsed = decodeScribe(args, &raw)
	}
	if !parsed {
		if j := extractJSON(strings.TrimSpace(res.Content())); j != "" {
			parsed = decodeScribe(json.RawMessage(j), &raw)
		}
	}
	if !parsed {
		return ScribeContent{}, false
	}
	c := ScribeContent{
		Subjective: cleanSection(string(raw.Subjective)),
		Objective:  cleanSection(string(raw.Objective)),
		Assessment: cleanSection(string(raw.Assessment)),
		Plan:       cleanSection(string(raw.Plan)),
	}
	for _, t := range raw.DiagnosisTerms {
		if t = cleanSection(t); t != "" {
			c.DiagnosisTerms = append(c.DiagnosisTerms, t)
		}
	}
	if c.Subjective == "" && c.Objective == "" && c.Assessment == "" && c.Plan == "" {
		return c, false
	}
	return c, true
}

// decodeScribe decodes field by field, so a field of an unexpected shape is
// skipped instead of failing the whole object.
func decodeScribe(b json.RawMessage, raw *RawScribe) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(b, &fields) != nil {
		return false
	}
	lookup := func(names ...string) json.RawMessage {
		for k, v := range fields {
			for _, n := range names {
				if strings.EqualFold(k, n) {
					return v
				}
			}
		}
		return nil
	}
	for _, f := range []struct {
		dst   any
		names []string
	}{
		{&raw.Subjective, []string{"subjective", "subjetivo", "s"}},
		{&raw.Objective, []string{"objective", "objetivo", "o"}},
		{&raw.Assessment, []string{"assessment", "analisis", "análisis", "a"}},
		{&raw.Plan, []string{"plan", "p"}},
		{&raw.DiagnosisTerms, []string{"diagnosis_terms", "diagnoses", "diagnosticos", "diagnósticos"}},
	} {
		if v := lookup(f.names...); v != nil {
			_ = json.Unmarshal(v, f.dst)
		}
	}
	return true
}

// placeholders the model writes instead of leaving a section empty.
var emptyPlaceholders = map[string]bool{
	"no consta": true, "no consta.": true, "n/a": true, "na": true, "ninguno": true, "ninguna": true,
	"-": true, "no referido": true, "no especificado": true, "sin datos": true, "null": true, "none": true,
}

func cleanSection(s string) string {
	s = strings.TrimSpace(s)
	if emptyPlaceholders[strings.ToLower(strings.TrimRight(s, "."))] || emptyPlaceholders[strings.ToLower(s)] {
		return ""
	}
	return s
}

// looseString accepts a string, null, a number or a list (joined by lines).
type looseString string

func (l *looseString) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*l = looseString(s)
		return nil
	}
	var list []any
	if json.Unmarshal(b, &list) == nil {
		var parts []string
		for _, v := range list {
			if v == nil {
				continue
			}
			parts = append(parts, strings.TrimSpace(fmt.Sprint(v)))
		}
		*l = looseString(strings.Join(parts, "\n"))
		return nil
	}
	var v any
	if json.Unmarshal(b, &v) == nil && v != nil {
		if _, isObject := v.(map[string]any); !isObject {
			*l = looseString(fmt.Sprint(v))
		}
	}
	return nil
}

// looseStrings accepts a list of strings or of {term|text|name} objects, a
// bare string (split on ";" and new lines), "" or null.
type looseStrings []string

func (l *looseStrings) UnmarshalJSON(b []byte) error {
	*l = nil
	var s string
	if json.Unmarshal(b, &s) == nil {
		for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == '\n' }) {
			if part = strings.TrimSpace(part); part != "" {
				*l = append(*l, part)
			}
		}
		return nil
	}
	var list []json.RawMessage
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	for _, item := range list {
		var str string
		if json.Unmarshal(item, &str) == nil {
			*l = append(*l, str)
			continue
		}
		var obj map[string]any
		if json.Unmarshal(item, &obj) == nil {
			for _, k := range []string{"term", "text", "name", "title", "diagnosis"} {
				if v, ok := obj[k].(string); ok {
					*l = append(*l, v)
					break
				}
			}
		}
	}
	return nil
}
