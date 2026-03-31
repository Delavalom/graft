# Graft

A Go-native framework for building AI agents and LLM-powered applications. Zero external SDK dependencies — just `net/http` and OpenTelemetry.

## Install

```bash
go get github.com/delavalom/graft
```

## Quick Start

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
        openai.WithModel("anthropic/claude-sonnet-4.6"),
    )

    greetTool := graft.NewTool("greet", "Greet someone by name",
        func(ctx context.Context, p struct {
            Name string `json:"name" description:"The person's name"`
        }) (string, error) {
            return fmt.Sprintf("Hello, %s!", p.Name), nil
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

## Features

- **Multi-provider**: OpenAI, Anthropic, Google (Gemini) — all via raw HTTP, no vendor SDKs
- **Type-safe tools**: Define tools as typed Go functions with auto-generated JSON schemas
- **Agent handoffs**: Route conversations between specialized agents
- **Guardrails**: Input validation (max tokens, content filtering) and output validation (JSON schema)
- **MCP integration**: Connect to MCP servers as a client, or expose graft tools as an MCP server
- **Graph orchestration**: LangGraph-style DAG execution with conditional routing and streaming
- **Session persistence**: Multi-turn conversations with memory and file-backed stores
- **Pluggable tracing**: Braintrust, LangSmith, OpenTelemetry, or bring your own
- **Durable execution**: Temporal, Hatchet, and Trigger.dev integrations
- **Streaming**: SSE HTTP handler adapter for real-time responses
- **Provider routing**: Fallback and round-robin strategies across providers

## Architecture

```
User messages -> Agent -> Runner.Run() -> LanguageModel.Generate()
  -> Tool calls?  -> Execute tools -> Append results -> Loop back
  -> Handoff?     -> Switch active agent -> Loop back
  -> No tool calls? -> Return Result
```

## Packages

| Package | Description |
|---------|-------------|
| `graft` | Core types: Agent, Runner, Tool, Message, Guardrail, Handoff, Hook |
| `provider/openai` | OpenAI, OpenRouter, Ollama, LM Studio |
| `provider/anthropic` | Anthropic Messages API |
| `provider/google` | Google Generative Language API (Gemini) |
| `provider/bedrock` | AWS Bedrock (Converse API) — Claude, Titan, Llama, Mistral |
| `provider` | Router (fallback/round-robin) and middleware chain |
| `guardrail` | Built-in guardrails: MaxTokens, ContentFilter, SchemaValidator |
| `mcp` | Model Context Protocol client and server |
| `graph` | Graph-based orchestration with conditional edges |
| `state` | Session persistence (memory and file stores) |
| `tracing` | Pluggable tracing: Braintrust, LangSmith, OpenTelemetry |
| `temporal` | Temporal durable workflow integration |
| `hatchet` | Hatchet durable function integration |
| `trigger` | Trigger.dev REST API integration |
| `stream` | SSE HTTP handler adapter |
| `otel` | OpenTelemetry instrumentation wrappers |

## Examples

| Example | Description |
|---------|-------------|
| [basic](examples/basic/) | Simple agent with a tool |
| [handoff](examples/handoff/) | Agent-to-agent routing |
| [streaming](examples/streaming/) | HTTP streaming with SSE |
| [multi-provider](examples/multi-provider/) | Fallback across providers |
| [guardrails](examples/guardrails/) | Input/output validation |
| [mcp-client](examples/mcp-client/) | Connect to an MCP server and use its tools |
| [mcp-server](examples/mcp-server/) | Expose graft tools as an MCP server |
| [graph](examples/graph/) | ReAct graph orchestration |
| [tracing](examples/tracing/) | Pluggable tracing with Braintrust |
| [state](examples/state/) | Persistent multi-turn sessions |
| [temporal](examples/temporal/) | Durable execution with Temporal |
| [hatchet](examples/hatchet/) | Durable functions with Hatchet |
| [trigger](examples/trigger/) | Background tasks with Trigger.dev |
| [bedrock](examples/bedrock/) | AWS Bedrock with Converse API |

Run any example:

```bash
export OPENROUTER_API_KEY=your-key
go run ./examples/basic/
```

## Design Principles

**Functional options everywhere**: Consistent API across agents, providers, and runners.

```go
agent := graft.NewAgent("name",
    graft.WithInstructions("..."),
    graft.WithTools(tool1, tool2),
    graft.WithGuardrails(guardrail.MaxTokens(1000)),
)
```

**Type-safe tools**: Struct fields become JSON schema automatically.

```go
tool := graft.NewTool("search", "Search for items",
    func(ctx context.Context, p struct {
        Query string `json:"query" description:"Search query"`
        Limit int    `json:"limit" description:"Max results"`
    }) (string, error) {
        // ...
    },
)
```

**Composable runners**: Wrap runners to add behavior without changing the agent.

```go
base := graft.NewDefaultRunner(model)
traced := tracing.NewTracedRunner(base, braintrustProvider)
persistent := state.NewSessionRunner(traced, store, sessionID)
```

## License

MIT
