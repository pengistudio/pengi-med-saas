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
	ai_models "pengi-med-saas/features/ai/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// Eval set of fictitious Spanish patient messages for the WhatsApp
// assistant (testdata/bot_eval_cases.json): FAQ questions, off-topic,
// prices not in the FAQ, symptoms, emergencies, asking for a person and
// prompt injections, each with the expected action. Offline (CI) it checks
// the pre-model triggers and the guards; against the real model:
//
//	AI_API_KEY=... go test ./features/ai/services -run TestBotEval -v -count=1
//
// AI_BASE_URL, AI_DEFAULT_SUMMARY_MODEL and AI_REASONING_EFFORT pick the
// provider and model. expect: "answer" (with must_mention), "handoff", or
// "safe" (a handoff, or an answer that passed the guards and has none of
// must_not_mention).

type botEvalFile struct {
	Knowledge struct {
		ClinicName  string                        `json:"clinic_name"`
		Address     string                        `json:"address"`
		Phone       string                        `json:"phone"`
		Hours       string                        `json:"hours"`
		FAQ         []whatsapp_models.BotFAQEntry `json:"faq"`
		BookingLink string                        `json:"booking_link"`
	} `json:"knowledge"`
	Cases []struct {
		Name           string   `json:"name"`
		Message        string   `json:"message"`
		Precheck       string   `json:"precheck"`
		Expect         string   `json:"expect"`
		MustMention    []string `json:"must_mention"`
		MustNotMention []string `json:"must_not_mention"`
	} `json:"cases"`
	GuardCases []struct {
		Reply string `json:"reply"`
		Want  string `json:"want"`
	} `json:"guard_cases"`
}

func loadBotEval(t *testing.T) (botEvalFile, BotKnowledge) {
	t.Helper()
	raw, err := os.ReadFile("testdata/bot_eval_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var f botEvalFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) < 15 {
		t.Fatalf("bot eval set has %d cases, want 15+", len(f.Cases))
	}
	k := f.Knowledge
	return f, BotKnowledge{ClinicName: k.ClinicName, Address: k.Address, Phone: k.Phone, Hours: k.Hours, FAQ: k.FAQ, BookingLink: k.BookingLink}
}

// TestBotEval_Offline runs in CI: every case's pre-model trigger and every
// guard case.
func TestBotEval_Offline(t *testing.T) {
	f, k := loadBotEval(t)
	for _, c := range f.Cases {
		if got := BotPrecheck(c.Message); got != c.Precheck {
			t.Errorf("%s: precheck = %q, want %q", c.Name, got, c.Precheck)
		}
		if c.Precheck != "" && c.Expect != "handoff" {
			t.Errorf("%s: a prechecked case can only expect a handoff", c.Name)
		}
		// The text only ever travels in the user role.
		req := BotChatRequest(k, c.Message, nil)
		if req.Messages[1].Role != "user" || !strings.Contains(req.Messages[1].Content, "Mensaje del paciente") ||
			strings.Contains(req.Messages[0].Content, "Mensaje del paciente") {
			t.Errorf("%s: patient text outside the user role", c.Name)
		}
	}
	for _, g := range f.GuardCases {
		if got := GuardBotReply(g.Reply, k); got != g.Want {
			t.Errorf("guard(%q) = %q, want %q", g.Reply, got, g.Want)
		}
	}
}

// TestBotEval runs the set against the real model; skipped without
// AI_API_KEY. Each case goes through the same steps as a real message:
// trigger, model, parse, guards.
func TestBotEval(t *testing.T) {
	cfg := config.Get()
	if cfg.AIAPIKey == "" {
		t.Skip("AI_API_KEY not set: eval skipped")
	}
	f, k := loadBotEval(t)
	client := ai.New(cfg.AIBaseURL, cfg.AIAPIKey)
	model := DefaultModel(ai_models.FunctionBot)
	passed := 0
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			action, reply, detail := whatsapp_models.BotActionHandoff, "", "precheck "+c.Precheck
			if reason := BotPrecheck(c.Message); reason == "" {
				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer cancel()
				req := BotChatRequest(k, c.Message, nil)
				req.Model = model
				start := time.Now()
				res, err := client.Chat(ctx, req)
				if err != nil {
					t.Fatalf("chat: %v", err)
				}
				answer, ok := ParseBotAnswer(res)
				switch {
				case !ok:
					detail = "unparseable: " + res.Content()
				case answer.Action != whatsapp_models.BotActionAnswer:
					detail = "model handoff: " + answer.Reason
				default:
					if guard := GuardBotReply(answer.Reply, k); guard != "" {
						detail = "guard " + guard + ": " + answer.Reply
					} else {
						action, reply, detail = whatsapp_models.BotActionAnswer, answer.Reply, "answer"
					}
				}
				t.Logf("%s in %s, tokens %d/%d: %s", model, time.Since(start).Round(time.Millisecond),
					res.Usage.PromptTokens, res.Usage.CompletionTokens, detail)
			}
			low := strings.ToLower(foldBot(reply))
			fail := false
			switch c.Expect {
			case "answer":
				if action != whatsapp_models.BotActionAnswer {
					t.Errorf("want an answer, got %s", detail)
					fail = true
				}
				for _, m := range c.MustMention {
					if !strings.Contains(low, strings.ToLower(foldBot(m))) {
						t.Errorf("answer lacks %q: %q", m, reply)
						fail = true
					}
				}
			case "handoff":
				if action != whatsapp_models.BotActionHandoff {
					t.Errorf("want a handoff, got answer %q", reply)
					fail = true
				}
			case "safe":
				for _, m := range c.MustNotMention {
					if strings.Contains(low, strings.ToLower(foldBot(m))) {
						t.Errorf("answer has %q: %q", m, reply)
						fail = true
					}
				}
			}
			if !fail {
				passed++
			}
		})
	}
	t.Logf("bot eval %s: %d/%d cases passed", model, passed, len(f.Cases))
}
