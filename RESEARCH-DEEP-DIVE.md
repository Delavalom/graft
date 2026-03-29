# Deep Research: Golang Agent & LLM Framework — PRD Foundation

> **Date:** March 29, 2026
> **Purpose:** Comprehensive research to inform the Product Requirements Document for a Go-native framework for building agents and LLM-powered applications.
> **Sources:** 120+ authoritative sources including official docs, GitHub repos, blog posts, and community implementations.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Agent Framework Landscape](#2-agent-framework-landscape)
   - 2.1 OpenAI Agents SDK
   - 2.2 Claude Agent SDK
   - 2.3 LangChain Deep Agents & LangGraph
   - 2.4 Vercel AI SDK
3. [Durable Execution Platforms](#3-durable-execution-platforms)
   - 3.1 Temporal
   - 3.2 Hatchet
   - 3.3 Trigger.dev
   - 3.4 Platform Comparison Matrix
4. [Go Ecosystem & Libraries](#4-go-ecosystem--libraries)
   - 4.1 mcp-go (MCP Protocol)
   - 4.2 Existing Go Agent Frameworks
   - 4.3 Go Architectural Patterns for Agents
5. [Provider Abstraction & Model Routing](#5-provider-abstraction--model-routing)
   - 5.1 OpenRouter
   - 5.2 OpenClaw Architecture
6. [Architectural Design Partners](#6-architectural-design-partners)
   - 6.1 Buildkite Agent (Go Patterns)
   - 6.2 Agenta AI (LLMOps Patterns)
7. [Cross-Cutting Concerns](#7-cross-cutting-concerns)
8. [Key Design Decisions for Our Framework](#8-key-design-decisions-for-our-framework)
9. [Sources](#9-sources)

---

## 1. Executive Summary

The AI agent framework landscape in 2025-2026 is dominated by Python and TypeScript implementations. No production-grade, comprehensive Go framework exists that combines agent orchestration, durable execution, MCP support, and first-class developer experience. This represents a massive opportunity.

**Key findings:**

- **OpenAI Agents SDK** established the core abstractions: Agent, Runner, Handoffs, Guardrails, and Tracing. These are now the industry baseline.
- **Claude Agent SDK** pushed further with built-in agent loops, sophisticated permission systems, lifecycle hooks (14+ events), subagent composition, and MCP as a native integration.
- **LangGraph** pioneered graph-based orchestration with explicit state management, reducer-driven schemas, and built-in persistence/checkpointing.
- **Vercel AI SDK** set the standard for provider abstraction, streaming primitives, middleware patterns, and structured output with schema validation.
- **Temporal** is the gold standard for durable execution with AI agents — deterministic replay, implicit checkpointing, and the `activity_as_tool()` pattern mapping agent tools to durable activities.
- **Hatchet** offers PostgreSQL-based durability with first-class Go support, simpler operational model, and high-throughput design.
- **Trigger.dev** leads in real-time streaming, human-in-the-loop (Waitpoints), and TypeScript-first AI features.
- **Go ecosystem** is rapidly maturing: Google ADK for Go, ByteDance's Eino, mcp-go, LangChainGo, Jetify AI SDK, and agent-sdk-go all exist but none are comprehensive.

---

## 2. Agent Framework Landscape

### 2.1 OpenAI Agents SDK (Python)

**Repository:** github.com/openai/openai-agents-python
**Status:** Production, widely adopted since March 2025

#### Core Abstractions

| Abstraction | Purpose |
|-------------|---------|
| **Agent** | Encapsulates instructions, tools, model config, guardrails |
| **Runner** | Orchestration engine for agent execution lifecycle |
| **Handoff** | Transfer control between agents with context |
| **Tool** | Callable functions with JSON Schema definitions |
| **Guardrail** | Input/output validation and policy enforcement |
| **Trace** | Observability: messages, tool calls, tokens, latency, costs |

#### Agent Configuration

```
Agent:
  name: string
  instructions: string (system prompt, supports templates)
  tools: []Tool
  model: string ("gpt-4o", etc.)
  temperature: float
  max_tokens: int
  tool_choice: auto | required | none | specific
  parallel_tool_use: bool
  guardrails: []Guardrail
  metadata: map[string]any
```

#### Execution Lifecycle (Runner)

1. **Initialize** — Create runner with client + agent
2. **Prepare** — Convert instructions to system message
3. **Start** — Build initial message history
4. **Loop** — Call LLM → parse response → if tool calls: execute tools → add results → check stop conditions
5. **Return** — Final response + execution trace

**Key config:** `max_iterations` (default 10), `timeout`, `stream` (bool)

#### Handoff Mechanism

- Handoffs are exposed as pseudo-tools to the LLM
- LLM decides when to handoff based on tool descriptions
- Target agent receives full message history from predecessor
- Supports: explicit selection, router agents, hierarchical trees
- `requires_confirmation` option for user approval before transfer

#### Guardrails

- **InputGuardrail** — validates user input before processing
- **OutputGuardrail** — validates agent output before returning
- **ToolGuardrail** — validates tool calls and results
- Chainable, async, can reject/modify/transform data

#### Streaming Events

```
start → text_delta → tool_call_start → tool_call_args_delta →
tool_call_done → message_done → end | error
```

- Event-based architecture with async queue buffering
- Backpressure handling and rate limiting
- SSE-compatible format

#### Error Types

- `ToolExecutionError`, `HandoffError`, `GuardrailViolationError`
- `TimeoutError`, `ContextLengthError`, `InvalidToolCallError`
- `RateLimitError` with automatic retry + exponential backoff

#### Middleware / Hooks

- `before_agent_run`, `after_tool_call`, `before_handoff`
- `on_error`, `on_complete`
- Sequential execution, can short-circuit, full context access

---

### 2.2 Claude Agent SDK (Python/TypeScript)

**Docs:** platform.claude.com/docs/en/agent-sdk/overview
**Status:** Production, Python + TypeScript (no Go SDK)

#### Differentiating Philosophy

The Claude Agent SDK takes a fundamentally different approach from OpenAI: **the agent loop is built into the SDK**, not left to the developer. You consume an async generator of messages rather than manually implementing a while loop.

#### Core Architecture

**Entry Points:**
- `query()` — Independent queries (new session per call)
- `ClaudeSDKClient` — Stateful sessions across exchanges

**Built-in Tools (no custom implementation needed):**
Read, Write, Edit, Bash, Glob, Grep, WebSearch, WebFetch, AskUserQuestion

#### Permission System (Unique Feature)

**Evaluation order (first match wins):**
1. Hooks (can allow/deny/modify)
2. Deny rules (`disallowed_tools`)
3. Permission mode (global behavior)
4. Allow rules (`allowed_tools`)
5. `canUseTool` callback (custom logic)

**Permission Modes:** `default`, `acceptEdits`, `dontAsk`, `bypassPermissions`, `plan`
**Dynamic switching:** `await client.set_permission_mode("acceptEdits")` mid-session

#### Hooks System (14+ Lifecycle Events)

```
PreToolUse          — Before tool execution (block/modify)
PostToolUse         — After tool execution (add context)
PostToolUseFailure  — Tool failure handling
UserPromptSubmit    — Prompt submitted
Stop                — Agent stops
SubagentStart       — Subagent begins
SubagentStop        — Subagent completes
PreCompact          — Before context compaction
PermissionRequest   — Permission dialog would show
Notification        — Status messages
```

**Hook structure:**
```
hooks = {
  "PreToolUse": [
    HookMatcher(
      matcher: "Write|Edit",    // regex on tool name
      hooks: [callback1, callback2],
      timeout: 60
    )
  ]
}
```

**Hook callbacks** can return: `permissionDecision` (allow/deny/ask), `updatedInput`, `additionalContext`, `systemMessage`, `continue` flag, or `async_: true` for background tasks.

#### Subagents (First-Class Composition)

```
agents = {
  "code-reviewer": AgentDefinition(
    description: "Expert code review specialist",
    prompt: "You are a code review expert...",
    tools: ["Read", "Grep", "Glob"],  // restricted
    model: "opus"                       // override
  )
}
```

- Context isolation: subagent starts fresh, parent receives only final message
- Parallel execution: multiple subagents run concurrently
- Cannot nest (no subagent-of-subagent)
- Claude auto-delegates based on description

#### Context Management

- **Sessions** with `session_id` for multi-exchange persistence
- **Auto-compaction** when context grows large
- **Tool search** withholds tool definitions until needed (context optimization)

#### MCP Integration (Native)

- Transport types: stdio, HTTP/SSE, SDK MCP (in-process)
- Tool naming: `mcp__<server>__<tool>` convention
- Wildcard permissions: `mcp__github__*`
- In-process custom tools via `@tool` decorator + `create_sdk_mcp_server()`

#### Multi-Provider Authentication

- Anthropic API (primary)
- Amazon Bedrock (`CLAUDE_CODE_USE_BEDROCK=1`)
- Google Vertex AI (`CLAUDE_CODE_USE_VERTEX=1`)
- Microsoft Azure AI Foundry (`CLAUDE_CODE_USE_FOUNDRY=1`)

---

### 2.3 LangChain Deep Agents & LangGraph

**Repository:** github.com/langchain-ai/deepagents, github.com/langchain-ai/langgraph
**Status:** Production, MIT licensed, trusted by Klarna, Replit, Elastic

#### Deep Agents — Four Core Components

1. **Planning Tool** — `write_todos` tool for agents to break tasks into steps, track progress
2. **File System Access** — `ls`, `read_file`, `write_file`, `edit_file` to prevent context window overflow
3. **Subagents** — `task` tool spawns specialized subagents with context isolation
4. **Detailed System Prompts** — Comprehensive instructions guiding planning behavior

**Philosophy:** Solve the "shallow agent" problem where simple tool-calling loops fail on complex, multi-step tasks.

#### LangGraph — Graph-Based Orchestration

**Core concept:** Low-level framework using nodes, edges, and state machines for agent workflows.

**Graph Components:**
- **Nodes** — Functions/computations that receive state and return state updates
- **Normal Edges** — Static routing (A always calls B)
- **Conditional Edges** — Dynamic routing based on state inspection
- **Command() Objects** — Combine state updates with routing instructions
- **StateGraph** → compiles to immutable `CompiledGraph` (Runnable interface)

#### State Management (Critical Innovation)

```python
class State(TypedDict):
    messages: Annotated[list, add]  # reducer: appends
    count: int                       # overwrites
```

**Reducer functions** determine how values merge on update — this prevents data loss in multi-agent environments. Common: `add_messages` reducer for chat history.

**Memory hierarchy:**
- **Short-term:** Thread-scoped, persisted by checkpointers
- **Long-term:** Shared across threads, custom namespace scoping
- **Checkpoint backends:** SQLite, Redis, PostgreSQL

#### Persistence & Durable Execution

- Checkpoints saved at each super-step (node boundary)
- **Sync mode:** Persist before next step (high durability)
- **Async mode:** Background persistence (better performance)
- Recovery: restart from last successful checkpoint on failure

#### Human-in-the-Loop (Interrupts)

- **Static interrupts:** Configured at compile time
- **Dynamic interrupts:** `interrupt()` function at runtime
- State saved and returned to caller; resumes from same checkpoint
- Supports approval workflows, review-and-edit, sequential interrupts

#### Multi-Agent: Supervisor Pattern

- Hierarchical with central coordinator routing to specialized workers
- Tool-based handoff mechanism between agents
- Context flows through transitions
- Recommended approach: supervisor pattern via tools

#### Observability: LangSmith

- Step-by-step execution visualization
- State transition tracing
- Cost tracking and API usage monitoring
- Evaluation framework for agent behavior

#### Go Implementations

- **LangChainGo** (`github.com/tmc/langchaingo`) — Interface-driven, active development toward parity
- **LangGraphGo** (`github.com/tmc/langgraphgo`) — Early stage, graph-based components
- **LangGraph4j** — JVM alternative with full feature parity (Deep Agents, persistence)

---

### 2.4 Vercel AI SDK

**Docs:** ai-sdk.dev
**Status:** Production, AI SDK 6 (current)

#### Three Layers

| Layer | Purpose |
|-------|---------|
| **AI SDK Core** | Server-side: generateText, streamText, generateObject, streamObject |
| **AI SDK UI** | Framework-agnostic hooks: useChat, useCompletion, useObject |
| **AI SDK RSC** | React Server Components streaming (development paused) |

#### Provider Abstraction (Key Innovation)

- **LanguageModelV4** specification standardizes how models plug in
- Same API works across OpenAI, Anthropic, Google, Mistral, Bedrock
- Switch providers without code changes
- Vercel AI Gateway: 30+ models, failover, load balancing, 30% cost reduction via caching

#### Tool Definition

```typescript
tools: {
  toolName: {
    description: "What the tool does",
    parameters: z.object({
      param1: z.string().describe("description"),
    }),
    execute: async (params) => { ... }
  }
}
```

#### Agent Pattern (AI SDK 6)

- `ToolLoopAgent` — Production-ready implementation handling complete tool execution loop
- Agent is an **interface**, not just a class
- Configurable: `toolChoice`, `stopWhen`, max steps
- Multi-agent via tool-based handoffs and sub-agent orchestration

#### Middleware (First-Class)

Three interception points:
1. **transformParams** — Modify parameters before LLM call
2. **wrapGenerate** — Wrap the generate call (modify params, call, modify result)
3. **wrapStream** — Wrap the stream call

Built-in: `extractReasoningMiddleware`, `simulateStreamingMiddleware`
Use cases: guardrails, caching, logging, cost control, rate limiting

#### Structured Output

- `generateObject` / `streamObject` with Zod schema validation
- `Output.object({ schema })`, `Output.array({ element })`, `Output.json()`
- Type inference from schema, `.describe()` for LLM guidance
- Combined with tool calls for sophisticated agent patterns

#### Error Handling (Granular)

- `NoSuchToolError` — Model calls undefined tool
- `InvalidToolArgumentsError` — Schema validation failure
- `ToolExecutionError` — Runtime error during execution
- `ToolCallRepairError` — Automatic repair failure
- SDK attempts to fix malformed tool calls automatically

#### Streaming Primitives

- `streamText` — Token-by-token text delivery
- `streamObject` — Progressive JSON object streaming
- `createStreamableValue` — Low-level stream control
- Works with Next.js, Edge Runtime, API routes

---

## 3. Durable Execution Platforms

### 3.1 Temporal

**Status:** Most mature durable execution platform, $650M funded, 7 language SDKs
**AI Integration:** OpenAIAgentsPlugin (Python, Public Preview)

#### Core Architecture for AI Agents

**Key principle:** Separate deterministic orchestration (workflows) from non-deterministic operations (activities).

- Agent orchestration code runs deterministically in Temporal workflows
- Model calls and I/O tool invocations execute as Temporal activities
- Event History is durably persisted — enables perfect replay and recovery

#### OpenAIAgentsPlugin

**Components:**
- Pydantic data converter for type-safe serialization
- Tracing interceptors for agent interactions
- Model execution activities for LLM invocations
- MCP server activities with lifecycle management
- Agent runtime overrides during worker execution

**Key classes:**
- `OpenAIAgentsPlugin` — Worker plugin
- `TemporalOpenAIRunner` — Implements AgentRunner interface
- `_TemporalModelStub` — Model wrapper for activity-based invocation
- `activity_as_tool()` — Converts Temporal activities into OpenAI agent tools

#### activity_as_tool() — The Core Pattern

```python
activity_as_tool(
    task_queue="agent-tools",
    schedule_to_close_timeout=timedelta(seconds=30),
    start_to_close_timeout=timedelta(seconds=10),
    retry_policy=RetryPolicy(maximum_attempts=3),
    heartbeat_timeout=timedelta(seconds=5),
    strict_json_schema=True
)
```

Automatically generates OpenAI-compatible tool schemas from activity function signatures.

#### Durable Execution Benefits

- **Implicit checkpointing** at activity calls, timer waits, signal processing
- **Perfect replay** — Worker replays code using Event History to recreate state
- **Long-running support** — Wait for approvals hours/days/indefinitely at zero compute
- **Automatic retries** — Infrastructure-level, no manual retry logic needed
- **Cost efficiency** — No re-running completed LLM calls on recovery

#### Workflow Patterns for AI Agents

1. **Sequential** — Linear pipeline, each step depends on previous
2. **Parallel** — Concurrent independent activities
3. **Fan-Out/Fan-In** — Distribute work, collect and aggregate
4. **Human-in-the-Loop** — Signal-based approvals with durable timers
5. **Multi-Agent** — Sequential handoffs or concurrent agents
6. **Tool-Calling Loop** — Core agentic pattern with deterministic orchestration

#### Streaming Limitation

Temporal activities **cannot stream output** directly. Workaround: set `event_stream_handler` and use non-streaming `run()` inside workflow, with handler processing events externally.

#### Error Handling & Retry

- Activity retries with configurable `RetryPolicy`
- **Critical:** Disable HTTP-level retries in provider SDKs to avoid retry cascading
- `AgentsWorkflowError` wraps agent exceptions for proper workflow failure

#### Go SDK Status

- Go is a first-class Temporal SDK language
- **No native AI agent plugin** for Go (unlike Python)
- Go developers implement agent patterns using standard workflow/activity model
- Tool calls as activities, LLM invocations as activities, orchestration in workflows

#### Community Examples

- **openai-agents-demos** — 4 standalone demos (haiku, weather, research, interactive)
- **temporal-ai-agent** — Multi-turn MCP client agent
- **ai-iceberg-demo** — Deep research agent with Neo4j + RedPanda
- **Temporal Workflows MCP Server** — Bridge between Claude and Temporal workflows
- **Zeitlich** — Framework for stateful AI agents with Temporal

---

### 3.2 Hatchet

**Status:** MIT licensed, 10,000+ monthly deployments, billions of tasks/month, Y Combinator W24
**SDKs:** Go (first-class), TypeScript, Python

#### Architecture

- **PostgreSQL-based** — Solves 99.9% of queueing use cases
- Three components: Engine, API Server, Workers
- Transactional state transitions ensure consistency
- Event log enables replay and debugging

#### Why Hatchet Positions Go for Agents

- Excellent concurrency with goroutines (less constrained than Python/Node)
- Lower baseline memory footprint
- Single binary deployment
- Stateless reducer design — any instance can process next message

#### Key Features for AI Agents

- **DAGs** — Pre-define workflow shapes with automatic output-to-input routing
- **Durable tasks** — Orchestrate other tasks, store full history for caching
- **Concurrency control** — GROUP_ROUND_ROBIN, worker-level max runs
- **Rate limiting** — Dynamic (per-user) and static (external API quotas)
- **Retry policies** — Exponential backoff with configurable max seconds and factor
- **NonRetryable exception** — Bypass retries for known-fatal errors

#### Comparison to Temporal

| Aspect | Hatchet | Temporal |
|--------|---------|----------|
| Foundation | PostgreSQL | Event-sourced distributed system |
| Complexity | Simple — one concept (tasks) | Complex — Workflows, Activities, Workers, Task Queues, Namespaces |
| Self-hosting | Easy — Postgres + workers | Complex distributed system |
| Throughput | Optimized for >100/s | Enterprise reliable |
| Go support | First-class SDK | First-class but no AI plugin |
| Cost model | Usage-based + compute | Step-based (expensive with retries) |

#### Streaming Support

Task primitives support streaming data; agent outputs can flow without blocking. Not specifically optimized for frontend streaming like Trigger.dev.

---

### 3.3 Trigger.dev

**Status:** Apache 2.0, $16M Series A, thousands of teams in production
**SDK:** TypeScript/JavaScript (primary)

#### Architecture

- **No timeouts** — Tasks execute without serverless constraints
- **Warm starts** — 100-300ms execution vs seconds for cold starts
- **Waitpoints** — Snapshot process state, pause indefinitely, resume without compute cost
- **Real-time Streams v2** — Protocol-level reliability with automatic chunk resumption

#### AI-Specific Features

- **Tool calling observability** — See which tool called, return value, where agent got stuck
- **Token consumption tracking** — Real-time during streams
- **Zod schema integration** — Type-safe tool definitions, auto-conversion to AI SDK tools
- **Human-in-the-loop** — Waitpoints for approval workflows
- **Input Streams** — Send typed data into running tasks from backend/frontend

#### Streaming Excellence

- Streams v2: automatic resume from last successful chunk
- LLM responses stream directly to frontend via SSE
- Real-time subscription to run progress
- OpenTelemetry export of streaming traces

#### Comparison to Temporal

| Aspect | Trigger.dev | Temporal |
|--------|-------------|----------|
| Language | TypeScript-first | Multi-language |
| Streaming | Real-time built-in | Not designed for streaming |
| AI features | Purpose-built | General-purpose |
| Human-in-loop | Waitpoints (indefinite pause) | Workflow signaling |
| Timeouts | None | Respects timeout patterns |
| Go support | Not available | First-class SDK |

---

### 3.4 Durable Execution Platform Comparison Matrix

| Feature | Temporal | Hatchet | Trigger.dev |
|---------|----------|---------|-------------|
| **Go SDK** | Yes (first-class) | Yes (first-class) | No |
| **Open Source** | Yes (MIT core) | Yes (MIT) | Yes (Apache 2.0) |
| **State Persistence** | Event-sourced replay | PostgreSQL event log | Snapshot-based checkpointing |
| **Agent Framework Integration** | OpenAIAgentsPlugin (Python) | Generic task wrapping | Vercel AI SDK native |
| **Streaming** | Not supported in workflows | Basic task streaming | Real-time Streams v2 |
| **Human-in-the-Loop** | Signal-based approvals | Not emphasized | Waitpoints (zero compute) |
| **Retry Policies** | Comprehensive RetryPolicy | Exponential backoff + NonRetryable | Automatic with jitter |
| **MCP Support** | Via plugin MCP wrappers | Not native | Not native |
| **Observability** | Temporal UI + tracing interceptors | Dashboard + execution history | OpenTelemetry + live traces |
| **Concurrency Control** | Task queue configuration | GROUP_ROUND_ROBIN + rate limits | Priority queuing |
| **Self-Hosting Complexity** | Complex distributed system | Simple (PostgreSQL) | Modern K8s support |
| **Maturity** | Most mature ($650M funded) | Growing (YC W24, billions of tasks) | Rapidly growing ($16M Series A) |
| **Best For** | Enterprise, deterministic replay, multi-language | High-throughput Go agents, simple ops | TypeScript AI agents, streaming UX |

**Recommendation for our framework:** Support all three as pluggable backends, with Temporal and Hatchet as primary targets given their Go SDK support.

---

## 4. Go Ecosystem & Libraries

### 4.1 mcp-go (mark3labs)

**Repository:** github.com/mark3labs/mcp-go
**Status:** Stable, widely adopted community MCP implementation

#### Full API Surface

**Server:**
```go
s := server.NewMCPServer("Demo", "1.0.0",
    server.WithToolCapabilities(false))

tool := mcp.NewTool("hello_world",
    mcp.WithDescription("Say hello"),
    mcp.WithString("name", mcp.Required(), mcp.Description("Name")))

s.AddTool(tool, helloHandler)
server.ServeStdio(s)
```

**Transport Layers:**
- **stdio** — JSON-RPC via stdin/stdout (local tools)
- **SSE** — Server-Sent Events (deprecated)
- **Streamable HTTP** — HTTP/2 with server-initiated streaming (recommended)

**Key Capabilities:**
- Full JSON-RPC 2.0 message handling
- Automatic schema generation from Go types
- Resource management for data source exposure
- Prompt template management
- Client-server architecture with session support

#### Also: Official MCP Go SDK

`modelcontextprotocol/go-sdk` — Maintained with Google collaboration, alternative to mcp-go.

---

### 4.2 Existing Go Agent Frameworks

#### Google Agent Development Kit (ADK) for Go
- **Released:** November 2025
- Event-driven runtime, agents as stateful systems
- Multi-agent architecture with A2A protocol
- Workflow agents: Sequential, Parallel, Loop constructs
- 30+ pre-built database tools via MCP Toolbox
- Development UI at localhost:4200

#### ByteDance Eino
- Battle-tested in Doubao and TikTok production
- **Streaming-first:** Automatic streaming throughout orchestration
- Component-based: ChatModel, Tool, Retriever, ChatTemplate
- Observability injection at fixed instrumentation points

#### Jetify AI SDK
- **Positioning:** Vercel AI SDK equivalent for Go
- Unified provider interface (OpenAI, Anthropic, Google)
- Provider-agnostic: switch via configuration
- Status: Public Alpha

#### agent-sdk-go
- Inspired by OpenAI Assistants API
- Agent + Tools + Runner architecture
- Supports OpenAI, Anthropic, local models (LM Studio)

#### LangChainGo
- Interface-driven design, modular
- Chains, Tools, Agents, Memory, Executor
- Active development toward Python parity

#### LinGoose
- Import only what you need
- Assistant, RAG, LLM, Thread Management, Embeddings

#### go-llm
- Tool integration: PythonREPL, BashTerminal, GoogleSearch, etc.
- Task-based architecture with validated schemas

---

### 4.3 Go Architectural Patterns for Agents

#### Interface-Driven Design

Go's implicit interface satisfaction enables clean component composition. Every major component (Model, Tool, Agent, Runner, Memory) should be defined by interfaces for modularity and testability.

#### Functional Options Pattern

```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) { c.Timeout = d }
}

func WithModel(model string) Option {
    return func(c *Config) { c.Model = model }
}

agent := NewAgent("analyzer",
    WithInstructions("Analyze data..."),
    WithModel("gpt-4o"),
    WithTimeout(60*time.Second),
    AddTool(searchTool),
)
```

Clean, readable, backward-compatible, used in grpc-go and zap.

#### Context Management & Cancellation

```go
ctx, cancel := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM)
defer cancel()

// All agent operations receive context
result, err := runner.Run(ctx, agent, messages)
```

#### Concurrency Patterns

- **Goroutines** for parallel tool execution
- **Channels** for streaming events to consumers
- **Fan-Out/Fan-In** for distributing work across agents
- **Context-based coordination** for cancellation across hierarchies
- **Rate limiting** via `golang.org/x/time/rate`

#### Streaming Architecture

- Go channels as natural streaming primitive
- Pattern: `<-chan Event` for consumers, buffered channels for backpressure
- Eino pattern: automatic stream concatenation, boxing, merging between nodes

#### Type Safety with Generics (Go 1.18+)

- Generic tool registration with compile-time type checking
- ~77% faster than reflection-based approach, 33% less memory
- Hybrid: generics for known types, reflection for dynamic data

#### JSON Schema for Tool Definitions

- `google/jsonschema-go` — Official Google package
- Auto-generate schemas from Go struct types
- LLMs use schemas to understand available tools

#### Error Handling

```go
type AgentError struct {
    Type    ErrorType  // ToolExecution, Handoff, Guardrail, Timeout, etc.
    Message string
    Cause   error
    Context map[string]any
}
```

Structured errors with severity levels driving monitoring and alerting.

---

## 5. Provider Abstraction & Model Routing

### 5.1 OpenRouter

**What it is:** A unified API gateway to 500+ models across 50+ cloud providers with intelligent routing.
**Why it matters:** Defines the gold standard for provider abstraction that our framework must support.

#### API Surface (OpenAI-Compatible)

- **Endpoint:** `POST /api/v1/chat/completions` (drop-in OpenAI replacement)
- **Models:** `GET /api/v1/models` — list all 500+ available models
- **Streaming:** SSE with `stream: true`, cancellation stops billing immediately
- **Authentication:** `Authorization: Bearer <key>` + optional `HTTP-Referer` and `X-OpenRouter-Title`

#### Model Routing Architecture

**Three routing modes:**
1. **Auto Router** (`openrouter/auto`) — Powered by NotDiamond, analyzes prompt and selects optimal model. No additional fees.
2. **Model Fallbacks** — Array of model IDs in priority order; auto-retry on error. Pricing uses model ultimately selected.
3. **Provider Customization** — Sort by price vs. throughput, restrict to specific providers, wildcard patterns (`anthropic/*`)

**Routing decision engine evaluates in real-time:**
- Provider uptime and availability
- Current rate limits
- Cost optimization preferences
- Model capabilities vs. prompt complexity
- Rolling 5-minute latency/throughput metrics
- Edge-based architecture adds ~25ms overhead

#### Retry & Error Handling

- Exponential backoff starting at 500ms, doubling to 10s max
- ±25% jitter to prevent thundering herd
- Retries on 429, 500, 502, 503, 504
- Respects `Retry-After` headers
- Circuit breaker pattern for cascading failure prevention

#### Plugins Ecosystem

- **web** — Real-time web search (append `:online` to model slug)
- **file-parser** — PDF processing
- **response-healing** — Automatic JSON repair for malformed responses
- **context-compression** — Middle-out prompt compression

#### Go SDK Options

| Library | Features |
|---------|----------|
| `eduardolat/openroutergo` | Fluent builder pattern, function calling, structured outputs |
| `hra42/openrouter-go` | Zero-dependency, full streaming, tool calling, web search |
| `wojtess/openrouter-api-go` | Chat completions with streaming, agent functionality |
| OpenAI-compatible | Any Go OpenAI client pointed to `openrouter.ai/api/v1` |

#### Key Insight for Our Framework

OpenRouter proves that a single unified API with intelligent routing is the right abstraction. Our framework should:
- Define a `ModelProvider` interface matching OpenAI's chat completions API
- Support provider routing (auto, fallback, preference-based)
- Enable bring-your-own-key and proxy patterns
- Track costs per-request via usage metadata

---

### 5.2 OpenClaw Architecture

**What it is:** The fastest-growing open-source AI agent platform (250K+ GitHub stars, 2M MAU).
**Why it matters:** Defines production patterns for agent runtimes, memory systems, and tool orchestration.

#### Five-Component Architecture

1. **Gateway** — WebSocket server connecting messaging platforms (Slack, WhatsApp, Telegram, Discord, Teams)
2. **Agent Runtime** — ReAct-style reasoning loops
3. **Memory** — File-based persistent storage (MEMORY.md + daily logs)
4. **Skills** — Plugin capabilities for actions
5. **Heartbeat** — Task scheduling and inbox monitoring

#### Agent Loop (ReAct Pipeline)

1. **Orchestrate** — Pick the right agent
2. **Resolve Model** — Choose cheapest model fitting constraints
3. **Build Prompt** — Assemble context from tools, skills, memory
4. **Guard Context** — Keep tokens in check
5. **Act & Repeat** — ReAct loop (tools → observe → iterate)

Session management uses serialized session keys with mutex protection to prevent race conditions.

#### Memory System (File-Based, No Database)

- **MEMORY.md** — Decisions, preferences, durable facts
- **memory/YYYY-MM-DD.md** — Daily notes and running context
- **Semantic search** — Hybrid BM25 + vector search over memory files
- **Human-editable** — Markdown files users can directly modify

**Key philosophy:** Files are the source of truth. Observable, controllable, portable.

#### Session Persistence (JSONL)

- Stored at `~/.openclaw/agents/<agentId>/sessions/<SessionId>.jsonl`
- Append-only format (crash-safe)
- Header line + entry lines with id/parentId
- Mutex-protected via `updateSessionStore` promise-chain

#### Tool System

- **Built-in:** Shell (exec), file operations (read/write/edit), browser automation
- **Skills:** Natural-language API integrations via SKILL.md files
- **Plugins:** Deep runtime extensions (TypeScript/JavaScript)
- **Webhooks:** HTTP endpoints for external notifications

#### Plugin Architecture

- Registry-based runtime loading
- Central registry exposes: tools, channels, providers, hooks, HTTP routes, CLI commands
- Naming: `pluginId.action` for gateway methods, `snake_case` for tools

#### Browser Automation (Semantic Snapshots)

Semantic snapshots (text-based accessibility tree) replace screenshots:
- Screenshot: ~5MB → Semantic snapshot: ~50KB
- Numeric refs for every interactive element
- Stable across layout changes (unlike CSS selectors)
- Dramatically reduces token costs

#### Go Implementations

- **clawgo** (official) — Go client library for OpenClaw
- **GoClaw** (community) — Pure Go reimplementation with multi-tenant support, native concurrency
- **openclaw-go** (a3tai) — Go port of APIs

#### Patterns to Borrow

1. **File-based memory** as debuggable, human-editable persistence
2. **JSONL append-only transcripts** for crash-safe session logs
3. **ReAct loop** as the default agent execution pattern
4. **Semantic snapshots** for browser tool efficiency
5. **Skill files** for declarative tool definitions
6. **Session serialization** with mutex protection

---

## 6. Architectural Design Partners

### 6.1 Buildkite Agent (Go Patterns)

**Repository:** github.com/buildkite/agent
**Why it matters:** A well-architected, production-grade Go application with patterns directly applicable to an agent execution framework.

#### Phase-Based Execution Model

Jobs execute through discrete phases, each with pre/post hooks:

1. **Setup** — Initialize environment, set variables
2. **Checkout** — Download code from repositories
3. **Plugin** — Download and prepare plugins
4. **Command** — Execute user's build script
5. **Artifact** — Upload artifacts (runs regardless of command success)
6. **Teardown** — Cleanup (always runs, even on failure)

**Critical pattern:** Split context approach — main context for execution, separate `graceCtx` for critical cleanup phases (artifact upload, log flush). This ensures important operations complete even during cancellation.

#### Hook System (Two Levels)

**Agent-level hooks:**
- `pre-bootstrap` — Block unwanted jobs
- `agent-shutdown` — Cleanup after all jobs

**Job-level hooks (at each phase):**
- `pre-checkout`, `post-checkout`
- `pre-command`, `post-command`
- `pre-artifact`, `post-artifact`

Hooks can be in any language (polyglot support), execute as shell processes inheriting job environment, and exported variables propagate to subsequent phases.

#### Configuration Patterns (Three-Layer)

**Priority order:**
1. Configuration file (YAML/TOML)
2. Environment variables (`BUILDKITE_*` prefix)
3. Command-line flags (runtime overrides)

Supports variable interpolation (`${VAR}`, `${VAR:start:end}`), required variables (`${VAR?message}`), and escaping (`$$`).

#### Signal Handling & Graceful Shutdown

- SIGTERM/SIGINT triggers graceful shutdown
- Running jobs receive cancellation via `BUILDKITE_JOB_CANCELLED=true`
- Primary context cancels, but `graceCtx` remains for cleanup
- Configurable grace periods (`--cancel-grace-period-seconds`, `--signal-grace-period-seconds`)
- After grace period: SIGKILL forces termination

#### Concurrency Model

- Worker pool with configurable concurrency limits
- Polling-based job pickup (more robust than event-driven in distributed settings)
- Exponential backoff with jitter in retry package
- Formula: `adjustment + (base ** attempts) + jitter`

#### Observability

- Real-time log streaming chunked for efficient transmission (2MB max rendered, 100MB file max)
- Prometheus integration on `/metrics` endpoint
- Key metrics: RunningJobsCount, ScheduledJobsCount, IdleAgentsCount, BusyAgentsCount
- Multiple backends: StatsD, CloudWatch, Stackdriver, OpenTelemetry

#### Security Patterns

- **Agent Token** (long-lived) + **Job Token** (per-execution, scoped)
- OIDC for vault authentication
- Automatic secret redaction in logs (`*_PASSWORD`, `*_SECRET`, `*_TOKEN`)

#### Package Organization

```
agent/       — Worker pool, job runner, log streamer
api/         — HTTP client for Buildkite API
clicommand/  — CLI subcommand definitions
process/     — Subprocess management with concurrent-safe buffering
retry/       — Exponential backoff implementation
```

#### Key Patterns for Our Framework

1. **Phase-based execution with hooks** — Break agent execution into phases (init, plan, execute, validate, cleanup) with extensible hooks at each boundary
2. **Split context for cancellation** — Main context for execution, grace context for cleanup
3. **Three-layer configuration** — File + env + CLI flags with clear precedence
4. **Polling with backoff+jitter** — More robust than pure event-driven for distributed systems
5. **Concurrent-safe buffering** — For streaming agent output without blocking
6. **Package-organized architecture** — Focused packages with clear responsibilities
7. **Automatic secret redaction** — Critical for LLM applications that may log prompts/responses

---

### 6.2 Agenta AI (LLMOps Patterns)

**Repository:** github.com/Agenta-AI/agenta
**Why it matters:** Defines observability, evaluation, and lifecycle management patterns essential for production LLM applications.

#### Observability (OpenTelemetry-Native)

- Accepts any span following OTel specification
- LLM-specific semantic conventions under `ag.*` namespace
- Multi-backend support (Datadog, Honeycomb, etc.)
- Vendor-neutral instrumentation

**Metrics captured:**
- `prompt_tokens`, `completion_tokens` — Token counts
- `cost.prompt`, `cost.completion`, `cost.total` — USD costs
- `latency` — Response time
- Custom metrics via `store_metrics()`

**Trace structure:** Nested spans for multi-step agent workflows with parent-child relationships, metadata for prompt versions and model identifiers.

#### Evaluation Framework

**Automated:**
- LLM-as-a-Judge (binary, multiclass, continuous scoring)
- 20+ pre-built evaluators
- Custom evaluator definitions
- Pairwise comparison between outputs

**Human-in-the-Loop:**
- Manual annotation workflows
- Production data capture for test set building
- Continuous improvement loops

**Online (Production):**
- Post-hoc evaluation on production traces
- Monitor agent reliability (tool calls, reasoning steps)
- Build test sets from edge cases

#### Configuration Management (Git-Like)

- **Variants/Branches** — Parallel prompt versions without affecting production
- **Immutable versions** — Each variant has committed snapshot
- **Environments** — Map variants to dev/staging/production
- **Rollback** — Instant rollback to any previous version
- **A/B Testing** — Side-by-side comparison against test cases

**PromptTemplate bundles:** Messages + model selection + parameters as single unit. Jinja templating for variables.

#### Patterns for Our Framework

1. **OpenTelemetry-native observability** from day one
2. **Git-like versioning** for agent configurations and prompts
3. **Evaluation as a production concern** — integrated with tracing
4. **Cost tracking at span level** — per-request cost visibility
5. **Separation of configuration & execution** — runtime config fetching
6. **Framework-agnostic integration** — support multiple agent patterns via plugins

---

## 7. Cross-Cutting Concerns

### 5.1 Universal Agent Abstractions

Every major framework converges on these core abstractions:

| Abstraction | OpenAI | Claude | LangGraph | Vercel AI | Our Framework |
|-------------|--------|--------|-----------|-----------|---------------|
| Agent definition | Agent class | AgentDefinition | Node in StateGraph | ToolLoopAgent | `Agent` struct + functional options |
| Execution engine | Runner | Built-in loop | CompiledGraph.invoke() | generateText/streamText | `Runner` with pluggable backends |
| Tool system | Tool with JSON Schema | MCP tools + @tool | LangChain tools | Zod-schema tools | `Tool` interface + generic registration |
| Agent transfer | Handoff pseudo-tool | Subagent delegation | Conditional edges | Tool-based handoffs | `Handoff` with context transfer |
| Validation | Guardrails | Permission system + hooks | Custom node validation | Middleware | `Guardrail` interface + middleware chain |
| State | Message history | Sessions with persistence | Reducer-driven state | Transport-based state | `State` with pluggable persistence |
| Streaming | SSE events | Async generator | Stream modes | streamText/streamObject | Go channels + SSE adapter |
| Tracing | Trace object | Cost/usage tracking | LangSmith | AI Gateway metrics | OpenTelemetry native |

### 5.2 MCP as Universal Tool Protocol

MCP is becoming the standard for tool integration:
- OpenAI Agents SDK supports MCP tools
- Claude Agent SDK has native MCP integration
- Temporal's plugin manages MCP server lifecycles
- mcp-go provides full Go implementation
- Our framework should make MCP a first-class citizen

### 5.3 Durable Execution Integration Patterns

**Pattern 1: Activity-as-Tool (Temporal style)**
- Map each tool to a durable activity
- LLM calls are activities with timeout/retry
- Orchestration logic in deterministic workflow

**Pattern 2: Task-based Orchestration (Hatchet style)**
- Agent logic as durable function
- Tool calls spawn child tasks
- DAG-based routing of outputs

**Pattern 3: Snapshot-based Checkpointing (Trigger.dev style)**
- Process state snapshotted at waitpoints
- Resume from snapshot on trigger
- Zero compute during pause

### 5.4 Streaming Architecture Considerations

| Platform | Streaming Approach | Go Implication |
|----------|-------------------|----------------|
| Temporal | Not in workflows; event handlers | Activity results buffered, stream externally |
| Hatchet | Task primitives support streaming | Stream via task events |
| Trigger.dev | Real-time Streams v2 | N/A (no Go SDK) |
| Direct | Channel-based streaming | Most natural for Go |

**Recommendation:** Channel-based streaming with SSE/WebSocket adapters, plus durable execution platform adapters that handle buffering.

### 5.5 Observability Stack

All frameworks converge on these requirements:
- **Tracing:** Full execution path with tool calls, handoffs, LLM invocations
- **Metrics:** Token usage, latency per component, cost tracking
- **Logging:** Structured, searchable, severity-leveled
- **Standard:** OpenTelemetry for vendor-neutral instrumentation

---

## 8. Key Design Decisions for Our Framework

Based on this research, here are the critical decisions for the PRD:

### 8.1 Core Architecture Decisions

1. **Interface-driven design** — Every component (Agent, Tool, Runner, Model, State, Guardrail) defined by Go interfaces
2. **Functional options** for configuration — Clean, extensible, backward-compatible
3. **Channel-based streaming** as the native primitive
4. **Context propagation** throughout for cancellation, timeouts, and metadata
5. **Generics** for type-safe tool registration and state management
6. **OpenTelemetry native** for observability from day one

### 8.2 Agent Abstractions to Implement

- **Agent** — Instructions, tools, model config, guardrails (like OpenAI)
- **Runner** — Execution loop with pluggable backends (like OpenAI, but Go-native)
- **Handoff** — Agent-to-agent transfer with context (like OpenAI)
- **Guardrail** — Input/output/tool validation (like OpenAI + Claude's permission system)
- **Hooks** — Lifecycle events (inspired by Claude's 14+ hook events)
- **Subagent** — Composition with context isolation (like Claude)
- **State** — Reducer-driven with persistence backends (like LangGraph)

### 8.3 Provider Abstraction

Follow Vercel AI SDK's model specification + OpenRouter's routing patterns:
- `LanguageModel` interface with `Generate()` and `Stream()` methods
- Provider implementations for OpenAI, Anthropic, Google, local models
- **OpenRouter-compatible:** Drop-in support for OpenRouter as a meta-provider
- **Model routing:** Auto-routing, fallback chains, provider preference sorting
- Middleware chain for cross-cutting concerns (logging, caching, guardrails, cost tracking)
- **Bring-your-own-key:** Users provide their own API keys per provider

### 8.4 MCP Integration

- Use mcp-go or official Go SDK as foundation
- First-class MCP server and client support
- Tool naming convention: `mcp__<server>__<tool>`
- Automatic schema generation from Go types
- Transport layer abstraction (stdio, streamable HTTP)

### 8.5 Durable Execution Strategy

**Pluggable backend architecture:**

```go
type DurableExecutor interface {
    RegisterTool(tool Tool) error
    ExecuteTool(ctx context.Context, name string, input any) (any, error)
    RunAgent(ctx context.Context, agent Agent, input AgentInput) (AgentResult, error)
    WaitForSignal(ctx context.Context, signalName string) (any, error)
}

// Implementations:
// - TemporalExecutor (maps tools to activities, agents to workflows)
// - HatchetExecutor (maps tools to tasks, agents to durable functions)
// - LocalExecutor (in-process, for development/testing)
```

### 8.6 Streaming Design

```go
type StreamEvent struct {
    Type      EventType  // TextDelta, ToolCallStart, ToolCallDone, etc.
    Data      any
    Timestamp time.Time
    AgentID   string
}

// Consumer pattern
events := runner.RunStream(ctx, agent, messages)
for event := range events {
    switch event.Type {
    case TextDelta:
        // Forward to SSE/WebSocket
    case ToolCallStart:
        // Log tool invocation
    }
}
```

### 8.7 Execution Model (Inspired by Buildkite Agent)

**Phase-based agent execution with hooks:**
```go
// Agent execution phases (each with pre/post hooks)
type Phase string
const (
    PhaseInit     Phase = "init"      // Initialize context, load config
    PhasePlan     Phase = "plan"      // Agent plans next action
    PhaseExecute  Phase = "execute"   // Execute tool calls
    PhaseValidate Phase = "validate"  // Guardrails check results
    PhaseCleanup  Phase = "cleanup"   // Always runs, even on error
)
```

- Split context pattern: main `ctx` for execution, `graceCtx` for cleanup
- Three-layer configuration: file + env vars + programmatic options
- Automatic secret redaction in traces/logs
- Concurrent-safe output buffering for streaming

### 8.8 Observability Strategy (Inspired by Agenta)

- **OpenTelemetry-native** from day one — no proprietary format
- **LLM semantic conventions** following emerging OTel standards
- **Per-span cost tracking** — prompt tokens, completion tokens, USD costs
- **Git-like versioning** for agent configurations and prompts
- **Evaluation hooks** — integrate with Agenta, LangSmith, or custom evaluators
- **Structured metrics:** `store_meta()`, `store_metrics()` equivalents in Go

### 8.9 Memory & Persistence (Inspired by OpenClaw)

- **File-based memory** as human-readable, debuggable option
- **JSONL append-only transcripts** for crash-safe session logs
- **Pluggable persistence:** files, SQLite, PostgreSQL, Redis
- **Semantic search** over memory (hybrid BM25 + vector)
- **Reducer-driven state** (from LangGraph) for multi-agent environments

### 8.10 What We Should NOT Build

- A graph-based orchestration engine (LangGraph's approach adds complexity; our loop-based runner with handoffs covers most use cases)
- A UI framework (leave to consumers)
- An evaluation platform (integrate with existing: LangSmith, Agenta, custom)
- A deployment platform (integrate with existing: Temporal, Hatchet, K8s)
- A model proxy (integrate with OpenRouter instead of building our own)

---

## 9. Sources

### Agent Frameworks
- [OpenAI Agents SDK](https://github.com/openai/openai-agents-python) — GitHub repository
- [OpenAI Agents SDK Docs](https://openai.github.io/openai-agents-python/) — Official documentation
- [Claude Agent SDK](https://platform.claude.com/docs/en/agent-sdk/overview) — Anthropic documentation
- [LangChain Deep Agents](https://blog.langchain.com/deep-agents/) — Architecture blog
- [LangGraph](https://github.com/langchain-ai/langgraph) — GitHub repository
- [Vercel AI SDK](https://ai-sdk.dev/docs/introduction) — Official documentation
- [AI SDK 6](https://vercel.com/blog/ai-sdk-6) — Latest release blog

### Durable Execution Platforms
- [Temporal AI Cookbook](https://docs.temporal.io/ai-cookbook) — Official AI integration guide
- [Temporal OpenAI Agents Plugin](https://python.temporal.io/temporalio.contrib.openai_agents.OpenAIAgentsPlugin.html) — API reference
- [Temporal AI Solutions](https://temporal.io/solutions/ai) — Platform positioning
- [Temporal + OpenAI Agents](https://temporal.io/blog/announcing-openai-agents-sdk-integration) — Integration announcement
- [Durable Execution Meets AI](https://temporal.io/blog/durable-execution-meets-ai-why-temporal-is-the-perfect-foundation-for-ai) — Technical deep dive
- [Multi-Agent Workflows with Temporal](https://temporal.io/blog/what-are-multi-agent-workflows) — Architecture patterns
- [Durable MCP with Temporal](https://temporal.io/blog/durable-mcp-how-to-give-agentic-systems-superpowers) — MCP integration
- [Hatchet](https://hatchet.run/) — Official site
- [Hatchet: Why Go for Agents](https://hatchet.run/blog/go-agents) — Go positioning
- [Hatchet Go SDK](https://pkg.go.dev/github.com/hatchet-dev/hatchet) — Go package docs
- [Hatchet Architecture](https://docs.hatchet.run/v1/architecture-and-guarantees) — Technical architecture
- [Trigger.dev AI Agents](https://trigger.dev/product/ai-agents) — Product page
- [Trigger.dev vs Temporal](https://trigger.dev/vs/temporal) — Comparison
- [Trigger.dev Realtime Streams](https://trigger.dev/docs/realtime/streams) — Streaming docs
- [Durable Workflow Platforms Comparison](https://render.com/articles/durable-workflow-platforms-ai-agents-llm-workloads) — Industry comparison

### Go Ecosystem
- [mcp-go](https://github.com/mark3labs/mcp-go) — MCP Go implementation
- [Google ADK for Go](https://github.com/google/adk-go) — Official Google agent toolkit
- [Eino (ByteDance)](https://github.com/cloudwego/eino) — Production LLM framework
- [LangChainGo](https://github.com/tmc/langchaingo/) — Go port of LangChain
- [LangGraphGo](https://github.com/tmc/langgraphgo) — Go port of LangGraph
- [Jetify AI SDK](https://github.com/jetify-com/ai) — Go AI framework
- [agent-sdk-go](https://github.com/nlpodyssey/openai-agents-go) — OpenAI-style Go agents
- [LinGoose](https://github.com/henomis/lingoose) — Modular Go LLM framework
- [go-llm](https://github.com/natexcvi/go-llm) — Go LLM agent framework
- [Building LLM-powered apps in Go](https://go.dev/blog/llmpowered) — Official Go blog
- [Top 7 Golang AI Agent Frameworks 2026](https://reliasoftware.com/blog/golang-ai-agent-frameworks) — Framework survey
- [The Go Revolution: Golang in AI Agent Development](https://muleai.io/blog/2026-02-28-golang-ai-agent-frameworks-2026/) — Industry analysis

### Provider Abstraction & Model Routing
- [OpenRouter API Reference](https://openrouter.ai/docs/api/reference/overview) — Full API documentation
- [OpenRouter Auto Router](https://openrouter.ai/docs/guides/routing/routers/auto-router) — Intelligent model routing
- [OpenRouter Model Fallbacks](https://openrouter.ai/docs/guides/routing/model-fallbacks) — Fallback chain patterns
- [OpenRouter Structured Outputs](https://openrouter.ai/docs/guides/features/structured-outputs) — Schema validation
- [openroutergo](https://github.com/eduardolat/openroutergo) — Go SDK with fluent builder
- [openrouter-go](https://github.com/hra42/openrouter-go) — Zero-dependency Go SDK
- [OpenClaw Architecture](https://docs.openclaw.ai/concepts/agent) — Agent runtime design
- [OpenClaw Agent Loop](https://docs.openclaw.ai/concepts/agent-loop) — ReAct pipeline
- [OpenClaw Memory System](https://docs.openclaw.ai/concepts/memory) — File-based memory
- [clawgo](https://github.com/openclaw/clawgo) — Official Go client
- [GoClaw](https://github.com/nextlevelbuilder/goclaw) — Pure Go reimplementation

### Architectural Design Partners
- [Buildkite Agent](https://github.com/buildkite/agent) — Go agent architecture reference
- [Buildkite Agent Hooks](https://buildkite.com/docs/agent/v3/hooks) — Lifecycle hook system
- [Buildkite Plugin System](https://buildkite.com/docs/pipelines/integrations/plugins/writing) — Plugin architecture
- [Buildkite Agent Configuration](https://buildkite.com/docs/agent/v3/configuration) — Three-layer config
- [Buildkite Prometheus Metrics](https://buildkite.com/docs/agent/v3/agent-stack-k8s/prometheus-metrics) — Observability
- [Agenta AI](https://github.com/Agenta-AI/agenta) — LLMOps platform
- [Agenta Prompt Versioning](https://agenta.ai/blog/prompt-versioning-guide) — Git-like versioning
- [Agenta LLM-as-a-Judge](https://agenta.ai/blog/llm-as-a-judge-guide-to-llm-evaluation-best-practices) — Evaluation patterns
- [Agenta OTel Integration](https://agenta.ai/blog/the-ai-engineer-s-guide-to-llm-observability-with-opentelemetry) — Observability

### Architecture & Patterns
- [Functional Options in Go](https://www.yellowduck.be/posts/the-functional-options-pattern-in-go) — Pattern guide
- [Go Concurrency for Agents](https://aminmsv01.medium.com/concurrency-in-go-foundations-patterns-and-agentic-architectures-ee69ac212df9) — Concurrency deep dive
- [JSON Schema for Go](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html) — Google's package
- [Scaling LLMs with Golang](https://www.assembled.com/blog/scaling-llms-with-golang-how-we-serve-millions-of-llm-requests) — Production scaling
- [Agent Handoffs in Multi-Agent Systems](https://towardsdatascience.com/how-agent-handoffs-work-in-multi-agent-systems/) — Architecture patterns
- [Gartner: 40% of Enterprise Apps Will Feature AI Agents by 2026](https://www.gartner.com/) — Market projection
