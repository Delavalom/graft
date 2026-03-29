package state_test

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/state"
)

// fakeModel is a minimal LanguageModel for testing.
type fakeModel struct {
	responses []graft.GenerateResult
	callIdx   int
}

func (f *fakeModel) ModelID() string { return "fake" }

func (f *fakeModel) Generate(_ context.Context, _ graft.GenerateParams) (*graft.GenerateResult, error) {
	if f.callIdx >= len(f.responses) {
		return &graft.GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "no more responses"},
		}, nil
	}
	r := f.responses[f.callIdx]
	f.callIdx++
	return &r, nil
}

func (f *fakeModel) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	ch := make(chan graft.StreamChunk)
	go func() {
		defer close(ch)
		result, err := f.Generate(ctx, params)
		if err != nil {
			return
		}
		if result.Message.Content != "" {
			ch <- graft.StreamChunk{Delta: graft.StreamEvent{Type: graft.EventTextDelta, Data: result.Message.Content}}
		}
		ch <- graft.StreamChunk{Delta: graft.StreamEvent{Type: graft.EventDone}, Usage: &result.Usage}
	}()
	return ch, nil
}

func TestSessionRunner_RunEmptySession(t *testing.T) {
	model := &fakeModel{
		responses: []graft.GenerateResult{
			{Message: graft.Message{Role: graft.RoleAssistant, Content: "Hello!"}},
		},
	}
	store := state.NewMemoryStore()
	sess := state.NewSession("test-agent")

	runner := state.NewSessionRunner(graft.NewDefaultRunner(model), store, sess.ID)
	agent := graft.NewAgent("test-agent")

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "Hello!" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Hello!")
	}

	// The session should be persisted with user + system + assistant messages.
	saved, err := store.Load(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Load after Run: %v", err)
	}
	if len(saved.Messages) == 0 {
		t.Error("expected saved messages, got 0")
	}
}

func TestSessionRunner_RunAppendsConversation(t *testing.T) {
	model := &fakeModel{
		responses: []graft.GenerateResult{
			{Message: graft.Message{Role: graft.RoleAssistant, Content: "First reply"}},
			{Message: graft.Message{Role: graft.RoleAssistant, Content: "Second reply"}},
		},
	}
	store := state.NewMemoryStore()
	sess := state.NewSession("agent")

	runner := state.NewSessionRunner(graft.NewDefaultRunner(model), store, sess.ID)
	agent := graft.NewAgent("agent")
	ctx := context.Background()

	// First exchange
	_, err := runner.Run(ctx, agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Turn 1"},
	})
	if err != nil {
		t.Fatalf("Run (1): %v", err)
	}

	// Second exchange — prior messages should be loaded and prepended
	result, err := runner.Run(ctx, agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Turn 2"},
	})
	if err != nil {
		t.Fatalf("Run (2): %v", err)
	}
	if result.LastAssistantText() != "Second reply" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Second reply")
	}

	saved, err := store.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Load after second Run: %v", err)
	}

	// Should contain messages from both exchanges.
	if len(saved.Messages) < 4 {
		t.Errorf("expected at least 4 saved messages (2 user + 2 assistant), got %d", len(saved.Messages))
	}
}

func TestSessionRunner_MultipleExchangesAccumulate(t *testing.T) {
	const exchanges = 3
	responses := make([]graft.GenerateResult, exchanges)
	for i := range responses {
		responses[i] = graft.GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "reply"},
		}
	}

	model := &fakeModel{responses: responses}
	store := state.NewMemoryStore()
	sess := state.NewSession("acc-agent")
	runner := state.NewSessionRunner(graft.NewDefaultRunner(model), store, sess.ID)
	agent := graft.NewAgent("acc-agent")
	ctx := context.Background()

	for i := 0; i < exchanges; i++ {
		_, err := runner.Run(ctx, agent, []graft.Message{
			{Role: graft.RoleUser, Content: "msg"},
		})
		if err != nil {
			t.Fatalf("Run[%d]: %v", i, err)
		}
	}

	saved, err := store.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Each exchange adds at least 1 user message + 1 assistant message.
	if len(saved.Messages) < exchanges*2 {
		t.Errorf("expected at least %d saved messages, got %d", exchanges*2, len(saved.Messages))
	}
}
