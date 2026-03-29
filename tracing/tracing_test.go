package tracing

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
)

// recordingProvider captures span events for testing.
type recordingProvider struct {
	runs       []RunInfo
	gens       []GenerationInfo
	tools      []ToolCallInfo
	flushed    int
	runEnded   int
	genEnded   int
	toolEnded  int
}

func (p *recordingProvider) StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan) {
	p.runs = append(p.runs, info)
	return ctx, &recordingRunSpan{provider: p}
}

func (p *recordingProvider) StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan) {
	p.gens = append(p.gens, info)
	return ctx, &recordingGenSpan{provider: p}
}

func (p *recordingProvider) StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan) {
	p.tools = append(p.tools, info)
	return ctx, &recordingToolSpan{provider: p}
}

func (p *recordingProvider) Flush(_ context.Context) error {
	p.flushed++
	return nil
}

type recordingRunSpan struct{ provider *recordingProvider }

func (s *recordingRunSpan) End(error)                  { s.provider.runEnded++ }
func (s *recordingRunSpan) SetMetadata(map[string]any) {}
func (s *recordingRunSpan) SetUsage(graft.Usage)       {}

type recordingGenSpan struct{ provider *recordingProvider }

func (s *recordingGenSpan) End(error)                  { s.provider.genEnded++ }
func (s *recordingGenSpan) SetMetadata(map[string]any) {}
func (s *recordingGenSpan) SetUsage(graft.Usage)       {}
func (s *recordingGenSpan) SetModel(string)            {}

type recordingToolSpan struct{ provider *recordingProvider }

func (s *recordingToolSpan) End(error)                  { s.provider.toolEnded++ }
func (s *recordingToolSpan) SetMetadata(map[string]any) {}
func (s *recordingToolSpan) SetResult(any)              {}

// fakeRunner is a minimal Runner for testing.
type fakeRunner struct {
	result *graft.Result
	err    error
}

func (r *fakeRunner) Run(_ context.Context, _ *graft.Agent, _ []graft.Message, _ ...graft.RunOption) (*graft.Result, error) {
	return r.result, r.err
}

func (r *fakeRunner) RunStream(_ context.Context, _ *graft.Agent, _ []graft.Message, _ ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	ch := make(chan graft.StreamEvent, 1)
	ch <- graft.StreamEvent{Type: graft.EventDone}
	close(ch)
	return ch, nil
}

func TestTracedRunnerRun(t *testing.T) {
	provider := &recordingProvider{}
	inner := &fakeRunner{result: &graft.Result{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: "hello"}},
	}}
	runner := NewTracedRunner(inner, provider)

	agent := graft.NewAgent("test-agent")
	msgs := []graft.Message{{Role: graft.RoleUser, Content: "hi"}}

	result, err := runner.Run(context.Background(), agent, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if result.LastAssistantText() != "hello" {
		t.Errorf("expected 'hello', got %s", result.LastAssistantText())
	}

	if len(provider.runs) != 1 {
		t.Errorf("expected 1 run span, got %d", len(provider.runs))
	}
	if provider.runs[0].AgentName != "test-agent" {
		t.Errorf("expected agent name 'test-agent', got %s", provider.runs[0].AgentName)
	}
	if provider.runEnded != 1 {
		t.Errorf("expected run span ended, got %d", provider.runEnded)
	}
	if provider.flushed != 1 {
		t.Errorf("expected 1 flush, got %d", provider.flushed)
	}
}

func TestTracedRunnerRunStream(t *testing.T) {
	provider := &recordingProvider{}
	inner := &fakeRunner{}
	runner := NewTracedRunner(inner, provider)

	agent := graft.NewAgent("stream-agent")
	ch, err := runner.RunStream(context.Background(), agent, nil)
	if err != nil {
		t.Fatal(err)
	}

	var events []graft.StreamEvent
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if provider.runEnded != 1 {
		t.Errorf("expected run span ended, got %d", provider.runEnded)
	}
}

func TestTracedRunnerCaptureInputDisabled(t *testing.T) {
	provider := &recordingProvider{}
	inner := &fakeRunner{result: &graft.Result{}}
	runner := NewTracedRunner(inner, provider, WithCaptureInput(false))

	agent := graft.NewAgent("test")
	msgs := []graft.Message{{Role: graft.RoleUser, Content: "secret"}}
	runner.Run(context.Background(), agent, msgs)

	if provider.runs[0].Messages != nil {
		t.Error("expected messages to not be captured")
	}
}

func TestMultiProvider(t *testing.T) {
	p1 := &recordingProvider{}
	p2 := &recordingProvider{}
	multi := NewMultiProvider(p1, p2)

	ctx := context.Background()
	info := RunInfo{AgentName: "multi-test"}
	_, span := multi.StartRun(ctx, info)
	span.End(nil)
	multi.Flush(ctx)

	if len(p1.runs) != 1 || len(p2.runs) != 1 {
		t.Errorf("expected 1 run in each provider, got %d and %d", len(p1.runs), len(p2.runs))
	}
	if p1.runEnded != 1 || p2.runEnded != 1 {
		t.Errorf("expected spans ended in each, got %d and %d", p1.runEnded, p2.runEnded)
	}
	if p1.flushed != 1 || p2.flushed != 1 {
		t.Errorf("expected flush in each, got %d and %d", p1.flushed, p2.flushed)
	}
}

func TestNoopProvider(t *testing.T) {
	noop := NewNoopProvider()
	ctx := context.Background()

	// Should not panic
	_, rs := noop.StartRun(ctx, RunInfo{})
	rs.End(nil)
	rs.SetMetadata(map[string]any{"k": "v"})
	rs.SetUsage(graft.Usage{})

	_, gs := noop.StartGeneration(ctx, GenerationInfo{})
	gs.End(nil)
	gs.SetModel("test")

	_, ts := noop.StartToolCall(ctx, ToolCallInfo{})
	ts.End(nil)
	ts.SetResult("result")

	if err := noop.Flush(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTracedRunnerWithMetadata(t *testing.T) {
	provider := &recordingProvider{}
	inner := &fakeRunner{result: &graft.Result{}}
	meta := map[string]any{"env": "test"}
	runner := NewTracedRunner(inner, provider, WithMetadata(meta))

	agent := graft.NewAgent("test")
	runner.Run(context.Background(), agent, nil)

	if provider.runs[0].Metadata["env"] != "test" {
		t.Error("expected metadata to be set")
	}
}
