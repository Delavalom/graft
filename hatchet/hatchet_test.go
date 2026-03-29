package hatchet

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/delavalom/graft"
)

// mockHatchetClient implements HatchetClient for testing.
type mockHatchetClient struct {
	taskResult any
	taskErr    error
	wfResult   any
	wfErr      error
}

func (m *mockHatchetClient) RunTask(_ context.Context, _ string, _ any) (TaskRun, error) {
	if m.taskErr != nil {
		return nil, m.taskErr
	}
	return &mockTaskRun{result: m.taskResult}, nil
}

func (m *mockHatchetClient) RunWorkflow(_ context.Context, _ string, _ any) (WorkflowRun, error) {
	if m.wfErr != nil {
		return nil, m.wfErr
	}
	return &mockWorkflowRun{result: m.wfResult}, nil
}

type mockTaskRun struct {
	result any
}

func (r *mockTaskRun) ID() string { return "task-123" }
func (r *mockTaskRun) Wait(_ context.Context) (any, error) {
	return r.result, nil
}

type mockWorkflowRun struct {
	result any
}

func (r *mockWorkflowRun) ID() string { return "wf-123" }
func (r *mockWorkflowRun) Wait(_ context.Context) (any, error) {
	return r.result, nil
}

func TestRunnerOptions(t *testing.T) {
	client := &mockHatchetClient{}
	runner := NewHatchetRunner(client,
		WithNamespace("custom"),
		WithConcurrency(5),
		WithRateLimit("api", 100),
	)
	if runner.cfg.namespace != "custom" {
		t.Errorf("expected namespace 'custom', got %s", runner.cfg.namespace)
	}
	if runner.cfg.concurrency != 5 {
		t.Errorf("expected concurrency 5, got %d", runner.cfg.concurrency)
	}
	if runner.cfg.rateLimits["api"] != 100 {
		t.Errorf("expected rate limit 100, got %d", runner.cfg.rateLimits["api"])
	}
}

func TestRunnerRun(t *testing.T) {
	output := TaskOutput{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: "result"}},
		Usage:    graft.Usage{PromptTokens: 5, CompletionTokens: 3},
	}
	client := &mockHatchetClient{wfResult: output}
	runner := NewHatchetRunner(client)
	agent := graft.NewAgent("test-agent")

	result, err := runner.Run(context.Background(), agent, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].Content != "result" {
		t.Errorf("expected 'result', got %s", result.Messages[0].Content)
	}
}

func TestRunnerRunError(t *testing.T) {
	client := &mockHatchetClient{wfErr: fmt.Errorf("hatchet down")}
	runner := NewHatchetRunner(client)
	agent := graft.NewAgent("test")

	_, err := runner.Run(context.Background(), agent, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunnerRunStream(t *testing.T) {
	output := TaskOutput{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: "streamed"}},
	}
	client := &mockHatchetClient{wfResult: output}
	runner := NewHatchetRunner(client)
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
	if events[len(events)-1].Type != graft.EventDone {
		t.Error("expected last event to be done")
	}
}

func TestTaskAsTool(t *testing.T) {
	client := &mockHatchetClient{taskResult: "task output"}
	schema := json.RawMessage(`{"type":"object"}`)
	tool := TaskAsTool("my-task", "does things", schema, client)

	if tool.Name() != "my-task" {
		t.Errorf("expected 'my-task', got %s", tool.Name())
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if result != "task output" {
		t.Errorf("expected 'task output', got %v", result)
	}
}

func TestTaskAsToolError(t *testing.T) {
	client := &mockHatchetClient{taskErr: fmt.Errorf("fail")}
	tool := TaskAsTool("fail", "fails", json.RawMessage(`{}`), client)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDAGWorkflow(t *testing.T) {
	dag := NewAgentDAG("test")
	if len(dag.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(dag.Steps))
	}
	if err := dag.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDAGValidationError(t *testing.T) {
	dag := &DAGWorkflow{
		Name: "bad",
		Steps: []DAGStep{
			{Name: "a", DependsOn: []string{"nonexistent"}},
		},
	}
	if err := dag.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRetryConfigNonRetryable(t *testing.T) {
	cfg := &RetryConfig{
		NonRetryable: []string{"auth_error", "not_found"},
	}
	if !cfg.IsNonRetryable("auth_error") {
		t.Error("expected auth_error to be non-retryable")
	}
	if cfg.IsNonRetryable("timeout") {
		t.Error("expected timeout to be retryable")
	}
}

// Ensure HatchetRunner satisfies graft.Runner.
var _ graft.Runner = (*HatchetRunner)(nil)
