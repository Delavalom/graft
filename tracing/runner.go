package tracing

import (
	"context"
	"log"

	"github.com/delavalom/graft"
)

// Option configures a TracedRunner.
type Option func(*tracedConfig)

type tracedConfig struct {
	captureInput  bool
	captureOutput bool
	metadata      map[string]any
}

// WithCaptureInput enables capturing input messages in traces.
func WithCaptureInput(enabled bool) Option {
	return func(c *tracedConfig) { c.captureInput = enabled }
}

// WithCaptureOutput enables capturing output messages in traces.
func WithCaptureOutput(enabled bool) Option {
	return func(c *tracedConfig) { c.captureOutput = enabled }
}

// WithMetadata sets default metadata added to all spans.
func WithMetadata(meta map[string]any) Option {
	return func(c *tracedConfig) { c.metadata = meta }
}

// TracedRunner wraps a graft.Runner with tracing instrumentation.
type TracedRunner struct {
	inner    graft.Runner
	provider TracingProvider
	cfg      tracedConfig
}

// NewTracedRunner creates a new tracing-instrumented runner.
func NewTracedRunner(runner graft.Runner, provider TracingProvider, opts ...Option) *TracedRunner {
	cfg := tracedConfig{
		captureInput:  true,
		captureOutput: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &TracedRunner{
		inner:    runner,
		provider: provider,
		cfg:      cfg,
	}
}

func (r *TracedRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	info := RunInfo{
		AgentName: agent.Name,
		Model:     agent.Model,
		Metadata:  r.cfg.metadata,
	}
	if r.cfg.captureInput {
		info.Messages = messages
	}

	ctx, span := r.provider.StartRun(ctx, info)

	result, err := r.inner.Run(ctx, agent, messages, opts...)

	if result != nil {
		span.SetUsage(result.Usage)
	}
	if r.cfg.metadata != nil {
		span.SetMetadata(r.cfg.metadata)
	}
	span.End(err)

	// Best-effort flush — tracing errors never bubble up
	if flushErr := r.provider.Flush(ctx); flushErr != nil {
		log.Printf("tracing: flush error: %v", flushErr)
	}

	return result, err
}

func (r *TracedRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	info := RunInfo{
		AgentName: agent.Name,
		Model:     agent.Model,
		Metadata:  r.cfg.metadata,
	}
	if r.cfg.captureInput {
		info.Messages = messages
	}

	ctx, span := r.provider.StartRun(ctx, info)

	ch, err := r.inner.RunStream(ctx, agent, messages, opts...)
	if err != nil {
		span.End(err)
		return nil, err
	}

	// Wrap channel to end span when done
	wrapped := make(chan graft.StreamEvent, 16)
	go func() {
		defer close(wrapped)
		defer span.End(nil)
		defer func() {
			if flushErr := r.provider.Flush(ctx); flushErr != nil {
				log.Printf("tracing: flush error: %v", flushErr)
			}
		}()
		for event := range ch {
			wrapped <- event
		}
	}()
	return wrapped, nil
}

// Ensure TracedRunner satisfies graft.Runner.
var _ graft.Runner = (*TracedRunner)(nil)
