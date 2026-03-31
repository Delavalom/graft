package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/mcp"
)

// This example shows how to expose graft tools as an MCP server.
// Any MCP client (Claude Desktop, other agents) can connect and use these tools.
func main() {
	// Define graft tools.
	greetTool := graft.NewTool("greet", "Greet someone by name",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return fmt.Sprintf("Hello, %s!", p.Name), nil
		},
	)

	calcTool := graft.NewTool("add", "Add two numbers",
		func(ctx context.Context, p struct {
			A float64 `json:"a" description:"First number"`
			B float64 `json:"b" description:"Second number"`
		}) (string, error) {
			return fmt.Sprintf("%.2f", p.A+p.B), nil
		},
	)

	// Create an MCP server and register the tools.
	server := mcp.NewServer("graft-tools", "1.0.0")
	server.AddTools([]graft.Tool{greetTool, calcTool})

	// Optionally add resources.
	server.AddResource(mcp.ResourceInfo{
		URI:         "info://version",
		Name:        "Version",
		Description: "Server version info",
	}, "graft-tools v1.0.0")

	// Serve over HTTP — clients connect via POST requests.
	fmt.Println("MCP server listening on :9090")
	if err := http.ListenAndServe(":9090", server.ServeHTTP()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
