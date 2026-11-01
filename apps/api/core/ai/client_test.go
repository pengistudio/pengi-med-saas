package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChat_NotConfigured(t *testing.T) {
	c := New("https://example.invalid/v1", "")
	if c.Configured() {
		t.Fatal("client without key must not be configured")
	}
	if _, err := c.Chat(context.Background(), ChatRequest{Model: "m"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestChat_SendsToolsAndReadsToolCallAndUsage(t *testing.T) {
	var got ChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"id":"c1","model":"m","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"t1","type":"function","function":{"name":"save","arguments":"{\"x\":1}"}}]}}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1/", "secret")
	tool := FunctionTool("save", "Save it", json.RawMessage(`{"type":"object","properties":{"x":{"type":"integer"}}}`))
	res, err := c.Chat(context.Background(), ChatRequest{
		Model:      "m",
		Messages:   []Message{{Role: "user", Content: "hola"}},
		Tools:      []Tool{tool},
		ToolChoice: ForceTool("save"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "m" || len(got.Tools) != 1 || got.Tools[0].Function.Name != "save" || got.ToolChoice == nil {
		t.Fatalf("request not forwarded: %+v", got)
	}
	args, ok := res.ToolArguments("save")
	if !ok || string(args) != `{"x":1}` {
		t.Fatalf("tool args = %s, %v", args, ok)
	}
	if _, ok := res.ToolArguments("other"); ok {
		t.Fatal("unexpected tool match")
	}
	if res.Usage.PromptTokens != 11 || res.Usage.CompletionTokens != 7 {
		t.Fatalf("usage = %+v", res.Usage)
	}
}

func TestChat_ProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "k").Chat(context.Background(), ChatRequest{Model: "m"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
		t.Fatalf("err = %v, want APIError 429", err)
	}
}

func TestChat_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "k").Chat(context.Background(), ChatRequest{Model: "m"}); err == nil {
		t.Fatal("want an error without choices")
	}
}

// The bearer key must only reach the configured host: a redirect is an error,
// never followed.
func TestChat_DoesNotFollowRedirects(t *testing.T) {
	hit := false
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	_, err := New(srv.URL, "k").Chat(context.Background(), ChatRequest{Model: "m"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTemporaryRedirect {
		t.Fatalf("err = %v, want APIError 307", err)
	}
	if hit {
		t.Fatal("redirect target was contacted")
	}
}
