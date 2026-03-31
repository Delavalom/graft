package agentworkflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/delavalom/graft"
	graftemporal "github.com/delavalom/graft/temporal"
	"go.temporal.io/sdk/activity"
)

// Activities holds worker-side dependencies for agent activities.
// Construct once per worker and register its methods as Temporal activities.
type Activities struct {
	models ModelProvider
	tools  *ToolRegistry
}

// NewActivities creates an Activities instance with the given model provider and tool registry.
func NewActivities(models ModelProvider, tools *ToolRegistry) *Activities {
	return &Activities{models: models, tools: tools}
}

// GenerateActivity performs a single LLM call as a Temporal activity.
// It deserializes the input, calls the language model, and serializes the response.
func (a *Activities) GenerateActivity(ctx context.Context, input graftemporal.GenerateActivityInput) (*graftemporal.GenerateActivityOutput, error) {
	model, err := a.models.Model(input.Model)
	if err != nil {
		return nil, fmt.Errorf("GenerateActivity: %w", err)
	}

	var messages []graft.Message
	if err := json.Unmarshal(input.Messages, &messages); err != nil {
		return nil, fmt.Errorf("GenerateActivity: unmarshal messages: %w", err)
	}

	var tools []graft.ToolDefinition
	if len(input.Tools) > 0 {
		if err := json.Unmarshal(input.Tools, &tools); err != nil {
			return nil, fmt.Errorf("GenerateActivity: unmarshal tools: %w", err)
		}
	}

	params := graft.GenerateParams{
		Messages:    messages,
		Tools:       tools,
		Temperature: input.Temperature,
		MaxTokens:   input.MaxTokens,
	}

	// Heartbeat before the potentially long LLM call
	activity.RecordHeartbeat(ctx, "calling model")

	result, err := model.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("GenerateActivity: model.Generate: %w", err)
	}

	msgBytes, err := json.Marshal(result.Message)
	if err != nil {
		return nil, fmt.Errorf("GenerateActivity: marshal message: %w", err)
	}

	usageBytes, err := json.Marshal(result.Usage)
	if err != nil {
		return nil, fmt.Errorf("GenerateActivity: marshal usage: %w", err)
	}

	return &graftemporal.GenerateActivityOutput{
		Message: msgBytes,
		Usage:   usageBytes,
	}, nil
}

// ToolActivity executes a single tool as a Temporal activity.
// It looks up the tool by name in the registry and calls Execute.
func (a *Activities) ToolActivity(ctx context.Context, input graftemporal.ToolActivityInput) (*graftemporal.ToolActivityOutput, error) {
	tool, ok := a.tools.Get(input.ToolName)
	if !ok {
		return &graftemporal.ToolActivityOutput{
			Result:  jsonBytes(fmt.Sprintf("unknown tool: %s", input.ToolName)),
			IsError: true,
		}, nil
	}

	activity.RecordHeartbeat(ctx, "executing tool: "+input.ToolName)

	output, err := tool.Execute(ctx, input.Arguments)
	if err != nil {
		return &graftemporal.ToolActivityOutput{
			Result:  jsonBytes(err.Error()),
			IsError: true,
		}, nil
	}

	resultBytes, err := marshalToolOutput(output)
	if err != nil {
		return &graftemporal.ToolActivityOutput{
			Result:  jsonBytes(fmt.Sprintf("failed to marshal tool result: %v", err)),
			IsError: true,
		}, nil
	}

	return &graftemporal.ToolActivityOutput{
		Result:  resultBytes,
		IsError: false,
	}, nil
}

func marshalToolOutput(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonBytes(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`""`)
	}
	return b
}
