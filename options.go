package graft

type AgentOption func(*Agent)

func WithInstructions(instructions string) AgentOption {
	return func(a *Agent) { a.Instructions = instructions }
}

func WithModel(model string) AgentOption {
	return func(a *Agent) { a.Model = model }
}

func WithTemperature(temp float64) AgentOption {
	return func(a *Agent) { a.Temperature = &temp }
}

func WithMaxTokens(max int) AgentOption {
	return func(a *Agent) { a.MaxTokens = &max }
}

func WithToolChoice(tc ToolChoice) AgentOption {
	return func(a *Agent) { a.ToolChoice = tc }
}

func WithTools(tools ...Tool) AgentOption {
	return func(a *Agent) { a.Tools = append(a.Tools, tools...) }
}

func WithGuardrails(guardrails ...Guardrail) AgentOption {
	return func(a *Agent) { a.Guardrails = append(a.Guardrails, guardrails...) }
}

func WithHandoffs(handoffs ...Handoff) AgentOption {
	return func(a *Agent) { a.Handoffs = append(a.Handoffs, handoffs...) }
}

func WithHooks(hooks *HookRegistry) AgentOption {
	return func(a *Agent) { a.Hooks = hooks }
}

func WithMetadata(meta map[string]any) AgentOption {
	return func(a *Agent) { a.Metadata = meta }
}

type RunOption func(*RunConfig)

type RunConfig struct {
	MaxIterations int
	ParallelTools bool
}

func DefaultRunConfig() RunConfig {
	return RunConfig{
		MaxIterations: 10,
		ParallelTools: true,
	}
}

func WithMaxIterations(n int) RunOption {
	return func(c *RunConfig) { c.MaxIterations = n }
}

func WithParallelTools(enabled bool) RunOption {
	return func(c *RunConfig) { c.ParallelTools = enabled }
}
