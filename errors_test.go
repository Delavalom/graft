package graft

import (
	"errors"
	"fmt"
	"testing"
)

func TestAgentErrorMessage(t *testing.T) {
	err := &AgentError{
		Type:    ErrToolExecution,
		Message: "tool search failed",
		Cause:   fmt.Errorf("connection refused"),
	}
	want := "graft: tool_execution: tool search failed: connection refused"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAgentErrorWithoutCause(t *testing.T) {
	err := &AgentError{
		Type:    ErrTimeout,
		Message: "max iterations exceeded",
	}
	want := "graft: timeout: max iterations exceeded"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAgentErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &AgentError{
		Type:    ErrProvider,
		Message: "API error",
		Cause:   cause,
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}

func TestAgentErrorIs(t *testing.T) {
	err := &AgentError{Type: ErrRateLimit, Message: "429"}
	if !errors.Is(err, ErrRateLimit) {
		t.Error("errors.Is should match ErrorType")
	}
	if errors.Is(err, ErrTimeout) {
		t.Error("errors.Is should not match different ErrorType")
	}
}

func TestNewAgentError(t *testing.T) {
	cause := fmt.Errorf("something broke")
	err := NewAgentError(ErrToolExecution, "search failed", cause)
	if err.Type != ErrToolExecution {
		t.Errorf("Type = %q, want %q", err.Type, ErrToolExecution)
	}
	if err.Cause != cause {
		t.Error("Cause not set")
	}
}
