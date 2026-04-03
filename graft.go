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
