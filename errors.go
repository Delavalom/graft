package graft

import "fmt"

type ErrorType string

const (
	ErrToolExecution   ErrorType = "tool_execution"
	ErrHandoff         ErrorType = "handoff"
	ErrGuardrail       ErrorType = "guardrail"
	ErrTimeout         ErrorType = "timeout"
	ErrContextLength   ErrorType = "context_length"
	ErrInvalidToolCall ErrorType = "invalid_tool_call"
	ErrRateLimit       ErrorType = "rate_limit"
	ErrProvider        ErrorType = "provider"
)

func (e ErrorType) Error() string {
	return string(e)
}

type AgentError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]any
}

func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("graft: %s: %s: %s", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("graft: %s: %s", e.Type, e.Message)
}

func (e *AgentError) Unwrap() error {
	return e.Cause
}

func (e *AgentError) Is(target error) bool {
	if t, ok := target.(ErrorType); ok {
		return e.Type == t
	}
	return false
}

func NewAgentError(errType ErrorType, message string, cause error) *AgentError {
	return &AgentError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}
