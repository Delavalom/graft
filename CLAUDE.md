# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Graft

Graft is a Go-native framework for building AI agents and LLM-powered applications. Module path: `github.com/delavalom/graft`. Go 1.25+. Only external dependency is OpenTelemetry.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Run tests for a specific package
go test ./provider/anthropic/

# Build (library — no main binary)
go build ./...

# Run an example
go run ./examples/basic/
```

No Makefile, no linter config, no CI pipeline currently.

## Architecture

### Core Loop

`Runner.Run()` drives the agent loop: send messages to the LLM, execute any tool calls, handle handoffs, repeat until the model responds without tool calls or max iterations (default 10) is hit.

```
User messages → Agent → Runner.Run() → LanguageModel.Generate()
  → Tool calls? → Execute tools → Append results → Loop back
  → Handoff? → Switch active agent, rebuild tool map → Loop back
  → No tool calls? → Return Result
```

### Package Layout

- **Root (`graft`)**: All core types and interfaces — Agent, Runner, Message, Tool, Guardrail, Handoff, Hook, Result, Stream, Errors, Options. This is the public API.
- **`provider/`**: Multi-provider LLM abstraction. Each sub-package (`anthropic/`, `openai/`, `google/`) implements the `LanguageModel` interface. Also contains `Router` (fallback/round-robin strategy) and `Middleware` (chain pattern).
- **`guardrail/`**: Built-in guardrails — MaxTokens, ContentFilter, SchemaValidator.
- **`stream/`**: SSE HTTP handler adapter (`SSEHandlerFromChannel`) and `Collect` helper.
- **`otel/`**: OpenTelemetry instrumentation — `TracingRunner` and `MetricsRunner` wrappers.
- **`internal/jsonschema/`**: Reflection-based JSON Schema generation from Go structs. Used by `NewTool` to auto-generate schemas from function parameter types.

### Key Interfaces

- **`LanguageModel`** (`llm.go`): `Generate()`, `Stream()`, `ModelID()` — implemented by each provider.
- **`Runner`** (`runner.go`): `Run()`, `RunStream()` — `DefaultRunner` is the standard implementation.
- **`Tool`** (`tool.go`): `Name()`, `Description()`, `Schema()`, `Execute()` — `NewTool[T]()` creates type-safe tools from typed Go functions.
- **`Guardrail`** (`guardrail.go`): `Name()`, `Type()`, `Validate()` — three types: Input, Output, Tool.

### Patterns

**Functional options everywhere**: `NewAgent("name", WithInstructions(...), WithTools(...))`, provider constructors (`anthropic.New(anthropic.WithAPIKey(...))`), and run config (`runner.Run(ctx, agent, msgs, WithMaxIterations(5))`).

**Tool type-safety**: `graft.NewTool("name", "desc", func(ctx, params MyStruct) (string, error) {...})` — the struct fields become the JSON schema via `internal/jsonschema`. Struct tags: `json:"name"` for field names, `description:"..."` for schema descriptions.

**Handoffs as synthetic tools**: Handoffs become tools named `handoff_<target-agent-name>` in the runner's tool map. When the LLM "calls" a handoff tool, the runner switches the active agent.

**Hook lifecycle**: HookRegistry fires events at: AgentStart/End, PreGenerate/PostGenerate, PreToolCall/PostToolCall/ToolCallError, PreHandoff/PostHandoff, GuardrailTrip. Hooks can deny tool calls via `HookResult.Allow`.

**Provider middleware**: `provider.Chain(model, middleware1, middleware2)` wraps a LanguageModel. Built-in: `WithLogging`.

### Provider Details

- **OpenAI** (`provider/openai/`): Works with OpenAI, OpenRouter, Ollama, LM Studio via custom BaseURL/headers.
- **Anthropic** (`provider/anthropic/`): Messages API, API version 2023-06-01.
- **Google** (`provider/google/`): Generative Language API (Gemini).

All providers use zero external SDK dependencies — raw HTTP with `net/http`.

## Conventions

- All provider tests hit real APIs via environment variables (`OPENROUTER_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`). There are no mocked provider tests.
- Root package tests use a `fakeModel` (in `runner_test.go`) to test the runner without API calls.
- Error types are in `errors.go` — use `NewAgentError(ErrType, message, cause)` with optional `Context` map. `ErrProvider` errors should be self-explanatory for end users who may not have provider console access.
- `ToolResult.Content` is `any` — strings pass through, everything else gets JSON-marshaled.
- `RunStream` currently wraps `Run` in a goroutine (not true token-level streaming from the runner yet).
