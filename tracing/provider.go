package tracing

import (
	"context"

	"github.com/delavalom/graft"
)

// TracingProvider is the interface for agent-specific tracing backends.
type TracingProvider interface {
	StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan)
	StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan)
	StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan)
	Flush(ctx context.Context) error
}

// RunSpan represents a top-level agent run span.
type RunSpan interface {
	End(err error)
	SetMetadata(meta map[string]any)
	SetUsage(usage graft.Usage)
}

// GenerationSpan represents an LLM generation span.
type GenerationSpan interface {
	End(err error)
	SetMetadata(meta map[string]any)
	SetUsage(usage graft.Usage)
	SetModel(model string)
}

// ToolCallSpan represents a tool invocation span.
type ToolCallSpan interface {
	End(err error)
	SetMetadata(meta map[string]any)
	SetResult(result any)
}

// RunInfo carries data for starting a run span.
type RunInfo struct {
	AgentName string
	Model     string
	Messages  []graft.Message
	Metadata  map[string]any
}

// GenerationInfo carries data for starting a generation span.
type GenerationInfo struct {
	AgentName string
	Model     string
	Messages  []graft.Message
}

// ToolCallInfo carries data for starting a tool call span.
type ToolCallInfo struct {
	ToolName  string
	Arguments []byte
}
