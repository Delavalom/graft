package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/hatchet"
	"github.com/delavalom/graft/provider/openai"
)

// This example shows how to run a graft agent as a Hatchet durable function.
// You need a running Hatchet server and a worker registered with the
// agent workflow.
//
// Hatchet provides serverless durable functions with built-in concurrency
// control, rate limiting, and DAG-based workflow orchestration.
func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)
	_ = model // Model is used by the Hatchet worker, not the client.

	// In production, create a real Hatchet client:
	//   c, err := hatchetclient.New()
	//   runner := hatchet.NewHatchetRunner(c, ...)
	//
	// Here we show the setup pattern:
	var hatchetClient hatchet.HatchetClient // = your Hatchet client

	runner := hatchet.NewHatchetRunner(hatchetClient,
		hatchet.WithNamespace("production"),
		hatchet.WithConcurrency(5),
		hatchet.WithRateLimit("openai", 100),
	)

	agent := graft.NewAgent("durable-assistant",
		graft.WithInstructions("You are a helpful assistant running as a durable function."),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "What are the advantages of durable functions?"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
