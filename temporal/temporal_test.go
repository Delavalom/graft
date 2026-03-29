package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/delavalom/graft"
)

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("expected max attempts 3, got %d", p.MaxAttempts)
	}
	if p.BackoffCoefficient != 2.0 {
		t.Errorf("expected backoff 2.0, got %f", p.BackoffCoefficient)
	}
}

func TestDefaultActivityConfig(t *testing.T) {
	cfg := DefaultActivityConfig()
	if cfg.StartToCloseTimeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", cfg.StartToCloseTimeout)
	}
	if cfg.RetryPolicy == nil {
		t.Fatal("expected retry policy")
	}
}

func TestRunnerOptions(t *testing.T) {
	client := &mockClient{}
	runner := NewTemporalRunner(client,
		WithTaskQueue("custom-queue"),
		WithWorkflowID("custom-id"),
		WithRetryPolicy(&RetryPolicy{MaxAttempts: 5}),
	)

	if runner.cfg.taskQueue != "custom-queue" {
		t.Errorf("expected task queue 'custom-queue', got %s", runner.cfg.taskQueue)
	}
	if runner.cfg.workflowID != "custom-id" {
		t.Errorf("expected workflow ID 'custom-id', got %s", runner.cfg.workflowID)
	}
	if runner.cfg.retryPolicy.MaxAttempts != 5 {
		t.Errorf("expected max attempts 5, got %d", runner.cfg.retryPolicy.MaxAttempts)
	}
}

// mockClient implements TemporalClient for testing.
type mockClient struct {
	executeErr error
	output     WorkflowOutput
}

func (m *mockClient) ExecuteWorkflow(_ context.Context, _ WorkflowOptions, _ string, _ ...any) (WorkflowRun, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return &mockWorkflowRun{output: m.output}, nil
}

func (m *mockClient) QueryWorkflow(_ context.Context, _, _, _ string, _ ...any) (EncodedValue, error) {
	return &mockEncodedValue{}, nil
}

func (m *mockClient) SignalWorkflow(_ context.Context, _, _, _ string, _ any) error {
	return nil
}

type mockWorkflowRun struct {
	output WorkflowOutput
}

func (r *mockWorkflowRun) GetID() string    { return "test-wf-id" }
func (r *mockWorkflowRun) GetRunID() string { return "test-run-id" }
func (r *mockWorkflowRun) Get(_ context.Context, valuePtr any) error {
	if ptr, ok := valuePtr.(*WorkflowOutput); ok {
		*ptr = r.output
	}
	return nil
}

type mockEncodedValue struct{}

func (v *mockEncodedValue) Get(valuePtr any) error {
	if ptr, ok := valuePtr.(*json.RawMessage); ok {
		*ptr = json.RawMessage(`{"status":"running"}`)
	}
	return nil
}

func TestRunnerRun(t *testing.T) {
	client := &mockClient{
		output: WorkflowOutput{
			Messages: []graft.Message{{Role: graft.RoleAssistant, Content: "done"}},
			Usage:    graft.Usage{PromptTokens: 10, CompletionTokens: 5},
		},
	}
	runner := NewTemporalRunner(client)
	agent := graft.NewAgent("test", graft.WithModel("gpt-4o"))
	msgs := []graft.Message{{Role: graft.RoleUser, Content: "hello"}}

	result, err := runner.Run(context.Background(), agent, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].Content != "done" {
		t.Errorf("expected 'done', got %s", result.Messages[0].Content)
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", result.Usage.PromptTokens)
	}
}

func TestRunnerRunError(t *testing.T) {
	client := &mockClient{executeErr: fmt.Errorf("temporal unavailable")}
	runner := NewTemporalRunner(client)
	agent := graft.NewAgent("test")

	_, err := runner.Run(context.Background(), agent, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunnerRunStream(t *testing.T) {
	client := &mockClient{
		output: WorkflowOutput{
			Messages: []graft.Message{{Role: graft.RoleAssistant, Content: "streamed"}},
		},
	}
	runner := NewTemporalRunner(client)
	agent := graft.NewAgent("test")

	ch, err := runner.RunStream(context.Background(), agent, nil)
	if err != nil {
		t.Fatal(err)
	}

	var events []graft.StreamEvent
	for e := range ch {
		events = append(events, e)
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	// Last event should be EventDone
	if events[len(events)-1].Type != graft.EventDone {
		t.Errorf("expected last event to be done, got %s", events[len(events)-1].Type)
	}
}

func TestActivityAsTool(t *testing.T) {
	executor := func(_ context.Context, name string, input any) (any, error) {
		return fmt.Sprintf("executed %s", name), nil
	}

	schema := json.RawMessage(`{"type":"object"}`)
	tool := ActivityAsTool("my-activity", "does stuff", schema, executor)

	if tool.Name() != "my-activity" {
		t.Errorf("expected name 'my-activity', got %s", tool.Name())
	}
	if tool.Description() != "does stuff" {
		t.Errorf("expected desc 'does stuff', got %s", tool.Description())
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if result != "executed my-activity" {
		t.Errorf("expected 'executed my-activity', got %v", result)
	}
}

func TestActivityAsToolError(t *testing.T) {
	executor := func(_ context.Context, _ string, _ any) (any, error) {
		return nil, fmt.Errorf("activity failed")
	}

	tool := ActivityAsTool("fail", "fails", json.RawMessage(`{}`), executor)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSignal(t *testing.T) {
	client := &mockClient{}
	runner := NewTemporalRunner(client)

	err := runner.Signal(context.Background(), "wf-id", SignalApprovalName, SignalApproval{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	client := &mockClient{}
	runner := NewTemporalRunner(client)

	result, err := runner.Query(context.Background(), "wf-id", QueryCurrentState)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestWorkflowInputSerialization(t *testing.T) {
	input := WorkflowInput{
		AgentName:    "test",
		Instructions: "do stuff",
		Model:        "gpt-4o",
		ToolNames:    []string{"search", "calc"},
		Messages:     []graft.Message{{Role: graft.RoleUser, Content: "hi"}},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	var parsed WorkflowInput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.AgentName != "test" {
		t.Errorf("expected agent name 'test', got %s", parsed.AgentName)
	}
	if len(parsed.ToolNames) != 2 {
		t.Errorf("expected 2 tool names, got %d", len(parsed.ToolNames))
	}
}

// Ensure TemporalRunner satisfies graft.Runner.
var _ graft.Runner = (*TemporalRunner)(nil)
