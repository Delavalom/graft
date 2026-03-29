package graft

import (
	"context"
	"testing"
)

func TestNewAgentDefaults(t *testing.T) {
	agent := NewAgent("test-agent")
	if agent.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", agent.Name, "test-agent")
	}
	if agent.ToolChoice != ToolChoiceAuto {
		t.Errorf("ToolChoice = %v, want %v", agent.ToolChoice, ToolChoiceAuto)
	}
	if agent.Instructions != "" {
		t.Errorf("Instructions = %q, want empty", agent.Instructions)
	}
}

func TestNewAgentWithOptions(t *testing.T) {
	temp := 0.7
	maxTok := 1000
	tool := NewTool("test", "test tool",
		func(ctx context.Context, p struct{}) (string, error) {
			return "", nil
		},
	)

	agent := NewAgent("my-agent",
		WithInstructions("Be helpful."),
		WithModel("openai/gpt-4o"),
		WithTemperature(temp),
		WithMaxTokens(maxTok),
		WithToolChoice(ToolChoiceRequired),
		WithTools(tool),
		WithMetadata(map[string]any{"env": "test"}),
	)

	if agent.Instructions != "Be helpful." {
		t.Errorf("Instructions = %q, want %q", agent.Instructions, "Be helpful.")
	}
	if agent.Model != "openai/gpt-4o" {
		t.Errorf("Model = %q, want %q", agent.Model, "openai/gpt-4o")
	}
	if *agent.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", *agent.Temperature)
	}
	if *agent.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %v, want 1000", *agent.MaxTokens)
	}
	if agent.ToolChoice != ToolChoiceRequired {
		t.Errorf("ToolChoice = %v, want %v", agent.ToolChoice, ToolChoiceRequired)
	}
	if len(agent.Tools) != 1 {
		t.Errorf("len(Tools) = %d, want 1", len(agent.Tools))
	}
	if agent.Metadata["env"] != "test" {
		t.Errorf("Metadata[env] = %v, want test", agent.Metadata["env"])
	}
}

func TestToolChoiceSpecific(t *testing.T) {
	tc := ToolChoiceSpecific("search")
	if tc != ToolChoice("specific:search") {
		t.Errorf("ToolChoiceSpecific = %q, want %q", tc, "specific:search")
	}
}
