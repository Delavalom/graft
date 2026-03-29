package tracing

import (
	"context"

	"github.com/delavalom/graft"
)

// NoopProvider is a no-op tracing provider for disabling tracing.
type NoopProvider struct{}

// NewNoopProvider creates a no-op provider.
func NewNoopProvider() *NoopProvider { return &NoopProvider{} }

func (p *NoopProvider) StartRun(ctx context.Context, _ RunInfo) (context.Context, RunSpan) {
	return ctx, noopRunSpan{}
}

func (p *NoopProvider) StartGeneration(ctx context.Context, _ GenerationInfo) (context.Context, GenerationSpan) {
	return ctx, noopGenSpan{}
}

func (p *NoopProvider) StartToolCall(ctx context.Context, _ ToolCallInfo) (context.Context, ToolCallSpan) {
	return ctx, noopToolSpan{}
}

func (p *NoopProvider) Flush(_ context.Context) error { return nil }

type noopRunSpan struct{}

func (noopRunSpan) End(error)                  {}
func (noopRunSpan) SetMetadata(map[string]any) {}
func (noopRunSpan) SetUsage(graft.Usage)       {}

type noopGenSpan struct{}

func (noopGenSpan) End(error)                  {}
func (noopGenSpan) SetMetadata(map[string]any) {}
func (noopGenSpan) SetUsage(graft.Usage)       {}
func (noopGenSpan) SetModel(string)            {}

type noopToolSpan struct{}

func (noopToolSpan) End(error)                  {}
func (noopToolSpan) SetMetadata(map[string]any) {}
func (noopToolSpan) SetResult(any)              {}

var _ TracingProvider = (*NoopProvider)(nil)
