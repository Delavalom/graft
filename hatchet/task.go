package hatchet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/delavalom/graft"
)

// TaskAsTool converts a named Hatchet task into a graft.Tool.
// When executed, the tool runs the task through the Hatchet client.
func TaskAsTool(name, description string, schema json.RawMessage, client HatchetClient, opts ...TaskToolOption) graft.Tool {
	cfg := taskToolConfig{
		retryConfig: DefaultRetryConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &hatchetTool{
		name:        name,
		description: description,
		schema:      schema,
		client:      client,
		retryConfig: cfg.retryConfig,
	}
}

type hatchetTool struct {
	name        string
	description string
	schema      json.RawMessage
	client      HatchetClient
	retryConfig *RetryConfig
}

func (t *hatchetTool) Name() string             { return t.name }
func (t *hatchetTool) Description() string       { return t.description }
func (t *hatchetTool) Schema() json.RawMessage   { return t.schema }

func (t *hatchetTool) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	run, err := t.client.RunTask(ctx, t.name, params)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrToolExecution, fmt.Sprintf("hatchet task %s failed to start", t.name), err)
	}
	result, err := run.Wait(ctx)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrToolExecution, fmt.Sprintf("hatchet task %s failed", t.name), err)
	}
	return result, nil
}

var _ graft.Tool = (*hatchetTool)(nil)
