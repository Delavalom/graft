package graft

import (
	"encoding/json"
	"testing"
)

func TestRoleConstants(t *testing.T) {
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %q, want %q", RoleSystem, "system")
	}
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want %q", RoleUser, "user")
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", RoleAssistant, "assistant")
	}
	if RoleTool != "tool" {
		t.Errorf("RoleTool = %q, want %q", RoleTool, "tool")
	}
}

func TestMessageJSON(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "hello",
		Metadata: map[string]any{"source": "test"},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Role != RoleUser {
		t.Errorf("Role = %q, want %q", got.Role, RoleUser)
	}
	if got.Content != "hello" {
		t.Errorf("Content = %q, want %q", got.Content, "hello")
	}
}

func TestToolCallJSON(t *testing.T) {
	tc := ToolCall{
		ID:        "call_123",
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolCall
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "call_123" {
		t.Errorf("ID = %q, want %q", got.ID, "call_123")
	}
	if got.Name != "search" {
		t.Errorf("Name = %q, want %q", got.Name, "search")
	}
}

func TestToolResultJSON(t *testing.T) {
	tr := ToolResult{
		CallID:  "call_123",
		Content: "result data",
		IsError: false,
	}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CallID != "call_123" {
		t.Errorf("CallID = %q, want %q", got.CallID, "call_123")
	}
	if got.IsError != false {
		t.Error("IsError = true, want false")
	}
}
