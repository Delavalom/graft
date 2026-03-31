package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// --- SigV4 Tests ---

func TestSignRequest_SetsRequiredHeaders(t *testing.T) {
	body := []byte(`{"messages":[]}`)
	req, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-3-5-sonnet-20241022-v2:0/converse", nil)
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	signRequest(req, body, creds, "us-east-1")

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("expected AWS4-HMAC-SHA256 prefix, got %s", auth[:30])
	}
	if !strings.Contains(auth, "Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Error("expected access key in credential")
	}
	if !strings.Contains(auth, "/us-east-1/bedrock/aws4_request") {
		t.Error("expected region/service in credential scope")
	}
	if !strings.Contains(auth, "SignedHeaders=") {
		t.Error("expected SignedHeaders in auth header")
	}

	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate == "" {
		t.Fatal("expected X-Amz-Date header")
	}
	if len(amzDate) != 16 || amzDate[8] != 'T' || amzDate[15] != 'Z' {
		t.Errorf("expected ISO 8601 basic format, got %s", amzDate)
	}
}

func TestSignRequest_WithSessionToken(t *testing.T) {
	body := []byte(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey:    "AKID",
		secretKey:    "SECRET",
		sessionToken: "SESSION_TOKEN_VALUE",
	}
	signRequest(req, body, creds, "us-east-1")

	token := req.Header.Get("X-Amz-Security-Token")
	if token != "SESSION_TOKEN_VALUE" {
		t.Errorf("expected session token header, got %q", token)
	}

	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "x-amz-security-token") {
		t.Error("expected x-amz-security-token in signed headers")
	}
}

func TestSignRequest_DeterministicSignature(t *testing.T) {
	body := []byte(`{"test": true}`)
	creds := credentials{accessKey: "AK", secretKey: "SK"}

	req1, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req1.Header.Set("Content-Type", "application/json")
	signRequest(req1, body, creds, "us-east-1")

	req2, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Amz-Date", req1.Header.Get("X-Amz-Date"))
	signRequest(req2, body, creds, "us-east-1")

	if req1.Header.Get("Authorization") != req2.Header.Get("Authorization") {
		t.Error("same inputs should produce same signature")
	}
}

// --- Event Stream Tests ---

func TestEventStreamDecoder_SingleEvent(t *testing.T) {
	payload := []byte(`{"role":"assistant"}`)
	frame := encodeEventStreamFrame("messageStart", payload)

	decoder := newEventStreamDecoder(bytes.NewReader(frame))
	eventType, data, err := decoder.readEvent()
	if err != nil {
		t.Fatalf("readEvent failed: %v", err)
	}
	if eventType != "messageStart" {
		t.Errorf("expected messageStart, got %s", eventType)
	}
	if string(data) != string(payload) {
		t.Errorf("expected %s, got %s", payload, data)
	}
}

func TestEventStreamDecoder_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(encodeEventStreamFrame("messageStart", []byte(`{"role":"assistant"}`)))
	buf.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"delta":{"text":"Hello"}}`)))
	buf.Write(encodeEventStreamFrame("messageStop", []byte(`{"stopReason":"end_turn"}`)))

	decoder := newEventStreamDecoder(&buf)

	var events []string
	for {
		eventType, _, err := decoder.readEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("readEvent failed: %v", err)
		}
		events = append(events, eventType)
	}

	expected := []string{"messageStart", "contentBlockDelta", "messageStop"}
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d", len(expected), len(events))
	}
	for i, e := range expected {
		if events[i] != e {
			t.Errorf("event %d: expected %s, got %s", i, e, events[i])
		}
	}
}

func TestEventStreamDecoder_EmptyPayload(t *testing.T) {
	frame := encodeEventStreamFrame("contentBlockStop", []byte(`{}`))
	decoder := newEventStreamDecoder(bytes.NewReader(frame))
	eventType, data, err := decoder.readEvent()
	if err != nil {
		t.Fatalf("readEvent failed: %v", err)
	}
	if eventType != "contentBlockStop" {
		t.Errorf("expected contentBlockStop, got %s", eventType)
	}
	if string(data) != "{}" {
		t.Errorf("expected {}, got %s", data)
	}
}

// --- Client and Generate Tests ---

func TestGenerate_SimpleText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := converseResponse{
			Output: converseOutput{
				Message: &converseMessage{
					Role:    "assistant",
					Content: []contentBlock{{Text: "Hello from Bedrock!"}},
				},
			},
			StopReason: "end_turn",
			Usage: converseUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"),
		WithAnonymousAuth(),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Message.Content != "Hello from Bedrock!" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "Hello from Bedrock!")
	}
	if result.Message.Role != graft.RoleAssistant {
		t.Errorf("Role = %q, want %q", result.Message.Role, graft.RoleAssistant)
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", result.Usage.CompletionTokens)
	}
}

func TestGenerate_WithToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := converseResponse{
			Output: converseOutput{
				Message: &converseMessage{
					Role: "assistant",
					Content: []contentBlock{
						{Text: "Let me check the weather."},
						{ToolUse: &toolUseBlock{
							ToolUseID: "tooluse_abc123",
							Name:      "get_weather",
							Input:     json.RawMessage(`{"city":"Seattle"}`),
						}},
					},
				},
			},
			StopReason: "tool_use",
			Usage: converseUsage{
				InputTokens:  15,
				OutputTokens: 20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"),
		WithAnonymousAuth(),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "What's the weather in Seattle?"}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Message.Content != "Let me check the weather." {
		t.Errorf("Content = %q, want %q", result.Message.Content, "Let me check the weather.")
	}
	if len(result.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.Message.ToolCalls))
	}

	tc := result.Message.ToolCalls[0]
	if tc.ID != "tooluse_abc123" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "tooluse_abc123")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if string(tc.Arguments) != `{"city":"Seattle"}` {
		t.Errorf("ToolCall.Arguments = %s, want %s", tc.Arguments, `{"city":"Seattle"}`)
	}
}

func TestGenerate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Access denied"}`))
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"),
		WithAnonymousAuth(),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention status code 403, got: %v", err)
	}
}

func TestGenerate_WithSystemPrompt(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		resp := converseResponse{
			Output: converseOutput{
				Message: &converseMessage{
					Role:    "assistant",
					Content: []contentBlock{{Text: "OK"}},
				},
			},
			Usage: converseUsage{InputTokens: 1, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("test-model"),
		WithAnonymousAuth(),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleSystem, Content: "You are a helpful assistant."},
			{Role: graft.RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse the captured request body.
	var reqBody converseRequest
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	// System should be in the top-level system field.
	if len(reqBody.System) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(reqBody.System))
	}
	if reqBody.System[0].Text != "You are a helpful assistant." {
		t.Errorf("system text = %q, want %q", reqBody.System[0].Text, "You are a helpful assistant.")
	}

	// System should NOT be in messages.
	for _, msg := range reqBody.Messages {
		if msg.Role == "system" {
			t.Error("system message should not appear in messages array")
		}
	}
}

func TestGenerate_SigV4Auth(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		resp := converseResponse{
			Output: converseOutput{
				Message: &converseMessage{
					Role:    "assistant",
					Content: []contentBlock{{Text: "OK"}},
				},
			},
			Usage: converseUsage{InputTokens: 1, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("test-model"),
		WithRegion("us-east-1"),
		WithCredentials("AKID_TEST", "SECRET_TEST"),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if capturedAuth == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(capturedAuth, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization should start with AWS4-HMAC-SHA256, got %q", capturedAuth)
	}
}

func TestModelID(t *testing.T) {
	client := New(
		WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"),
		WithAnonymousAuth(),
	)
	if got := client.ModelID(); got != "anthropic.claude-3-5-sonnet-20241022-v2:0" {
		t.Errorf("ModelID() = %q, want %q", got, "anthropic.claude-3-5-sonnet-20241022-v2:0")
	}
}

func TestNew_EnvironmentVariables(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ENV_AKID")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")
	t.Setenv("AWS_SESSION_TOKEN", "ENV_SESSION")
	t.Setenv("AWS_REGION", "eu-west-1")

	client := New(WithModel("test-model"))

	if client.creds.accessKey != "ENV_AKID" {
		t.Errorf("accessKey = %q, want %q", client.creds.accessKey, "ENV_AKID")
	}
	if client.creds.secretKey != "ENV_SECRET" {
		t.Errorf("secretKey = %q, want %q", client.creds.secretKey, "ENV_SECRET")
	}
	if client.creds.sessionToken != "ENV_SESSION" {
		t.Errorf("sessionToken = %q, want %q", client.creds.sessionToken, "ENV_SESSION")
	}
	if client.region != "eu-west-1" {
		t.Errorf("region = %q, want %q", client.region, "eu-west-1")
	}
	if client.baseURL != "https://bedrock-runtime.eu-west-1.amazonaws.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://bedrock-runtime.eu-west-1.amazonaws.com")
	}
}

func TestNew_WithCustomHeaders(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-Custom-Header")
		resp := converseResponse{
			Output: converseOutput{
				Message: &converseMessage{
					Role:    "assistant",
					Content: []contentBlock{{Text: "OK"}},
				},
			},
			Usage: converseUsage{InputTokens: 1, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithModel("test-model"),
		WithAnonymousAuth(),
		WithHeader("X-Custom-Header", "custom-value"),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if capturedHeader != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", capturedHeader, "custom-value")
	}
}
