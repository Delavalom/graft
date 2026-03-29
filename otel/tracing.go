package otel

import (
	"context"
	"time"

	"github.com/delavalom/graft"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/delavalom/graft/otel"

type Option func(*config)

type config struct {
	tracerProvider trace.TracerProvider
}

func defaultConfig() config {
	return config{tracerProvider: otel.GetTracerProvider()}
}

func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) { c.tracerProvider = tp }
}

func InstrumentRunner(runner graft.Runner, opts ...Option) graft.Runner {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	tracer := cfg.tracerProvider.Tracer(tracerName)
	return &tracingRunner{inner: runner, tracer: tracer}
}

type tracingRunner struct {
	inner  graft.Runner
	tracer trace.Tracer
}

func (t *tracingRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	ctx, span := t.tracer.Start(ctx, "agent.run",
		trace.WithAttributes(
			attribute.String(AttrAgentName, agent.Name),
			attribute.String(AttrModelID, agent.Model),
		),
	)
	defer span.End()

	start := time.Now()
	result, err := t.inner.Run(ctx, agent, messages, opts...)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int(AttrPromptTokens, result.Usage.PromptTokens),
		attribute.Int(AttrCompletionTokens, result.Usage.CompletionTokens),
		attribute.Int(AttrTotalTokens, result.Usage.TotalTokens()),
		attribute.Int64(AttrToolDuration, duration.Milliseconds()),
	)

	return result, nil
}

func (t *tracingRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	ctx, span := t.tracer.Start(ctx, "agent.run_stream",
		trace.WithAttributes(
			attribute.String(AttrAgentName, agent.Name),
			attribute.String(AttrModelID, agent.Model),
		),
	)

	events, err := t.inner.RunStream(ctx, agent, messages, opts...)
	if err != nil {
		span.RecordError(err)
		span.End()
		return nil, err
	}

	wrapped := make(chan graft.StreamEvent, 64)
	go func() {
		defer close(wrapped)
		defer span.End()
		for ev := range events {
			wrapped <- ev
		}
	}()

	return wrapped, nil
}
