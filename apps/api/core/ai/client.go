// Package ai is a small client for OpenAI-compatible chat completions APIs
// (DigitalOcean Inference by default): plain REST, tool calling, and the
// token usage of each call so callers can log and bill it.
//
// It knows nothing about tenants, quotas, prompts or where its settings come
// from: features/ai resolves the provider of each call (backoffice settings,
// else the AI_* env), builds a Client for it and decides what may be sent
// (see Redactor for identifiers).
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pengi-med-saas/core/config"
)

// ErrNotConfigured is returned by every call when no API key is set.
var ErrNotConfigured = errors.New("ai: no API key configured")

// defaultTimeout bounds one completion; open models answer a summary in a
// few seconds, a stuck upstream must not hold the request forever.
const defaultTimeout = 60 * time.Second

// maxErrorBody caps the upstream error body kept in APIError.
const maxErrorBody = 2048

// Client calls POST {baseURL}/chat/completions with a bearer key.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New returns a client for baseURL (e.g. https://inference.do-ai.run/v1).
// An empty apiKey yields a client whose calls fail with ErrNotConfigured.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http: &http.Client{
			Timeout: defaultTimeout,
			// Never follow a redirect: the bearer key must only reach the
			// configured host. A 3xx comes back as an APIError.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// Configured reports whether the client has an API key.
func (c *Client) Configured() bool { return c != nil && c.apiKey != "" && c.baseURL != "" }

// Message is one chat message. Content is a plain string (no multimodal).
type Message struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool is a function the model may call; Parameters is a JSON Schema object.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function FunctionSpec `json:"function"`
}

// FunctionSpec describes a callable function.
type FunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// FunctionTool builds a Tool from a name, description and JSON Schema.
func FunctionTool(name, description string, schema json.RawMessage) Tool {
	return Tool{Type: "function", Function: FunctionSpec{Name: name, Description: description, Parameters: schema}}
}

// ToolCall is a call the model made to one of the request's tools.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall carries the function name and its JSON arguments (a string,
// as the OpenAI API sends them).
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatRequest is the body of a chat completion.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  any       `json:"tool_choice,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	// ReasoningEffort ("none", "low", ...) is sent only when set: reasoning
	// models served locally (Ollama) are slow with thinking on.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// ForceTool returns the tool_choice value that makes the model call name.
func ForceTool(name string) any {
	return map[string]any{"type": "function", "function": map[string]string{"name": name}}
}

// Usage is the token count of one completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Choice is one completion alternative (only the first is used).
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatResponse is the answer of a chat completion.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// APIError is a non-2xx answer from the provider.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ai: provider answered %d: %s", e.Status, e.Body)
}

// Chat runs one chat completion. It does not retry: the caller decides
// whether a failed use is retried (and counted).
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	if req.Model == "" {
		return nil, errors.New("ai: model is required")
	}
	if req.ReasoningEffort == "" {
		req.ReasoningEffort = config.Get().AIReasoningEffort
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ai: encode request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai: request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBody))
		return nil, &APIError{Status: res.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	var out ChatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ai: decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("ai: response has no choices")
	}
	return &out, nil
}

// ToolArguments returns the arguments of the first call to the tool name in
// the first choice, or false when the model didn't call it.
func (r *ChatResponse) ToolArguments(name string) (json.RawMessage, bool) {
	if r == nil || len(r.Choices) == 0 {
		return nil, false
	}
	for _, call := range r.Choices[0].Message.ToolCalls {
		if call.Function.Name == name && strings.TrimSpace(call.Function.Arguments) != "" {
			return json.RawMessage(call.Function.Arguments), true
		}
	}
	return nil, false
}

// Content returns the text of the first choice.
func (r *ChatResponse) Content() string {
	if r == nil || len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}
