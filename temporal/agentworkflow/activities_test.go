package agentworkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/delavalom/graft"
	graftemporal "github.com/delavalom/graft/temporal"
	"go.temporal.io/sdk/testsuite"
)

// fakeModel implements graft.LanguageModel for testing.
type fakeModel struct {
	response graft.GenerateResult
	err      error
}

func (f *fakeModel) Generate(_ context.Context, _ graft.GenerateParams) (*graft.GenerateResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &f.response, nil
}

func (f *fakeModel) Stream(_ context.Context, _ graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeModel) ModelID() string { return "fake-model" }

// fakeTool implements graft.Tool for testing.
type fakeTool struct {
	name   string
	result any
	err    error
}

func (f *fakeTool) Name() string                { return f.name }
func (f *fakeTool) Description() string          { return "test tool" }
func (f *fakeTool) Schema() json.RawMessage      { return json.RawMessage(`{"type":"object"}`) }
func (f *fakeTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestGenerateActivity_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	model := &fakeModel{
		response: graft.GenerateResult{
			Message: graft.Message{
				Role:    graft.RoleAssistant,
				Content: "Hello!",
			},
			Usage: graft.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
			},
		},
	}

	models := NewSingleModelProvider(model)
	tools := NewToolRegistry()
	a := NewActivities(models, tools)
	env.RegisterActivity(a.GenerateActivity)

	msgs, _ := json.Marshal([]graft.Message{{Role: graft.RoleUser, Content: "Hi"}})
	toolDefs, _ := json.Marshal([]graft.ToolDefinition{})

	result, err := env.ExecuteActivity(a.GenerateActivity, graftemporal.GenerateActivityInput{
		Model:    "fake-model",
		Messages: msgs,
		Tools:    toolDefs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output graftemporal.GenerateActivityOutput
	if err := result.Get(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	var msg graft.Message
	if err := json.Unmarshal(output.Message, &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	if msg.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", msg.Content)
	}

	var usage graft.Usage
	if err := json.Unmarshal(output.Usage, &usage); err != nil {
		t.Fatalf("failed to unmarshal usage: %v", err)
	}
	if usage.PromptTokens != 10 || usage.CompletionTokens != 5 {
		t.Errorf("unexpected usage: %+v", usage)
	}
}

func TestGenerateActivity_ModelError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	model := &fakeModel{err: fmt.Errorf("rate limited")}
	models := NewSingleModelProvider(model)
	tools := NewToolRegistry()
	a := NewActivities(models, tools)
	env.RegisterActivity(a.GenerateActivity)

	msgs, _ := json.Marshal([]graft.Message{{Role: graft.RoleUser, Content: "Hi"}})

	_, err := env.ExecuteActivity(a.GenerateActivity, graftemporal.GenerateActivityInput{
		Model:    "fake-model",
		Messages: msgs,
		Tools:    []byte(`[]`),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToolActivity_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	tools := NewToolRegistry()
	tools.Register(&fakeTool{name: "search", result: "42 results found"})
	a := NewActivities(nil, tools)
	env.RegisterActivity(a.ToolActivity)

	result, err := env.ExecuteActivity(a.ToolActivity, graftemporal.ToolActivityInput{
		ToolName:  "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output graftemporal.ToolActivityOutput
	if err := result.Get(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}
	if output.IsError {
		t.Error("expected IsError=false")
	}

	var content string
	if err := json.Unmarshal(output.Result, &content); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if content != "42 results found" {
		t.Errorf("expected '42 results found', got %q", content)
	}
}

func TestToolActivity_UnknownTool(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	tools := NewToolRegistry()
	a := NewActivities(nil, tools)
	env.RegisterActivity(a.ToolActivity)

	result, err := env.ExecuteActivity(a.ToolActivity, graftemporal.ToolActivityInput{
		ToolName:  "nonexistent",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output graftemporal.ToolActivityOutput
	if err := result.Get(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}
	if !output.IsError {
		t.Error("expected IsError=true for unknown tool")
	}
}

func TestToolActivity_ToolError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	tools := NewToolRegistry()
	tools.Register(&fakeTool{name: "fail", err: fmt.Errorf("tool failed")})
	a := NewActivities(nil, tools)
	env.RegisterActivity(a.ToolActivity)

	result, err := env.ExecuteActivity(a.ToolActivity, graftemporal.ToolActivityInput{
		ToolName:  "fail",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output graftemporal.ToolActivityOutput
	if err := result.Get(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}
	if !output.IsError {
		t.Error("expected IsError=true for tool error")
	}
}
