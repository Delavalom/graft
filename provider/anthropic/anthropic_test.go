package anthropic_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/anthropic"
)

func TestGenerate_SimpleText(t *testing.T) {
	response := map[string]any{
		"id":   "msg_abc123",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{"type": "text", "text": "Hello from Claude!"},
		},
		"model":       "claude-3-5-sonnet-20241022",
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  12,
			"output_tokens": 4,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected /messages, got %s", r.URL.Path)
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header '2023-06-01', got %q", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header 'test-key', got %q", r.Header.Get("x-api-key"))
		}

		// Verify system message is extracted
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["system"] != "You are a helpful assistant." {
			t.Errorf("expected system field, got %v", body["system"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := anthropic.New(
		anthropic.WithBaseURL(srv.URL),
		anthropic.WithAPIKey("test-key"),
		anthropic.WithModel("claude-3-5-sonnet-20241022"),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleSystem, Content: "You are a helpful assistant."},
			{Role: graft.RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Message.Content != "Hello from Claude!" {
		t.Errorf("expected 'Hello from Claude!', got %q", result.Message.Content)
	}
	if result.Usage.PromptTokens != 12 {
		t.Errorf("expected input_tokens=12, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 4 {
		t.Errorf("expected output_tokens=4, got %d", result.Usage.CompletionTokens)
	}
}

func TestGenerate_ToolUse(t *testing.T) {
	toolInput := json.RawMessage(`{"city":"New York"}`)
	response := map[string]any{
		"id":   "msg_tool123",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{
				"type":  "tool_use",
				"id":    "toolu_abc",
				"name":  "get_weather",
				"input": json.RawMessage(toolInput),
			},
		},
		"model":       "claude-3-5-sonnet-20241022",
		"stop_reason": "tool_use",
		"usage": map[string]any{
			"input_tokens":  25,
			"output_tokens": 10,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := anthropic.New(
		anthropic.WithBaseURL(srv.URL),
		anthropic.WithAPIKey("test-key"),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "What's the weather in New York?"},
		},
		Tools: []graft.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get weather for a city",
				Schema:      json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(result.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.Message.ToolCalls))
	}

	tc := result.Message.ToolCalls[0]
	if tc.ID != "toolu_abc" {
		t.Errorf("expected tool call id 'toolu_abc', got %q", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got %q", tc.Name)
	}

	var args map[string]string
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["city"] != "New York" {
		t.Errorf("expected city 'New York', got %q", args["city"])
	}
}

func TestModelID(t *testing.T) {
	client := anthropic.New(anthropic.WithModel("claude-3-haiku-20240307"))
	if client.ModelID() != "claude-3-haiku-20240307" {
		t.Errorf("expected claude-3-haiku-20240307, got %q", client.ModelID())
	}
}
