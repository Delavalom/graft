package otel

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
)

type noopModel struct{}

func (n *noopModel) ModelID() string { return "noop" }
func (n *noopModel) Generate(ctx context.Context, params graft.GenerateParams) (*graft.GenerateResult, error) {
	return &graft.GenerateResult{
		Message: graft.Message{Role: graft.RoleAssistant, Content: "ok"},
		Usage:   graft.Usage{PromptTokens: 10, CompletionTokens: 5},
	}, nil
}
func (n *noopModel) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	ch := make(chan graft.StreamChunk)
	close(ch)
	return ch, nil
}

func TestInstrumentRunnerReturnsRunner(t *testing.T) {
	model := &noopModel{}
	runner := graft.NewDefaultRunner(model)
	instrumented := InstrumentRunner(runner)

	var _ graft.Runner = instrumented

	result, err := instrumented.Run(context.Background(), graft.NewAgent("test"), []graft.Message{
		{Role: graft.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.LastAssistantText() != "ok" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "ok")
	}
}

func TestAttributeConstants(t *testing.T) {
	attrs := []string{
		AttrAgentName, AttrModelID, AttrProviderName,
		AttrPromptTokens, AttrCompletionTokens, AttrTotalTokens,
		AttrCostUSD, AttrToolName, AttrToolDuration, AttrIterationCount,
	}
	for _, a := range attrs {
		if a == "" {
			t.Error("found empty attribute constant")
		}
	}
}
