package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	codeReviewer := graft.NewAgent("code-reviewer",
		graft.WithInstructions("You are an expert code reviewer. Analyze code for bugs, performance, and style."),
	)

	triage := graft.NewAgent("triage",
		graft.WithInstructions("You are a triage agent. Route code questions to the code reviewer."),
		graft.WithHandoffs(graft.Handoff{
			Target:      codeReviewer,
			Description: "Transfer to code reviewer for code analysis questions",
		}),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), triage, []graft.Message{
		{Role: graft.RoleUser, Content: "Review this Go function: func add(a, b int) int { return a - b }"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
