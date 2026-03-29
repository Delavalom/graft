package graft

import (
	"context"
	"encoding/json"

	"github.com/delavalom/graft/internal/jsonschema"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (any, error)
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

func ToolDefFromTool(t Tool) ToolDefinition {
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Schema:      t.Schema(),
	}
}

type typedTool[P any, R any] struct {
	name        string
	description string
	schema      json.RawMessage
	fn          func(ctx context.Context, params P) (R, error)
}

func NewTool[P any, R any](name, description string, fn func(ctx context.Context, params P) (R, error)) Tool {
	schema, err := jsonschema.Generate[P]()
	if err != nil {
		schema = json.RawMessage(`{"type":"object"}`)
	}
	return &typedTool[P, R]{
		name:        name,
		description: description,
		schema:      schema,
		fn:          fn,
	}
}

func (t *typedTool[P, R]) Name() string           { return t.name }
func (t *typedTool[P, R]) Description() string     { return t.description }
func (t *typedTool[P, R]) Schema() json.RawMessage { return t.schema }

func (t *typedTool[P, R]) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	var p P
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, NewAgentError(ErrInvalidToolCall, "failed to unmarshal tool parameters", err)
	}
	return t.fn(ctx, p)
}
