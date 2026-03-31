package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/trigger"
)

// This example shows how to run a graft agent via Trigger.dev's REST API.
// You need a Trigger.dev project with a task configured to handle agent
// execution payloads.
//
// Trigger.dev provides background jobs with event streaming, waitpoints
// for human-in-the-loop approvals, and scheduled execution.
func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)
	_ = model // Model is used by the Trigger.dev task, not the client.

	runner := trigger.NewTriggerRunner("https://api.trigger.dev",
		trigger.WithAPIKey(os.Getenv("TRIGGER_API_KEY")),
		trigger.WithProjectID("my-project"),
		trigger.WithEnvironment("production"),
	)

	agent := graft.NewAgent("background-assistant",
		graft.WithInstructions("You are a helpful assistant running as a background task."),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Generate a summary of recent AI developments"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
