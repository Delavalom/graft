package graft

import (
	"encoding/json"
	"fmt"
	"time"
)

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

// ProviderError holds structured information about an HTTP error from a provider.
type ProviderError struct {
	StatusCode   int
	ProviderCode string
	ProviderName string
	Retryable    bool
	RetryAfter   time.Duration
	Guidance     string
}

// providerErrorBody is used to parse common provider error JSON shapes.
type providerErrorBody struct {
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"` // some providers use int, others string
		Type    string `json:"type"`
	} `json:"error"`
	// Anthropic / Google top-level shape
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// NewProviderError builds an *AgentError of type ErrProvider with structured
// ProviderError context. It maps common HTTP status codes to actionable guidance
// and attempts to parse provider-specific error messages from body.
func NewProviderError(statusCode int, providerName string, body []byte) *AgentError {
	pe := &ProviderError{
		StatusCode:   statusCode,
		ProviderName: providerName,
	}

	// Try to extract a provider error message / code from the JSON body.
	if len(body) > 0 {
		var parsed providerErrorBody
		if err := json.Unmarshal(body, &parsed); err == nil {
			if parsed.Error != nil {
				pe.ProviderCode = fmt.Sprintf("%v", parsed.Error.Code)
				if parsed.Error.Message != "" {
					pe.Guidance = parsed.Error.Message
				}
			} else if parsed.Message != "" {
				pe.Guidance = parsed.Message
				if parsed.Code != 0 {
					pe.ProviderCode = fmt.Sprintf("%d", parsed.Code)
				}
			}
		}
	}

	// Map status codes to guidance and retryability, overriding body-derived
	// guidance only when no provider message was found.
	guidance, retryable := guidanceForStatus(statusCode)
	if pe.Guidance == "" {
		pe.Guidance = guidance
	}
	pe.Retryable = retryable

	msg := fmt.Sprintf("%s: HTTP %d: %s", providerName, statusCode, pe.Guidance)

	ae := &AgentError{
		Type:    ErrProvider,
		Message: msg,
		Context: map[string]any{
			"provider_error": pe,
		},
	}
	return ae
}

// guidanceForStatus returns actionable guidance and retryability for an HTTP status code.
func guidanceForStatus(statusCode int) (guidance string, retryable bool) {
	switch statusCode {
	case 401:
		return "Invalid API key. Check your API key configuration.", false
	case 402:
		return "Insufficient credits. Add credits to your provider account.", false
	case 403:
		return "Access denied. Your API key may not have permission for this model.", false
	case 404:
		return "Model not found. Verify the model ID is correct.", false
	case 429:
		return "Rate limited.", true
	case 500, 502, 503:
		return "Provider server error. This is temporary.", true
	default:
		return fmt.Sprintf("Unexpected status code %d.", statusCode), false
	}
}

// IsRetryable reports whether the error is safe to retry.
// For non-provider errors it always returns false.
func (e *AgentError) IsRetryable() bool {
	if e.Context == nil {
		return false
	}
	pe, ok := e.Context["provider_error"].(*ProviderError)
	if !ok {
		return false
	}
	return pe.Retryable
}

// StatusCode returns the HTTP status code stored inside a provider error,
// or 0 if this AgentError was not created from a provider HTTP response.
func (e *AgentError) StatusCode() int {
	if e.Context == nil {
		return 0
	}
	pe, ok := e.Context["provider_error"].(*ProviderError)
	if !ok {
		return 0
	}
	return pe.StatusCode
}
