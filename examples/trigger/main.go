// Trigger.dev example.
//
// TriggerRunner dispatches agent runs to Trigger.dev tasks over HTTPS and
// polls for completion. Unlike the Hatchet and Temporal runtimes, Trigger.dev
// does not require a Go worker — the task that executes the agent lives in
// your Trigger.dev project (usually TypeScript) and is the side that
// configures the LLM provider.
//
// This client-side program is runnable against a real Trigger.dev project.
// Set TRIGGER_API_KEY and TRIGGER_PROJECT_ID before running:
//
//	export TRIGGER_API_KEY="tr_..."
//	export TRIGGER_PROJECT_ID="proj_..."
//	go run ./examples/trigger
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/trigger"
)

func main() {
	runner := trigger.NewTriggerRunner("https://api.trigger.dev",
		trigger.WithAPIKey(os.Getenv("TRIGGER_API_KEY")),
		trigger.WithProjectID(os.Getenv("TRIGGER_PROJECT_ID")),
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
