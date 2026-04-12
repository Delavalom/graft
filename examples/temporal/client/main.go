// Temporal client example.
//
// This program shows how to start a Graft agent as a Temporal workflow and
// wait for its result. It is runnable as-is because it uses an in-process
// demo client that mimics the Temporal SDK surface.
//
// In production, replace demoClient{} with a real *client.Client from
// go.temporal.io/sdk/client. The surrounding code does not change: Graft
// talks to Temporal through the TemporalClient / WorkflowRun interfaces
// defined in the temporal package.
//
// Run:
//
//	go run ./examples/temporal/client
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/temporal"
)

func main() {
	runner := temporal.NewTemporalRunner(demoClient{},
		temporal.WithTaskQueue("ai-agents"),
		temporal.WithRetryPolicy(&temporal.RetryPolicy{
			MaxAttempts:        5,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaxInterval:        30 * time.Second,
		}),
	)

	agent := graft.NewAgent("durable-assistant",
		graft.WithInstructions("You are a helpful assistant running in a durable workflow."),
	)

	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Summarize the benefits of durable execution"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}

// demoClient is an in-process stand-in for go.temporal.io/sdk/client.Client.
// It implements temporal.TemporalClient by logging the dispatch and returning
// a canned WorkflowOutput so the example runs without infrastructure.
//
// Swap for the real SDK in production:
//
//	c, err := client.Dial(client.Options{})
//	if err != nil { ... }
//	runner := temporal.NewTemporalRunner(c, ...)
type demoClient struct{}

func (demoClient) ExecuteWorkflow(ctx context.Context, options temporal.WorkflowOptions, workflow string, args ...any) (temporal.WorkflowRun, error) {
	payload, _ := json.Marshal(args)
	log.Printf("[demo] ExecuteWorkflow queue=%s workflow=%s args=%s", options.TaskQueue, workflow, string(payload))
	return demoRun{id: "wf-demo-1", runID: "run-demo-1"}, nil
}

func (demoClient) QueryWorkflow(ctx context.Context, workflowID, runID, queryType string, args ...any) (temporal.EncodedValue, error) {
	return nil, fmt.Errorf("demoClient: QueryWorkflow not implemented")
}

func (demoClient) SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg any) error {
	log.Printf("[demo] SignalWorkflow wf=%s signal=%s", workflowID, signalName)
	return nil
}

type demoRun struct {
	id    string
	runID string
}

func (r demoRun) GetID() string    { return r.id }
func (r demoRun) GetRunID() string { return r.runID }

func (r demoRun) Get(ctx context.Context, valuePtr any) error {
	out, ok := valuePtr.(*temporal.WorkflowOutput)
	if !ok {
		return fmt.Errorf("demoClient: unexpected valuePtr type %T", valuePtr)
	}
	out.Messages = []graft.Message{
		{
			Role:    graft.RoleAssistant,
			Content: "(demo) Durable workflows survive worker crashes, replay deterministically, and support signals for human-in-the-loop gates.",
		},
	}
	return nil
}
