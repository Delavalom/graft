package graph

import (
	"context"

	"github.com/delavalom/graft"
)

// NamedNode pairs a name with a node function for convenience constructors.
type NamedNode[S any] struct {
	Name string
	Func NodeFunc[S]
}

// ReactAgentGraph creates a pre-built ReAct pattern graph:
// generate → check for tool calls → execute tools → loop back.
func ReactAgentGraph(agent *graft.Agent, runner graft.Runner) (*CompiledGraph[*MessageState], error) {
	g := NewGraph[*MessageState]("react-agent")

	// Node: generate — run the agent
	g.AddNode("generate", func(ctx context.Context, state *MessageState) (*MessageState, error) {
		result, err := runner.Run(ctx, agent, state.GetMessages())
		if err != nil {
			return state, err
		}
		state.SetMessages(append(state.GetMessages(), result.Messages...))
		state.Set("last_result", result)
		return state, nil
	})

	// Node: tools — execute tool calls (already handled by runner, this is a passthrough)
	g.AddNode("tools", func(ctx context.Context, state *MessageState) (*MessageState, error) {
		// The runner already executed tools in the generate step.
		// This node exists for the graph structure to be explicit.
		return state, nil
	})

	g.SetEntryPoint(START)
	g.AddEdge(START, "generate")

	// After generate, check if there were tool calls (meaning we should loop)
	g.AddConditionalEdge("generate", func(_ context.Context, state *MessageState) string {
		if v, ok := state.Get("last_result"); ok {
			if result, ok := v.(*graft.Result); ok {
				for _, msg := range result.Messages {
					if len(msg.ToolCalls) > 0 {
						return "continue"
					}
				}
			}
		}
		return "done"
	}, map[string]string{
		"continue": "tools",
		"done":     END,
	})

	g.AddEdge("tools", "generate")

	return g.Compile()
}
