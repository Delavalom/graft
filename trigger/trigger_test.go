package trigger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunnerOptions(t *testing.T) {
	runner := NewTriggerRunner("http://localhost",
		WithAPIKey("test-key"),
		WithProjectID("proj-123"),
		WithEnvironment("prod"),
		WithPollInterval(5*time.Second),
	)
	if runner.cfg.apiKey != "test-key" {
		t.Errorf("expected api key 'test-key', got %s", runner.cfg.apiKey)
	}
	if runner.cfg.projectID != "proj-123" {
		t.Errorf("expected project ID 'proj-123', got %s", runner.cfg.projectID)
	}
	if runner.cfg.environment != "prod" {
		t.Errorf("expected environment 'prod', got %s", runner.cfg.environment)
	}
	if runner.cfg.pollInterval != 5*time.Second {
		t.Errorf("expected poll interval 5s, got %v", runner.cfg.pollInterval)
	}
}

func TestClientTriggerTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks/trigger" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong authorization header")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "run-123"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	runID, err := client.TriggerTask(context.Background(), "my-task", map[string]string{"input": "data"})
	if err != nil {
		t.Fatal(err)
	}
	if runID != "run-123" {
		t.Errorf("expected run-123, got %s", runID)
	}
}

func TestClientGetRunStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/run-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(RunStatus{
			ID:     "run-123",
			Status: "COMPLETED",
			Output: map[string]string{"result": "done"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	status, err := client.GetRunStatus(context.Background(), "run-123")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", status.Status)
	}
}

func TestClientSendWaitpointToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/waitpoints/complete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["token"] != "tok-123" {
			t.Errorf("expected token tok-123, got %v", body["token"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	err := client.SendWaitpointToken(context.Background(), "tok-123", ApprovalData{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSSEParsing(t *testing.T) {
	sseData := "event: OUTPUT\ndata: {\"text\":\"hello\"}\n\nevent: STATUS_UPDATE\ndata: {\"status\":\"done\"}\n\n"

	ch := make(chan RunEvent, 10)
	ctx := context.Background()
	go func() {
		parseSSE(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	var events []RunEvent
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "OUTPUT" {
		t.Errorf("expected type OUTPUT, got %s", events[0].Type)
	}
	if events[1].Type != "STATUS_UPDATE" {
		t.Errorf("expected type STATUS_UPDATE, got %s", events[1].Type)
	}
}

func TestClientSubscribeToRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		w.Write([]byte("event: OUTPUT\ndata: {\"text\":\"hi\"}\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.SubscribeToRun(ctx, "run-123")
	if err != nil {
		t.Fatal(err)
	}

	event := <-ch
	if event.Type != "OUTPUT" {
		t.Errorf("expected OUTPUT, got %s", event.Type)
	}
}

func TestWaitpoint(t *testing.T) {
	wp := NewApprovalWaitpoint("Approve this action")
	if wp.Type != WaitpointApproval {
		t.Errorf("expected approval type, got %s", wp.Type)
	}
	if wp.Description != "Approve this action" {
		t.Errorf("unexpected description: %s", wp.Description)
	}
}

func TestMapEventType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"OUTPUT", "text_delta"},
		{"ERROR", "error"},
		{"STATUS_UPDATE", "message_done"},
		{"UNKNOWN", "text_delta"},
	}
	for _, tt := range tests {
		result := mapEventType(tt.input)
		if string(result) != tt.expected {
			t.Errorf("mapEventType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	cfg := DefaultStreamConfig()
	if cfg.BufferSize != 64 {
		t.Errorf("expected buffer size 64, got %d", cfg.BufferSize)
	}
	if cfg.MaxReconnects != 5 {
		t.Errorf("expected max reconnects 5, got %d", cfg.MaxReconnects)
	}
}

func TestClientHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.TriggerTask(context.Background(), "task", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
