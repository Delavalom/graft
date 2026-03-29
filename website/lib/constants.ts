export const REPO_URL = "https://github.com/delavalom/graft";
export const MODULE_PATH = "github.com/delavalom/graft";

export const providers = [
  "Anthropic",
  "OpenAI",
  "Google Gemini",
  "OpenRouter",
  "Ollama",
  "LM Studio",
];

export const useCases = [
  {
    title: "Customer Support Agent",
    description: "Route conversations to specialized agents with type-safe handoffs.",
    code: `agent := graft.NewAgent("support",
  graft.WithTools(lookupOrder, refund),
  graft.WithHandoffs(handoff.To(billing)),
)`,
  },
  {
    title: "Multi-Agent Orchestration",
    description: "Compose sub-agents that run in parallel with isolated context.",
    code: `graft.RunSubAgentsParallel(ctx, runner,
  []graft.SubAgent{
    {Agent: researcher},
    {Agent: writer},
  }, messages)`,
  },
  {
    title: "Durable AI Pipelines",
    description: "Survive crashes with Temporal or Hatchet-backed execution.",
    code: `runner := temporal.NewRunner(client,
  temporal.WithTaskQueue("agents"),
)
result, _ := runner.Run(ctx, agent, msgs)`,
  },
  {
    title: "Observable RAG",
    description: "Full OpenTelemetry tracing for every token and tool call.",
    code: `runner := otel.InstrumentRunner(
  graft.NewDefaultRunner(model),
  otel.WithTracerProvider(tp),
)`,
  },
];

export const dxSnippets = [
  {
    label: "Define a tool",
    code: `tool := graft.NewTool("search", "Search the web",
  func(ctx context.Context, p struct {
    Query string \`json:"query"\`
  }) (string, error) {
    return search(p.Query), nil
  })`,
  },
  {
    label: "Switch providers",
    code: `model := openai.New(openai.WithModel("gpt-4o"))
model := anthropic.New(anthropic.WithModel("claude-sonnet-4-20250514"))
model := google.New(google.WithModel("gemini-2.5-pro"))`,
  },
  {
    label: "Add observability",
    code: `runner := graft.NewDefaultRunner(model)
runner = otel.InstrumentRunner(runner,
  otel.WithTracerProvider(tp))`,
  },
];

export const features = [
  { icon: "Wrench", title: "Type-Safe Tools", description: "Struct tags become JSON Schema via reflection." },
  { icon: "Route", title: "Multi-Provider", description: "Anthropic, OpenAI, Gemini. Fallback + round-robin." },
  { icon: "ArrowLeftRight", title: "Agent Handoffs", description: "LLM-driven routing between specialized agents." },
  { icon: "Webhook", title: "Lifecycle Hooks", description: "14+ events: pre/post generate, tool calls, errors." },
  { icon: "Shield", title: "Guardrails", description: "Input, output, and tool validation out of the box." },
  { icon: "Activity", title: "OpenTelemetry", description: "Tracing and metrics from day one. Vendor-neutral." },
  { icon: "Database", title: "Session State", description: "Memory and file-backed persistence. Transparent." },
  { icon: "Users", title: "SubAgents", description: "Context-isolated child agents. Run in parallel." },
  { icon: "Radio", title: "Streaming + SSE", description: "Go channels with built-in SSE HTTP adapter." },
  { icon: "Plug", title: "MCP Protocol", description: "Client and server support for the MCP ecosystem." },
  { icon: "Timer", title: "Temporal", description: "Durable execution with deterministic replay." },
  { icon: "Container", title: "Hatchet", description: "PostgreSQL-powered high-throughput durability." },
  { icon: "Workflow", title: "Graph Orchestration", description: "State machines, reducers, checkpointing." },
  { icon: "GitBranch", title: "Trigger.dev", description: "Waitpoints, warm starts, zero-timeout tasks." },
  { icon: "Eye", title: "Pluggable Tracing", description: "Braintrust, LangSmith, OTel, or custom providers." },
];

export const comparison = {
  headers: ["", "Graft", "LangChain", "OpenAI SDK", "Vercel AI SDK"],
  rows: [
    { label: "Language", values: ["Go-native", "Python", "Python", "TypeScript"], graftWins: true },
    { label: "Durable Execution", values: ["Temporal + Hatchet + Trigger.dev", "Checkpoints", "None", "None"], graftWins: true },
    { label: "MCP Support", values: ["Client + Server", "Limited", "Basic", "None"], graftWins: true },
    { label: "Observability", values: ["OTel + Braintrust + LangSmith", "LangSmith", "Custom", "Middleware"], graftWins: true },
    { label: "Graph Orchestration", values: ["Built-in", "LangGraph (separate)", "None", "None"], graftWins: true },
    { label: "Dependencies", values: ["1 (OTel)", "Many", "Several", "Many"], graftWins: true },
    { label: "Tool Type Safety", values: ["Generics + reflection", "Runtime", "Runtime", "Zod schemas"], graftWins: true },
    { label: "Streaming", values: ["Go channels + SSE", "Async iteration", "SSE", "streamText"], graftWins: true },
  ],
};
