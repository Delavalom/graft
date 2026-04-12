// Hatchet client example.
//
// This program shows how to dispatch a Graft agent run through a Hatchet
// HatchetRunner. It is runnable as-is because it uses an in-process demo
// client that mimics the Hatchet SDK interface.
//
// In production, replace `demoClient{}` with a real *hatchet.Client from
// github.com/hatchet-dev/hatchet-go. The surrounding code does not change:
// Graft talks to Hatchet through the HatchetClient / WorkflowRun interfaces
// defined in the hatchet package.
//
// Run:
//
//	go run ./examples/hatchet/client
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/hatchet"
)

func main() {
	runner := hatchet.NewHatchetRunner(demoClient{},
		hatchet.WithNamespace("graft-demo"),
		hatchet.WithConcurrency(5),
		hatchet.WithRateLimit("openai", 100),
	)

	agent := graft.NewAgent("durable-assistant",
		graft.WithInstructions("You are a helpful assistant running as a durable function."),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "What are the advantages of durable functions?"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}

// demoClient is an in-process stand-in for a real Hatchet client. It
// implements the hatchet.HatchetClient interface by logging the dispatch
// and returning a canned response so the example runs without infrastructure.
//
// Swap this for a real Hatchet client in production:
//
//	c, err := hatchetclient.New()
//	if err != nil { ... }
//	runner := hatchet.NewHatchetRunner(c, ...)
type demoClient struct{}

func (demoClient) RunTask(ctx context.Context, taskName string, input any) (hatchet.TaskRun, error) {
	log.Printf("[demo] RunTask name=%s input=%s", taskName, mustJSON(input))
	return demoRun{id: "task-demo-1"}, nil
}

func (demoClient) RunWorkflow(ctx context.Context, workflowName string, input any) (hatchet.WorkflowRun, error) {
	log.Printf("[demo] RunWorkflow name=%s input=%s", workflowName, mustJSON(input))
	return demoRun{id: "wf-demo-1"}, nil
}

type demoRun struct{ id string }

func (r demoRun) ID() string { return r.id }

func (r demoRun) Wait(ctx context.Context) (any, error) {
	return hatchet.TaskOutput{
		Messages: []graft.Message{
			{
				Role:    graft.RoleAssistant,
				Content: "(demo) Durable functions survive crashes, auto-retry failed steps, and give you per-step observability.",
			},
		},
	}, nil
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return string(b)
}
