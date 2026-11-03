package ai_services

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/ai"
	"pengi-med-saas/core/config"
	ai_models "pengi-med-saas/features/ai/models"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// Eval set of fictitious Spanish consultation notes
// (testdata/scribe_eval_cases.json), run against the real model before
// switching the scribe's production model:
//
//	AI_API_KEY=... go test ./features/ai/services -run TestScribeEval -v -count=1
//
// AI_BASE_URL, AI_DEFAULT_SCRIBE_MODEL (else AI_DEFAULT_SUMMARY_MODEL) and
// AI_REASONING_EFFORT pick provider and model. Each case lists terms the
// draft must keep, terms it must not contain (identifiers, or doses,
// drugs and diagnoses the notes don't have: inventions), sections that must
// stay empty and the diagnosis terms expected. Without AI_API_KEY only the
// offline checks run.

type scribeEvalCase struct {
	Name    string `json:"name"`
	Patient struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		AgeYears  int    `json:"age_years"`
		Gender    string `json:"gender"`
	} `json:"patient"`
	Motive string        `json:"motive"`
	Vitals *ScribeVitals `json:"vitals"`
	Notes  string        `json:"notes"`
	Expect struct {
		MustMention    []string `json:"must_mention"`
		MustNotMention []string `json:"must_not_mention"`
		EmptySections  []string `json:"empty_sections"`
		TermsAny       []string `json:"terms_any"`
	} `json:"expect"`
}

func loadScribeEvalCases(t *testing.T) []scribeEvalCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/scribe_eval_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []scribeEvalCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 8 || len(cases) > 10 {
		t.Fatalf("scribe eval set has %d cases, want 8–10", len(cases))
	}
	return cases
}

func (c scribeEvalCase) input() string {
	p := clinical_models.Patient{
		FirstName: c.Patient.FirstName, LastName: c.Patient.LastName,
		BirthDate: evalNow.AddDate(-c.Patient.AgeYears, -1, 0), Gender: c.Patient.Gender,
	}
	return BuildScribeInput(p, ScribeRequest{Notes: c.Notes, Motive: c.Motive, Vitals: c.Vitals}, evalNow)
}

// isIdentifier tells identifiers in must_not_mention apart from invented
// clinical facts (only the former are checked offline).
func (c scribeEvalCase) isIdentifier(term string) bool {
	return strings.ContainsAny(term, "0123456789@") ||
		strings.Contains(c.Patient.FirstName+" "+c.Patient.LastName, term) ||
		term == "Diciembre" || term == "Salazar" || term == "mtorres"
}

// TestScribeEval_InputsAreDeidentified runs offline (CI): no case sends the
// patient's names, ID numbers, contacts, address or a professional's name.
func TestScribeEval_InputsAreDeidentified(t *testing.T) {
	for _, c := range loadScribeEvalCases(t) {
		prompt := strings.ToLower(c.input())
		leaks := strings.Fields(c.Patient.FirstName + " " + c.Patient.LastName)
		for _, term := range c.Expect.MustNotMention {
			if c.isIdentifier(term) {
				leaks = append(leaks, term)
			}
		}
		for _, leak := range leaks {
			// Whole words: "rosa" inside "dolorosa" is not a leak.
			if containsWord(prompt, strings.ToLower(leak)) {
				t.Errorf("%s: prompt leaks %q:\n%s", c.Name, leak, prompt)
			}
		}
	}
}

// TestScribeEval runs the eval set against the real model; skipped without
// AI_API_KEY.
func TestScribeEval(t *testing.T) {
	cfg := config.Get()
	if cfg.AIAPIKey == "" {
		t.Skip("AI_API_KEY not set: eval skipped")
	}
	client := ai.New(cfg.AIBaseURL, cfg.AIAPIKey)
	model := DefaultModel(ai_models.FunctionScribe)
	passed, cases := 0, loadScribeEvalCases(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			req := ScribeChatRequest(c.input())
			req.Model = model
			start := time.Now()
			res, err := client.Chat(ctx, req)
			if err != nil {
				t.Fatalf("chat: %v", err)
			}
			content, ok := ParseScribe(res)
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
				if containsWord(text, strings.ToLower(foldForEval(term))) {
					t.Errorf("mentions %q (identifier or not in the notes)", term)
					fail = true
				}
			}
			sections := map[string]string{"subjective": content.Subjective, "objective": content.Objective, "assessment": content.Assessment, "plan": content.Plan}
			for _, s := range c.Expect.EmptySections {
				if strings.TrimSpace(sections[s]) != "" {
					t.Errorf("section %s should be empty, got %q", s, sections[s])
					fail = true
				}
			}
			if len(c.Expect.TermsAny) > 0 {
				terms := strings.ToLower(foldForEval(strings.Join(content.DiagnosisTerms, " | ")))
				found := false
				for _, want := range c.Expect.TermsAny {
					found = found || strings.Contains(terms, strings.ToLower(foldForEval(want)))
				}
				if !found {
					t.Errorf("diagnosis terms %q lack any of %q", content.DiagnosisTerms, c.Expect.TermsAny)
					fail = true
				}
			}
			if !fail {
				passed++
			}
		})
	}
	t.Logf("scribe eval %s: %d/%d cases passed", model, passed, len(cases))
}

// containsWord reports whether word appears in text as a whole word (or
// phrase): "rosa" inside "dolorosa" doesn't count.
func containsWord(text, word string) bool {
	return regexp.MustCompile(`(^|[^\p{L}\d])` + regexp.QuoteMeta(word) + `($|[^\p{L}\d])`).MatchString(text)
}
