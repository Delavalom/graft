package otel

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
