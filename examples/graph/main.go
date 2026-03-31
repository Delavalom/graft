package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/graph"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	lookupTool := graft.NewTool("lookup_weather", "Look up the weather for a city",
		func(ctx context.Context, p struct {
			City string `json:"city" description:"The city to look up"`
		}) (string, error) {
			return fmt.Sprintf("The weather in %s is 72°F and sunny.", p.City), nil
		},
	)

	agent := graft.NewAgent("weather-agent",
		graft.WithInstructions("You are a weather assistant. Use the lookup_weather tool to answer weather questions."),
		graft.WithTools(lookupTool),
	)

	runner := graft.NewDefaultRunner(model)

	// Build a ReAct graph: generate → check tool calls → execute → loop.
	compiled, err := graph.ReactAgentGraph(agent, runner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile graph: %v\n", err)
		os.Exit(1)
	}

	// Initialize state with the user's message.
	state := graph.NewMessageState()
	state.Messages = []graft.Message{
		{Role: graft.RoleUser, Content: "What's the weather in Tokyo?"},
	}

	// Run the graph to completion.
	finalState, err := compiled.Run(context.Background(), state)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print the last assistant message.
	msgs := finalState.GetMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == graft.RoleAssistant && msgs[i].Content != "" {
			fmt.Println(msgs[i].Content)
			break
		}
	}
}
