---
name: graft
description: Use when writing Go code that imports github.com/delavalom/graft. Enforces Graft conventions — functional options, generic NewTool, interface-driven runners, vendor-native provider IDs, and zero-SDK HTTP providers.
---

# Graft Skill

You are writing Go code that uses [Graft](https://github.com/delavalom/graft), a Go-native framework for LLM agents. This skill encodes the conventions the package is built around. Follow them strictly when generating or modifying Graft code.

## Detection

This skill applies when any of the following are true:
- The current module's `go.mod` contains `github.com/delavalom/graft`.
- A file being edited imports `github.com/delavalom/graft` or a sub-package (`provider/*`, `hatchet`, `temporal`, `trigger`, `mcp`, `graph`, `state`, `tracing`, `stream`, `otel`, `guardrail`).
- The user asks about Graft, "the agent framework," or mentions any of its sub-packages by name.

## Non-negotiable conventions

### 1. Functional options

Every constructor takes `...Option` (or a typed equivalent). Never set struct fields on returned values when an option exists.

```go
// CORRECT
agent := graft.NewAgent("assistant",
    graft.WithInstructions("..."),
    graft.WithTools(tool1, tool2),
    graft.WithModel("gpt-4o"),
)

// WRONG — breaks the pattern, silently misses default wiring
agent := &graft.Agent{Name: "assistant", Instructions: "...", Tools: []graft.Tool{tool1}}
```

The struct-literal form is supported for legacy code but must not be used in new code.

### 2. Type-safe tools via `NewTool`

Always use the generic constructor. It auto-generates the JSON Schema from the parameter struct.

```go
// CORRECT
type Params struct {
    Query string `json:"query" description:"The search query"`
    Limit int    `json:"limit,omitempty" description:"Max results (default 10)"`
}

tool := graft.NewTool("search", "Search the knowledge base.",
    func(ctx context.Context, p Params) ([]Result, error) {
        return doSearch(ctx, p.Query, p.Limit)
    },
)
```

Rules:
- Every struct field needs a `json` tag.
- Use `description` tags — the LLM reads them.
- Return type `R` can be anything; strings pass through, other types JSON-marshal.
- Never hand-implement `graft.Tool` unless the schema is genuinely runtime-dynamic.

### 3. Role constants, not string literals

```go
// CORRECT
{Role: graft.RoleUser, Content: "Hi"}
{Role: graft.RoleAssistant, Content: "Hello"}

// WRONG
{Role: "user", Content: "Hi"}
```

Available: `RoleUser`, `RoleAssistant`, `RoleSystem`, `RoleTool`.

### 4. Errors via `NewAgentError`

```go
return graft.NewAgentError(graft.ErrProvider, "openai: request failed", err)
```

Don't wrap with `fmt.Errorf` when callers may need `.Type()`, `.IsRetryable()`, or `.Context`. `ErrProvider` messages must be self-explanatory for users who can't see the provider console.

### 5. Runner is an interface — design for swap

Never assume `*graft.DefaultRunner`. Accept `graft.Runner`:

```go
// CORRECT
func Serve(r graft.Runner, agent *graft.Agent) { ... }

// WRONG — hard-codes the default, breaks hatchet/temporal/trigger
func Serve(r *graft.DefaultRunner, agent *graft.Agent) { ... }
```

### 6. Vendor-native model IDs

Do not invent model strings. Use what the provider actually ships:

| Provider | Example IDs |
|----------|-------------|
| OpenAI | `gpt-4o`, `gpt-4o-mini`, `o1`, `o3` |
| Anthropic | `claude-sonnet-4-20250514`, `claude-opus-4-20250514`, `claude-haiku-4-5-20251001` |
| Google | `gemini-2.0-flash`, `gemini-2.5-pro` |
| Bedrock | `anthropic.claude-sonnet-4-20250514-v1:0`, `amazon.titan-text-premier-v1:0`, `meta.llama3-70b-instruct-v1:0` |
| OpenRouter | `anthropic/claude-sonnet-4.6`, `openai/gpt-4o`, etc. |

### 7. No external SDKs in providers

The four in-tree providers use `net/http` only. If you add a new provider, do the same. Do **not** add `github.com/openai/openai-go` or similar.

### 8. Durable runtimes inject real SDK clients

Graft defines small interfaces (`hatchet.HatchetClient`, `temporal.TemporalClient`) to avoid SDK dependencies. The real SDK client is supplied at construction:

```go
// Hatchet — production
c, _ := hatchetclient.New()
runner := hatchet.NewHatchetRunner(c, hatchet.WithNamespace("prod"))

// Temporal — production
c, _ := client.Dial(client.Options{})
runner := temporal.NewTemporalRunner(c, temporal.WithTaskQueue("ai-agents"))
```

For demos, see `examples/hatchet/client` and `examples/temporal/client` for minimal `HatchetClient` / `TemporalClient` stubs that implement the interface in-process.

### 9. Handoffs are synthetic tools

Handoffs become tools named `handoff_<target-agent-name>`. Don't try to implement handoffs via custom tools — use `graft.Handoff{Target, Description, Filter}`.

### 10. Streaming caveat

`Runner.RunStream` currently wraps `Run` in a goroutine and emits **lifecycle events** (tool-call start/end, completion, errors). It does not emit per-token tokens from inside the agent loop. For per-token streaming, call `LanguageModel.Stream` directly.

Do not write docs or comments claiming token-level streaming from the runner.

## Project layout

Keep Graft code close to the standard layout:

```
cmd/
    myagent/
        main.go           # entry point, wires model + runner + agent
internal/
    agents/
        support.go        # agent definitions
        billing.go
    tools/
        weather.go        # tool definitions
        search.go
```

Put agents in `internal/agents/` and tools in `internal/tools/`. Don't scatter `graft.NewTool` calls across feature packages — tools are a shared resource and benefit from being indexed in one place.

## Common recipes

### Add a new provider

Copy `provider/openai/` as a template. Implement `LanguageModel.Generate` and `.Stream` using `net/http`. Map vendor-specific message/tool-call shapes in the provider; do not leak them into `graft.Message`.

### Add a new guardrail

```go
type myGuard struct{ /* ... */ }
func (g *myGuard) Name() string { return "my-guard" }
func (g *myGuard) Type() graft.GuardrailType { return graft.GuardrailInput }
func (g *myGuard) Validate(ctx context.Context, d *graft.ValidationData) (*graft.ValidationResult, error) { ... }
```

Return `ValidationResult{Pass: false, Reason: "..."}` to trip. Use types `GuardrailInput`, `GuardrailOutput`, `GuardrailTool`.

### Add a new hook

```go
hooks := graft.NewHookRegistry()
hooks.On(graft.HookPostToolCall, func(ctx context.Context, p *graft.HookPayload) (*graft.HookResult, error) {
    audit.Log(p.Agent.Name, p.ToolCall.Name, p.ToolCall.Arguments)
    return nil, nil
})
agent := graft.NewAgent("a", graft.WithHooks(hooks))
```

Returning `&HookResult{Allow: ptrFalse}` denies the current operation. Multiple callbacks for the same event run in registration order.

## Testing

- Runner tests use a `fakeModel` pattern (see `runner_test.go`). Follow it.
- Provider tests hit real APIs via env vars (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, `OPENROUTER_API_KEY`). There are no mocked provider tests. Gate network tests with a build tag or skip on missing env vars.
- `go test ./...` should pass without network by default. If you add a network-dependent test, make it skip gracefully.

## Anti-patterns — reject these

- Using `any` in place of typed params on `NewTool` — loses the schema.
- Returning `error` from a tool without `NewAgentError` when the agent needs to distinguish retryable vs terminal.
- Calling `runner.Run` inside a hook callback — causes re-entrancy. Use `SubAgent` or a separate `Runner` instance.
- Creating a new `Runner` per request when a single `*DefaultRunner` would do. Runners are safe to share.
- Hardcoding the OpenAI base URL. Use `WithBaseURL` for gateways.
- Writing "token-level streaming" docs. See convention 10.

## When in doubt

Read:
- `CLAUDE.md` in the repo root — authoritative conventions.
- `examples/*/main.go` (or `examples/*/client/main.go`) — canonical usage.
- Docs at [`website/content/docs/`](../../website/content/docs/) — user-facing concepts.
