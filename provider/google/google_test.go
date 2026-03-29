package google_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/google"
)

func TestGenerate_SimpleText(t *testing.T) {
	response := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"role": "model",
					"parts": []map[string]any{
						{"text": "Hello from Gemini!"},
					},
				},
				"finishReason": "STOP",
				"index":        0,
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     8,
			"candidatesTokenCount": 4,
			"totalTokenCount":      12,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Path should be /v1beta/models/{model}:generateContent
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Errorf("expected generateContent in path, got %s", r.URL.Path)
		}
		// API key should be in query param
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("expected key=test-key, got %q", r.URL.Query().Get("key"))
		}

		// Verify system instruction is extracted
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["systemInstruction"] == nil {
			t.Error("expected systemInstruction field")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := google.New(
		google.WithBaseURL(srv.URL),
		google.WithAPIKey("test-key"),
		google.WithModel("gemini-1.5-pro"),
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

	if result.Message.Content != "Hello from Gemini!" {
		t.Errorf("expected 'Hello from Gemini!', got %q", result.Message.Content)
	}
	if result.Usage.PromptTokens != 8 {
		t.Errorf("expected promptTokenCount=8, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 4 {
		t.Errorf("expected candidatesTokenCount=4, got %d", result.Usage.CompletionTokens)
	}
}

func TestGenerate_UserModelRoles(t *testing.T) {
	// Verify that assistant messages are sent as "model" role
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role":  "model",
						"parts": []map[string]any{{"text": "response"}},
					},
					"finishReason": "STOP",
					"index":        0,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     10,
				"candidatesTokenCount": 2,
				"totalTokenCount":      12,
			},
		})
	}))
	defer srv.Close()

	client := google.New(
		google.WithBaseURL(srv.URL),
		google.WithAPIKey("test-key"),
		google.WithModel("gemini-1.5-flash"),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hello"},
			{Role: graft.RoleAssistant, Content: "Hi there!"},
			{Role: graft.RoleUser, Content: "How are you?"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	contents, ok := capturedBody["contents"].([]any)
	if !ok {
		t.Fatalf("expected contents array, got %T", capturedBody["contents"])
	}
	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}

	// The assistant message should have role "model"
	assistantMsg, _ := contents[1].(map[string]any)
	if assistantMsg["role"] != "model" {
		t.Errorf("expected assistant role to be 'model', got %q", assistantMsg["role"])
	}
}

func TestModelID(t *testing.T) {
	client := google.New(google.WithModel("gemini-1.5-flash"))
	if client.ModelID() != "gemini-1.5-flash" {
		t.Errorf("expected gemini-1.5-flash, got %q", client.ModelID())
	}
}
