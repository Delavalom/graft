# Graft: Go-Native Agent & LLM Framework — Core Design Spec

> **Date:** 2026-03-29
> **Status:** Approved
> **Scope:** MVP core library

---

## Overview

Graft is a Go library (`go get github.com/delavalom/graft`) for building AI agents and LLM-powered applications. It provides agent orchestration, multi-provider abstraction, tool execution, streaming, guardrails, lifecycle hooks, and OpenTelemetry observability.

### Design Principles

- **Interface-driven** — every major component defined by Go interfaces for modularity and testability
- **Functional options** — clean, extensible, backward-compatible configuration
- **Channel-based streaming** — Go channels as the native streaming primitive
- **Context propagation** — cancellation, timeouts, and metadata throughout
- **Generics** — type-safe tool registration and schema generation
- **OTel-native** — observability from day one, no proprietary format
- **Durable-execution-ready** — interfaces designed so Temporal/Hatchet/Trigger.dev backends can wrap the same definitions without refactoring

### Architecture: Domain-Split Packages

```
graft/             — Core types: Agent, Tool, Runner, Handoff, StreamEvent
graft/provider/    — LanguageModel interface + provider implementations
graft/hook/        — Lifecycle hook system
graft/guardrail/   — Input/output/tool validation
graft/stream/      — SSE adapter
graft/otel/        — OpenTelemetry instrumentation
```

---

## 1. Core Types (`graft/`)

### Agent

```go
type Agent struct {
    Name         string
    Instructions string              // system prompt, supports template vars
    Tools        []Tool
    Model        string              // e.g. "anthropic/claude-sonnet-4-20250514", "openai/gpt-4o"
    Temperature  *float64
    MaxTokens    *int
    ToolChoice   ToolChoice          // Auto | Required | None | Specific("tool_name")
    Guardrails   []Guardrail         // Guardrail interface defined in root package
    Handoffs     []Handoff
    Hooks        *HookRegistry       // HookRegistry defined in root package
    Metadata     map[string]any
}
```

**Import design note:** The `Guardrail` and `HookRegistry` interfaces/types are defined in the root `graft/` package to avoid circular imports. The `graft/guardrail/` and `graft/hook/` packages provide implementations and helpers that depend on the root types, not the other way around.

Created via functional options:

```go
agent := graft.NewAgent("analyzer",
    graft.WithInstructions("Analyze data..."),
    graft.WithModel("anthropic/claude-sonnet-4-20250514"),
    graft.WithTools(searchTool, calcTool),
    graft.WithGuardrails(myGuardrail),
)
```

### Tool

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage        // JSON Schema for parameters
    Execute(ctx context.Context, params json.RawMessage) (any, error)
}
```

Generic helper for type-safe registration with auto-generated JSON Schema:

```go
tool := graft.NewTool("search", "Search the web",
    func(ctx context.Context, p SearchParams) (SearchResult, error) { ... })
```

### Messages

```go
type Role string // System, User, Assistant, Tool

type Message struct {
    Role      Role
    Content   string
    ToolCalls []ToolCall
    ToolResult *ToolResult
    Metadata  map[string]any
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type ToolResult struct {
    CallID  string
    Content any
    IsError bool
}
```

### Runner

```go
type Runner interface {
    Run(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (*Result, error)
    RunStream(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (<-chan StreamEvent, error)
}

type Result struct {
    Messages  []Message
    FinalText string
    Usage     Usage
    Cost      *Cost
    Trace     *Trace
}
```

Run options: `WithMaxIterations(n)`, `WithTimeout(d)`, `WithParallelTools(bool)`.

### Handoff

```go
type Handoff struct {
    Target      *Agent
    Description string              // exposed to LLM as tool description
    Filter      func(ctx context.Context, messages []Message) bool
}
```

Handoffs are exposed as pseudo-tools to the LLM. When the model calls a handoff tool, the runner swaps the active agent and continues the loop with the full message history. Same pattern as OpenAI Agents SDK.

### StreamEvent

```go
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

type StreamEvent struct {
    Type      EventType
    Data      any
    AgentID   string
    Timestamp time.Time
}
```

### Error Types

```go
type ErrorType string
const (
    ErrToolExecution    ErrorType = "tool_execution"
    ErrHandoff          ErrorType = "handoff"
    ErrGuardrail        ErrorType = "guardrail"
    ErrTimeout          ErrorType = "timeout"
    ErrContextLength    ErrorType = "context_length"
    ErrInvalidToolCall  ErrorType = "invalid_tool_call"
    ErrRateLimit        ErrorType = "rate_limit"
    ErrProvider         ErrorType = "provider"
)

type AgentError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]any
}
```

---

## 2. Provider Abstraction (`graft/provider/`)

### LanguageModel Interface

```go
type LanguageModel interface {
    Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
    Stream(ctx context.Context, params StreamParams) (<-chan StreamChunk, error)
    ModelID() string
}

type GenerateParams struct {
    Messages    []graft.Message
    Tools       []graft.ToolDefinition
    Temperature *float64
    MaxTokens   *int
    ToolChoice  graft.ToolChoice
    Stop        []string
    Metadata    map[string]any
}

type GenerateResult struct {
    Message graft.Message
    Usage   graft.Usage
    Cost    *graft.Cost
}

type StreamChunk struct {
    Delta graft.StreamEvent
    Usage *graft.Usage
}
```

### Provider Implementations

Each in its own sub-package:

- `graft/provider/openai/` — OpenAI API (also works for OpenRouter, Ollama, LM Studio via BaseURL)
- `graft/provider/anthropic/` — Anthropic Messages API
- `graft/provider/google/` — Google Gemini API

OpenRouter usage — point the OpenAI provider at OpenRouter's base URL:

```go
model := openai.New(
    openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    openai.WithBaseURL("https://openrouter.ai/api/v1"),
    openai.WithModel("anthropic/claude-sonnet-4-20250514"),
)
```

### Model Router

```go
type Router struct {
    Models   []LanguageModel
    Strategy RoutingStrategy // Fallback | RoundRobin | CostOptimized
}
```

Falls back to next model on retryable errors (429, 500, 502, 503, 504). Respects Retry-After headers, exponential backoff with jitter.

### Middleware

```go
type Middleware func(LanguageModel) LanguageModel
```

Wraps `LanguageModel` — same interface in, same interface out. Composable.

Built-in: `WithLogging`, `WithCostTracking`, `WithRetry`, `WithRateLimit`, `WithCache`.

---

## 3. Hook System (`graft/hook/`)

### Events

```go
type Event string
const (
    EventAgentStart       Event = "agent_start"
    EventAgentEnd         Event = "agent_end"
    EventPreToolCall      Event = "pre_tool_call"
    EventPostToolCall     Event = "post_tool_call"
    EventToolCallError    Event = "tool_call_error"
    EventPreHandoff       Event = "pre_handoff"
    EventPostHandoff      Event = "post_handoff"
    EventPreGenerate      Event = "pre_generate"
    EventPostGenerate     Event = "post_generate"
    EventGuardrailTrip    Event = "guardrail_trip"
    EventPhaseStart       Event = "phase_start"
    EventPhaseEnd         Event = "phase_end"
)
```

### Callbacks

```go
type HookCallback func(ctx context.Context, payload *HookPayload) (*HookResult, error)

type HookPayload struct {
    Event    Event
    Agent    *graft.Agent
    ToolCall *graft.ToolCall
    Messages []graft.Message
    Metadata map[string]any
}

type HookResult struct {
    Allow         *bool
    ModifiedInput json.RawMessage
    AdditionalCtx string
    SkipExecution bool
}
```

Sequential execution in registration order. `nil` return = passthrough. First `Allow = false` short-circuits.

---

## 4. Guardrails (`graft/guardrail/`)

### Interface

```go
type Guardrail interface {
    Name() string
    Type() Type           // Input | Output | Tool
    Validate(ctx context.Context, data *ValidationData) (*ValidationResult, error)
}

type ValidationResult struct {
    Pass     bool
    Message  string
    Modified any           // optional: transformed data instead of rejection
}
```

### Built-in Guardrails

- `MaxTokens(limit)` — reject if input exceeds token estimate
- `ContentFilter(patterns)` — regex-based content filtering
- `RequireToolConfirmation(toolNames...)` — pause for confirmation
- `SchemaValidator(schema)` — validate structured output against JSON Schema

Evaluated in order. First failure stops. Modified data passes to subsequent guardrails.

---

## 5. Streaming (`graft/stream/`)

### SSE Adapter

```go
func SSEHandler(runner graft.Runner, agent *graft.Agent) http.HandlerFunc
```

Parses messages from request body, sets `Content-Type: text/event-stream`, streams events as SSE, respects context cancellation.

### Collect Helper

```go
func Collect(events <-chan graft.StreamEvent) (*graft.Result, error)
```

Drains channel, assembles full Result. Useful for testing or wrapping stream in a blocking call.

---

## 6. Observability (`graft/otel/`)

### Tracing

```go
func InstrumentRunner(runner graft.Runner, opts ...Option) graft.Runner
```

Span hierarchy:

```
agent.run (root)
  ├── llm.generate
  ├── tool.execute
  ├── guardrail.validate
  └── handoff
```

### Semantic Attributes

Following OTel LLM conventions:

- `graft.agent.name`, `graft.model.id`, `graft.provider.name`
- `graft.usage.prompt_tokens`, `graft.usage.completion_tokens`, `graft.usage.total_tokens`
- `graft.cost.total_usd`
- `graft.tool.name`, `graft.tool.duration_ms`
- `graft.run.iterations`

### Metrics

```go
func InstrumentMetrics(runner graft.Runner, meter metric.Meter) graft.Runner
```

Emits: `graft.run.duration`, `graft.run.iterations`, `graft.llm.tokens.prompt`, `graft.llm.tokens.completion`, `graft.llm.cost.usd`, `graft.tool.duration`, `graft.tool.errors`.

---

## 7. Runner Execution Loop

### DefaultRunner

```go
type DefaultRunner struct {
    model      provider.LanguageModel
    middleware []provider.Middleware
}
```

### Loop

1. **INIT** — resolve agent config, fire `agent_start` hook
2. **PREPARE** — build system message from instructions, inject tool schemas
3. **LOOP:**
   - a. GENERATE — call model with messages + tools
   - b. PARSE — extract text and/or tool calls
   - c. CHECK — no tool calls? → output guardrails → return
   - d. EXECUTE — for each tool call: pre hook → input guardrail → execute → post hook → tool guardrail
   - e. HANDOFF — if tool call matches handoff: swap agent, carry messages, continue
   - f. LIMIT — check max_iterations/timeout
   - g. REPEAT
4. **CLEANUP** — always runs (even on error/cancel), `agent_end` hook

### Split Context (Buildkite pattern)

Main `ctx` for execution, separate `graceCtx` (30s timeout) for cleanup hooks. Ensures critical operations (logging, tracing flush) complete even during cancellation.

### Parallel Tool Execution

When model returns multiple tool calls and `ParallelTools` is enabled (default true), tool executions run concurrently via goroutines with `sync.WaitGroup`.

### Durable Execution Ready

The Runner interface is where durable execution plugs in. A future `TemporalRunner` replaces `executeLoop` with workflow orchestration and `executeTool` with activity invocations — same `Agent`, `Tool`, and `Message` types, different execution strategy.

---

## 8. Package Layout

```
github.com/delavalom/graft/
├── graft.go              — NewAgent(), NewTool(), core constructors
├── agent.go              — Agent struct
├── tool.go               — Tool interface + generic helper
├── runner.go             — Runner interface + DefaultRunner
├── message.go            — Message, ToolCall, ToolResult, Role
├── stream.go             — StreamEvent, EventType
├── handoff.go            — Handoff struct
├── errors.go             — AgentError, ErrorType
├── result.go             — Result, Usage, Cost, Trace
├── options.go            — RunOption, RunConfig, functional options
│
├── provider/
│   ├── model.go          — LanguageModel interface
│   ├── router.go         — Router (fallback, round-robin)
│   ├── middleware.go      — Middleware type + built-ins
│   ├── openai/           — OpenAI + OpenRouter + Ollama
│   ├── anthropic/        — Anthropic Messages API
│   └── google/           — Google Gemini API
│
├── hook/
│   ├── events.go         — Event constants
│   ├── hook.go           — HookCallback, HookPayload, HookResult
│   └── registry.go       — Registry
│
├── guardrail/
│   ├── guardrail.go      — Guardrail interface
│   └── builtin.go        — Built-in guardrails
│
├── stream/
│   ├── sse.go            — SSEHandler
│   └── collect.go        — Collect helper
│
├── otel/
│   ├── tracing.go        — InstrumentRunner
│   ├── metrics.go        — InstrumentMetrics
│   └── attributes.go     — Semantic attributes
│
└── examples/
    ├── basic/            — Simple agent with one tool
    ├── multi-provider/   — OpenRouter fallback routing
    ├── streaming/        — SSE endpoint
    └── handoff/          — Multi-agent with handoffs
```

---

## 9. MVP Scope

### In

- Agent definition with functional options
- Tool system with generics and JSON Schema generation
- DefaultRunner with Run + RunStream
- Multi-provider: OpenAI-compatible, Anthropic, Google
- Model router with fallback
- Provider middleware (retry, rate limit, logging, caching, cost tracking)
- Handoffs (agent-to-agent via pseudo-tools)
- Hook system (12 lifecycle events)
- Guardrails (input, output, tool) with built-ins
- Channel-based streaming + SSE adapter
- OpenTelemetry tracing + metrics
- Split context for graceful shutdown
- Parallel tool execution
- Examples

### Out (Future Packages)

- MCP client/server support
- Temporal durable execution backend
- Hatchet durable execution backend
- Trigger.dev durable execution backend
- Subagent composition with context isolation
- State persistence and memory system
- Graph-based orchestration
- Enriched provider error messages
- Pluggable tracing providers (Braintrust, LangSmith, etc.)

---

## 10. What We Explicitly Do NOT Build

- A graph-based orchestration engine (loop + handoffs covers most cases)
- A UI framework (consumers build their own)
- An evaluation platform (integrate with Agenta, LangSmith, Braintrust)
- A deployment platform (integrate with Temporal, Hatchet, K8s)
- A model proxy (integrate with OpenRouter)
