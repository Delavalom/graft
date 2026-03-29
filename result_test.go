package graft

import (
	"testing"
	"time"
)

func TestUsageTotalTokens(t *testing.T) {
	u := Usage{PromptTokens: 100, CompletionTokens: 50}
	if got := u.TotalTokens(); got != 150 {
		t.Errorf("TotalTokens() = %d, want 150", got)
	}
}

func TestResultLastAssistantText(t *testing.T) {
	r := &Result{
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
			{Role: RoleAssistant, Content: "hello there"},
			{Role: RoleAssistant, Content: "how can I help?"},
		},
	}
	if got := r.LastAssistantText(); got != "how can I help?" {
		t.Errorf("LastAssistantText() = %q, want %q", got, "how can I help?")
	}
}

func TestResultLastAssistantTextEmpty(t *testing.T) {
	r := &Result{
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
		},
	}
	if got := r.LastAssistantText(); got != "" {
		t.Errorf("LastAssistantText() = %q, want empty", got)
	}
}

func TestCostTotal(t *testing.T) {
	c := Cost{InputCostUSD: 0.01, OutputCostUSD: 0.03}
	if got := c.TotalUSD(); got != 0.04 {
		t.Errorf("TotalUSD() = %f, want 0.04", got)
	}
}

func TestTraceAddSpan(t *testing.T) {
	tr := NewTrace("agent-1")
	tr.AddSpan(Span{
		Name:      "llm.generate",
		StartTime: time.Now(),
		Duration:  500 * time.Millisecond,
	})
	if len(tr.Spans) != 1 {
		t.Errorf("len(Spans) = %d, want 1", len(tr.Spans))
	}
	if tr.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", tr.AgentID, "agent-1")
	}
}
