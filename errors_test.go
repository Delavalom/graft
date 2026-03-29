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

// --- ProviderError / NewProviderError tests ---

func TestNewProviderError_401(t *testing.T) {
	err := NewProviderError(401, "openai", nil)
	if err.Type != ErrProvider {
		t.Errorf("Type = %q, want %q", err.Type, ErrProvider)
	}
	pe := providerErrorFrom(t, err)
	if pe.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", pe.StatusCode)
	}
	if pe.Retryable {
		t.Error("401 should not be retryable")
	}
	wantGuidance := "Invalid API key. Check your API key configuration."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_402(t *testing.T) {
	err := NewProviderError(402, "anthropic", nil)
	pe := providerErrorFrom(t, err)
	if pe.Retryable {
		t.Error("402 should not be retryable")
	}
	wantGuidance := "Insufficient credits. Add credits to your provider account."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_403(t *testing.T) {
	err := NewProviderError(403, "google", nil)
	pe := providerErrorFrom(t, err)
	if pe.Retryable {
		t.Error("403 should not be retryable")
	}
	wantGuidance := "Access denied. Your API key may not have permission for this model."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_404(t *testing.T) {
	err := NewProviderError(404, "openai", nil)
	pe := providerErrorFrom(t, err)
	if pe.Retryable {
		t.Error("404 should not be retryable")
	}
	wantGuidance := "Model not found. Verify the model ID is correct."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_429(t *testing.T) {
	err := NewProviderError(429, "openai", nil)
	pe := providerErrorFrom(t, err)
	if !pe.Retryable {
		t.Error("429 should be retryable")
	}
	wantGuidance := "Rate limited."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_500(t *testing.T) {
	err := NewProviderError(500, "openai", nil)
	pe := providerErrorFrom(t, err)
	if !pe.Retryable {
		t.Error("500 should be retryable")
	}
	wantGuidance := "Provider server error. This is temporary."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_502(t *testing.T) {
	err := NewProviderError(502, "anthropic", nil)
	pe := providerErrorFrom(t, err)
	if !pe.Retryable {
		t.Error("502 should be retryable")
	}
	wantGuidance := "Provider server error. This is temporary."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_503(t *testing.T) {
	err := NewProviderError(503, "google", nil)
	pe := providerErrorFrom(t, err)
	if !pe.Retryable {
		t.Error("503 should be retryable")
	}
	wantGuidance := "Provider server error. This is temporary."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestNewProviderError_ProviderName(t *testing.T) {
	err := NewProviderError(401, "anthropic", nil)
	pe := providerErrorFrom(t, err)
	if pe.ProviderName != "anthropic" {
		t.Errorf("ProviderName = %q, want %q", pe.ProviderName, "anthropic")
	}
}

func TestNewProviderError_JSONBody_OpenAIShape(t *testing.T) {
	body := []byte(`{"error":{"message":"You exceeded your current quota","code":"insufficient_quota","type":"requests"}}`)
	err := NewProviderError(429, "openai", body)
	pe := providerErrorFrom(t, err)
	if pe.Guidance != "You exceeded your current quota" {
		t.Errorf("Guidance = %q, want provider message from body", pe.Guidance)
	}
	if pe.ProviderCode != "insufficient_quota" {
		t.Errorf("ProviderCode = %q, want %q", pe.ProviderCode, "insufficient_quota")
	}
}

func TestNewProviderError_JSONBody_AnthropicShape(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	err := NewProviderError(401, "anthropic", body)
	pe := providerErrorFrom(t, err)
	if pe.Guidance != "invalid x-api-key" {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, "invalid x-api-key")
	}
}

func TestNewProviderError_JSONBody_TopLevelMessage(t *testing.T) {
	body := []byte(`{"message":"API key not valid. Please pass a valid API key.","code":400}`)
	err := NewProviderError(401, "google", body)
	pe := providerErrorFrom(t, err)
	if pe.Guidance != "API key not valid. Please pass a valid API key." {
		t.Errorf("Guidance = %q, want top-level message from body", pe.Guidance)
	}
}

func TestNewProviderError_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	err := NewProviderError(500, "openai", body)
	pe := providerErrorFrom(t, err)
	// Falls back to status-code guidance
	wantGuidance := "Provider server error. This is temporary."
	if pe.Guidance != wantGuidance {
		t.Errorf("Guidance = %q, want %q", pe.Guidance, wantGuidance)
	}
}

func TestAgentError_IsRetryable_True(t *testing.T) {
	err := NewProviderError(429, "openai", nil)
	if !err.IsRetryable() {
		t.Error("IsRetryable() should return true for 429")
	}
}

func TestAgentError_IsRetryable_False(t *testing.T) {
	err := NewProviderError(401, "openai", nil)
	if err.IsRetryable() {
		t.Error("IsRetryable() should return false for 401")
	}
}

func TestAgentError_IsRetryable_NonProvider(t *testing.T) {
	err := &AgentError{Type: ErrToolExecution, Message: "boom"}
	if err.IsRetryable() {
		t.Error("IsRetryable() should return false for non-provider errors")
	}
}

func TestAgentError_StatusCode(t *testing.T) {
	err := NewProviderError(403, "google", nil)
	if got := err.StatusCode(); got != 403 {
		t.Errorf("StatusCode() = %d, want 403", got)
	}
}

func TestAgentError_StatusCode_Zero_NonProvider(t *testing.T) {
	err := &AgentError{Type: ErrTimeout, Message: "timed out"}
	if got := err.StatusCode(); got != 0 {
		t.Errorf("StatusCode() = %d, want 0 for non-provider error", got)
	}
}

func TestNewProviderError_ErrorMessage_Contains_ProviderName(t *testing.T) {
	err := NewProviderError(401, "openai", nil)
	msg := err.Error()
	if !contains(msg, "openai") {
		t.Errorf("Error() = %q, expected to contain provider name", msg)
	}
	if !contains(msg, "401") {
		t.Errorf("Error() = %q, expected to contain status code", msg)
	}
}

// helpers

func providerErrorFrom(t *testing.T, ae *AgentError) *ProviderError {
	t.Helper()
	if ae.Context == nil {
		t.Fatal("AgentError.Context is nil")
	}
	pe, ok := ae.Context["provider_error"].(*ProviderError)
	if !ok {
		t.Fatalf("AgentError.Context[provider_error] is %T, want *ProviderError", ae.Context["provider_error"])
	}
	return pe
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
