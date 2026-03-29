package temporal

import (
	"context"
	"encoding/json"

	"github.com/delavalom/graft"
)

// ActivityAsTool converts a named Temporal activity into a graft.Tool.
// When the tool is executed, it invokes the activity through the workflow context.
// This is the Go equivalent of Temporal's Python `activity_as_tool()` pattern.
func ActivityAsTool(name, description string, schema json.RawMessage, executor ActivityExecutor, opts ...ActivityToolOption) graft.Tool {
	cfg := activityToolConfig{
		activityConfig: DefaultActivityConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &activityTool{
		name:        name,
		description: description,
		schema:      schema,
		executor:    executor,
		config:      cfg.activityConfig,
	}
}

// ActivityExecutor is a function that executes a Temporal activity.
// Users provide this from their workflow context.
type ActivityExecutor func(ctx context.Context, activityName string, input any) (any, error)

// activityTool wraps a Temporal activity as a graft.Tool.
type activityTool struct {
	name        string
	description string
	schema      json.RawMessage
	executor    ActivityExecutor
	config      *ActivityConfig
}

func (t *activityTool) Name() string             { return t.name }
func (t *activityTool) Description() string       { return t.description }
func (t *activityTool) Schema() json.RawMessage   { return t.schema }

func (t *activityTool) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	result, err := t.executor(ctx, t.name, params)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrToolExecution, "temporal activity failed: "+t.name, err)
	}
	return result, nil
}

// Ensure activityTool satisfies graft.Tool.
var _ graft.Tool = (*activityTool)(nil)
