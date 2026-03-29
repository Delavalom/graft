package graft

type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"
	ToolChoiceRequired ToolChoice = "required"
	ToolChoiceNone     ToolChoice = "none"
)

func ToolChoiceSpecific(name string) ToolChoice {
	return ToolChoice("specific:" + name)
}

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
	SubAgents    []*SubAgent
	Hooks        *HookRegistry
	Metadata     map[string]any
}
