package tracing

import (
	"context"
	"fmt"

	"github.com/delavalom/graft"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OTelProvider bridges the TracingProvider interface to OpenTelemetry.
type OTelProvider struct {
	tracer trace.Tracer
}

// NewOTelProvider creates a TracingProvider backed by OTel.
func NewOTelProvider(tracer trace.Tracer) *OTelProvider {
	return &OTelProvider{tracer: tracer}
}

func (p *OTelProvider) StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan) {
	ctx, span := p.tracer.Start(ctx, "agent.run",
		trace.WithAttributes(
			attribute.String("graft.agent.name", info.AgentName),
			attribute.String("graft.model.id", info.Model),
		),
	)
	return ctx, &otelRunSpan{span: span}
}

func (p *OTelProvider) StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan) {
	ctx, span := p.tracer.Start(ctx, "llm.generate",
		trace.WithAttributes(
			attribute.String("graft.agent.name", info.AgentName),
			attribute.String("graft.model.id", info.Model),
		),
	)
	return ctx, &otelGenSpan{span: span}
}

func (p *OTelProvider) StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan) {
	ctx, span := p.tracer.Start(ctx, "tool.execute",
		trace.WithAttributes(
			attribute.String("graft.tool.name", info.ToolName),
		),
	)
	return ctx, &otelToolSpan{span: span}
}

func (p *OTelProvider) Flush(_ context.Context) error { return nil }

type otelRunSpan struct{ span trace.Span }

func (s *otelRunSpan) End(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
	s.span.End()
}

func (s *otelRunSpan) SetMetadata(meta map[string]any) {
	for k, v := range meta {
		s.span.SetAttributes(attribute.String("graft.meta."+k, fmt.Sprint(v)))
	}
}

func (s *otelRunSpan) SetUsage(usage graft.Usage) {
	s.span.SetAttributes(
		attribute.Int("graft.usage.prompt_tokens", usage.PromptTokens),
		attribute.Int("graft.usage.completion_tokens", usage.CompletionTokens),
		attribute.Int("graft.usage.total_tokens", usage.TotalTokens()),
	)
}

type otelGenSpan struct{ span trace.Span }

func (s *otelGenSpan) End(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
	s.span.End()
}

func (s *otelGenSpan) SetMetadata(meta map[string]any) {
	for k, v := range meta {
		s.span.SetAttributes(attribute.String("graft.meta."+k, fmt.Sprint(v)))
	}
}

func (s *otelGenSpan) SetModel(model string) {
	s.span.SetAttributes(attribute.String("graft.model.id", model))
}

func (s *otelGenSpan) SetUsage(usage graft.Usage) {
	s.span.SetAttributes(
		attribute.Int("graft.usage.prompt_tokens", usage.PromptTokens),
		attribute.Int("graft.usage.completion_tokens", usage.CompletionTokens),
		attribute.Int("graft.usage.total_tokens", usage.TotalTokens()),
	)
}

type otelToolSpan struct{ span trace.Span }

func (s *otelToolSpan) End(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
	s.span.End()
}

func (s *otelToolSpan) SetMetadata(meta map[string]any) {
	for k, v := range meta {
		s.span.SetAttributes(attribute.String("graft.meta."+k, fmt.Sprint(v)))
	}
}

func (s *otelToolSpan) SetResult(result any) {
	s.span.SetAttributes(attribute.String("graft.tool.result", fmt.Sprint(result)))
}

var _ TracingProvider = (*OTelProvider)(nil)
