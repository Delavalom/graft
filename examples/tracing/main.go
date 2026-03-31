package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/tracing"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	// Set up Braintrust as the tracing provider.
	// Traces will appear in your Braintrust dashboard.
	bt := tracing.NewBraintrustProvider(
		os.Getenv("BRAINTRUST_API_KEY"),
		tracing.WithBraintrustProject("my-agent"),
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant."),
	)

	// Wrap the runner with tracing — the underlying runner is unchanged.
	baseRunner := graft.NewDefaultRunner(model)
	runner := tracing.NewTracedRunner(baseRunner, bt,
		tracing.WithCaptureInput(true),
		tracing.WithCaptureOutput(true),
		tracing.WithMetadata(map[string]any{
			"environment": "development",
			"version":     "1.0.0",
		}),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Explain the difference between goroutines and threads"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
