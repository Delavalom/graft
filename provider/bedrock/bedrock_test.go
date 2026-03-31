package bedrock

import (
	"encoding/json"
	"testing"

	"github.com/delavalom/graft"
)

func TestConvertMessages_SystemExtracted(t *testing.T) {
	msgs := []graft.Message{
		{Role: graft.RoleSystem, Content: "You are helpful."},
		{Role: graft.RoleUser, Content: "Hello"},
		{Role: graft.RoleSystem, Content: "Be concise."},
	}

	system, out := convertMessages(msgs)

	if len(system) != 2 {
		t.Fatalf("expected 2 system blocks, got %d", len(system))
	}
	if system[0].Text != "You are helpful." {
		t.Errorf("system[0].Text = %q, want %q", system[0].Text, "You are helpful.")
	}
	if system[1].Text != "Be concise." {
		t.Errorf("system[1].Text = %q, want %q", system[1].Text, "Be concise.")
	}

	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
	if out[0].Role != "user" {
		t.Errorf("out[0].Role = %q, want %q", out[0].Role, "user")
	}
	if out[0].Content[0].Text != "Hello" {
		t.Errorf("out[0].Content[0].Text = %q, want %q", out[0].Content[0].Text, "Hello")
	}
}

func TestConvertMessages_AssistantWithToolCalls(t *testing.T) {
	msgs := []graft.Message{
		{
			Role:    graft.RoleAssistant,
			Content: "Let me search for that.",
			ToolCalls: []graft.ToolCall{
				{
					ID:        "call_123",
					Name:      "search",
					Arguments: json.RawMessage(`{"query":"golang"}`),
				},
			},
		},
	}

	_, out := convertMessages(msgs)

	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
	msg := out[0]
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(msg.Content))
	}

	// First block: text
	if msg.Content[0].Text != "Let me search for that." {
		t.Errorf("text block = %q, want %q", msg.Content[0].Text, "Let me search for that.")
	}

	// Second block: toolUse
	tu := msg.Content[1].ToolUse
	if tu == nil {
		t.Fatal("expected toolUse block, got nil")
	}
	if tu.ToolUseID != "call_123" {
		t.Errorf("toolUseId = %q, want %q", tu.ToolUseID, "call_123")
	}
	if tu.Name != "search" {
		t.Errorf("name = %q, want %q", tu.Name, "search")
	}
	if string(tu.Input) != `{"query":"golang"}` {
		t.Errorf("input = %s, want %s", tu.Input, `{"query":"golang"}`)
	}
}

func TestConvertMessages_ToolResult(t *testing.T) {
	msgs := []graft.Message{
		{
			Role: graft.RoleTool,
			ToolResult: &graft.ToolResult{
				CallID:  "call_123",
				Content: "search results here",
				IsError: false,
			},
		},
	}

	_, out := convertMessages(msgs)

	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
	msg := out[0]
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}

	tr := msg.Content[0].ToolResult
	if tr == nil {
		t.Fatal("expected toolResult block, got nil")
	}
	if tr.ToolUseID != "call_123" {
		t.Errorf("toolUseId = %q, want %q", tr.ToolUseID, "call_123")
	}
	if tr.Status != "success" {
		t.Errorf("status = %q, want %q", tr.Status, "success")
	}
	if len(tr.Content) != 1 || tr.Content[0].Text != "search results here" {
		t.Errorf("content text = %q, want %q", tr.Content[0].Text, "search results here")
	}
}

func TestConvertMessages_ToolResultError(t *testing.T) {
	msgs := []graft.Message{
		{
			Role: graft.RoleTool,
			ToolResult: &graft.ToolResult{
				CallID:  "call_456",
				Content: "something went wrong",
				IsError: true,
			},
		},
	}

	_, out := convertMessages(msgs)

	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}

	tr := out[0].Content[0].ToolResult
	if tr == nil {
		t.Fatal("expected toolResult block, got nil")
	}
	if tr.Status != "error" {
		t.Errorf("status = %q, want %q", tr.Status, "error")
	}
	if tr.ToolUseID != "call_456" {
		t.Errorf("toolUseId = %q, want %q", tr.ToolUseID, "call_456")
	}
}

func TestConvertTools(t *testing.T) {
	tools := []graft.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			Schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
	}

	out := convertTools(tools)

	if len(out) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out))
	}

	spec := out[0].ToolSpec
	if spec.Name != "search" {
		t.Errorf("name = %q, want %q", spec.Name, "search")
	}
	if spec.Description != "Search the web" {
		t.Errorf("description = %q, want %q", spec.Description, "Search the web")
	}

	// inputSchema.json should wrap the original schema
	expectedSchema := `{"type":"object","properties":{"query":{"type":"string"}}}`
	if string(spec.InputSchema.JSON) != expectedSchema {
		t.Errorf("inputSchema.json = %s, want %s", spec.InputSchema.JSON, expectedSchema)
	}
}

func TestConvertTools_Empty(t *testing.T) {
	out := convertTools(nil)
	if out != nil {
		t.Errorf("expected nil, got %v", out)
	}

	out = convertTools([]graft.ToolDefinition{})
	if out != nil {
		t.Errorf("expected nil for empty slice, got %v", out)
	}
}
