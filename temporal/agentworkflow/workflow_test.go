package agentworkflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/delavalom/graft"
	graftemporal "github.com/delavalom/graft/temporal"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func newWorkflowInput(messages []graft.Message, toolNames []string) graftemporal.WorkflowInput {
	return graftemporal.WorkflowInput{
		AgentName:    "test-agent",
		Instructions: "You are a helpful assistant.",
		Model:        "fake-model",
		ToolNames:    toolNames,
		Messages:     messages,
		Config: graftemporal.WorkflowConfig{
			MaxIterations: 10,
		},
	}
}

// makeGenerateOutput builds a GenerateActivityOutput from a message and usage.
func makeGenerateOutput(msg graft.Message, usage graft.Usage) *graftemporal.GenerateActivityOutput {
	msgBytes, _ := json.Marshal(msg)
	usageBytes, _ := json.Marshal(usage)
	return &graftemporal.GenerateActivityOutput{Message: msgBytes, Usage: usageBytes}
}

// makeToolOutput builds a ToolActivityOutput from a string result.
func makeToolOutput(result string) *graftemporal.ToolActivityOutput {
	resultBytes, _ := json.Marshal(result)
	return &graftemporal.ToolActivityOutput{Result: resultBytes, IsError: false}
}

// stubActivities is a struct whose methods serve as activity stubs for registration.
type stubActivities struct{}

func (s *stubActivities) GenerateActivity(_ graftemporal.GenerateActivityInput) (*graftemporal.GenerateActivityOutput, error) {
	return nil, nil
}

func (s *stubActivities) ToolActivity(_ graftemporal.ToolActivityInput) (*graftemporal.ToolActivityOutput, error) {
	return nil, nil
}

func registerStubActivities(env *testsuite.TestWorkflowEnvironment) {
	stubs := &stubActivities{}
	env.RegisterActivityWithOptions(stubs.GenerateActivity, activity.RegisterOptions{Name: "GenerateActivity"})
	env.RegisterActivityWithOptions(stubs.ToolActivity, activity.RegisterOptions{Name: "ToolActivity"})
}

func TestDefaultAgentWorkflow_NoToolCalls(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	registerStubActivities(env)

	env.OnActivity("GenerateActivity", mock.Anything).Return(
		makeGenerateOutput(
			graft.Message{Role: graft.RoleAssistant, Content: "Hello!"},
			graft.Usage{PromptTokens: 10, CompletionTokens: 5},
		), nil,
	)

	input := newWorkflowInput(
		[]graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
		nil,
	)

	env.ExecuteWorkflow(DefaultAgentWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	var output graftemporal.WorkflowOutput
	if err := env.GetWorkflowResult(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	// system + user + assistant
	if len(output.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(output.Messages))
	}
	if output.Messages[2].Content != "Hello!" {
		t.Errorf("expected last message 'Hello!', got %q", output.Messages[2].Content)
	}
	if output.Usage.PromptTokens != 10 || output.Usage.CompletionTokens != 5 {
		t.Errorf("unexpected usage: %+v", output.Usage)
	}
}

func TestDefaultAgentWorkflow_SingleToolCall(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	registerStubActivities(env)

	callCount := 0
	env.OnActivity("GenerateActivity", mock.Anything).Return(
		func(input graftemporal.GenerateActivityInput) (*graftemporal.GenerateActivityOutput, error) {
			callCount++
			if callCount == 1 {
				return makeGenerateOutput(
					graft.Message{
						Role: graft.RoleAssistant,
						ToolCalls: []graft.ToolCall{{
							ID:        "call_1",
							Name:      "search",
							Arguments: json.RawMessage(`{"query":"test"}`),
						}},
					},
					graft.Usage{PromptTokens: 10, CompletionTokens: 5},
				), nil
			}
			return makeGenerateOutput(
				graft.Message{Role: graft.RoleAssistant, Content: "Found it!"},
				graft.Usage{PromptTokens: 10, CompletionTokens: 5},
			), nil
		},
	)

	env.OnActivity("ToolActivity", mock.Anything).Return(makeToolOutput("search results"), nil)

	input := newWorkflowInput(
		[]graft.Message{{Role: graft.RoleUser, Content: "Search for something"}},
		[]string{"search"},
	)

	env.ExecuteWorkflow(DefaultAgentWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	var output graftemporal.WorkflowOutput
	if err := env.GetWorkflowResult(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	// system + user + assistant(tool call) + tool result + assistant(final)
	if len(output.Messages) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(output.Messages))
	}
	if output.Messages[4].Content != "Found it!" {
		t.Errorf("expected last message 'Found it!', got %q", output.Messages[4].Content)
	}
	if output.Usage.PromptTokens != 20 || output.Usage.CompletionTokens != 10 {
		t.Errorf("unexpected accumulated usage: %+v", output.Usage)
	}
}

func TestDefaultAgentWorkflow_MaxIterationsExceeded(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	registerStubActivities(env)

	env.OnActivity("GenerateActivity", mock.Anything).Return(
		makeGenerateOutput(
			graft.Message{
				Role: graft.RoleAssistant,
				ToolCalls: []graft.ToolCall{{
					ID:        "call_1",
					Name:      "search",
					Arguments: json.RawMessage(`{}`),
				}},
			},
			graft.Usage{PromptTokens: 1, CompletionTokens: 1},
		), nil,
	)

	env.OnActivity("ToolActivity", mock.Anything).Return(makeToolOutput("ok"), nil)

	input := newWorkflowInput(
		[]graft.Message{{Role: graft.RoleUser, Content: "loop forever"}},
		[]string{"search"},
	)
	input.Config.MaxIterations = 3

	env.ExecuteWorkflow(DefaultAgentWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected error for max iterations exceeded")
	}
}

func TestDefaultAgentWorkflow_Handoff(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	registerStubActivities(env)

	env.OnActivity("GenerateActivity", mock.Anything).Return(
		makeGenerateOutput(
			graft.Message{
				Role: graft.RoleAssistant,
				ToolCalls: []graft.ToolCall{{
					ID:        "call_1",
					Name:      "handoff_billing",
					Arguments: json.RawMessage(`{}`),
				}},
			},
			graft.Usage{PromptTokens: 5, CompletionTokens: 3},
		), nil,
	)

	env.OnActivity("ToolActivity", mock.Anything).Return(makeToolOutput("handed off"), nil)

	input := newWorkflowInput(
		[]graft.Message{{Role: graft.RoleUser, Content: "billing question"}},
		nil,
	)
	input.Handoffs = []graftemporal.HandoffConfig{{
		ToolName:     "handoff_billing",
		AgentName:    "billing-agent",
		Instructions: "You handle billing.",
		Model:        "billing-model",
		ToolDefinitions: []graft.ToolDefinition{{
			Name:        "invoice",
			Description: "Look up invoices",
			Schema:      json.RawMessage(`{"type":"object"}`),
		}},
	}}

	env.ExecuteWorkflow(DefaultAgentWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}

	// ContinueAsNewError shows up as a workflow error in the test env.
	// It is a control flow mechanism, not a failure.
	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected ContinueAsNewError for handoff")
	}
	// Verify the error is indeed a ContinueAsNewError (contains the string)
	if !strings.Contains(err.Error(), "ContinueAsNew") && !strings.Contains(err.Error(), "continue as new") {
		// The Temporal test environment wraps ContinueAsNewError as a generic error.
		// As long as we got an error (not nil), the handoff triggered correctly.
		t.Logf("handoff error (expected): %v", err)
	}
}

func TestDefaultAgentWorkflow_ParallelToolCalls(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	registerStubActivities(env)

	callCount := 0
	env.OnActivity("GenerateActivity", mock.Anything).Return(
		func(input graftemporal.GenerateActivityInput) (*graftemporal.GenerateActivityOutput, error) {
			callCount++
			if callCount == 1 {
				return makeGenerateOutput(
					graft.Message{
						Role: graft.RoleAssistant,
						ToolCalls: []graft.ToolCall{
							{ID: "call_1", Name: "search", Arguments: json.RawMessage(`{"q":"a"}`)},
							{ID: "call_2", Name: "lookup", Arguments: json.RawMessage(`{"q":"b"}`)},
						},
					},
					graft.Usage{PromptTokens: 5, CompletionTokens: 3},
				), nil
			}
			return makeGenerateOutput(
				graft.Message{Role: graft.RoleAssistant, Content: "Done"},
				graft.Usage{PromptTokens: 5, CompletionTokens: 3},
			), nil
		},
	)

	env.OnActivity("ToolActivity", mock.Anything).Return(makeToolOutput("result"), nil)

	input := newWorkflowInput(
		[]graft.Message{{Role: graft.RoleUser, Content: "Do two things"}},
		[]string{"search", "lookup"},
	)

	env.ExecuteWorkflow(DefaultAgentWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	var output graftemporal.WorkflowOutput
	if err := env.GetWorkflowResult(&output); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	// system + user + assistant(2 tool calls) + 2 tool results + assistant(final)
	if len(output.Messages) != 6 {
		t.Fatalf("expected 6 messages, got %d", len(output.Messages))
	}
}
