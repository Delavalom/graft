# Graft Core MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Graft Go library — a production-grade framework for building AI agents with multi-provider support, tools, streaming, guardrails, hooks, and OpenTelemetry observability.

**Architecture:** Domain-split packages under `github.com/delavalom/graft`. Root package defines core interfaces and types. Sub-packages (`provider/`, `hook/`, `guardrail/`, `stream/`, `otel/`) provide implementations. Functional options for configuration. Channels for streaming. Interfaces designed for future durable execution backends.

**Tech Stack:** Go 1.22+, `encoding/json`, `google/jsonschema-go` (schema generation), `go.opentelemetry.io/otel` (tracing/metrics), standard `net/http` (SSE adapter). No heavy dependencies in core.

**Spec:** `docs/superpowers/specs/2026-03-29-graft-core-design.md`

---

## File Structure

```
github.com/delavalom/graft/
├── go.mod
├── go.sum
├── graft.go                    — Package doc, NewAgent constructor, functional options
├── agent.go                    — Agent struct, ToolChoice type
├── tool.go                     — Tool interface, NewTool generic helper, ToolDefinition
├── message.go                  — Message, Role, ToolCall, ToolResult
├── stream.go                   — StreamEvent, EventType constants
├── handoff.go                  — Handoff struct, handoff-as-tool adapter
├── errors.go                   — AgentError, ErrorType constants, error helpers
├── result.go                   — Result, Usage, Cost, Trace
├── options.go                  — RunOption, RunConfig
├── runner.go                   — Runner interface, DefaultRunner implementation
├── guardrail.go                — Guardrail interface, Type, ValidationData, ValidationResult
│
├── provider/
│   ├── model.go                — LanguageModel interface, GenerateParams, GenerateResult, StreamChunk
│   ├── middleware.go           — Middleware type, WithLogging, WithRetry, WithRateLimit
│   ├── router.go               — Router, RoutingStrategy
│   ├── openai/
│   │   ├── openai.go           — OpenAI provider implementation
│   │   └── openai_test.go
│   ├── anthropic/
│   │   ├── anthropic.go        — Anthropic provider implementation
│   │   └── anthropic_test.go
│   └── google/
│       ├── google.go           — Google Gemini provider implementation
│       └── google_test.go
│
├── hook/
│   ├── events.go               — Event constants
│   ├── hook.go                 — HookCallback, HookPayload, HookResult
│   └── registry.go             — Registry implementation
│
├── guardrail/
│   └── builtin.go              — MaxTokens, ContentFilter, RequireToolConfirmation, SchemaValidator
│
├── stream/
│   ├── sse.go                  — SSEHandler
│   └── collect.go              — Collect helper
│
├── otel/
│   ├── tracing.go              — InstrumentRunner
│   ├── metrics.go              — InstrumentMetrics
│   └── attributes.go           — Semantic attribute constants
│
├── internal/
│   └── jsonschema/
│       └── generate.go         — JSON Schema generation from Go types via reflection
│
└── examples/
    ├── basic/
    │   └── main.go
    ├── multi-provider/
    │   └── main.go
    ├── streaming/
    │   └── main.go
    └── handoff/
        └── main.go
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `graft.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/delavalom/delavalom-labs/graft
go mod init github.com/delavalom/graft
```

- [ ] **Step 2: Create root package file with package doc**

Create `graft.go`:

```go
// Package graft is a Go-native framework for building AI agents and LLM-powered applications.
//
// Graft provides agent orchestration, multi-provider abstraction, tool execution,
// streaming, guardrails, lifecycle hooks, and OpenTelemetry observability.
//
// Basic usage:
//
//	agent := graft.NewAgent("assistant",
//	    graft.WithInstructions("You are a helpful assistant."),
//	    graft.WithTools(myTool),
//	)
//	runner := graft.NewDefaultRunner(model)
//	result, err := runner.Run(ctx, agent, messages)
package graft
```

- [ ] **Step 3: Commit**

```bash
git add go.mod graft.go
git commit -m "feat: initialize graft Go module and package doc"
```

---

## Task 2: Core Message Types

**Files:**
- Create: `message.go`
- Create: `message_test.go`

- [ ] **Step 1: Write the failing test**

Create `message_test.go`:

```go
package graft

import (
	"encoding/json"
	"testing"
)

func TestRoleConstants(t *testing.T) {
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %q, want %q", RoleSystem, "system")
	}
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want %q", RoleUser, "user")
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", RoleAssistant, "assistant")
	}
	if RoleTool != "tool" {
		t.Errorf("RoleTool = %q, want %q", RoleTool, "tool")
	}
}

func TestMessageJSON(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "hello",
		Metadata: map[string]any{
			"source": "test",
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Role != RoleUser {
		t.Errorf("Role = %q, want %q", got.Role, RoleUser)
	}
	if got.Content != "hello" {
		t.Errorf("Content = %q, want %q", got.Content, "hello")
	}
}

func TestToolCallJSON(t *testing.T) {
	tc := ToolCall{
		ID:        "call_123",
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ToolCall
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "call_123" {
		t.Errorf("ID = %q, want %q", got.ID, "call_123")
	}
	if got.Name != "search" {
		t.Errorf("Name = %q, want %q", got.Name, "search")
	}
}

func TestToolResultJSON(t *testing.T) {
	tr := ToolResult{
		CallID:  "call_123",
		Content: "result data",
		IsError: false,
	}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CallID != "call_123" {
		t.Errorf("CallID = %q, want %q", got.CallID, "call_123")
	}
	if got.IsError != false {
		t.Error("IsError = true, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run TestRole
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `message.go`:

```go
package graft

import "encoding/json"

// Role represents the sender of a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role            `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolResult *ToolResult     `json:"tool_result,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// ToolCall represents a request from the model to call a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of executing a tool call.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Content any    `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add message.go message_test.go
git commit -m "feat: add core message types (Message, ToolCall, ToolResult, Role)"
```

---

## Task 3: Error Types

**Files:**
- Create: `errors.go`
- Create: `errors_test.go`

- [ ] **Step 1: Write the failing test**

Create `errors_test.go`:

```go
package graft

import (
	"errors"
	"fmt"
	"testing"
)

func TestAgentErrorMessage(t *testing.T) {
	err := &AgentError{
		Type:    ErrToolExecution,
		Message: "tool search failed",
		Cause:   fmt.Errorf("connection refused"),
	}
	want := "graft: tool_execution: tool search failed: connection refused"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAgentErrorWithoutCause(t *testing.T) {
	err := &AgentError{
		Type:    ErrTimeout,
		Message: "max iterations exceeded",
	}
	want := "graft: timeout: max iterations exceeded"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAgentErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &AgentError{
		Type:    ErrProvider,
		Message: "API error",
		Cause:   cause,
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}

func TestAgentErrorIs(t *testing.T) {
	err := &AgentError{Type: ErrRateLimit, Message: "429"}
	if !errors.Is(err, ErrRateLimit) {
		t.Error("errors.Is should match ErrorType")
	}
	if errors.Is(err, ErrTimeout) {
		t.Error("errors.Is should not match different ErrorType")
	}
}

func TestNewAgentError(t *testing.T) {
	cause := fmt.Errorf("something broke")
	err := NewAgentError(ErrToolExecution, "search failed", cause)
	if err.Type != ErrToolExecution {
		t.Errorf("Type = %q, want %q", err.Type, ErrToolExecution)
	}
	if err.Cause != cause {
		t.Error("Cause not set")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run TestAgentError
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `errors.go`:

```go
package graft

import "fmt"

// ErrorType categorizes agent errors for programmatic handling.
type ErrorType string

const (
	ErrToolExecution   ErrorType = "tool_execution"
	ErrHandoff         ErrorType = "handoff"
	ErrGuardrail       ErrorType = "guardrail"
	ErrTimeout         ErrorType = "timeout"
	ErrContextLength   ErrorType = "context_length"
	ErrInvalidToolCall ErrorType = "invalid_tool_call"
	ErrRateLimit       ErrorType = "rate_limit"
	ErrProvider        ErrorType = "provider"
)

// Error implements the error interface so ErrorType can be used with errors.Is.
func (e ErrorType) Error() string {
	return string(e)
}

// AgentError is a structured error returned by Graft operations.
type AgentError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]any
}

// Error returns a human-readable error string.
func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("graft: %s: %s: %s", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("graft: %s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *AgentError) Unwrap() error {
	return e.Cause
}

// Is reports whether this error matches the target. Supports matching against ErrorType.
func (e *AgentError) Is(target error) bool {
	if t, ok := target.(ErrorType); ok {
		return e.Type == t
	}
	return false
}

// NewAgentError creates a new AgentError.
func NewAgentError(errType ErrorType, message string, cause error) *AgentError {
	return &AgentError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v -run TestAgentError
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add errors.go errors_test.go
git commit -m "feat: add structured error types (AgentError, ErrorType)"
```

---

## Task 4: Result Types

**Files:**
- Create: `result.go`
- Create: `result_test.go`

- [ ] **Step 1: Write the failing test**

Create `result_test.go`:

```go
package graft

import (
	"testing"
	"time"
)

func TestUsageTotalTokens(t *testing.T) {
	u := Usage{PromptTokens: 100, CompletionTokens: 50}
	if got := u.TotalTokens(); got != 150 {
		t.Errorf("TotalTokens() = %d, want 150", got)
	}
}

func TestResultFinalText(t *testing.T) {
	r := &Result{
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
			{Role: RoleAssistant, Content: "hello there"},
			{Role: RoleAssistant, Content: "how can I help?"},
		},
	}
	if got := r.LastAssistantText(); got != "how can I help?" {
		t.Errorf("LastAssistantText() = %q, want %q", got, "how can I help?")
	}
}

func TestResultLastAssistantTextEmpty(t *testing.T) {
	r := &Result{
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
		},
	}
	if got := r.LastAssistantText(); got != "" {
		t.Errorf("LastAssistantText() = %q, want empty", got)
	}
}

func TestCostTotal(t *testing.T) {
	c := Cost{InputCostUSD: 0.01, OutputCostUSD: 0.03}
	if got := c.TotalUSD(); got != 0.04 {
		t.Errorf("TotalUSD() = %f, want 0.04", got)
	}
}

func TestTraceAddSpan(t *testing.T) {
	tr := NewTrace("agent-1")
	tr.AddSpan(Span{
		Name:      "llm.generate",
		StartTime: time.Now(),
		Duration:  500 * time.Millisecond,
	})
	if len(tr.Spans) != 1 {
		t.Errorf("len(Spans) = %d, want 1", len(tr.Spans))
	}
	if tr.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", tr.AgentID, "agent-1")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run "TestUsage|TestResult|TestCost|TestTrace"
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `result.go`:

```go
package graft

import "time"

// Usage tracks token consumption for an agent run.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// TotalTokens returns prompt + completion tokens.
func (u Usage) TotalTokens() int {
	return u.PromptTokens + u.CompletionTokens
}

// Cost tracks monetary cost for an agent run.
type Cost struct {
	InputCostUSD  float64 `json:"input_cost_usd"`
	OutputCostUSD float64 `json:"output_cost_usd"`
}

// TotalUSD returns the total cost in USD.
func (c Cost) TotalUSD() float64 {
	return c.InputCostUSD + c.OutputCostUSD
}

// Span represents a single operation within a trace.
type Span struct {
	Name       string         `json:"name"`
	StartTime  time.Time      `json:"start_time"`
	Duration   time.Duration  `json:"duration"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Children   []Span         `json:"children,omitempty"`
}

// Trace tracks the execution path of an agent run.
type Trace struct {
	AgentID   string    `json:"agent_id"`
	StartTime time.Time `json:"start_time"`
	Spans     []Span    `json:"spans"`
}

// NewTrace creates a new Trace for an agent.
func NewTrace(agentID string) *Trace {
	return &Trace{
		AgentID:   agentID,
		StartTime: time.Now(),
	}
}

// AddSpan appends a span to the trace.
func (t *Trace) AddSpan(s Span) {
	t.Spans = append(t.Spans, s)
}

// Result holds the output of an agent run.
type Result struct {
	Messages []Message `json:"messages"`
	Usage    Usage     `json:"usage"`
	Cost     *Cost     `json:"cost,omitempty"`
	Trace    *Trace    `json:"trace,omitempty"`
}

// LastAssistantText returns the content of the last assistant message.
func (r *Result) LastAssistantText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == RoleAssistant && r.Messages[i].Content != "" {
			return r.Messages[i].Content
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v -run "TestUsage|TestResult|TestCost|TestTrace"
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add result.go result_test.go
git commit -m "feat: add result types (Result, Usage, Cost, Trace)"
```

---

## Task 5: StreamEvent Types

**Files:**
- Create: `stream.go`
- Create: `stream_test.go`

- [ ] **Step 1: Write the failing test**

Create `stream_test.go`:

```go
package graft

import (
	"testing"
	"time"
)

func TestStreamEventTypes(t *testing.T) {
	events := []EventType{
		EventTextDelta,
		EventToolCallStart,
		EventToolCallDelta,
		EventToolCallDone,
		EventToolResultDone,
		EventMessageDone,
		EventHandoff,
		EventError,
		EventDone,
	}
	seen := make(map[EventType]bool)
	for _, e := range events {
		if seen[e] {
			t.Errorf("duplicate event type: %q", e)
		}
		seen[e] = true
		if e == "" {
			t.Error("event type is empty string")
		}
	}
	if len(events) != 9 {
		t.Errorf("expected 9 event types, got %d", len(events))
	}
}

func TestStreamEventConstruction(t *testing.T) {
	ev := StreamEvent{
		Type:      EventTextDelta,
		Data:      "hello",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}
	if ev.Type != EventTextDelta {
		t.Errorf("Type = %q, want %q", ev.Type, EventTextDelta)
	}
	if ev.Data.(string) != "hello" {
		t.Errorf("Data = %v, want %q", ev.Data, "hello")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run TestStreamEvent
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `stream.go`:

```go
package graft

import "time"

// EventType identifies the kind of streaming event.
type EventType string

const (
	EventTextDelta      EventType = "text_delta"
	EventToolCallStart  EventType = "tool_call_start"
	EventToolCallDelta  EventType = "tool_call_delta"
	EventToolCallDone   EventType = "tool_call_done"
	EventToolResultDone EventType = "tool_result_done"
	EventMessageDone    EventType = "message_done"
	EventHandoff        EventType = "handoff"
	EventError          EventType = "error"
	EventDone           EventType = "done"
)

// StreamEvent represents a single event emitted during streaming execution.
type StreamEvent struct {
	Type      EventType `json:"type"`
	Data      any       `json:"data"`
	AgentID   string    `json:"agent_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v -run TestStreamEvent
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add stream.go stream_test.go
git commit -m "feat: add streaming event types (StreamEvent, EventType)"
```

---

## Task 6: JSON Schema Generation (Internal)

**Files:**
- Create: `internal/jsonschema/generate.go`
- Create: `internal/jsonschema/generate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/jsonschema/generate_test.go`:

```go
package jsonschema

import (
	"encoding/json"
	"testing"
)

type SimpleParams struct {
	Query string `json:"query" description:"The search query"`
	Limit int    `json:"limit,omitempty" description:"Max results"`
}

func TestGenerateFromType(t *testing.T) {
	schema, err := Generate[SimpleParams]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}

	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not a map")
	}
	if _, ok := props["query"]; !ok {
		t.Error("missing property: query")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("missing property: limit")
	}
}

type NestedParams struct {
	Name    string       `json:"name"`
	Options SubOptions   `json:"options"`
}

type SubOptions struct {
	Verbose bool `json:"verbose"`
}

func TestGenerateNested(t *testing.T) {
	schema, err := Generate[NestedParams]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	props := m["properties"].(map[string]any)
	opts, ok := props["options"].(map[string]any)
	if !ok {
		t.Fatal("options property not a map")
	}
	if opts["type"] != "object" {
		t.Errorf("options.type = %v, want object", opts["type"])
	}
}

func TestGenerateEmpty(t *testing.T) {
	type Empty struct{}
	schema, err := Generate[Empty]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/jsonschema/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write minimal implementation**

Create `internal/jsonschema/generate.go`:

```go
// Package jsonschema generates JSON Schema from Go types using reflection.
package jsonschema

import (
	"encoding/json"
	"reflect"
)

// Generate produces a JSON Schema from a Go struct type using generics.
func Generate[T any]() (json.RawMessage, error) {
	var zero T
	t := reflect.TypeOf(zero)
	schema := generateType(t)
	return json.Marshal(schema)
}

// GenerateFromType produces a JSON Schema from a reflect.Type.
func GenerateFromType(t reflect.Type) (json.RawMessage, error) {
	schema := generateType(t)
	return json.Marshal(schema)
}

func generateType(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return generateObject(t)
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{
			"type":  "array",
			"items": generateType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": generateType(t.Elem()),
		}
	default:
		return map[string]any{"type": "string"}
	}
}

func generateObject(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		omitempty := false
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			parts := splitTag(tag)
			if parts[0] != "" {
				name = parts[0]
			}
			for _, p := range parts[1:] {
				if p == "omitempty" {
					omitempty = true
				}
			}
		}

		prop := generateType(field.Type)

		if desc := field.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}

		properties[name] = prop

		if !omitempty {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func splitTag(tag string) []string {
	var parts []string
	current := ""
	for _, c := range tag {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/jsonschema/... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/
git commit -m "feat: add internal JSON Schema generation from Go types"
```

---

## Task 7: Tool Interface and Generic Helper

**Files:**
- Create: `tool.go`
- Create: `tool_test.go`

- [ ] **Step 1: Write the failing test**

Create `tool_test.go`:

```go
package graft

import (
	"context"
	"encoding/json"
	"testing"
)

type SearchParams struct {
	Query string `json:"query" description:"Search query"`
	Limit int    `json:"limit,omitempty" description:"Max results"`
}

type SearchResult struct {
	Results []string `json:"results"`
}

func TestNewToolName(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{Results: []string{"r1"}}, nil
		},
	)
	if got := tool.Name(); got != "search" {
		t.Errorf("Name() = %q, want %q", got, "search")
	}
}

func TestNewToolDescription(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	if got := tool.Description(); got != "Search the web" {
		t.Errorf("Description() = %q, want %q", got, "Search the web")
	}
}

func TestNewToolSchema(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	schema := tool.Schema()
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
	props := m["properties"].(map[string]any)
	if _, ok := props["query"]; !ok {
		t.Error("schema missing 'query' property")
	}
}

func TestNewToolExecute(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{Results: []string{"result for: " + p.Query}}, nil
		},
	)

	input := json.RawMessage(`{"query":"golang","limit":5}`)
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	sr, ok := result.(SearchResult)
	if !ok {
		t.Fatalf("result type = %T, want SearchResult", result)
	}
	if len(sr.Results) != 1 || sr.Results[0] != "result for: golang" {
		t.Errorf("Results = %v, want [result for: golang]", sr.Results)
	}
}

func TestNewToolExecuteInvalidJSON(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestToolDefinitionFromTool(t *testing.T) {
	tool := NewTool("greet", "Greet someone",
		func(ctx context.Context, p struct{ Name string }) (string, error) {
			return "hi " + p.Name, nil
		},
	)
	def := ToolDefFromTool(tool)
	if def.Name != "greet" {
		t.Errorf("Name = %q, want %q", def.Name, "greet")
	}
	if def.Description != "Greet someone" {
		t.Errorf("Description = %q, want %q", def.Description, "Greet someone")
	}
	if len(def.Schema) == 0 {
		t.Error("Schema is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run "TestNewTool|TestToolDef"
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `tool.go`:

```go
package graft

import (
	"context"
	"encoding/json"

	"github.com/delavalom/graft/internal/jsonschema"
)

// Tool defines a callable function that an agent can invoke.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (any, error)
}

// ToolDefinition is a schema-only representation of a tool for provider APIs.
// It contains no execute function — just the metadata needed for the LLM.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

// ToolDefFromTool creates a ToolDefinition from a Tool.
func ToolDefFromTool(t Tool) ToolDefinition {
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Schema:      t.Schema(),
	}
}

// typedTool is the generic implementation of Tool.
type typedTool[P any, R any] struct {
	name        string
	description string
	schema      json.RawMessage
	fn          func(ctx context.Context, params P) (R, error)
}

// NewTool creates a type-safe Tool from a Go function. The parameter type P must
// be a struct — its JSON Schema is auto-generated for the LLM.
func NewTool[P any, R any](name, description string, fn func(ctx context.Context, params P) (R, error)) Tool {
	schema, err := jsonschema.Generate[P]()
	if err != nil {
		// Schema generation from a struct should not fail at runtime.
		// If it does, store an empty schema and let validation catch it.
		schema = json.RawMessage(`{"type":"object"}`)
	}
	return &typedTool[P, R]{
		name:        name,
		description: description,
		schema:      schema,
		fn:          fn,
	}
}

func (t *typedTool[P, R]) Name() string              { return t.name }
func (t *typedTool[P, R]) Description() string        { return t.description }
func (t *typedTool[P, R]) Schema() json.RawMessage    { return t.schema }

func (t *typedTool[P, R]) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	var p P
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, NewAgentError(ErrInvalidToolCall, "failed to unmarshal tool parameters", err)
	}
	return t.fn(ctx, p)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v -run "TestNewTool|TestToolDef"
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add tool.go tool_test.go
git commit -m "feat: add Tool interface with generic type-safe NewTool helper"
```

---

## Task 8: Agent Struct and Functional Options

**Files:**
- Create: `agent.go`
- Create: `options.go`
- Modify: `graft.go`
- Create: `agent_test.go`

- [ ] **Step 1: Write the failing test**

Create `agent_test.go`:

```go
package graft

import (
	"context"
	"testing"
)

func TestNewAgentDefaults(t *testing.T) {
	agent := NewAgent("test-agent")
	if agent.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", agent.Name, "test-agent")
	}
	if agent.ToolChoice != ToolChoiceAuto {
		t.Errorf("ToolChoice = %v, want %v", agent.ToolChoice, ToolChoiceAuto)
	}
	if agent.Instructions != "" {
		t.Errorf("Instructions = %q, want empty", agent.Instructions)
	}
}

func TestNewAgentWithOptions(t *testing.T) {
	temp := 0.7
	maxTok := 1000
	tool := NewTool("test", "test tool",
		func(ctx context.Context, p struct{}) (string, error) {
			return "", nil
		},
	)

	agent := NewAgent("my-agent",
		WithInstructions("Be helpful."),
		WithModel("openai/gpt-4o"),
		WithTemperature(temp),
		WithMaxTokens(maxTok),
		WithToolChoice(ToolChoiceRequired),
		WithTools(tool),
		WithMetadata(map[string]any{"env": "test"}),
	)

	if agent.Instructions != "Be helpful." {
		t.Errorf("Instructions = %q, want %q", agent.Instructions, "Be helpful.")
	}
	if agent.Model != "openai/gpt-4o" {
		t.Errorf("Model = %q, want %q", agent.Model, "openai/gpt-4o")
	}
	if *agent.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", *agent.Temperature)
	}
	if *agent.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %v, want 1000", *agent.MaxTokens)
	}
	if agent.ToolChoice != ToolChoiceRequired {
		t.Errorf("ToolChoice = %v, want %v", agent.ToolChoice, ToolChoiceRequired)
	}
	if len(agent.Tools) != 1 {
		t.Errorf("len(Tools) = %d, want 1", len(agent.Tools))
	}
	if agent.Metadata["env"] != "test" {
		t.Errorf("Metadata[env] = %v, want test", agent.Metadata["env"])
	}
}

func TestToolChoiceSpecific(t *testing.T) {
	tc := ToolChoiceSpecific("search")
	if tc != ToolChoice("specific:search") {
		t.Errorf("ToolChoiceSpecific = %q, want %q", tc, "specific:search")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run "TestNewAgent|TestToolChoice"
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Write the Agent struct**

Create `agent.go`:

```go
package graft

// ToolChoice controls how the model selects tools.
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"
	ToolChoiceRequired ToolChoice = "required"
	ToolChoiceNone     ToolChoice = "none"
)

// ToolChoiceSpecific forces the model to call a specific tool by name.
func ToolChoiceSpecific(name string) ToolChoice {
	return ToolChoice("specific:" + name)
}

// Agent defines an AI agent with instructions, tools, and configuration.
type Agent struct {
	Name         string
	Instructions string
	Tools        []Tool
	Model        string
	Temperature  *float64
	MaxTokens    *int
	ToolChoice   ToolChoice
	Guardrails   []Guardrail
	Handoffs     []Handoff
	Hooks        *HookRegistry
	Metadata     map[string]any
}
```

- [ ] **Step 4: Write the functional options and constructor**

Update `graft.go` to add the constructor:

```go
package graft

// NewAgent creates a new Agent with the given name and options.
func NewAgent(name string, opts ...AgentOption) *Agent {
	a := &Agent{
		Name:       name,
		ToolChoice: ToolChoiceAuto,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}
```

Create `options.go`:

```go
package graft

// AgentOption configures an Agent.
type AgentOption func(*Agent)

// WithInstructions sets the agent's system prompt.
func WithInstructions(instructions string) AgentOption {
	return func(a *Agent) { a.Instructions = instructions }
}

// WithModel sets the model identifier (e.g. "openai/gpt-4o").
func WithModel(model string) AgentOption {
	return func(a *Agent) { a.Model = model }
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temp float64) AgentOption {
	return func(a *Agent) { a.Temperature = &temp }
}

// WithMaxTokens sets the maximum tokens for the response.
func WithMaxTokens(max int) AgentOption {
	return func(a *Agent) { a.MaxTokens = &max }
}

// WithToolChoice sets how the model selects tools.
func WithToolChoice(tc ToolChoice) AgentOption {
	return func(a *Agent) { a.ToolChoice = tc }
}

// WithTools adds tools to the agent.
func WithTools(tools ...Tool) AgentOption {
	return func(a *Agent) { a.Tools = append(a.Tools, tools...) }
}

// WithGuardrails adds guardrails to the agent.
func WithGuardrails(guardrails ...Guardrail) AgentOption {
	return func(a *Agent) { a.Guardrails = append(a.Guardrails, guardrails...) }
}

// WithHandoffs adds handoff targets to the agent.
func WithHandoffs(handoffs ...Handoff) AgentOption {
	return func(a *Agent) { a.Handoffs = append(a.Handoffs, handoffs...) }
}

// WithHooks sets the hook registry for the agent.
func WithHooks(hooks *HookRegistry) AgentOption {
	return func(a *Agent) { a.Hooks = hooks }
}

// WithMetadata sets metadata on the agent.
func WithMetadata(meta map[string]any) AgentOption {
	return func(a *Agent) { a.Metadata = meta }
}

// RunOption configures a single Run invocation.
type RunOption func(*RunConfig)

// RunConfig holds per-run configuration.
type RunConfig struct {
	MaxIterations int
	ParallelTools bool
}

// DefaultRunConfig returns the default run configuration.
func DefaultRunConfig() RunConfig {
	return RunConfig{
		MaxIterations: 10,
		ParallelTools: true,
	}
}

// WithMaxIterations sets the maximum number of agent loop iterations.
func WithMaxIterations(n int) RunOption {
	return func(c *RunConfig) { c.MaxIterations = n }
}

// WithParallelTools enables or disables parallel tool execution.
func WithParallelTools(enabled bool) RunOption {
	return func(c *RunConfig) { c.ParallelTools = enabled }
}
```

- [ ] **Step 5: Add stub types for Guardrail, Handoff, HookRegistry** (needed for Agent to compile)

Add to `guardrail.go`:

```go
package graft

import "context"

// GuardrailType categorizes when a guardrail runs.
type GuardrailType string

const (
	GuardrailInput  GuardrailType = "input"
	GuardrailOutput GuardrailType = "output"
	GuardrailTool   GuardrailType = "tool"
)

// Guardrail validates data at various points in the agent execution loop.
type Guardrail interface {
	Name() string
	Type() GuardrailType
	Validate(ctx context.Context, data *ValidationData) (*ValidationResult, error)
}

// ValidationData is the input to a guardrail check.
type ValidationData struct {
	Messages   []Message
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Agent      *Agent
}

// ValidationResult is the output of a guardrail check.
type ValidationResult struct {
	Pass     bool
	Message  string
	Modified any
}
```

Add to `handoff.go`:

```go
package graft

import "context"

// Handoff defines a transfer target for agent-to-agent handoff.
// Handoffs are exposed as pseudo-tools to the LLM.
type Handoff struct {
	Target      *Agent
	Description string
	Filter      func(ctx context.Context, messages []Message) bool
}
```

Add to `hook.go` (root package, temporary — registry will move to hook/ package later):

```go
package graft

import "context"

// HookEvent identifies a lifecycle event in the agent execution loop.
type HookEvent string

const (
	HookAgentStart    HookEvent = "agent_start"
	HookAgentEnd      HookEvent = "agent_end"
	HookPreToolCall   HookEvent = "pre_tool_call"
	HookPostToolCall  HookEvent = "post_tool_call"
	HookToolCallError HookEvent = "tool_call_error"
	HookPreHandoff    HookEvent = "pre_handoff"
	HookPostHandoff   HookEvent = "post_handoff"
	HookPreGenerate   HookEvent = "pre_generate"
	HookPostGenerate  HookEvent = "post_generate"
	HookGuardrailTrip HookEvent = "guardrail_trip"
)

// HookPayload carries context for a hook invocation.
type HookPayload struct {
	Event    HookEvent
	Agent    *Agent
	ToolCall *ToolCall
	Messages []Message
	Metadata map[string]any
}

// HookResult controls execution flow after a hook runs.
type HookResult struct {
	Allow         *bool
	ModifiedInput []byte
	AdditionalCtx string
	SkipExecution bool
}

// HookCallback is a function invoked at a lifecycle event.
type HookCallback func(ctx context.Context, payload *HookPayload) (*HookResult, error)

// HookRegistry stores registered hooks for lifecycle events.
type HookRegistry struct {
	hooks map[HookEvent][]HookCallback
}

// NewHookRegistry creates an empty HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{hooks: make(map[HookEvent][]HookCallback)}
}

// On registers a callback for a lifecycle event.
func (r *HookRegistry) On(event HookEvent, cb HookCallback) {
	r.hooks[event] = append(r.hooks[event], cb)
}

// Run executes all hooks for an event sequentially. Returns the first non-nil
// HookResult, or nil if all hooks pass through.
func (r *HookRegistry) Run(ctx context.Context, payload *HookPayload) (*HookResult, error) {
	if r == nil {
		return nil, nil
	}
	callbacks := r.hooks[payload.Event]
	for _, cb := range callbacks {
		result, err := cb(ctx, payload)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if result.Allow != nil && !*result.Allow {
				return result, nil
			}
			if result.SkipExecution {
				return result, nil
			}
		}
	}
	return nil, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./... -v -run "TestNewAgent|TestToolChoice"
```

Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add agent.go options.go graft.go agent_test.go guardrail.go handoff.go hook.go
git commit -m "feat: add Agent struct, functional options, Guardrail, Handoff, HookRegistry"
```

---

## Task 9: Hook System Tests

**Files:**
- Create: `hook_test.go`

- [ ] **Step 1: Write the tests**

Create `hook_test.go`:

```go
package graft

import (
	"context"
	"testing"
)

func TestHookRegistryOn(t *testing.T) {
	reg := NewHookRegistry()
	called := false
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		called = true
		return nil, nil
	})

	_, err := reg.Run(context.Background(), &HookPayload{Event: HookPreToolCall})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("hook was not called")
	}
}

func TestHookRegistryDeny(t *testing.T) {
	reg := NewHookRegistry()
	deny := false
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		deny := false
		return &HookResult{Allow: &deny}, nil
	})
	// Second hook should not run
	secondCalled := false
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		secondCalled = true
		return nil, nil
	})

	result, err := reg.Run(context.Background(), &HookPayload{Event: HookPreToolCall})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil || result.Allow == nil || *result.Allow != false {
		t.Error("expected deny result")
	}
	if secondCalled {
		t.Error("second hook should not have been called after deny")
	}
	_ = deny
}

func TestHookRegistryPassthrough(t *testing.T) {
	reg := NewHookRegistry()
	reg.On(HookAgentStart, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		return nil, nil // passthrough
	})

	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for passthrough")
	}
}

func TestHookRegistryNoHooks(t *testing.T) {
	reg := NewHookRegistry()
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentEnd})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no hooks registered")
	}
}

func TestHookRegistryNilSafe(t *testing.T) {
	var reg *HookRegistry
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil registry")
	}
}

func TestHookRegistryMultipleEvents(t *testing.T) {
	reg := NewHookRegistry()
	startCalled := false
	endCalled := false

	reg.On(HookAgentStart, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		startCalled = true
		return nil, nil
	})
	reg.On(HookAgentEnd, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		endCalled = true
		return nil, nil
	})

	reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if !startCalled {
		t.Error("start hook not called")
	}
	if endCalled {
		t.Error("end hook should not be called for start event")
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./... -v -run "TestHookRegistry"
```

Expected: All PASS (implementation was already written in Task 8).

- [ ] **Step 3: Commit**

```bash
git add hook_test.go
git commit -m "test: add comprehensive tests for HookRegistry"
```

---

## Task 10: Provider — LanguageModel Interface

**Files:**
- Create: `provider/model.go`
- Create: `provider/middleware.go`
- Create: `provider/router.go`
- Create: `provider/router_test.go`

- [ ] **Step 1: Write the failing test**

Create `provider/router_test.go`:

```go
package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/delavalom/graft"
)

// mockModel is a test double for LanguageModel.
type mockModel struct {
	id        string
	err       error
	result    *GenerateResult
	callCount int
}

func (m *mockModel) ModelID() string { return m.id }

func (m *mockModel) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockModel) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}

func TestRouterFallback(t *testing.T) {
	failing := &mockModel{id: "fail", err: fmt.Errorf("503 service unavailable")}
	working := &mockModel{
		id: "work",
		result: &GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "ok"},
		},
	}

	router := NewRouter(StrategyFallback, failing, working)
	result, err := router.Generate(context.Background(), GenerateParams{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "ok" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "ok")
	}
	if failing.callCount != 1 {
		t.Errorf("failing.callCount = %d, want 1", failing.callCount)
	}
	if working.callCount != 1 {
		t.Errorf("working.callCount = %d, want 1", working.callCount)
	}
}

func TestRouterAllFail(t *testing.T) {
	m1 := &mockModel{id: "m1", err: fmt.Errorf("error 1")}
	m2 := &mockModel{id: "m2", err: fmt.Errorf("error 2")}

	router := NewRouter(StrategyFallback, m1, m2)
	_, err := router.Generate(context.Background(), GenerateParams{})
	if err == nil {
		t.Fatal("expected error when all models fail")
	}
}

func TestRouterSingleModel(t *testing.T) {
	m := &mockModel{
		id: "m1",
		result: &GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "hello"},
		},
	}

	router := NewRouter(StrategyFallback, m)
	result, err := router.Generate(context.Background(), GenerateParams{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "hello" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "hello")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./provider/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the LanguageModel interface**

Create `provider/model.go`:

```go
// Package provider defines the LanguageModel interface and provider implementations.
package provider

import (
	"context"

	"github.com/delavalom/graft"
)

// LanguageModel is the interface that LLM providers implement.
type LanguageModel interface {
	Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
	Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error)
	ModelID() string
}

// GenerateParams configures an LLM generation request.
type GenerateParams struct {
	Messages    []graft.Message        `json:"messages"`
	Tools       []graft.ToolDefinition `json:"tools,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	ToolChoice  graft.ToolChoice       `json:"tool_choice,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

// GenerateResult holds the output of an LLM generation.
type GenerateResult struct {
	Message graft.Message `json:"message"`
	Usage   graft.Usage   `json:"usage"`
	Cost    *graft.Cost   `json:"cost,omitempty"`
}

// StreamChunk is a single chunk from a streaming LLM response.
type StreamChunk struct {
	Delta graft.StreamEvent `json:"delta"`
	Usage *graft.Usage      `json:"usage,omitempty"`
}
```

- [ ] **Step 4: Write the Middleware type**

Create `provider/middleware.go`:

```go
package provider

import (
	"context"
	"log/slog"
	"time"
)

// Middleware wraps a LanguageModel to add cross-cutting behavior.
type Middleware func(LanguageModel) LanguageModel

// Chain applies middleware in order: first middleware is outermost wrapper.
func Chain(model LanguageModel, mw ...Middleware) LanguageModel {
	for i := len(mw) - 1; i >= 0; i-- {
		model = mw[i](model)
	}
	return model
}

// --- Logging Middleware ---

type loggingModel struct {
	inner  LanguageModel
	logger *slog.Logger
}

// WithLogging returns middleware that logs generate calls.
func WithLogging(logger *slog.Logger) Middleware {
	return func(m LanguageModel) LanguageModel {
		return &loggingModel{inner: m, logger: logger}
	}
}

func (l *loggingModel) ModelID() string { return l.inner.ModelID() }

func (l *loggingModel) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	start := time.Now()
	l.logger.Info("llm.generate.start", "model", l.inner.ModelID(), "messages", len(params.Messages))
	result, err := l.inner.Generate(ctx, params)
	duration := time.Since(start)
	if err != nil {
		l.logger.Error("llm.generate.error", "model", l.inner.ModelID(), "duration", duration, "error", err)
		return nil, err
	}
	l.logger.Info("llm.generate.done",
		"model", l.inner.ModelID(),
		"duration", duration,
		"prompt_tokens", result.Usage.PromptTokens,
		"completion_tokens", result.Usage.CompletionTokens,
	)
	return result, nil
}

func (l *loggingModel) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	l.logger.Info("llm.stream.start", "model", l.inner.ModelID(), "messages", len(params.Messages))
	return l.inner.Stream(ctx, params)
}
```

- [ ] **Step 5: Write the Router**

Create `provider/router.go`:

```go
package provider

import (
	"context"
	"fmt"
)

// RoutingStrategy determines how the router selects models.
type RoutingStrategy string

const (
	StrategyFallback   RoutingStrategy = "fallback"
	StrategyRoundRobin RoutingStrategy = "round_robin"
)

// Router routes requests across multiple LanguageModel instances.
type Router struct {
	models   []LanguageModel
	strategy RoutingStrategy
	next     int // for round-robin
}

// NewRouter creates a Router with the given strategy and models.
func NewRouter(strategy RoutingStrategy, models ...LanguageModel) *Router {
	return &Router{
		models:   models,
		strategy: strategy,
	}
}

func (r *Router) ModelID() string {
	if len(r.models) > 0 {
		return "router:" + r.models[0].ModelID()
	}
	return "router:empty"
}

func (r *Router) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	switch r.strategy {
	case StrategyFallback:
		return r.generateFallback(ctx, params)
	default:
		return r.generateFallback(ctx, params)
	}
}

func (r *Router) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	switch r.strategy {
	case StrategyFallback:
		return r.streamFallback(ctx, params)
	default:
		return r.streamFallback(ctx, params)
	}
}

func (r *Router) generateFallback(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	var lastErr error
	for _, model := range r.models {
		result, err := model.Generate(ctx, params)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all models failed, last error: %w", lastErr)
}

func (r *Router) streamFallback(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	var lastErr error
	for _, model := range r.models {
		ch, err := model.Stream(ctx, params)
		if err == nil {
			return ch, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all models failed, last error: %w", lastErr)
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./provider/... -v
```

Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add provider/
git commit -m "feat: add provider package (LanguageModel, Router, Middleware)"
```

---

## Task 11: DefaultRunner Implementation

**Files:**
- Create: `runner.go`
- Create: `runner_test.go`

- [ ] **Step 1: Write the failing test**

Create `runner_test.go`:

```go
package graft

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/delavalom/graft/provider"
)

// fakeModel is a test double that returns predefined responses.
type fakeModel struct {
	responses []provider.GenerateResult
	callIdx   int
}

func (f *fakeModel) ModelID() string { return "fake" }

func (f *fakeModel) Generate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	if f.callIdx >= len(f.responses) {
		return &provider.GenerateResult{
			Message: Message{Role: RoleAssistant, Content: "no more responses"},
		}, nil
	}
	r := f.responses[f.callIdx]
	f.callIdx++
	return &r, nil
}

func (f *fakeModel) Stream(ctx context.Context, params provider.GenerateParams) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		result, err := f.Generate(ctx, params)
		if err != nil {
			return
		}
		if result.Message.Content != "" {
			ch <- provider.StreamChunk{
				Delta: StreamEvent{Type: EventTextDelta, Data: result.Message.Content},
			}
		}
		ch <- provider.StreamChunk{
			Delta: StreamEvent{Type: EventDone},
			Usage: &result.Usage,
		}
	}()
	return ch, nil
}

func TestRunnerSimpleResponse(t *testing.T) {
	model := &fakeModel{
		responses: []provider.GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "Hello!"}},
		},
	}

	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "Hello!" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Hello!")
	}
}

func TestRunnerToolCall(t *testing.T) {
	model := &fakeModel{
		responses: []provider.GenerateResult{
			// First response: model calls a tool
			{Message: Message{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{{
					ID:        "call_1",
					Name:      "greet",
					Arguments: json.RawMessage(`{"name":"Alice"}`),
				}},
			}},
			// Second response: model returns final text
			{Message: Message{Role: RoleAssistant, Content: "I greeted Alice for you."}},
		},
	}

	greetTool := NewTool("greet", "Greet someone",
		func(ctx context.Context, p struct{ Name string }) (string, error) {
			return "Hello, " + p.Name + "!", nil
		},
	)

	agent := NewAgent("test", WithTools(greetTool))
	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "Greet Alice"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "I greeted Alice for you." {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "I greeted Alice for you.")
	}
	// Should have 4 messages: user, assistant(tool_call), tool(result), assistant(final)
	if len(result.Messages) != 4 {
		t.Errorf("len(Messages) = %d, want 4", len(result.Messages))
	}
}

func TestRunnerMaxIterations(t *testing.T) {
	// Model always returns tool calls — should hit max iterations
	model := &fakeModel{
		responses: []provider.GenerateResult{
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c2", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c3", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
		},
	}

	noopTool := NewTool("noop", "No-op",
		func(ctx context.Context, p struct{}) (string, error) {
			return "done", nil
		},
	)

	agent := NewAgent("test", WithTools(noopTool))
	runner := NewDefaultRunner(model)
	_, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "loop forever"},
	}, WithMaxIterations(2))
	if err == nil {
		t.Fatal("expected error for max iterations exceeded")
	}
}

func TestRunnerStream(t *testing.T) {
	model := &fakeModel{
		responses: []provider.GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "streamed!"}},
		},
	}

	runner := NewDefaultRunner(model)
	events, err := runner.RunStream(context.Background(), NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}

	var gotText string
	for ev := range events {
		if ev.Type == EventTextDelta {
			gotText += ev.Data.(string)
		}
	}
	if gotText != "streamed!" {
		t.Errorf("streamed text = %q, want %q", gotText, "streamed!")
	}
}

func TestRunnerHandoff(t *testing.T) {
	targetAgent := NewAgent("specialist",
		WithInstructions("You are a specialist."),
	)

	model := &fakeModel{
		responses: []provider.GenerateResult{
			// First call: model triggers handoff
			{Message: Message{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{{
					ID:        "call_h",
					Name:      "handoff_specialist",
					Arguments: json.RawMessage(`{}`),
				}},
			}},
			// Second call (now specialist agent): final response
			{Message: Message{Role: RoleAssistant, Content: "Specialist here!"}},
		},
	}

	agent := NewAgent("generalist",
		WithHandoffs(Handoff{
			Target:      targetAgent,
			Description: "Transfer to specialist",
		}),
	)

	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "I need a specialist"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "Specialist here!" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Specialist here!")
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	model := &fakeModel{
		responses: []provider.GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "ok"}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	runner := NewDefaultRunner(model)
	_, err := runner.Run(ctx, NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./... -v -run "TestRunner"
```

Expected: FAIL — `NewDefaultRunner` not defined.

- [ ] **Step 3: Write the Runner interface and DefaultRunner**

Create `runner.go`:

```go
package graft

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/delavalom/graft/provider"
)

// Runner executes an agent against a conversation.
type Runner interface {
	Run(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (*Result, error)
	RunStream(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (<-chan StreamEvent, error)
}

// DefaultRunner implements Runner using a LanguageModel.
type DefaultRunner struct {
	model provider.LanguageModel
}

// NewDefaultRunner creates a Runner backed by the given model.
func NewDefaultRunner(model provider.LanguageModel) *DefaultRunner {
	return &DefaultRunner{model: model}
}

// Run executes the agent loop synchronously and returns the final result.
func (r *DefaultRunner) Run(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (*Result, error) {
	cfg := DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, NewAgentError(ErrTimeout, "context cancelled before start", err)
	}

	// Fire agent_start hook
	if agent.Hooks != nil {
		_, err := agent.Hooks.Run(ctx, &HookPayload{Event: HookAgentStart, Agent: agent, Messages: messages})
		if err != nil {
			return nil, err
		}
	}

	// Grace context for cleanup
	graceCtx, graceCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer graceCancel()
	defer func() {
		if agent.Hooks != nil {
			agent.Hooks.Run(graceCtx, &HookPayload{Event: HookAgentEnd, Agent: agent, Messages: messages})
		}
	}()

	// Build tool map (agent tools + handoff pseudo-tools)
	toolMap := r.buildToolMap(agent)

	// Build tool definitions for the provider
	toolDefs := r.buildToolDefs(agent, toolMap)

	// Copy messages to avoid mutating the input
	msgs := make([]Message, len(messages))
	copy(msgs, messages)

	// Prepend system message if agent has instructions
	if agent.Instructions != "" {
		msgs = append([]Message{{Role: RoleSystem, Content: agent.Instructions}}, msgs...)
	}

	var totalUsage Usage
	activeAgent := agent

	for i := 0; i < cfg.MaxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return nil, NewAgentError(ErrTimeout, "context cancelled during execution", err)
		}

		// Fire pre_generate hook
		if activeAgent.Hooks != nil {
			activeAgent.Hooks.Run(ctx, &HookPayload{Event: HookPreGenerate, Agent: activeAgent, Messages: msgs})
		}

		// Call the model
		params := provider.GenerateParams{
			Messages:    msgs,
			Tools:       toolDefs,
			Temperature: activeAgent.Temperature,
			MaxTokens:   activeAgent.MaxTokens,
			ToolChoice:  activeAgent.ToolChoice,
		}

		genResult, err := r.model.Generate(ctx, params)
		if err != nil {
			return nil, NewAgentError(ErrProvider, "model generation failed", err)
		}

		// Fire post_generate hook
		if activeAgent.Hooks != nil {
			activeAgent.Hooks.Run(ctx, &HookPayload{Event: HookPostGenerate, Agent: activeAgent, Messages: msgs})
		}

		// Accumulate usage
		totalUsage.PromptTokens += genResult.Usage.PromptTokens
		totalUsage.CompletionTokens += genResult.Usage.CompletionTokens

		// Append assistant message
		msgs = append(msgs, genResult.Message)

		// No tool calls — we're done
		if len(genResult.Message.ToolCalls) == 0 {
			return &Result{
				Messages: msgs,
				Usage:    totalUsage,
			}, nil
		}

		// Execute tool calls
		toolResults, handoffAgent, err := r.executeToolCalls(ctx, activeAgent, toolMap, genResult.Message.ToolCalls, cfg.ParallelTools)
		if err != nil {
			return nil, err
		}

		// Append tool results to messages
		for _, tr := range toolResults {
			msgs = append(msgs, Message{
				Role:       RoleTool,
				ToolResult: &tr,
			})
		}

		// Handle handoff: swap active agent
		if handoffAgent != nil {
			activeAgent = handoffAgent
			toolMap = r.buildToolMap(activeAgent)
			toolDefs = r.buildToolDefs(activeAgent, toolMap)

			// Replace system message for new agent
			if activeAgent.Instructions != "" && len(msgs) > 0 && msgs[0].Role == RoleSystem {
				msgs[0] = Message{Role: RoleSystem, Content: activeAgent.Instructions}
			}
		}
	}

	return nil, NewAgentError(ErrTimeout, fmt.Sprintf("max iterations (%d) exceeded", cfg.MaxIterations), nil)
}

// RunStream executes the agent loop and returns a channel of streaming events.
func (r *DefaultRunner) RunStream(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (<-chan StreamEvent, error) {
	cfg := DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := ctx.Err(); err != nil {
		return nil, NewAgentError(ErrTimeout, "context cancelled before start", err)
	}

	events := make(chan StreamEvent, 64)

	go func() {
		defer close(events)

		// For MVP, delegate to the model's Stream method for single-turn,
		// then fall back to Run for multi-turn tool loops.
		// This gives streaming for the final text response.
		result, err := r.Run(ctx, agent, messages, opts...)
		if err != nil {
			events <- StreamEvent{
				Type:      EventError,
				Data:      err,
				AgentID:   agent.Name,
				Timestamp: time.Now(),
			}
			return
		}

		// Emit text delta for the final assistant response
		text := result.LastAssistantText()
		if text != "" {
			events <- StreamEvent{
				Type:      EventTextDelta,
				Data:      text,
				AgentID:   agent.Name,
				Timestamp: time.Now(),
			}
		}

		events <- StreamEvent{
			Type:      EventDone,
			AgentID:   agent.Name,
			Timestamp: time.Now(),
		}
	}()

	return events, nil
}

// buildToolMap creates a name-to-Tool map including handoff pseudo-tools.
func (r *DefaultRunner) buildToolMap(agent *Agent) map[string]Tool {
	m := make(map[string]Tool)
	for _, t := range agent.Tools {
		m[t.Name()] = t
	}
	for _, h := range agent.Handoffs {
		name := "handoff_" + h.Target.Name
		m[name] = &handoffTool{handoff: h, name: name}
	}
	return m
}

// buildToolDefs creates ToolDefinitions for the provider from the tool map.
func (r *DefaultRunner) buildToolDefs(agent *Agent, toolMap map[string]Tool) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(toolMap))
	for _, t := range toolMap {
		defs = append(defs, ToolDefFromTool(t))
	}
	return defs
}

// executeToolCalls runs tool calls, optionally in parallel.
// Returns the results and an optional handoff target agent.
func (r *DefaultRunner) executeToolCalls(
	ctx context.Context,
	agent *Agent,
	toolMap map[string]Tool,
	calls []ToolCall,
	parallel bool,
) ([]ToolResult, *Agent, error) {
	results := make([]ToolResult, len(calls))
	var handoffAgent *Agent

	if parallel && len(calls) > 1 {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for i, call := range calls {
			wg.Add(1)
			go func(idx int, tc ToolCall) {
				defer wg.Done()
				tr, ha, err := r.executeSingleTool(ctx, agent, toolMap, tc)
				mu.Lock()
				defer mu.Unlock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				results[idx] = tr
				if ha != nil {
					handoffAgent = ha
				}
			}(i, call)
		}
		wg.Wait()

		if firstErr != nil {
			return nil, nil, firstErr
		}
	} else {
		for i, call := range calls {
			tr, ha, err := r.executeSingleTool(ctx, agent, toolMap, call)
			if err != nil {
				return nil, nil, err
			}
			results[i] = tr
			if ha != nil {
				handoffAgent = ha
			}
		}
	}

	return results, handoffAgent, nil
}

// executeSingleTool runs one tool call with hooks.
func (r *DefaultRunner) executeSingleTool(
	ctx context.Context,
	agent *Agent,
	toolMap map[string]Tool,
	call ToolCall,
) (ToolResult, *Agent, error) {
	tool, ok := toolMap[call.Name]
	if !ok {
		return ToolResult{
			CallID:  call.ID,
			Content: fmt.Sprintf("unknown tool: %s", call.Name),
			IsError: true,
		}, nil, nil
	}

	// Pre-tool hook
	if agent.Hooks != nil {
		hookResult, err := agent.Hooks.Run(ctx, &HookPayload{
			Event:    HookPreToolCall,
			Agent:    agent,
			ToolCall: &call,
			Messages: nil,
		})
		if err != nil {
			return ToolResult{CallID: call.ID, Content: err.Error(), IsError: true}, nil, nil
		}
		if hookResult != nil && hookResult.Allow != nil && !*hookResult.Allow {
			return ToolResult{CallID: call.ID, Content: "tool call denied by hook", IsError: true}, nil, nil
		}
	}

	// Execute
	output, err := tool.Execute(ctx, call.Arguments)

	// Check for handoff
	if ht, ok := tool.(*handoffTool); ok && err == nil {
		return ToolResult{
			CallID:  call.ID,
			Content: fmt.Sprintf("Handed off to %s", ht.handoff.Target.Name),
		}, ht.handoff.Target, nil
	}

	if err != nil {
		// Post-tool error hook
		if agent.Hooks != nil {
			agent.Hooks.Run(ctx, &HookPayload{
				Event:    HookToolCallError,
				Agent:    agent,
				ToolCall: &call,
			})
		}
		return ToolResult{
			CallID:  call.ID,
			Content: err.Error(),
			IsError: true,
		}, nil, nil
	}

	// Marshal output to string for the message
	content := formatToolOutput(output)

	// Post-tool hook
	if agent.Hooks != nil {
		agent.Hooks.Run(ctx, &HookPayload{
			Event:    HookPostToolCall,
			Agent:    agent,
			ToolCall: &call,
		})
	}

	return ToolResult{
		CallID:  call.ID,
		Content: content,
	}, nil, nil
}

// formatToolOutput converts a tool's return value to a string for messages.
func formatToolOutput(v any) any {
	switch val := v.(type) {
	case string:
		return val
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	}
}

// handoffTool is a pseudo-tool that triggers agent handoff.
type handoffTool struct {
	handoff Handoff
	name    string
}

func (h *handoffTool) Name() string           { return h.name }
func (h *handoffTool) Description() string     { return h.handoff.Description }
func (h *handoffTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (h *handoffTool) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	return fmt.Sprintf("Handing off to %s", h.handoff.Target.Name), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v -run "TestRunner"
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add runner.go runner_test.go
git commit -m "feat: add DefaultRunner with tool execution, handoffs, hooks, and streaming"
```

---

## Task 12: Built-in Guardrails

**Files:**
- Create: `guardrail/builtin.go`
- Create: `guardrail/builtin_test.go`

- [ ] **Step 1: Write the failing test**

Create `guardrail/builtin_test.go`:

```go
package guardrail

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
)

func TestMaxTokensPass(t *testing.T) {
	g := MaxTokens(100)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "short"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass for short message, got fail: %s", result.Message)
	}
}

func TestMaxTokensFail(t *testing.T) {
	g := MaxTokens(5) // ~5 tokens ≈ 20 chars
	longMsg := "This is a message that definitely exceeds five tokens worth of content and should fail the guardrail check"
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: longMsg}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for long message, got pass")
	}
}

func TestContentFilterPass(t *testing.T) {
	g := ContentFilter([]string{`(?i)password`, `(?i)secret`})
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "hello world"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
}

func TestContentFilterFail(t *testing.T) {
	g := ContentFilter([]string{`(?i)password`, `(?i)secret`})
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "my Password is 1234"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for message containing 'password'")
	}
}

func TestContentFilterName(t *testing.T) {
	g := ContentFilter([]string{`test`})
	if g.Name() != "content_filter" {
		t.Errorf("Name() = %q, want %q", g.Name(), "content_filter")
	}
	if g.Type() != graft.GuardrailInput {
		t.Errorf("Type() = %v, want %v", g.Type(), graft.GuardrailInput)
	}
}

func TestSchemaValidatorPass(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	g := SchemaValidator(schema)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: `{"name":"Alice"}`}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
}

func TestSchemaValidatorFail(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	g := SchemaValidator(schema)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: `{"age":30}`}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for missing required field")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./guardrail/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the built-in guardrails**

Create `guardrail/builtin.go`:

```go
// Package guardrail provides built-in guardrail implementations for Graft agents.
package guardrail

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/delavalom/graft"
)

// MaxTokens rejects input where total message content exceeds the estimated token limit.
// Uses a rough estimate of 4 characters per token.
func MaxTokens(limit int) graft.Guardrail {
	return &maxTokensGuardrail{limit: limit}
}

type maxTokensGuardrail struct {
	limit int
}

func (g *maxTokensGuardrail) Name() string           { return "max_tokens" }
func (g *maxTokensGuardrail) Type() graft.GuardrailType { return graft.GuardrailInput }

func (g *maxTokensGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	totalChars := 0
	for _, msg := range data.Messages {
		totalChars += len(msg.Content)
	}
	estimatedTokens := totalChars / 4
	if estimatedTokens > g.limit {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("estimated %d tokens exceeds limit of %d", estimatedTokens, g.limit),
		}, nil
	}
	return &graft.ValidationResult{Pass: true}, nil
}

// ContentFilter rejects messages matching any of the provided regex patterns.
func ContentFilter(patterns []string) graft.Guardrail {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return &contentFilterGuardrail{patterns: compiled}
}

type contentFilterGuardrail struct {
	patterns []*regexp.Regexp
}

func (g *contentFilterGuardrail) Name() string           { return "content_filter" }
func (g *contentFilterGuardrail) Type() graft.GuardrailType { return graft.GuardrailInput }

func (g *contentFilterGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	for _, msg := range data.Messages {
		for _, p := range g.patterns {
			if p.MatchString(msg.Content) {
				return &graft.ValidationResult{
					Pass:    false,
					Message: fmt.Sprintf("content matches blocked pattern: %s", p.String()),
				}, nil
			}
		}
	}
	return &graft.ValidationResult{Pass: true}, nil
}

// SchemaValidator validates that the last assistant message's content is valid JSON
// matching the provided JSON Schema. This is a simplified validator that checks
// required fields and basic types.
func SchemaValidator(schema json.RawMessage) graft.Guardrail {
	return &schemaValidatorGuardrail{schema: schema}
}

type schemaValidatorGuardrail struct {
	schema json.RawMessage
}

func (g *schemaValidatorGuardrail) Name() string           { return "schema_validator" }
func (g *schemaValidatorGuardrail) Type() graft.GuardrailType { return graft.GuardrailOutput }

func (g *schemaValidatorGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	// Find the last assistant message
	var content string
	for i := len(data.Messages) - 1; i >= 0; i-- {
		if data.Messages[i].Role == graft.RoleAssistant {
			content = data.Messages[i].Content
			break
		}
	}

	if content == "" {
		return &graft.ValidationResult{Pass: true}, nil
	}

	// Parse the content as JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("content is not valid JSON: %v", err),
		}, nil
	}

	// Parse schema to check required fields
	var schemaDef struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(g.schema, &schemaDef); err != nil {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("invalid schema: %v", err),
		}, nil
	}

	// Check required fields
	var missing []string
	for _, field := range schemaDef.Required {
		if _, ok := parsed[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")),
		}, nil
	}

	return &graft.ValidationResult{Pass: true}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./guardrail/... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add guardrail/
git commit -m "feat: add built-in guardrails (MaxTokens, ContentFilter, SchemaValidator)"
```

---

## Task 13: SSE Adapter and Stream Collect

**Files:**
- Create: `stream/sse.go`
- Create: `stream/collect.go`
- Create: `stream/sse_test.go`
- Create: `stream/collect_test.go`

- [ ] **Step 1: Write the failing tests**

Create `stream/collect_test.go`:

```go
package stream

import (
	"testing"
	"time"

	"github.com/delavalom/graft"
)

func TestCollect(t *testing.T) {
	ch := make(chan graft.StreamEvent, 3)
	ch <- graft.StreamEvent{Type: graft.EventTextDelta, Data: "Hello", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventTextDelta, Data: " world", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	result, err := Collect(ch)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.LastAssistantText() != "Hello world" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Hello world")
	}
}

func TestCollectError(t *testing.T) {
	ch := make(chan graft.StreamEvent, 2)
	ch <- graft.StreamEvent{Type: graft.EventError, Data: "something broke", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	_, err := Collect(ch)
	if err == nil {
		t.Fatal("expected error from Collect")
	}
}

func TestCollectEmpty(t *testing.T) {
	ch := make(chan graft.StreamEvent, 1)
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	result, err := Collect(ch)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.LastAssistantText() != "" {
		t.Errorf("expected empty text, got %q", result.LastAssistantText())
	}
}
```

Create `stream/sse_test.go`:

```go
package stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/delavalom/graft"
)

func TestSSEHandler(t *testing.T) {
	events := make(chan graft.StreamEvent, 3)
	events <- graft.StreamEvent{Type: graft.EventTextDelta, Data: "Hi", Timestamp: time.Now()}
	events <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(events)

	handler := SSEHandlerFromChannel(events)
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: text_delta") {
		t.Errorf("body missing text_delta event:\n%s", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Errorf("body missing done event:\n%s", body)
	}
}

func TestSSEEventFormat(t *testing.T) {
	ev := graft.StreamEvent{
		Type: graft.EventTextDelta,
		Data: "hello",
	}
	line := formatSSEEvent(ev)
	if !strings.HasPrefix(line, "event: text_delta\n") {
		t.Errorf("unexpected prefix: %q", line)
	}
	// Data should be valid JSON
	dataLine := strings.TrimPrefix(strings.Split(line, "\n")[1], "data: ")
	var m map[string]any
	if err := json.Unmarshal([]byte(dataLine), &m); err != nil {
		t.Errorf("data is not valid JSON: %v, line: %q", err, dataLine)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./stream/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the Collect helper**

Create `stream/collect.go`:

```go
// Package stream provides adapters for consuming Graft streaming events.
package stream

import (
	"fmt"

	"github.com/delavalom/graft"
)

// Collect drains a StreamEvent channel and assembles a Result.
func Collect(events <-chan graft.StreamEvent) (*graft.Result, error) {
	var textParts []string
	var lastErr error

	for ev := range events {
		switch ev.Type {
		case graft.EventTextDelta:
			if s, ok := ev.Data.(string); ok {
				textParts = append(textParts, s)
			}
		case graft.EventError:
			switch v := ev.Data.(type) {
			case error:
				lastErr = v
			case string:
				lastErr = fmt.Errorf("%s", v)
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	fullText := ""
	for _, p := range textParts {
		fullText += p
	}

	var messages []graft.Message
	if fullText != "" {
		messages = append(messages, graft.Message{
			Role:    graft.RoleAssistant,
			Content: fullText,
		})
	}

	return &graft.Result{
		Messages: messages,
	}, nil
}
```

- [ ] **Step 4: Write the SSE adapter**

Create `stream/sse.go`:

```go
package stream

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/delavalom/graft"
)

// SSEHandlerFromChannel creates an http.HandlerFunc that streams events as SSE.
func SSEHandlerFromChannel(events <-chan graft.StreamEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		ctx := r.Context()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				line := formatSSEEvent(ev)
				fmt.Fprint(w, line)
				flusher.Flush()

				if ev.Type == graft.EventDone {
					return
				}
			}
		}
	}
}

// formatSSEEvent formats a StreamEvent as an SSE message.
func formatSSEEvent(ev graft.StreamEvent) string {
	data, _ := json.Marshal(map[string]any{
		"type":     ev.Type,
		"data":     ev.Data,
		"agent_id": ev.AgentID,
	})
	return fmt.Sprintf("event: %s\ndata: %s\n\n", ev.Type, data)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./stream/... -v
```

Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add stream/
git commit -m "feat: add stream package (SSE adapter, Collect helper)"
```

---

## Task 14: OpenTelemetry Instrumentation

**Files:**
- Create: `otel/attributes.go`
- Create: `otel/tracing.go`
- Create: `otel/metrics.go`
- Create: `otel/tracing_test.go`

- [ ] **Step 1: Add OpenTelemetry dependencies**

```bash
cd /Users/delavalom/delavalom-labs/graft
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/trace
go get go.opentelemetry.io/otel/metric
go get go.opentelemetry.io/otel/attribute
```

- [ ] **Step 2: Write the failing test**

Create `otel/tracing_test.go`:

```go
package otel

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

type noopModel struct{}

func (n *noopModel) ModelID() string { return "noop" }
func (n *noopModel) Generate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	return &provider.GenerateResult{
		Message: graft.Message{Role: graft.RoleAssistant, Content: "ok"},
		Usage:   graft.Usage{PromptTokens: 10, CompletionTokens: 5},
	}, nil
}
func (n *noopModel) Stream(ctx context.Context, params provider.GenerateParams) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	close(ch)
	return ch, nil
}

func TestInstrumentRunnerReturnsRunner(t *testing.T) {
	model := &noopModel{}
	runner := graft.NewDefaultRunner(model)
	instrumented := InstrumentRunner(runner)

	// Should still satisfy the Runner interface
	var _ graft.Runner = instrumented

	result, err := instrumented.Run(context.Background(), graft.NewAgent("test"), []graft.Message{
		{Role: graft.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "ok" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "ok")
	}
}

func TestAttributeConstants(t *testing.T) {
	// Verify attribute constants are non-empty
	attrs := []string{
		AttrAgentName,
		AttrModelID,
		AttrProviderName,
		AttrPromptTokens,
		AttrCompletionTokens,
		AttrTotalTokens,
		AttrCostUSD,
		AttrToolName,
		AttrToolDuration,
		AttrIterationCount,
	}
	for _, a := range attrs {
		if a == "" {
			t.Error("found empty attribute constant")
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./otel/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 4: Write the attributes**

Create `otel/attributes.go`:

```go
// Package otel provides OpenTelemetry instrumentation for Graft agents.
package otel

// Semantic attribute keys for Graft spans and metrics.
const (
	AttrAgentName        = "graft.agent.name"
	AttrModelID          = "graft.model.id"
	AttrProviderName     = "graft.provider.name"
	AttrPromptTokens     = "graft.usage.prompt_tokens"
	AttrCompletionTokens = "graft.usage.completion_tokens"
	AttrTotalTokens      = "graft.usage.total_tokens"
	AttrCostUSD          = "graft.cost.total_usd"
	AttrToolName         = "graft.tool.name"
	AttrToolDuration     = "graft.tool.duration_ms"
	AttrIterationCount   = "graft.run.iterations"
)
```

- [ ] **Step 5: Write the tracing wrapper**

Create `otel/tracing.go`:

```go
package otel

import (
	"context"
	"time"

	"github.com/delavalom/graft"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/delavalom/graft/otel"

// InstrumentRunner wraps a Runner with OpenTelemetry tracing.
func InstrumentRunner(runner graft.Runner, opts ...Option) graft.Runner {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	tracer := cfg.tracerProvider.Tracer(tracerName)
	return &tracingRunner{
		inner:  runner,
		tracer: tracer,
	}
}

// Option configures the tracing instrumentation.
type Option func(*config)

type config struct {
	tracerProvider trace.TracerProvider
}

func defaultConfig() config {
	return config{
		tracerProvider: otel.GetTracerProvider(),
	}
}

// WithTracerProvider sets a custom TracerProvider.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) { c.tracerProvider = tp }
}

type tracingRunner struct {
	inner  graft.Runner
	tracer trace.Tracer
}

func (t *tracingRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	ctx, span := t.tracer.Start(ctx, "agent.run",
		trace.WithAttributes(
			attribute.String(AttrAgentName, agent.Name),
			attribute.String(AttrModelID, agent.Model),
		),
	)
	defer span.End()

	start := time.Now()
	result, err := t.inner.Run(ctx, agent, messages, opts...)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int(AttrPromptTokens, result.Usage.PromptTokens),
		attribute.Int(AttrCompletionTokens, result.Usage.CompletionTokens),
		attribute.Int(AttrTotalTokens, result.Usage.TotalTokens()),
		attribute.Int64(AttrToolDuration, duration.Milliseconds()),
	)

	return result, nil
}

func (t *tracingRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	ctx, span := t.tracer.Start(ctx, "agent.run_stream",
		trace.WithAttributes(
			attribute.String(AttrAgentName, agent.Name),
			attribute.String(AttrModelID, agent.Model),
		),
	)

	events, err := t.inner.RunStream(ctx, agent, messages, opts...)
	if err != nil {
		span.RecordError(err)
		span.End()
		return nil, err
	}

	// Wrap the channel to end the span when streaming completes
	wrapped := make(chan graft.StreamEvent, 64)
	go func() {
		defer close(wrapped)
		defer span.End()
		for ev := range events {
			wrapped <- ev
		}
	}()

	return wrapped, nil
}
```

- [ ] **Step 6: Write the metrics wrapper**

Create `otel/metrics.go`:

```go
package otel

import (
	"context"
	"time"

	"github.com/delavalom/graft"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// InstrumentMetrics wraps a Runner with OpenTelemetry metrics.
func InstrumentMetrics(runner graft.Runner, meter metric.Meter) graft.Runner {
	runDuration, _ := meter.Float64Histogram("graft.run.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Agent run duration in milliseconds"))

	promptTokens, _ := meter.Int64Counter("graft.llm.tokens.prompt",
		metric.WithDescription("Total prompt tokens consumed"))

	completionTokens, _ := meter.Int64Counter("graft.llm.tokens.completion",
		metric.WithDescription("Total completion tokens consumed"))

	toolErrors, _ := meter.Int64Counter("graft.tool.errors",
		metric.WithDescription("Tool execution failures"))

	return &metricsRunner{
		inner:            runner,
		runDuration:      runDuration,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		toolErrors:       toolErrors,
	}
}

type metricsRunner struct {
	inner            graft.Runner
	runDuration      metric.Float64Histogram
	promptTokens     metric.Int64Counter
	completionTokens metric.Int64Counter
	toolErrors       metric.Int64Counter
}

func (m *metricsRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	start := time.Now()
	attrs := metric.WithAttributes(attribute.String(AttrAgentName, agent.Name))

	result, err := m.inner.Run(ctx, agent, messages, opts...)

	duration := float64(time.Since(start).Milliseconds())
	m.runDuration.Record(ctx, duration, attrs)

	if err != nil {
		m.toolErrors.Add(ctx, 1, attrs)
		return nil, err
	}

	m.promptTokens.Add(ctx, int64(result.Usage.PromptTokens), attrs)
	m.completionTokens.Add(ctx, int64(result.Usage.CompletionTokens), attrs)

	return result, nil
}

func (m *metricsRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	return m.inner.RunStream(ctx, agent, messages, opts...)
}
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
go test ./otel/... -v
```

Expected: All PASS.

- [ ] **Step 8: Commit**

```bash
git add otel/
git commit -m "feat: add OpenTelemetry instrumentation (tracing, metrics, attributes)"
```

---

## Task 15: OpenAI-Compatible Provider

**Files:**
- Create: `provider/openai/openai.go`
- Create: `provider/openai/openai_test.go`

- [ ] **Step 1: Write the failing test**

Create `provider/openai/openai_test.go`:

```go
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

func TestNewClient(t *testing.T) {
	c := New(
		WithAPIKey("test-key"),
		WithModel("gpt-4o"),
	)
	if c.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", c.ModelID(), "gpt-4o")
	}
}

func TestNewClientWithBaseURL(t *testing.T) {
	c := New(
		WithAPIKey("test-key"),
		WithBaseURL("https://openrouter.ai/api/v1"),
		WithModel("anthropic/claude-sonnet-4-20250514"),
	)
	if c.ModelID() != "anthropic/claude-sonnet-4-20250514" {
		t.Errorf("ModelID() = %q, want %q", c.ModelID(), "anthropic/claude-sonnet-4-20250514")
	}
}

func TestGenerate(t *testing.T) {
	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		resp := map[string]any{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello from mock!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithModel("gpt-4o"),
	)

	result, err := c.Generate(context.Background(), provider.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "Hello from mock!" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "Hello from mock!")
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", result.Usage.CompletionTokens)
	}
}

func TestGenerateWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":      "chatcmpl-456",
			"object":  "chat.completion",
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": nil,
						"tool_calls": []map[string]any{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]any{
									"name":      "search",
									"arguments": `{"query":"golang"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(WithAPIKey("key"), WithBaseURL(server.URL), WithModel("gpt-4o"))
	result, err := c.Generate(context.Background(), provider.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "search for golang"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(result.Message.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.Message.ToolCalls))
	}
	tc := result.Message.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Name != "search" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "search")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./provider/openai/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the OpenAI provider**

Create `provider/openai/openai.go`:

```go
// Package openai provides an OpenAI-compatible LanguageModel implementation.
// Works with OpenAI, OpenRouter, Ollama, LM Studio, and any OpenAI-compatible API.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client implements provider.LanguageModel for OpenAI-compatible APIs.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	headers map[string]string
}

// Option configures the OpenAI client.
type Option func(*Client)

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithBaseURL sets the base URL (for OpenRouter, Ollama, etc).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithModel sets the model identifier.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.http = client }
}

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers[key] = value }
}

// New creates an OpenAI-compatible client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		http:    http.DefaultClient,
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) ModelID() string { return c.model }

// Generate sends a chat completion request and returns the result.
func (c *Client) Generate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	reqBody := c.buildRequest(params)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)), nil)
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "unmarshal response", err)
	}

	return c.parseResponse(&chatResp)
}

// Stream sends a streaming chat completion request.
func (c *Client) Stream(ctx context.Context, params provider.GenerateParams) (<-chan provider.StreamChunk, error) {
	reqBody := c.buildRequest(params)
	reqBody["stream"] = true

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "stream request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan provider.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.readSSEStream(resp.Body, ch)
	}()

	return ch, nil
}

func (c *Client) buildRequest(params provider.GenerateParams) map[string]any {
	messages := make([]map[string]any, 0, len(params.Messages))
	for _, msg := range params.Messages {
		m := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
		if msg.Role == graft.RoleTool && msg.ToolResult != nil {
			m["tool_call_id"] = msg.ToolResult.CallID
			content, _ := json.Marshal(msg.ToolResult.Content)
			m["content"] = string(content)
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": string(tc.Arguments),
					},
				})
			}
			m["tool_calls"] = toolCalls
		}
		messages = append(messages, m)
	}

	req := map[string]any{
		"model":    c.model,
		"messages": messages,
	}

	if len(params.Tools) > 0 {
		tools := make([]map[string]any, 0, len(params.Tools))
		for _, t := range params.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  json.RawMessage(t.Schema),
				},
			})
		}
		req["tools"] = tools
	}

	if params.Temperature != nil {
		req["temperature"] = *params.Temperature
	}
	if params.MaxTokens != nil {
		req["max_tokens"] = *params.MaxTokens
	}

	return req
}

func (c *Client) parseResponse(resp *chatCompletionResponse) (*provider.GenerateResult, error) {
	if len(resp.Choices) == 0 {
		return nil, graft.NewAgentError(graft.ErrProvider, "no choices in response", nil)
	}

	choice := resp.Choices[0]
	msg := graft.Message{
		Role:    graft.RoleAssistant,
		Content: choice.Message.Content,
	}

	for _, tc := range choice.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, graft.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	return &provider.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
		},
	}, nil
}

func (c *Client) readSSEStream(body io.Reader, ch chan<- provider.StreamChunk) {
	// SSE stream parsing — simplified for MVP
	buf := make([]byte, 4096)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			// For MVP, emit the raw text. Full SSE parsing will be improved.
			ch <- provider.StreamChunk{
				Delta: graft.StreamEvent{
					Type: graft.EventTextDelta,
					Data: string(buf[:n]),
				},
			}
		}
		if err != nil {
			break
		}
	}
}

// --- OpenAI API response types ---

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int          `json:"index"`
	Message      respMessage  `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type respMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []respToolCall `json:"tool_calls,omitempty"`
}

type respToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function respFunction `json:"function"`
}

type respFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./provider/openai/... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add provider/openai/
git commit -m "feat: add OpenAI-compatible provider (works with OpenRouter, Ollama, LM Studio)"
```

---

## Task 16: Anthropic Provider

**Files:**
- Create: `provider/anthropic/anthropic.go`
- Create: `provider/anthropic/anthropic_test.go`

- [ ] **Step 1: Write the failing test**

Create `provider/anthropic/anthropic_test.go`:

```go
package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

func TestNewClient(t *testing.T) {
	c := New(WithAPIKey("test-key"), WithModel("claude-sonnet-4-20250514"))
	if c.ModelID() != "claude-sonnet-4-20250514" {
		t.Errorf("ModelID() = %q, want %q", c.ModelID(), "claude-sonnet-4-20250514")
	}
}

func TestGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q, want test-key", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}

		resp := map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": "Hello from Anthropic!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  15,
				"output_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(WithAPIKey("test-key"), WithBaseURL(server.URL), WithModel("claude-sonnet-4-20250514"))
	result, err := c.Generate(context.Background(), provider.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "Hello from Anthropic!" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "Hello from Anthropic!")
	}
	if result.Usage.PromptTokens != 15 {
		t.Errorf("PromptTokens = %d, want 15", result.Usage.PromptTokens)
	}
}

func TestGenerateWithToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "msg_456",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{
					"type": "tool_use",
					"id":   "toolu_abc",
					"name": "search",
					"input": map[string]any{
						"query": "golang",
					},
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]any{
				"input_tokens":  20,
				"output_tokens": 15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(WithAPIKey("key"), WithBaseURL(server.URL), WithModel("claude-sonnet-4-20250514"))
	result, err := c.Generate(context.Background(), provider.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "search"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(result.Message.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(result.Message.ToolCalls))
	}
	if result.Message.ToolCalls[0].Name != "search" {
		t.Errorf("ToolCall.Name = %q, want search", result.Message.ToolCalls[0].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./provider/anthropic/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the Anthropic provider**

Create `provider/anthropic/anthropic.go`:

```go
// Package anthropic provides a LanguageModel implementation for the Anthropic Messages API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

const defaultBaseURL = "https://api.anthropic.com/v1"

// Client implements provider.LanguageModel for Anthropic.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	version string
}

// Option configures the Anthropic client.
type Option func(*Client)

func WithAPIKey(key string) Option     { return func(c *Client) { c.apiKey = key } }
func WithBaseURL(url string) Option    { return func(c *Client) { c.baseURL = url } }
func WithModel(model string) Option    { return func(c *Client) { c.model = model } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// New creates an Anthropic client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		http:    http.DefaultClient,
		version: "2023-06-01",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) ModelID() string { return c.model }

func (c *Client) Generate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	reqBody := c.buildRequest(params)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)), nil)
	}

	var msgResp messageResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "unmarshal response", err)
	}

	return c.parseResponse(&msgResp)
}

func (c *Client) Stream(ctx context.Context, params provider.GenerateParams) (<-chan provider.StreamChunk, error) {
	reqBody := c.buildRequest(params)
	reqBody["stream"] = true

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "stream request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan provider.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				ch <- provider.StreamChunk{
					Delta: graft.StreamEvent{Type: graft.EventTextDelta, Data: string(buf[:n])},
				}
			}
			if err != nil {
				break
			}
		}
	}()

	return ch, nil
}

func (c *Client) buildRequest(params provider.GenerateParams) map[string]any {
	// Anthropic separates system from messages
	var system string
	messages := make([]map[string]any, 0, len(params.Messages))

	for _, msg := range params.Messages {
		if msg.Role == graft.RoleSystem {
			system = msg.Content
			continue
		}

		if msg.Role == graft.RoleTool && msg.ToolResult != nil {
			content, _ := json.Marshal(msg.ToolResult.Content)
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": msg.ToolResult.CallID,
						"content":     string(content),
					},
				},
			})
			continue
		}

		m := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
		messages = append(messages, m)
	}

	req := map[string]any{
		"model":      c.model,
		"messages":   messages,
		"max_tokens": 4096,
	}

	if system != "" {
		req["system"] = system
	}

	if params.MaxTokens != nil {
		req["max_tokens"] = *params.MaxTokens
	}
	if params.Temperature != nil {
		req["temperature"] = *params.Temperature
	}

	if len(params.Tools) > 0 {
		tools := make([]map[string]any, 0, len(params.Tools))
		for _, t := range params.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": json.RawMessage(t.Schema),
			})
		}
		req["tools"] = tools
	}

	return req
}

func (c *Client) parseResponse(resp *messageResponse) (*provider.GenerateResult, error) {
	msg := graft.Message{Role: graft.RoleAssistant}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			msg.Content += block.Text
		case "tool_use":
			input, _ := json.Marshal(block.Input)
			msg.ToolCalls = append(msg.ToolCalls, graft.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: input,
			})
		}
	}

	return &provider.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
		},
	}, nil
}

// --- Anthropic API response types ---

type messageResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      anthropicUsage `json:"usage"`
}

type contentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./provider/anthropic/... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add provider/anthropic/
git commit -m "feat: add Anthropic Messages API provider"
```

---

## Task 17: Google Gemini Provider

**Files:**
- Create: `provider/google/google.go`
- Create: `provider/google/google_test.go`

- [ ] **Step 1: Write the failing test**

Create `provider/google/google_test.go`:

```go
package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

func TestNewClient(t *testing.T) {
	c := New(WithAPIKey("test-key"), WithModel("gemini-2.0-flash"))
	if c.ModelID() != "gemini-2.0-flash" {
		t.Errorf("ModelID() = %q, want %q", c.ModelID(), "gemini-2.0-flash")
	}
}

func TestGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key in query params
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("API key = %q, want test-key", r.URL.Query().Get("key"))
		}

		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": "Hello from Gemini!"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     12,
				"candidatesTokenCount": 6,
				"totalTokenCount":      18,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(WithAPIKey("test-key"), WithBaseURL(server.URL), WithModel("gemini-2.0-flash"))
	result, err := c.Generate(context.Background(), provider.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "Hello from Gemini!" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "Hello from Gemini!")
	}
	if result.Usage.PromptTokens != 12 {
		t.Errorf("PromptTokens = %d, want 12", result.Usage.PromptTokens)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./provider/google/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Write the Google Gemini provider**

Create `provider/google/google.go`:

```go
// Package google provides a LanguageModel implementation for the Google Gemini API.
package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// Client implements provider.LanguageModel for Google Gemini.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

type Option func(*Client)

func WithAPIKey(key string) Option     { return func(c *Client) { c.apiKey = key } }
func WithBaseURL(url string) Option    { return func(c *Client) { c.baseURL = url } }
func WithModel(model string) Option    { return func(c *Client) { c.model = model } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

func New(opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		http:    http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) ModelID() string { return c.model }

func (c *Client) Generate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	reqBody := c.buildRequest(params)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)), nil)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "unmarshal response", err)
	}

	return c.parseResponse(&geminiResp)
}

func (c *Client) Stream(ctx context.Context, params provider.GenerateParams) (<-chan provider.StreamChunk, error) {
	reqBody := c.buildRequest(params)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "stream request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewAgentError(graft.ErrProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan provider.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				ch <- provider.StreamChunk{
					Delta: graft.StreamEvent{Type: graft.EventTextDelta, Data: string(buf[:n])},
				}
			}
			if err != nil {
				break
			}
		}
	}()

	return ch, nil
}

func (c *Client) buildRequest(params provider.GenerateParams) map[string]any {
	contents := make([]map[string]any, 0, len(params.Messages))

	var systemInstruction string
	for _, msg := range params.Messages {
		if msg.Role == graft.RoleSystem {
			systemInstruction = msg.Content
			continue
		}

		role := "user"
		if msg.Role == graft.RoleAssistant {
			role = "model"
		}

		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": msg.Content}},
		})
	}

	req := map[string]any{
		"contents": contents,
	}

	if systemInstruction != "" {
		req["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": systemInstruction}},
		}
	}

	if params.Temperature != nil || params.MaxTokens != nil {
		genConfig := map[string]any{}
		if params.Temperature != nil {
			genConfig["temperature"] = *params.Temperature
		}
		if params.MaxTokens != nil {
			genConfig["maxOutputTokens"] = *params.MaxTokens
		}
		req["generationConfig"] = genConfig
	}

	return req
}

func (c *Client) parseResponse(resp *geminiResponse) (*provider.GenerateResult, error) {
	if len(resp.Candidates) == 0 {
		return nil, graft.NewAgentError(graft.ErrProvider, "no candidates in response", nil)
	}

	candidate := resp.Candidates[0]
	var content string
	for _, part := range candidate.Content.Parts {
		content += part.Text
	}

	return &provider.GenerateResult{
		Message: graft.Message{
			Role:    graft.RoleAssistant,
			Content: content,
		},
		Usage: graft.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}

// --- Gemini API response types ---

type geminiResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
}

type candidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiContent struct {
	Parts []part `json:"parts"`
	Role  string `json:"role"`
}

type part struct {
	Text string `json:"text"`
}

type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./provider/google/... -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add provider/google/
git commit -m "feat: add Google Gemini API provider"
```

---

## Task 18: Examples

**Files:**
- Create: `examples/basic/main.go`
- Create: `examples/streaming/main.go`
- Create: `examples/handoff/main.go`
- Create: `examples/multi-provider/main.go`

- [ ] **Step 1: Write the basic example**

Create `examples/basic/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	greetTool := graft.NewTool("greet", "Greet someone by name",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return fmt.Sprintf("Hello, %s! Welcome to Graft.", p.Name), nil
		},
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant. Use the greet tool when asked to greet someone."),
		graft.WithTools(greetTool),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Please greet Alice"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
```

- [ ] **Step 2: Write the streaming example**

Create `examples/streaming/main.go`:

```go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/stream"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant."),
	)

	runner := graft.NewDefaultRunner(model)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		events, err := runner.RunStream(r.Context(), agent, []graft.Message{
			{Role: graft.RoleUser, Content: r.URL.Query().Get("q")},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stream.SSEHandlerFromChannel(events).ServeHTTP(w, r)
	})

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
```

- [ ] **Step 3: Write the handoff example**

Create `examples/handoff/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	codeReviewer := graft.NewAgent("code-reviewer",
		graft.WithInstructions("You are an expert code reviewer. Analyze code for bugs, performance, and style."),
	)

	triage := graft.NewAgent("triage",
		graft.WithInstructions("You are a triage agent. Route code questions to the code reviewer."),
		graft.WithHandoffs(graft.Handoff{
			Target:      codeReviewer,
			Description: "Transfer to code reviewer for code analysis questions",
		}),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), triage, []graft.Message{
		{Role: graft.RoleUser, Content: "Review this Go function: func add(a, b int) int { return a - b }"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
```

- [ ] **Step 4: Write the multi-provider example**

Create `examples/multi-provider/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	// Set up two providers with fallback routing
	primary := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	fallback := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("openai/gpt-4o"),
	)

	// Router falls back to gpt-4o if Claude fails
	router := provider.NewRouter(provider.StrategyFallback, primary, fallback)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant."),
	)

	runner := graft.NewDefaultRunner(router)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "What is the capital of France?"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
```

- [ ] **Step 5: Verify examples compile**

```bash
go build ./examples/basic/ && go build ./examples/streaming/ && go build ./examples/handoff/ && go build ./examples/multi-provider/
```

Expected: All compile successfully.

- [ ] **Step 6: Commit**

```bash
git add examples/
git commit -m "feat: add examples (basic, streaming, handoff, multi-provider)"
```

---

## Task 19: Full Test Suite Run and Tidy

**Files:**
- Modify: `go.mod` (tidy)

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/delavalom/delavalom-labs/graft
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 2: Tidy dependencies**

```bash
go mod tidy
```

- [ ] **Step 3: Run go vet**

```bash
go vet ./...
```

Expected: No issues.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: tidy go.mod and verify full test suite"
```

---

## Summary

| Task | What it builds | Key files |
|------|----------------|-----------|
| 1 | Project scaffolding | `go.mod`, `graft.go` |
| 2 | Message types | `message.go` |
| 3 | Error types | `errors.go` |
| 4 | Result types | `result.go` |
| 5 | Stream event types | `stream.go` |
| 6 | JSON Schema generation | `internal/jsonschema/` |
| 7 | Tool interface + NewTool | `tool.go` |
| 8 | Agent + functional options | `agent.go`, `options.go`, `guardrail.go`, `handoff.go`, `hook.go` |
| 9 | Hook system tests | `hook_test.go` |
| 10 | Provider package (interface, router, middleware) | `provider/` |
| 11 | DefaultRunner | `runner.go` |
| 12 | Built-in guardrails | `guardrail/builtin.go` |
| 13 | SSE adapter + Collect | `stream/` |
| 14 | OTel instrumentation | `otel/` |
| 15 | OpenAI provider | `provider/openai/` |
| 16 | Anthropic provider | `provider/anthropic/` |
| 17 | Google Gemini provider | `provider/google/` |
| 18 | Examples | `examples/` |
| 19 | Full test + tidy | `go.mod`, `go.sum` |
