package graft

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeModel struct {
	responses []GenerateResult
	callIdx   int
}

func (f *fakeModel) ModelID() string { return "fake" }

func (f *fakeModel) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	if f.callIdx >= len(f.responses) {
		return &GenerateResult{
			Message: Message{Role: RoleAssistant, Content: "no more responses"},
		}, nil
	}
	r := f.responses[f.callIdx]
	f.callIdx++
	return &r, nil
}

func (f *fakeModel) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		result, err := f.Generate(ctx, params)
		if err != nil {
			return
		}
		if result.Message.Content != "" {
			ch <- StreamChunk{Delta: StreamEvent{Type: EventTextDelta, Data: result.Message.Content}}
		}
		ch <- StreamChunk{Delta: StreamEvent{Type: EventDone}, Usage: &result.Usage}
	}()
	return ch, nil
}

func TestRunnerSimpleResponse(t *testing.T) {
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "Hello!"}},
		},
	}
	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "Hello!" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Hello!")
	}
}

func TestRunnerToolCall(t *testing.T) {
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{{ID: "call_1", Name: "greet", Arguments: json.RawMessage(`{"Name":"Alice"}`)}},
			}},
			{Message: Message{Role: RoleAssistant, Content: "I greeted Alice for you."}},
		},
	}
	greetTool := NewTool("greet", "Greet someone",
		func(ctx context.Context, p struct{ Name string }) (string, error) {
			return "Hello, " + p.Name + "!", nil
		},
	)
	agent := NewAgent("test", WithTools(greetTool))
	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "Greet Alice"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "I greeted Alice for you." {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "I greeted Alice for you.")
	}
	if len(result.Messages) != 4 {
		t.Errorf("len(Messages) = %d, want 4", len(result.Messages))
	}
}

func TestRunnerMaxIterations(t *testing.T) {
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c2", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
			{Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c3", Name: "noop", Arguments: json.RawMessage(`{}`)}}}},
		},
	}
	noopTool := NewTool("noop", "No-op",
		func(ctx context.Context, p struct{}) (string, error) { return "done", nil },
	)
	agent := NewAgent("test", WithTools(noopTool))
	runner := NewDefaultRunner(model)
	_, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "loop forever"},
	}, WithMaxIterations(2))
	if err == nil {
		t.Fatal("expected error for max iterations exceeded")
	}
}

func TestRunnerStream(t *testing.T) {
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "streamed!"}},
		},
	}
	runner := NewDefaultRunner(model)
	events, err := runner.RunStream(context.Background(), NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	var gotText string
	for ev := range events {
		if ev.Type == EventTextDelta {
			gotText += ev.Data.(string)
		}
	}
	if gotText != "streamed!" {
		t.Errorf("streamed text = %q, want %q", gotText, "streamed!")
	}
}

func TestRunnerHandoff(t *testing.T) {
	targetAgent := NewAgent("specialist", WithInstructions("You are a specialist."))
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{{ID: "call_h", Name: "handoff_specialist", Arguments: json.RawMessage(`{}`)}},
			}},
			{Message: Message{Role: RoleAssistant, Content: "Specialist here!"}},
		},
	}
	agent := NewAgent("generalist",
		WithHandoffs(Handoff{Target: targetAgent, Description: "Transfer to specialist"}),
	)
	runner := NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []Message{
		{Role: RoleUser, Content: "I need a specialist"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "Specialist here!" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Specialist here!")
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	model := &fakeModel{
		responses: []GenerateResult{
			{Message: Message{Role: RoleAssistant, Content: "ok"}},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := NewDefaultRunner(model)
	_, err := runner.Run(ctx, NewAgent("test"), []Message{
		{Role: RoleUser, Content: "Hi"},
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
