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
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	greetTool := graft.NewTool("greet", "Greet someone by name",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return fmt.Sprintf("Hello, %s! Welcome to Graft.", p.Name), nil
		},
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant. Use the greet tool when asked to greet someone."),
		graft.WithTools(greetTool),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Please greet Alice"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
