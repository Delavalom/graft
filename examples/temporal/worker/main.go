// Temporal worker example.
//
// A Temporal worker registers the AgentWorkflow and its activities. Graft
// provides the workflow + activity implementations in the
// github.com/delavalom/graft/temporal/agentworkflow sub-package; you wire
// them to your real go.temporal.io/sdk/worker.Worker.
//
// This file shows the registration pattern. It compiles and runs, but it
// exits immediately because no real worker is attached. Add the temporal
// SDK and replace the pseudocode to make it a live worker.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	// 1. Configure the model used by GenerateActivity.
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	// 2. Register the tools the agent can call via ToolActivity.
	greetTool := graft.NewTool("greet", "Greet someone by name.",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return "Hello, " + p.Name + "!", nil
		},
	)
	tools := []graft.Tool{greetTool}

	// 3. Register AgentWorkflow and activities on a Temporal worker.
	//
	//    Pseudocode — adapt to your go.temporal.io/sdk version:
	//
	//        import (
	//            "go.temporal.io/sdk/client"
	//            "go.temporal.io/sdk/worker"
	//            "github.com/delavalom/graft/temporal/agentworkflow"
	//        )
	//
	//        c, err := client.Dial(client.Options{})
	//        if err != nil { log.Fatal(err) }
	//        defer c.Close()
	//
	//        models := agentworkflow.NewSingleModelProvider(model)
	//        registry := agentworkflow.NewToolRegistry()
	//        registry.Register(tools...)
	//
	//        w := worker.New(c, "ai-agents", worker.Options{})
	//        agentworkflow.RegisterAgentWorkflow(w)
	//        agentworkflow.RegisterAgentActivities(w, models, registry)
	//
	//        if err := w.Run(worker.InterruptCh()); err != nil {
	//            log.Fatal(err)
	//        }
	_ = model
	_ = tools

	fmt.Println("worker pattern — wire handlers to go.temporal.io/sdk to run for real")
}
