package tracing

import (
	"context"
	"log"

	"github.com/delavalom/graft"
)

// MultiProvider fans out trace events to multiple providers.
// Errors from individual providers are logged, not propagated.
type MultiProvider struct {
	providers []TracingProvider
}

// NewMultiProvider creates a provider that sends to all given providers.
func NewMultiProvider(providers ...TracingProvider) *MultiProvider {
	return &MultiProvider{providers: providers}
}

func (m *MultiProvider) StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan) {
	spans := make([]RunSpan, len(m.providers))
	for i, p := range m.providers {
		ctx, spans[i] = p.StartRun(ctx, info)
	}
	return ctx, &multiRunSpan{spans: spans}
}

func (m *MultiProvider) StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan) {
	spans := make([]GenerationSpan, len(m.providers))
	for i, p := range m.providers {
		ctx, spans[i] = p.StartGeneration(ctx, info)
	}
	return ctx, &multiGenSpan{spans: spans}
}

func (m *MultiProvider) StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan) {
	spans := make([]ToolCallSpan, len(m.providers))
	for i, p := range m.providers {
		ctx, spans[i] = p.StartToolCall(ctx, info)
	}
	return ctx, &multiToolSpan{spans: spans}
}

func (m *MultiProvider) Flush(ctx context.Context) error {
	for _, p := range m.providers {
		if err := p.Flush(ctx); err != nil {
			log.Printf("tracing: multi flush error: %v", err)
		}
	}
	return nil
}

type multiRunSpan struct{ spans []RunSpan }

func (s *multiRunSpan) End(err error) {
	for _, sp := range s.spans {
		sp.End(err)
	}
}

func (s *multiRunSpan) SetMetadata(meta map[string]any) {
	for _, sp := range s.spans {
		sp.SetMetadata(meta)
	}
}

func (s *multiRunSpan) SetUsage(usage graft.Usage) {
	for _, sp := range s.spans {
		sp.SetUsage(usage)
	}
}

type multiGenSpan struct{ spans []GenerationSpan }

func (s *multiGenSpan) End(err error) {
	for _, sp := range s.spans {
		sp.End(err)
	}
}

func (s *multiGenSpan) SetMetadata(meta map[string]any) {
	for _, sp := range s.spans {
		sp.SetMetadata(meta)
	}
}

func (s *multiGenSpan) SetUsage(usage graft.Usage) {
	for _, sp := range s.spans {
		sp.SetUsage(usage)
	}
}

func (s *multiGenSpan) SetModel(model string) {
	for _, sp := range s.spans {
		sp.SetModel(model)
	}
}

type multiToolSpan struct{ spans []ToolCallSpan }

func (s *multiToolSpan) End(err error) {
	for _, sp := range s.spans {
		sp.End(err)
	}
}

func (s *multiToolSpan) SetMetadata(meta map[string]any) {
	for _, sp := range s.spans {
		sp.SetMetadata(meta)
	}
}

func (s *multiToolSpan) SetResult(result any) {
	for _, sp := range s.spans {
		sp.SetResult(result)
	}
}

var _ TracingProvider = (*MultiProvider)(nil)
