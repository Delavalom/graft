package graph

import (
	"context"
	"encoding/json"

	"github.com/delavalom/graft"
)

// NodeFunc is a function that processes state in a graph node.
type NodeFunc[S any] func(ctx context.Context, state S) (S, error)

// RouterFunc is a function that returns the name of the next edge to follow.
type RouterFunc[S any] func(ctx context.Context, state S) string

// AgentNode wraps a graft agent as a graph node.
// The state type must implement StateWithMessages so the node can
// extract messages, run the agent, and store the result messages back.
func AgentNode[S StateWithMessages](agent *graft.Agent, runner graft.Runner) NodeFunc[S] {
	return func(ctx context.Context, state S) (S, error) {
		msgs := state.GetMessages()
		result, err := runner.Run(ctx, agent, msgs)
		if err != nil {
			return state, err
		}
		state.SetMessages(append(msgs, result.Messages...))
		return state, nil
	}
}

// ToolNode wraps a graft tool as a graph node.
// It expects the state to have a "tool_input" key in Values (for MessageState)
// or uses empty params. The tool result is stored back in state.
func ToolNode[S any](tool graft.Tool) NodeFunc[S] {
	return func(ctx context.Context, state S) (S, error) {
		params := json.RawMessage(`{}`)

		// If state is a MessageState, check for tool_input
		if ms, ok := any(state).(*MessageState); ok {
			if input, exists := ms.Get("tool_input"); exists {
				if raw, ok := input.(json.RawMessage); ok {
					params = raw
				} else if b, err := json.Marshal(input); err == nil {
					params = b
				}
			}
		}

		result, err := tool.Execute(ctx, params)
		if err != nil {
			return state, err
		}

		// Store result back if possible
		if ms, ok := any(state).(*MessageState); ok {
			ms.Set("tool_result", result)
		}
		return state, nil
	}
}
