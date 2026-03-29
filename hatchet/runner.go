package hatchet

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/delavalom/graft"
)

// HatchetRunner wraps graft agent execution in Hatchet durable functions.
type HatchetRunner struct {
	client HatchetClient
	cfg    runnerConfig
}

// NewHatchetRunner creates a new Hatchet-backed runner.
func NewHatchetRunner(client HatchetClient, opts ...Option) *HatchetRunner {
	cfg := runnerConfig{
		namespace:   "graft",
		concurrency: 10,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &HatchetRunner{client: client, cfg: cfg}
}

// TaskInput is the serializable input for a Hatchet task.
type TaskInput struct {
	AgentName    string          `json:"agent_name"`
	Instructions string          `json:"instructions"`
	Model        string          `json:"model"`
	ToolNames    []string        `json:"tool_names"`
	Messages     []graft.Message `json:"messages"`
	MaxIterations int            `json:"max_iterations"`
}

// TaskOutput is the serializable output from a Hatchet task.
type TaskOutput struct {
	Messages []graft.Message `json:"messages"`
	Usage    graft.Usage     `json:"usage"`
}

// Run executes the agent loop as a Hatchet workflow.
func (r *HatchetRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	cfg := graft.DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	toolNames := make([]string, len(agent.Tools))
	for i, t := range agent.Tools {
		toolNames[i] = t.Name()
	}

	input := TaskInput{
		AgentName:    agent.Name,
		Instructions: agent.Instructions,
		Model:        agent.Model,
		ToolNames:    toolNames,
		Messages:     messages,
		MaxIterations: cfg.MaxIterations,
	}

	workflowName := fmt.Sprintf("%s/agent-%s", r.cfg.namespace, agent.Name)
	run, err := r.client.RunWorkflow(ctx, workflowName, input)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "hatchet: failed to start workflow", err)
	}

	rawResult, err := run.Wait(ctx)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "hatchet: workflow execution failed", err)
	}

	// Parse the result
	var output TaskOutput
	switch v := rawResult.(type) {
	case TaskOutput:
		output = v
	case *TaskOutput:
		output = *v
	default:
		// Try JSON roundtrip
		data, err := json.Marshal(rawResult)
		if err != nil {
			return nil, graft.NewAgentError(graft.ErrProvider, "hatchet: failed to marshal result", err)
		}
		if err := json.Unmarshal(data, &output); err != nil {
			return nil, graft.NewAgentError(graft.ErrProvider, "hatchet: failed to parse result", err)
		}
	}

	return &graft.Result{
		Messages: output.Messages,
		Usage:    output.Usage,
	}, nil
}

// RunStream starts the workflow and streams events.
func (r *HatchetRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	ch := make(chan graft.StreamEvent, 16)

	go func() {
		defer close(ch)
		result, err := r.Run(ctx, agent, messages, opts...)
		if err != nil {
			ch <- graft.StreamEvent{
				Type:      graft.EventError,
				Data:      err.Error(),
				Timestamp: time.Now(),
			}
			return
		}
		if result != nil && len(result.Messages) > 0 {
			last := result.Messages[len(result.Messages)-1]
			ch <- graft.StreamEvent{
				Type:      graft.EventMessageDone,
				Data:      last.Content,
				Timestamp: time.Now(),
			}
		}
		ch <- graft.StreamEvent{
			Type:      graft.EventDone,
			Timestamp: time.Now(),
		}
	}()
	return ch, nil
}

// Ensure HatchetRunner satisfies graft.Runner.
var _ graft.Runner = (*HatchetRunner)(nil)
