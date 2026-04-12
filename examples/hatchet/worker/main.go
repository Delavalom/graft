// Hatchet worker example.
//
// Hatchet workers register handlers for tasks and workflows. Graft defines
// the agent-loop DAG shape (see hatchet.NewAgentDAG); you implement the
// three step handlers — llm-generate, tool-execute, loop-check — using
// your real hatchet-go worker.
//
// This file shows the registration pattern. It compiles and runs, but it
// exits immediately because no real worker is attached. Wire the handlers
// to hatchet-go to make it a live worker.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/hatchet"
	"github.com/delavalom/graft/provider/openai"
)

func main() {
	// 1. Define the model used by the llm-generate step.
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4.6"),
	)

	// 2. Define the tools your agent can call.
	greetTool := graft.NewTool("greet", "Greet someone by name.",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return "Hello, " + p.Name + "!", nil
		},
	)
	tools := []graft.Tool{greetTool}

	// 3. Build the standard agent-loop DAG. Graft only defines the shape;
	//    your Hatchet worker registers the three step handlers below.
	dag := hatchet.NewAgentDAG("durable-assistant")
	if err := dag.Validate(); err != nil {
		panic(err)
	}

	// 4. Register the step handlers on your hatchet-go worker.
	//
	//    Pseudocode — adapt to your hatchet-go version:
	//
	//        w := hatchet.NewWorker(cfg)
	//        w.RegisterAction("llm-generate", func(ctx, in TaskInput) (TaskOutput, error) {
	//            return runLLMStep(ctx, model, in)
	//        })
	//        w.RegisterAction("tool-execute", func(ctx, in ToolInput) (ToolOutput, error) {
	//            return runToolStep(ctx, tools, in)
	//        })
	//        w.RegisterAction("loop-check", func(ctx, in LoopInput) (LoopOutput, error) {
	//            return decideContinue(in)
	//        })
	//        w.Start()
	_ = model
	_ = tools

	fmt.Printf("DAG: %s with %d steps — wire handlers to hatchet-go to run for real\n", dag.Name, len(dag.Steps))
}

// runLLMStep would call model.Generate(ctx, params) using the TaskInput
// received from Hatchet and return a TaskOutput containing the new message.
// Shown here as a reference signature you can copy into your worker.
func runLLMStep(ctx context.Context, model graft.LanguageModel, in hatchet.TaskInput) (hatchet.TaskOutput, error) {
	_, _ = model, in
	return hatchet.TaskOutput{}, fmt.Errorf("not implemented — wire to hatchet-go")
}
