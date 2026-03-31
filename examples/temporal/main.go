package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/temporal"
)

// This example shows how to run a graft agent as a durable Temporal workflow.
// You need a running Temporal server and a worker registered with the
// AgentWorkflow function.
//
// The TemporalRunner wraps a standard graft agent — the same agent definition
// works with DefaultRunner or TemporalRunner.
func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)
	_ = model // Model is used by the Temporal worker, not the client.

	// In production, create a real Temporal client:
	//   c, err := client.Dial(client.Options{})
	//   runner := temporal.NewTemporalRunner(c, ...)
	//
	// Here we show the setup pattern:
	var temporalClient temporal.TemporalClient // = your *client.Client

	runner := temporal.NewTemporalRunner(temporalClient,
		temporal.WithTaskQueue("ai-agents"),
		temporal.WithRetryPolicy(&temporal.RetryPolicy{
			MaxAttempts:        5,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaxInterval:        30 * time.Second,
		}),
	)

	agent := graft.NewAgent("durable-assistant",
		graft.WithInstructions("You are a helpful assistant running in a durable workflow."),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Summarize the benefits of durable execution"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
