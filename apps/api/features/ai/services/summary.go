package ai_services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/ai"
	ai_models "pengi-med-saas/features/ai/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// Errors of Summarizer.Get, mapped by the handler to E-AI-* codes.
var (
	ErrNotConfigured   = errors.New("ai summary: AI is not configured")
	ErrNotEnabled      = errors.New("ai summary: the clinic has not enabled AI")
	ErrQuotaExceeded   = errors.New("ai summary: monthly quota reached")
	ErrNoHistory       = errors.New("ai summary: nothing to summarize")
	ErrPatientNotFound = errors.New("ai summary: patient not found")
	ErrProvider        = errors.New("ai summary: provider failed")
)

// Completer is the part of ai.Client a summary needs (a fake in tests).
type Completer interface {
	Configured() bool
	Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error)
}

// SummaryItem is one statement of the summary and the consultations it comes
// from (record ids; empty when it comes from the patient's background).
type SummaryItem struct {
	Text    string `json:"text"`
	Sources []uint `json:"sources"`
}

// SummaryContent is the structured summary the model returns.
type SummaryContent struct {
	Overview           string        `json:"overview"`
	ActiveProblems     []SummaryItem `json:"active_problems"`
	Background         string        `json:"background"`
	Allergies          string        `json:"allergies"`
	CurrentMedications []SummaryItem `json:"current_medications"`
	RelevantExams      []SummaryItem `json:"relevant_exams"`
	RecentEvolution    string        `json:"recent_evolution"`
	Alerts             []SummaryItem `json:"alerts"`
}

// Summary is what the patient file shows: always an AI draft, never part of
// the record.
type Summary struct {
	PatientID   uint            `json:"patient_id"`
	Content     SummaryContent  `json:"content"`
	Sources     []SummarySource `json:"sources"`
	Included    int             `json:"included_consultations"`
	Omitted     int             `json:"omitted_consultations"`
	Model       string          `json:"model"`
	GeneratedAt time.Time       `json:"generated_at"`
	Cached      bool            `json:"cached"`
	AIDraft     bool            `json:"ai_draft"`
}

// storedSummary is AISummary.Content.
type storedSummary struct {
	Content  SummaryContent  `json:"content"`
	Sources  []SummarySource `json:"sources"`
	Included int             `json:"included_consultations"`
	Omitted  int             `json:"omitted_consultations"`
}

// Dialer builds the client of one call from the resolved provider. It runs
// per call, so a provider changed in backoffice applies to the next use.
type Dialer func(p Provider) Completer

// DialProvider is the production Dialer: an ai.Client for p.
func DialProvider(p Provider) Completer { return ai.New(p.BaseURL, p.APIKey) }

// Summarizer generates and caches patient history summaries.
type Summarizer struct {
	Dial Dialer
	Now  func() time.Time
}

// NewSummarizer returns a Summarizer that dials the resolved provider of
// each call (DialProvider when dial is nil).
func NewSummarizer(dial Dialer) *Summarizer {
	if dial == nil {
		dial = DialProvider
	}
	return &Summarizer{Dial: dial, Now: time.Now}
}

// Cached returns the patient's cached summary when it still matches the
// patient's data, or nil. It never calls the model nor counts a use.
// tdb must be bound to the patient's tenant.
func (s *Summarizer) Cached(tdb *gorm.DB, patientID uint) (*Summary, error) {
	fp, err := Fingerprint(tdb, patientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	return cachedFor(tdb, patientID, fp)
}

func cachedFor(tdb *gorm.DB, patientID uint, fingerprint string) (*Summary, error) {
	var row ai_models.AISummary
	if err := tdb.Where("patient_id = ?", patientID).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 || row.SourceFingerprint != fingerprint {
		return nil, nil
	}
	var stored storedSummary
	if err := json.Unmarshal(row.Content, &stored); err != nil {
		return nil, nil // unreadable cache: regenerate
	}
	return &Summary{
		PatientID: patientID, Content: stored.Content, Sources: stored.Sources,
		Included: stored.Included, Omitted: stored.Omitted,
		Model: row.AIModel, GeneratedAt: row.UpdatedAt, Cached: true, AIDraft: true,
	}, nil
}

// Enabled reports whether the tenant opted in to AI functions (clinic
// settings, off by default).
func Enabled(db *gorm.DB, tenantID uint) (bool, error) {
	var t tenant_models.Tenant
	if err := db.Select("id", "ui_settings").First(&t, tenantID).Error; err != nil {
		return false, err
	}
	return tenant_models.ParseUISettings(t.UISettings).AI.Enabled, nil
}

// Generate returns the patient's summary: the cached one when the data
// didn't change, else a new one from the model, which counts one use against
// the monthly quota and is logged (tokens and cost, not the prompt).
// tdb must be bound to tenantID; db is the unbound handle.
func (s *Summarizer) Generate(ctx context.Context, db, tdb *gorm.DB, tenantID, userID, patientID uint) (*Summary, error) {
	enabled, err := Enabled(db, tenantID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrNotEnabled
	}
	fp, err := Fingerprint(tdb, patientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	if cached, err := cachedFor(tdb, patientID, fp); err != nil || cached != nil {
		return cached, err
	}
	now := s.Now()
	provider, client, err := openFunction(s.Dial, db, tenantID, ai_models.FunctionSummary, now)
	if err != nil {
		return nil, err
	}
	history, err := LoadHistory(tdb, patientID)
	if err != nil {
		return nil, err
	}
	input := BuildSummaryInput(history, now)
	if input.Empty {
		return nil, ErrNoHistory
	}

	model := provider.Model
	res, err := callAndLog(ctx, db, client, provider, SummaryRequest(model, input.Prompt),
		UsageEntry{TenantID: tenantID, UserID: userID, Function: ai_models.FunctionSummary}, now)
	if err != nil {
		return nil, err
	}
	content, ok := ParseSummary(res, input.Sources)
	if !ok {
		return nil, fmt.Errorf("%w: unusable answer", ErrProvider)
	}

	stored, err := json.Marshal(storedSummary{Content: content, Sources: input.Sources, Included: input.Included, Omitted: input.Omitted})
	if err != nil {
		return nil, err
	}
	row := ai_models.AISummary{TenantID: tenantID, PatientID: patientID}
	if err := tdb.Where("patient_id = ?", patientID).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	row.SourceFingerprint, row.Content, row.AIModel, row.GeneratedByUserID = fp, stored, model, userID
	if err := tdb.Save(&row).Error; err != nil {
		return nil, err
	}
	return &Summary{
		PatientID: patientID, Content: content, Sources: input.Sources,
		Included: input.Included, Omitted: input.Omitted,
		Model: model, GeneratedAt: row.UpdatedAt, Cached: false, AIDraft: true,
	}, nil
}

// summaryToolName is the function the model calls with the summary (DO's
// API documents no response_format, so structure comes from tool calling).
const summaryToolName = "registrar_resumen"

const summarySystemPrompt = `Eres un asistente para médicos. Recibes la historia clínica estructurada de UN paciente (sin datos de identificación) y escribes un resumen para el médico tratante antes de la consulta.

Reglas:
- Usa SOLO los datos proporcionados. No inventes datos, valores, fechas ni diagnósticos. Si algo no consta, omítelo.
- No propongas diagnósticos nuevos, tratamientos ni conductas: resume, no recomiendes.
- Español, estilo clínico, conciso; frases cortas.
- Cada afirmación que venga de una consulta debe citar su etiqueta (C1, C2, …) en "sources".
- Los resultados de exámenes no están incluidos: indica solo qué se pidió y su estado.
- Las alertas son hechos del registro que el médico no debería pasar por alto (alergias, paciente crítico, exámenes urgentes sin resultados, controles pendientes).
- Entrega el resumen llamando a la función registrar_resumen.`

var summaryItemSchema = `{"type":"object","properties":{"text":{"type":"string"},"sources":{"type":"array","items":{"type":"string"},"description":"Etiquetas de las consultas de origen, p. ej. [\"C2\",\"C5\"]"}},"required":["text"]}`

var summaryToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "overview": {"type": "string", "description": "Resumen general en 2 a 4 frases."},
    "active_problems": {"type": "array", "items": ` + summaryItemSchema + `, "description": "Problemas o diagnósticos activos."},
    "background": {"type": "string", "description": "Antecedentes personales, familiares y quirúrgicos relevantes."},
    "allergies": {"type": "string", "description": "Alergias registradas, o vacío si no constan."},
    "current_medications": {"type": "array", "items": ` + summaryItemSchema + `, "description": "Medicación de las recetas más recientes."},
    "relevant_exams": {"type": "array", "items": ` + summaryItemSchema + `, "description": "Exámenes pedidos y su estado."},
    "recent_evolution": {"type": "string", "description": "Evolución en las últimas consultas."},
    "alerts": {"type": "array", "items": ` + summaryItemSchema + `}
  },
  "required": ["overview", "active_problems", "current_medications", "alerts"]
}`)

// SummaryRequest is the chat request of a summary over the de-identified
// history text.
func SummaryRequest(model, history string) ai.ChatRequest {
	temperature := 0.2
	return ai.ChatRequest{
		Model: model,
		Messages: []ai.Message{
			{Role: "system", Content: summarySystemPrompt},
			{Role: "user", Content: "Historia clínica:\n\n" + history},
		},
		Tools:       []ai.Tool{ai.FunctionTool(summaryToolName, "Registra el resumen estructurado de la historia clínica.", summaryToolSchema)},
		ToolChoice:  ai.ForceTool(summaryToolName),
		Temperature: &temperature,
		// Reasoning models (gpt-oss) spend tokens thinking before the call.
		MaxTokens: 4000,
	}
}

// rawSummary is the tool arguments as the model writes them (labels, not ids).
type rawSummary struct {
	Overview           string   `json:"overview"`
	ActiveProblems     rawItems `json:"active_problems"`
	Background         string   `json:"background"`
	Allergies          string   `json:"allergies"`
	CurrentMedications rawItems `json:"current_medications"`
	RelevantExams      rawItems `json:"relevant_exams"`
	RecentEvolution    string   `json:"recent_evolution"`
	Alerts             rawItems `json:"alerts"`
}

type rawItem struct {
	Text    string   `json:"text"`
	Sources []string `json:"sources"`
}

// rawItems is a list of items that tolerates what small models send instead
// of an empty array: "" or null (nothing) or a bare string (one item).
type rawItems []rawItem

func (r *rawItems) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*r = nil
		if strings.TrimSpace(s) != "" {
			*r = rawItems{{Text: s}}
		}
		return nil
	}
	var list []rawItem
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	*r = list
	return nil
}

// ParseSummary reads the model's answer: the tool call, else JSON in the
// text, else the text as the overview. Labels become record ids; labels the
// input didn't have are dropped. False when there is nothing usable.
func ParseSummary(res *ai.ChatResponse, sources []SummarySource) (SummaryContent, bool) {
	var raw rawSummary
	parsed := false
	if args, ok := res.ToolArguments(summaryToolName); ok {
		parsed = json.Unmarshal(args, &raw) == nil
	}
	text := strings.TrimSpace(res.Content())
	if !parsed && text != "" {
		if j := extractJSON(text); j != "" && json.Unmarshal([]byte(j), &raw) == nil && raw.Overview != "" {
			parsed = true
		} else {
			raw = rawSummary{Overview: text}
			parsed = true
		}
	}
	if !parsed {
		return SummaryContent{}, false
	}
	byLabel := map[string]uint{}
	for _, s := range sources {
		byLabel[strings.ToUpper(s.Label)] = s.MedicalRecordID
	}
	items := func(in rawItems) []SummaryItem {
		out := make([]SummaryItem, 0, len(in))
		for _, it := range in {
			if strings.TrimSpace(it.Text) == "" {
				continue
			}
			ids := []uint{}
			seen := map[uint]bool{}
			for _, l := range it.Sources {
				if id, ok := byLabel[strings.ToUpper(strings.Trim(strings.TrimSpace(l), "[]"))]; ok && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
			out = append(out, SummaryItem{Text: strings.TrimSpace(it.Text), Sources: ids})
		}
		return out
	}
	content := SummaryContent{
		Overview:           strings.TrimSpace(raw.Overview),
		ActiveProblems:     items(raw.ActiveProblems),
		Background:         strings.TrimSpace(raw.Background),
		Allergies:          strings.TrimSpace(raw.Allergies),
		CurrentMedications: items(raw.CurrentMedications),
		RelevantExams:      items(raw.RelevantExams),
		RecentEvolution:    strings.TrimSpace(raw.RecentEvolution),
		Alerts:             items(raw.Alerts),
	}
	if content.Overview == "" && len(content.ActiveProblems) == 0 && content.RecentEvolution == "" {
		return content, false
	}
	return content, true
}

// extractJSON returns the outermost {...} of text (models sometimes wrap it
// in a code fence), or "".
func extractJSON(text string) string {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return ""
	}
	return text[start : end+1]
}
