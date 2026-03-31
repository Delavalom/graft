package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/state"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant. Remember details from our conversation."),
	)

	// Use a file-based store so sessions survive process restarts.
	// Sessions are saved as JSON files in the given directory.
	store, err := state.NewFileStore("/tmp/graft-sessions")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create store: %v\n", err)
		os.Exit(1)
	}

	// The session ID ties multiple runs together into one conversation.
	sessionID := "user-123-session"

	baseRunner := graft.NewDefaultRunner(model)
	runner := state.NewSessionRunner(baseRunner, store, sessionID)

	// First turn — the session runner saves the conversation automatically.
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "My name is Alice and I work on distributed systems."},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Turn 1:", result.LastAssistantText())

	// Second turn — previous messages are loaded automatically from the store.
	result, err = runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "What's my name and what do I work on?"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Turn 2:", result.LastAssistantText())
}
