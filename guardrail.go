package graft

import "context"

type GuardrailType string

const (
	GuardrailInput  GuardrailType = "input"
	GuardrailOutput GuardrailType = "output"
	GuardrailTool   GuardrailType = "tool"
)

type Guardrail interface {
	Name() string
	Type() GuardrailType
	Validate(ctx context.Context, data *ValidationData) (*ValidationResult, error)
}

type ValidationData struct {
	Messages   []Message
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Agent      *Agent
}

type ValidationResult struct {
	Pass     bool
	Message  string
	Modified any
}
