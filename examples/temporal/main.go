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

// This example shows how to run a graft agent as a durable Temporal workflow
// with per-step durability. Each LLM call and tool execution is a separate
// Temporal activity, so on failure only the failed step reruns.
//
// Two sides are needed:
//
//  1. Worker side — registers the workflow and activities with a Temporal worker.
//     See workerSetup() below for the pattern.
//
//  2. Client side — uses TemporalRunner to start and wait for the workflow.
//     See clientSetup() below for the pattern.
func main() {
	// Client side: start and wait for workflow
	clientSetup()
}

// workerSetup shows how to register the DefaultAgentWorkflow and its activities
// on a Temporal worker. This runs in a separate process from the client.
//
// To use the per-step durable workflow, import the agentworkflow sub-package:
//
//	import "github.com/delavalom/graft/temporal/agentworkflow"
//
// Then register with your worker:
//
//	// Create the model provider — activities use this to call the LLM
//	model := openai.New(
//		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
//		openai.WithBaseURL("https://openrouter.ai/api/v1"),
//		openai.WithModel("anthropic/claude-sonnet-4.6"),
//	)
//	models := agentworkflow.NewSingleModelProvider(model)
//
//	// Register tools that the agent can use
//	toolRegistry := agentworkflow.NewToolRegistry()
//	toolRegistry.Register(myTool1, myTool2)
//
//	// Register workflow and activities
//	w := worker.New(temporalClient, "ai-agents", worker.Options{})
//	agentworkflow.RegisterAgentWorkflow(w)
//	agentworkflow.RegisterAgentActivities(w, models, toolRegistry)
//
//	// Start the worker
//	w.Run(worker.InterruptCh())
//
// Each LLM call runs as a GenerateActivity and each tool call runs as a
// ToolActivity. On worker crash, Temporal replays completed activities from
// event history and resumes from the last checkpoint.
func workerSetup() {}

// clientSetup shows the client-side pattern for starting a durable agent workflow.
func clientSetup() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)
	_ = model // Model is used by the worker, not the client.

	// In production, create a real Temporal client:
	//   c, err := client.Dial(client.Options{})
	//   runner := temporal.NewTemporalRunner(c, ...)
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
