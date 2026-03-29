package otel

import (
	"context"
	"time"

	"github.com/delavalom/graft"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func InstrumentMetrics(runner graft.Runner, meter metric.Meter) graft.Runner {
	runDuration, _ := meter.Float64Histogram("graft.run.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Agent run duration in milliseconds"))

	promptTokens, _ := meter.Int64Counter("graft.llm.tokens.prompt",
		metric.WithDescription("Total prompt tokens consumed"))

	completionTokens, _ := meter.Int64Counter("graft.llm.tokens.completion",
		metric.WithDescription("Total completion tokens consumed"))

	toolErrors, _ := meter.Int64Counter("graft.tool.errors",
		metric.WithDescription("Tool execution failures"))

	return &metricsRunner{
		inner:            runner,
		runDuration:      runDuration,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		toolErrors:       toolErrors,
	}
}

type metricsRunner struct {
	inner            graft.Runner
	runDuration      metric.Float64Histogram
	promptTokens     metric.Int64Counter
	completionTokens metric.Int64Counter
	toolErrors       metric.Int64Counter
}

func (m *metricsRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	start := time.Now()
	attrs := metric.WithAttributes(attribute.String(AttrAgentName, agent.Name))

	result, err := m.inner.Run(ctx, agent, messages, opts...)

	duration := float64(time.Since(start).Milliseconds())
	m.runDuration.Record(ctx, duration, attrs)

	if err != nil {
		m.toolErrors.Add(ctx, 1, attrs)
		return nil, err
	}

	m.promptTokens.Add(ctx, int64(result.Usage.PromptTokens), attrs)
	m.completionTokens.Add(ctx, int64(result.Usage.CompletionTokens), attrs)

	return result, nil
}

func (m *metricsRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	return m.inner.RunStream(ctx, agent, messages, opts...)
}
