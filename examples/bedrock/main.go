package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/bedrock"
)

func main() {
	// Uses AWS credentials from environment variables:
	//   AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	//
	// For proxy/service-mesh deployments, use:
	//   bedrock.WithBaseURL("http://127.0.0.1:19193"),
	//   bedrock.WithAnonymousAuth(),
	//   bedrock.WithHeader("X-Project", "my-project"),

	model := bedrock.New(
		bedrock.WithRegion("us-east-1"),
		bedrock.WithModel("anthropic.claude-sonnet-4-20250514-v1:0"),
	)

	greetTool := graft.NewTool("greet", "Greet someone by name",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return fmt.Sprintf("Hello, %s! Welcome to Graft on Bedrock.", p.Name), nil
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
