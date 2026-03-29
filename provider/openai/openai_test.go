package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
)

func TestGenerate_SimpleText(t *testing.T) {
	response := map[string]any{
		"id":     "chatcmpl-abc123",
		"object": "chat.completion",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello, world!",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := openai.New(
		openai.WithBaseURL(srv.URL),
		openai.WithAPIKey("test-key"),
		openai.WithModel("gpt-4o"),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Message.Content != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", result.Message.Content)
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("expected prompt_tokens=10, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("expected completion_tokens=5, got %d", result.Usage.CompletionTokens)
	}
}

func TestGenerate_ToolCall(t *testing.T) {
	response := map[string]any{
		"id":     "chatcmpl-tool123",
		"object": "chat.completion",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []map[string]any{
						{
							"id":   "call_abc",
							"type": "function",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": `{"location":"San Francisco"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     20,
			"completion_tokens": 15,
			"total_tokens":      35,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := openai.New(
		openai.WithBaseURL(srv.URL),
		openai.WithAPIKey("test-key"),
		openai.WithModel("gpt-4o"),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "What's the weather in San Francisco?"},
		},
		Tools: []graft.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get the current weather",
				Schema:      json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
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
	if tc.ID != "call_abc" {
		t.Errorf("expected tool call id 'call_abc', got %q", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got %q", tc.Name)
	}

	var args map[string]string
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["location"] != "San Francisco" {
		t.Errorf("expected location 'San Francisco', got %q", args["location"])
	}
}

func TestGenerate_CustomBaseURL(t *testing.T) {
	// Simulates OpenRouter or other compatible endpoint
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Verify custom header
		if r.Header.Get("HTTP-Referer") != "https://myapp.com" {
			t.Errorf("expected HTTP-Referer header, got %q", r.Header.Get("HTTP-Referer"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "chatcmpl-or123",
			"object": "chat.completion",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Response from OpenRouter",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 4,
				"total_tokens":      9,
			},
		})
	}))
	defer srv.Close()

	client := openai.New(
		openai.WithBaseURL(srv.URL),
		openai.WithAPIKey("openrouter-key"),
		openai.WithModel("anthropic/claude-3-opus"),
		openai.WithHeader("HTTP-Referer", "https://myapp.com"),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hello via OpenRouter"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !called {
		t.Error("handler was never called")
	}
	if result.Message.Content != "Response from OpenRouter" {
		t.Errorf("unexpected content: %q", result.Message.Content)
	}
}

func TestModelID(t *testing.T) {
	client := openai.New(openai.WithModel("gpt-3.5-turbo"))
	if client.ModelID() != "gpt-3.5-turbo" {
		t.Errorf("expected gpt-3.5-turbo, got %q", client.ModelID())
	}
}
