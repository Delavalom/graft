package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/mcp"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	ctx := context.Background()

	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	// Connect to an MCP server via stdio (e.g., a filesystem server).
	// Replace with your own MCP server command.
	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp")
	transport, err := mcp.NewStdioTransport(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	client := mcp.NewClient(transport, mcp.WithServerName("filesystem"))
	defer client.Close()

	// Initialize the MCP connection.
	if _, err := client.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "MCP init failed: %v\n", err)
		os.Exit(1)
	}

	// Convert MCP tools into graft tools.
	// Tools are named mcp__filesystem__<tool_name>.
	tools, err := client.AsTools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get MCP tools: %v\n", err)
		os.Exit(1)
	}

	agent := graft.NewAgent("file-assistant",
		graft.WithInstructions("You are a file system assistant. Use the available tools to help the user with file operations."),
		graft.WithTools(tools...),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(ctx, agent, []graft.Message{
		{Role: graft.RoleUser, Content: "List the files in /tmp"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
