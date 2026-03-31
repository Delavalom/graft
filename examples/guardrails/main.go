package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/guardrail"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	// Output guardrail: ensure the model responds with valid JSON
	// containing required fields "name" and "summary".
	schema := json.RawMessage(`{
		"type": "object",
		"required": ["name", "summary"]
	}`)

	agent := graft.NewAgent("structured-responder",
		graft.WithInstructions("You respond to every question with a JSON object containing 'name' (topic name) and 'summary' (brief answer). Output only valid JSON, no markdown."),
		graft.WithGuardrails(
			// Input guardrail: reject messages exceeding ~1000 tokens.
			guardrail.MaxTokens(1000),
			// Input guardrail: block messages containing profanity patterns.
			guardrail.ContentFilter([]string{`(?i)\b(badword1|badword2)\b`}),
			// Output guardrail: validate the assistant's response is JSON with required fields.
			guardrail.SchemaValidator(schema),
		),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Tell me about Go programming"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
