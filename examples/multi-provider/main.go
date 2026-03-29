package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	primary := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	fallback := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("openai/gpt-4o"),
	)

	router := provider.NewRouter(provider.StrategyFallback, primary, fallback)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant."),
	)

	runner := graft.NewDefaultRunner(router)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "What is the capital of France?"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
