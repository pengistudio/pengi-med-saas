package ai_services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/config"
	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/gorm"
)

// Diagnosis systems (UISettings.Clinical.DiagnosisSystem).
const (
	DiagnosisSystemCIE10 = "cie10"
	DiagnosisSystemCIE11 = "cie11"
)

// maxSuggestedDiagnoses caps the chips offered after one scribe use.
const maxSuggestedDiagnoses = 5

// DiagnosisSuggestion is a code that exists in the clinic's catalog, found
// from a term the model proposed. Code and Title have the shape the
// diagnosis picker stores (clinical_models.DiagnosisItem).
type DiagnosisSuggestion struct {
	Term  string `json:"term"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

// DiagnosisResolver finds the catalog entry for a diagnosis term, or nil.
// The model never supplies codes: only terms, resolved here, so a suggestion
// can only be a code the catalog has.
type DiagnosisResolver interface {
	Resolve(ctx context.Context, term string) (*DiagnosisSuggestion, error)
}

// ResolverFactory returns the resolver of a diagnosis system.
type ResolverFactory func(db *gorm.DB, system string) DiagnosisResolver

// DefaultResolvers resolves CIE-10 against the cie10_codes table and CIE-11
// against the icd-api container (same sources as the diagnosis picker).
func DefaultResolvers(db *gorm.DB, system string) DiagnosisResolver {
	if system == DiagnosisSystemCIE10 {
		return CIE10Resolver{DB: db}
	}
	cfg := config.Get()
	return CIE11Resolver{
		BaseURL: cfg.ICD11APIURL, Release: cfg.ICD11Release, Language: cfg.ICD11Language,
		HTTP: &http.Client{Timeout: 5 * time.Second},
	}
}

// ResolveDiagnoses turns the model's terms into catalog suggestions: one per
// term at most, duplicates and unresolved terms dropped, at most
// maxSuggestedDiagnoses. A resolver error drops that term only.
func ResolveDiagnoses(ctx context.Context, r DiagnosisResolver, terms []string) []DiagnosisSuggestion {
	out := []DiagnosisSuggestion{}
	if r == nil {
		return out
	}
	seenCode, seenTerm := map[string]bool{}, map[string]bool{}
	for _, term := range terms {
		term = strings.TrimSpace(term)
		key := strings.ToLower(foldAccentsForSearch(term))
		if utf8.RuneCountInString(term) < 3 || seenTerm[key] {
			continue
		}
		seenTerm[key] = true
		s, err := r.Resolve(ctx, term)
		if err != nil || s == nil || s.Code == "" || seenCode[s.Code] {
			continue
		}
		seenCode[s.Code] = true
		s.Term = term
		out = append(out, *s)
		if len(out) == maxSuggestedDiagnoses {
			break
		}
	}
	return out
}

// CIE10Resolver searches the seeded CIE-10 catalog by title: the whole term
// first, else every significant word of it; the shortest matching title
// wins (the most generic entry, e.g. "Hipertensión esencial (primaria)").
type CIE10Resolver struct {
	DB *gorm.DB
}

// likeEscaper neutralizes LIKE wildcards in model-produced terms.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// maxTermRunes caps a model-produced diagnosis term before it is resolved.
const maxTermRunes = 100

func (r CIE10Resolver) Resolve(ctx context.Context, term string) (*DiagnosisSuggestion, error) {
	if utf8.RuneCountInString(term) > maxTermRunes {
		term = string([]rune(term)[:maxTermRunes])
	}
	try := func(words []string) (*DiagnosisSuggestion, error) {
		q := r.DB.WithContext(ctx).Model(&clinical_models.Cie10Code{})
		for _, w := range words {
			q = q.Where(`LOWER(title) LIKE ? ESCAPE '\'`, "%"+likeEscaper.Replace(strings.ToLower(w))+"%")
		}
		var code clinical_models.Cie10Code
		if err := q.Order("LENGTH(title) ASC, code ASC").Limit(1).Find(&code).Error; err != nil {
			return nil, err
		}
		if code.Code == "" {
			return nil, nil
		}
		return &DiagnosisSuggestion{Code: code.Code, Title: code.Title}, nil
	}
	if s, err := try([]string{term}); err != nil || s != nil {
		return s, err
	}
	words := significantWords(term)
	if len(words) < 2 {
		return nil, nil
	}
	return try(words)
}

// significantWords are the words of 4+ letters (drops "de", "con", "la"…).
func significantWords(term string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(term, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '(' || r == ')' || r == '/' || r == '-'
	}) {
		if utf8.RuneCountInString(w) >= 4 {
			out = append(out, w)
		}
	}
	return out
}

// CIE11Resolver asks the WHO ICD-11 API container (icd-api) for the term and
// takes its first coded result, as the picker's search does.
type CIE11Resolver struct {
	BaseURL  string
	Release  string
	Language string
	HTTP     *http.Client
}

func (r CIE11Resolver) Resolve(ctx context.Context, term string) (*DiagnosisSuggestion, error) {
	params := url.Values{}
	params.Set("q", term)
	params.Set("flatResults", "true")
	params.Set("highlightingEnabled", "false")
	endpoint := strings.TrimRight(r.BaseURL, "/") + "/icd/release/11/" + r.Release + "/mms/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Api-Version", "v2")
	req.Header.Set("Accept-Language", r.Language)
	client := r.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("icd-11 search answered %d", res.StatusCode)
	}
	var body struct {
		DestinationEntities []struct {
			TheCode string `json:"theCode"`
			Title   string `json:"title"`
		} `json:"destinationEntities"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body); err != nil {
		return nil, err
	}
	for _, e := range body.DestinationEntities {
		if e.TheCode != "" {
			return &DiagnosisSuggestion{Code: e.TheCode, Title: e.Title}, nil
		}
	}
	return nil, nil
}

func foldAccentsForSearch(s string) string {
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n").Replace(strings.ToLower(s))
}
